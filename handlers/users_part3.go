package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

func ActivateUserByAdmin(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "users.update")
	if !ok {
		return
	}

	// Extract user ID from path variables
	userID := c.Param("user_id")

	// Activate user in database
	err := models.ActivateUserByID(models.DB, userID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to activate user with ID " + userID,
				Code:        http.StatusInternalServerError,
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
			Description: userWithID + userID + " activated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User activated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeactivateUserByAdmin deactivates a user account.
// This endpoint is restricted to administrators.
//
// @Summary      Deactivate a user by an admin
// @Description  Deactivate a user account by their ID. Requires admin privileges.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        user_id  path      string  true  "User ID"
// @Success      200      {object}  map[string]any "User deactivated successfully"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id}/de-activate [DELETE]
func DeactivateUserByAdmin(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "users.update")
	if !ok {
		return
	}

	// Extract user ID from path variables
	userID := c.Param("user_id")

	// Deactivate user in database
	err := models.DeactivateUserByID(models.DB, userID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to deactivate user with ID " + userID,
				Code:        http.StatusInternalServerError,
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
			Description: userWithID + userID + " deactivated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User deactivated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// GetUserDetails retrieves the profile details of the currently authenticated user.
//
// @Summary      View User Details
// @Description  Get the profile details of the currently authenticated user.
// @Tags         Users
// @Produce      json
// @Success      200  {object}  map[string]any "User details fetched successfully"
// @Failure      401  {object}  map[string]string      "Unauthorized"
// @Failure      500  {object}  map[string]string      "Internal server error"
// @Router       /api/user/me [get]
func GetUserDetails(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(c.Request.Context())
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

	// Fetch user details from database
	user, err := models.GetUserByUserID(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch user with ID " + authUser.ID + " details",
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with user details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + authUser.ID + " details fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   user,
		Message:   "User details",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// CreateAddress adds a new address for the authenticated user.
//
// @Summary      Create user address
// @Description  Add a new address to the user's profile.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        address  body      dtos.UserAdress  true  "Address Details"
// @Success      201      {object}  map[string]any "User address added successfully"
// @Failure      400      {object}  map[string]string      "Invalid request payload"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/user/profile/addresses [post]
func CreateAddress(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context and not authenticated",
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not found in context",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Decode request body
	input, ok := DecodeRequestBody[dtos.UserAdress](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateGinStructAndRespond(input, c, requestSummary, start, "Users") {
		return
	}

	// Create address in database
	err := models.CreateUserAddress(models.DB, *input, authUser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create address for user with ID " + authUser.ID,
				Code:        http.StatusInternalServerError,
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
			Description: "Address for user with ID " + authUser.ID + " added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "User address added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AdminCreateAddress allows an admin to create an address for a specific user.
// This endpoint is restricted to administrators.
//
// @Summary      Create user address (Admin)
// @Description  Create an address for a user as an admin.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        address  body      dtos.AdminUserAddress  true  "Admin User Address Details"
// @Success      201      {object}  map[string]any "User address added successfully"
// @Failure      400      {object}  map[string]string      "Invalid request payload"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/profile/addresses [post]
