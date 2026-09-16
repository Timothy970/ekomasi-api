package handlers

import (
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

func VerifyUserUpdateHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not authenticated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Decode request body
	req, ok := DecodeRequestBody[dtos.VerifyUserUpdate](c, requestSummary, start)
	if !ok {
		return
	}
	// Rate limiting for OTP verification
	identifier := authuser.ID
	isAllowed, retryAfter, err := utils.CheckRateLimit(c.Request.Context(), "verify_user_update:"+identifier, 3, 5*time.Minute)
	if err != nil {
		log.Printf("Rate limit error: %v", err)
	}
	if !isAllowed {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "OTP verification requested too soon for user with ID " + authuser.ID,
				Code:        http.StatusTooManyRequests,
			},
			Message:   fmt.Sprintf("Please wait %d seconds before verifying OTP again", retryAfter),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	// Verify OTP atomically (checks and deletes in one step to prevent race conditions)
	isValid, err := AtomicVerifyOTP(authuser.ID, req.OTP)
	if err != nil || !isValid {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Invalid or expired OTP for user update",
				Code:        http.StatusUnauthorized,
			},
			Message:   "Invalid or expired OTP",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Retrieve pending update from Redis
	tempKey := "pending_user_update:" + authuser.ID
	val, err := Redis.Get(c.Request.Context(), tempKey).Result()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Pending update not found or expired",
				Code:        http.StatusNotFound,
			},
			Message:   "Pending update session expired. Please try updating your profile again.",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	var input dtos.RegisterRequest
	if err := json.Unmarshal([]byte(val), &input); err != nil {
		log.Printf("Failed to unmarshal pending user update: %v", err)
		return
	}

	// Apply update in database
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	user, err := models.FindByIdAndUpdate(models.DB, input, authuser.ID, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to apply profile update after verification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Cleanup Redis (OTP already invalidated by AtomicVerifyOTP)
	Redis.Del(c.Request.Context(), tempKey)

	//clear rate limit for OTP verification
	utils.ClearRateLimit(c.Request.Context(), "verify_user_update:"+identifier)
	// Respond with success
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Profile successfully updated after OTP verification",
			Code:        http.StatusOK,
		},
		Payload:   user,
		Message:   "Profile details updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// Handler to add a partner
// Summary: Add a partner
// Description: Add a new partner to the system. This endpoint is restricted to administrators.
// Tags: Admin
// Accept: json
// Produce: json
// Param: partner body dtos.Partner true "Partner Details"
// Success: 201 {object} map[string]any "Partner added successfully"
// Failure: 400 {object} map[string]string "Invalid request payload"
// Failure: 401 {object} map[string]string "Unauthorized"
// Failure: 500 {object} map[string]string "Internal server error"
// Router: /api/admin/partners [post]
func AddPartner(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "users.create")
	if !ok {
		return
	}
	url, err := utils.ParseAndUploadFile(c.Request, "image", 20)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to upload image : " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}
	req := dtos.Partner{
		Name:  c.Request.FormValue("name"),
		Image: url,
	}
	// Create partner in database
	err = models.CreatePartner(models.DB, req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create partner",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}
	// Respond with success message
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Partner added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Partner added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// Handler to get all partners
// Summary: Get all partners
// Description: Retrieve a list of all partners in the system.
// Tags: Users
// Accept: json
// Produce: json
// Success: 200 {object} map[string]any "Partners fetched successfully"
// Failure: 500 {object} map[string]string "Internal server error"
// Router: /api/partners [get]
func GetAllPartners(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	partners, err := models.GetAllPartners(models.DB)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch partners",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Partners fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   partners,
		Message:   "Partners fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// Handler to delete a partner
// Summary: Delete a partner
// Description: Delete a partner from the system by ID. This endpoint is restricted to administrators.
// Tags: Admin
// Accept: json
// Produce: json
// Param: partner_id path string true "Partner ID"
// Success: 200 {object} map[string]any "Partner deleted successfully"
// Failure: 400 {object} map[string]string "Invalid partner ID"
// Failure: 401 {object} map[string]string "Unauthorized"
// Failure: 500 {object} map[string]string "Internal server error"
// Router: /api/admin/partners/{partner_id} [delete]
func DeletePartner(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "users.delete")
	if !ok {
		return
	}
	partnerID := c.Param("partner_id")
	err := models.DeletePartnerByID(models.DB, partnerID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete partner with ID " + partnerID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Partner with ID " + partnerID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Partner deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DownloadUsersCSVHandler downloads the user list as a CSV.
//
// @Summary      Download users CSV
// @Description  Download a CSV file containing all users or filtered results by search, role, or isAdmin
// @Tags         Admin
// @Produce      text/csv
// @Param        q        query     string  false  "Search query (name, phone, email)"
// @Param        role     query     string  false  "Filter by role"
// @Param        isAdmin  query     string  false  "Filter by admin status"
// @Success      200      {file}    file
// @Security     BearerAuth
// @Router       /api/admin/users/csv [get]
