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

func AdminCreateAddress(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "users.create")
	if !ok {
		return
	}

	// Decode request body
	req, ok := DecodeRequestBody[dtos.AdminUserAddress](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}

	// Map AdminUserAddress to UserAdress struct
	input := dtos.UserAdress{
		Address:   req.Address,
		Country:   req.Country,
		Apartment: req.Apartment,
		City:      req.City,
		ZipCode:   req.ZipCode,
	}

	// Create address in database
	err := models.CreateUserAddress(models.DB, input, req.UserID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create address for user with ID " + req.UserID,
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
			Description: "Address for user with ID " + req.UserID + " added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "User address added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetUserAddress retrieves all addresses associated with the authenticated user.
//
// @Summary      Get user addresses
// @Description  Retrieve a list of addresses for the currently authenticated user.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]any "User addresses fetched successfully"
// @Failure      401  {object}  map[string]string      "Unauthorized"
// @Failure      500  {object}  map[string]string      "Internal server error"
// @Router       /api/user/profile/addresses [get]
func GetUserAddress(c *gin.Context) {
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

	// Fetch addresses from database
	addresses, err := models.GetUserAddresses(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch addresses for user with ID " + authUser.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with list of addresses
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Addresses for user with ID " + authUser.ID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   addresses,
		Message:   "User address fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// UpdateAddress updates an existing address for the authenticated user.
//
// @Summary      Update user address
// @Description  Update details of an existing address for the user.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        address_id  path      string           true  "Address ID"
// @Param        address     body      dtos.UserAdress  true  "Updated Address Details"
// @Success      200         {object}  map[string]any "User address updated successfully"
// @Failure      400         {object}  map[string]string      "Invalid request payload"
// @Failure      401         {object}  map[string]string      "Unauthorized"
// @Failure      500         {object}  map[string]string      "Internal server error"
// @Router       /api/user/profile/addresses/{address_id} [patch]
func UpdateAddress(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Extract address ID from path variables
	addressID := c.Param("address_id")

	// Decode request body
	input, ok := DecodeRequestBody[dtos.UserAdress](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateGinStructAndRespond(input, c, requestSummary, start, "Users") {
		return
	}

	// Update address in database
	err := models.UpdateUserAddress(models.DB, addressID, authUser.ID, input)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update address for user with ID " + authUser.ID,
				Code:        http.StatusBadGateway,
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
			Description: userWithID + authUser.ID + " address updated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "User address updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func AdminUpdateAddress(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "users.update")
	addressID := c.Param("address_id")
	req, ok := DecodeRequestBody[dtos.AdminUserAddress](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}
	input := dtos.UserAdress{
		Address:   req.Address,
		Country:   req.Country,
		Apartment: req.Apartment,
		City:      req.City,
		ZipCode:   req.ZipCode,
	}
	err := models.UpdateUserAddress(models.DB, addressID, req.UserID, &input)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update address for user with ID " + req.UserID,
				Code:        http.StatusBadGateway,
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
			Description: userWithID + req.UserID + " address updated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "User address updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteAddress deletes an address for the authenticated user.
//
// @Summary      Delete user address
// @Description  Delete an address from the user's profile.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        address_id  path      string  true  "Address ID"
// @Success      200         {object}  map[string]any "User address deleted successfully"
// @Failure      401         {object}  map[string]string      "Unauthorized"
// @Failure      500         {object}  map[string]string      "Internal server error"
// @Router       /api/user/profile/addresses/{address_id} [delete]
