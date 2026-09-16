package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

func UpdateUser(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context or not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Decode request body
	input, ok := DecodeRequestBody[dtos.RegisterRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate mandatory fields and phone number format
	if !validateUpdateInput(c, input, start, requestSummary) {
		return
	}

	// Rate limiting for OTP resend
	if !checkUserUpdateRateLimit(c, authuser.ID, start, requestSummary) {
		return
	}

	currentUser, err := models.GetUserByUserID(models.DB, authuser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch current user details",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Error retrieving user",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Validate uniqueness against other users
	emailChanged, phoneChanged, valid := validateUserUpdateUniqueness(c, authuser.ID, input, currentUser, start, requestSummary)
	if !valid {
		return
	}

	// Generate and store OTP in Redis
	otp := "2025"
	if err := StoreOTPInRedis(authuser.ID, otp, 10*time.Minute); err != nil {
		log.Printf("Failed to store OTP in Redis for user update: %v", err)
		return
	}

	// Store the pending update data in Redis
	tempKey := "pending_user_update:" + authuser.ID
	tempData, _ := json.Marshal(input)
	if err := Redis.Set(context.Background(), tempKey, tempData, 10*time.Minute).Err(); err != nil {
		log.Printf("Failed to store pending user update in Redis: %v", err)
		c.String(http.StatusInternalServerError, "Failed to initiate user update")
		return
	}

	dispatchUserUpdateOTP(currentUser, input, emailChanged, phoneChanged, otp)

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Verification required for sensitive field update",
			Code:        http.StatusAccepted,
		},
		Message:   "Verification required. Please enter the OTP sent to your new email/phone to finalize the update.",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func validateUpdateInput(c *gin.Context, input *dtos.RegisterRequest, start time.Time, requestSummary string) bool {
	if input.Email == "" && input.Phonenumber == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Missing mandatory fields email or phone number",
				Code:        http.StatusBadRequest,
			},
			Message:   mandatory,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return false
	}

	if input.Phonenumber != "" && !utils.IsValidKenyanPhone(input.Phonenumber) {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Invalid phone number format",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid phone number",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return false
	}

	return true
}

func checkUserUpdateRateLimit(c *gin.Context, userID string, start time.Time, requestSummary string) bool {
	isAllowed, retryAfter, err := utils.CheckRateLimit(c.Request.Context(), "resend_otp:"+userID, 3, 5*time.Minute)
	if err != nil {
		log.Printf("Rate limit error: %v", err)
	}
	if !isAllowed {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "OTP resend requested too soon for user with ID " + userID,
				Code:        http.StatusTooManyRequests,
			},
			Message:   fmt.Sprintf("Please wait %d seconds before requesting a new OTP", retryAfter),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return false
	}
	return true
}

func validateUserUpdateUniqueness(c *gin.Context, userID string, input *dtos.RegisterRequest, currentUser *dtos.Users, start time.Time, requestSummary string) (emailChanged bool, phoneChanged bool, valid bool) {
	emailChanged = input.Email != "" && input.Email != currentUser.Email
	phoneChanged = input.Phonenumber != "" && input.Phonenumber != currentUser.Phone

	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	if emailChanged {
		if exists, _ := models.EmailExistsForOtherUser(models.DB, userID, input.Email, tenantID); exists {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Email already exists for another user",
					Code:        http.StatusConflict,
				},
				Message:   "Email already exists",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return emailChanged, phoneChanged, false
		}
	}
	if phoneChanged {
		if exists, _ := models.PhoneExistsForOtherUser(models.DB, userID, input.Phonenumber, tenantID); exists {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Phone number already exists for another user",
					Code:        http.StatusConflict,
				},
				Message:   "Phone number already exists",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return emailChanged, phoneChanged, false
		}
	}
	return emailChanged, phoneChanged, true
}

func dispatchUserUpdateOTP(currentUser *dtos.Users, input *dtos.RegisterRequest, emailChanged, phoneChanged bool, otp string) {
	dispatchReq := dtos.LoginRequest{}
	if emailChanged {
		dispatchReq.Email = input.Email
	}
	if phoneChanged {
		dispatchReq.Phone = input.Phonenumber
	}

	dispatchUser := &dtos.User{
		Email: input.Email,
		Phone: input.Phonenumber,
	}
	if dispatchUser.Email == "" {
		dispatchUser.Email = currentUser.Email
	}
	if dispatchUser.Phone == "" {
		dispatchUser.Phone = currentUser.Phone
	}

	dispatchOTP(dispatchUser, otp, dispatchReq)
}

// DeleteUserByAdmin deletes a user by their ID.
// This endpoint is restricted to administrators.
//
// @Summary      Delete a user by an admin
// @Description  Permanently remove a user from the system by their ID. Requires admin privileges.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        user_id  path      string  true  "User ID"
// @Success      200      {object}  map[string]any "User deleted successfully"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id} [DELETE]
func DeleteUserByAdmin(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "users.delete")
	if !ok {
		return
	}

	// Extract user ID from path variables
	userID := c.Param("user_id")

	// Delete user from database
	err := models.DeleteUserByID(models.DB, userID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete user with ID " + userID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + userID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ActivateUserByAdmin activates a user account.
// This endpoint is restricted to administrators.
//
// @Summary      Activate a user by an admin
// @Description  Activate a user account by their ID. Requires admin privileges.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        user_id  path      string  true  "User ID"
// @Success      200      {object}  map[string]any "User activated successfully"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id}/activate [PATCH]
