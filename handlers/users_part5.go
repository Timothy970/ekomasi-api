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

func DeleteAddress(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context and not validated",
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Extract address ID from path variables
	addressID := c.Param("address_id")

	// Delete address from database
	err := models.DeleteUserAddress(models.DB, addressID, authUser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete address for user with ID " + authUser.ID,
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
			Description: userWithID + authUser.ID + " address deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User address deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AdminDeleteAddress allows an admin to delete an address for a specific user.
// This endpoint is restricted to administrators.
//
// @Summary      Delete user address (Admin)
// @Description  Delete an address for a user as an admin.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        address_id  path      string                  true  "Address ID"
// @Param        user        body      dtos.AdminDeleteUserAddress true "User ID Details"
// @Success      200         {object}  map[string]any "User address deleted successfully"
// @Failure      400         {object}  map[string]string      "Invalid request payload"
// @Failure      401         {object}  map[string]string      "Unauthorized"
// @Failure      500         {object}  map[string]string      "Internal server error"
// @Router       /api/admin/profile/addresses/{address_id} [delete]
func AdminDeleteAddress(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "users.delete")
	if !ok {
		return
	}

	// Extract address ID from path variables
	addressID := c.Param("address_id")

	// Decode request body
	req, ok := DecodeRequestBody[dtos.AdminDeleteUserAddress](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}

	// Delete address from database
	err := models.DeleteUserAddress(models.DB, addressID, req.UserID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete address for user with ID " + req.UserID,
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
			Description: userWithID + req.UserID + " address deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User address deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// UpdateUserByAdmin updates a user's details by an admin.
// This endpoint is restricted to administrators.
//
// @Summary      Update a user by an admin
// @Description  Update details of a user by their ID. Requires admin privileges.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        user_id  path      string                true  "User ID"
// @Param        user     body      dtos.RegisterRequest  true  "User Update Details"
// @Success      200      {object}  map[string]any "User updated successfully"
// @Failure      400      {object}  map[string]string      "Invalid request payload"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id} [PATCH]
func UpdateUserByAdmin(c *gin.Context) {
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

	// Decode request body
	input, ok := DecodeRequestBody[dtos.RegisterRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate mandatory fields
	if (input.Email == "" || input.Phonenumber == "") && input.RoleID == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Missing mandatory fields email, phone number or role ID",
				Code:        http.StatusBadRequest,
			},
			Message:   mandatory,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Update user in database
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	user, err := models.FindByIdAndUpdate(models.DB, *input, userID, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update user with ID " + userID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with updated user details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + userID + " details updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   user,
		Message:   "User details updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetUserBasedProductsRecommendations retrieves product recommendations for the authenticated user.
// Recommendations are based on purchase history and wishlist.
//
// @Summary      Get recommended products
// @Description  Retrieve product recommendations for the authenticated user based on their purchase history and wishlist.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        page   query     int     false  "Page number (default 1)"
// @Param        size   query     int     false  "Number of items per page (default 10)"
// @Success      200    {object}  map[string]any "Recommended products fetched successfully"
// @Failure      401    {object}  map[string]string      "Unauthorized"
// @Failure      404    {object}  map[string]string      "No recommendations found"
// @Failure      500    {object}  map[string]string      "Internal server error"
// @Router       /api/user/products/recommendations [get]
