package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

var (
	mandatory  = "Include mandatory fields"
	userWithID = "User with ID "
)

// AddUser creates a new user in the system.
// This endpoint is restricted to administrators.
//
// @Summary      Create a user
// @Description  Create a new user with the provided details. Requires admin privileges.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      dtos.RegisterRequest  true  "User Registration Details"
// @Success      201   {object}  map[string]interface{} "User created successfully"
// @Failure      400   {object}  map[string]string      "Invalid request payload or missing mandatory fields"
// @Failure      401   {object}  map[string]string      "Unauthorized"
// @Failure      500   {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users [post]
func AddUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.create")
	if !ok {
		return
	}

	// Decode the request body into the RegisterRequest struct
	input, ok := DecodeRequestBody[dtos.RegisterRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate mandatory field: RoleID
	if input.RoleID == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Missing mandatory field role ID",
				Code:        http.StatusBadRequest,
			},
			Message:   "Role ID is mandatory",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Validate mandatory fields: Email or Phone number must be present
	if input.Email == "" && input.Phonenumber == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Missing mandatory fields, email or phone number",
				Code:        http.StatusBadRequest,
			},
			Message:   "Missing mandatory fields, email or phone number",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Create the user in the database
	user, err := models.CreateUser(models.DB, *input)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create user",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with the created user details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "User created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   user,
		Message:   "User added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetAllUsers retrieves a paginated list of all users.
// This endpoint is restricted to administrators.
//
// @Summary      List all Users
// @Description  Retrieve a list of users with pagination and optional filtering. Requires admin privileges.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        page   query     int     false  "Page number (default 1)"
// @Param        limit  query     int     false  "Number of items per page (default 10)"
// @Param        q      query     string  false  "Search query (name, phone, email)"
// @Param        role   query     string  false  "Filter by role"
// @Success      200    {object}  map[string]interface{} "Users fetched successfully"
// @Failure      400    {object}  map[string]string      "Invalid request parameters"
// @Failure      401    {object}  map[string]string      "Unauthorized"
// @Failure      500    {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users [GET]
func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "")
	if !ok {
		return
	}

	// Parse pagination parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	// Get filter parameters: query (name, phone, email) and role
	q := r.URL.Query().Get("q")
	role := r.URL.Query().Get("role")
	offset := (page - 1) * limit

	// Fetch users from database with pagination and filters
	users, meta, err := models.GetAllUsersWithPagination(models.DB, limit, offset, q, role)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch users: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Construct response with users and pagination metadata
	response := map[string]interface{}{
		"users":      users,
		"pagination": meta,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Users fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Users",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetUserByID retrieves a specific user by their ID.
// This endpoint is restricted to administrators.
//
// @Summary      Get user by id
// @Description  Retrieve details of a specific user by their unique ID. Requires admin privileges.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user_id  path      string  true  "User ID"
// @Success      200      {object}  map[string]interface{} "User fetched successfully"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id} [GET]
func GetUserByID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "")
	if !ok {
		return
	}

	// Extract user ID from path variables
	userID := mux.Vars(r)["user_id"]

	// Fetch user details from database
	user, err := models.GetUserByUserID(models.DB, userID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch user with ID " + userID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with user details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + userID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   user,
		Message:   "User fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateUser updates the authenticated user's profile details.
//
// @Summary      Update a user
// @Description  Update the profile details of the currently authenticated user.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      dtos.RegisterRequest  true  "User Update Details"
// @Success      200   {object}  map[string]interface{} "User updated successfully"
// @Failure      400   {object}  map[string]string      "Invalid request payload or validation error"
// @Failure      401   {object}  map[string]string      "Unauthorized"
// @Failure      404   {object}  map[string]string      "User not found"
// @Failure      500   {object}  map[string]string      "Internal server error"
// @Router       /api/user/me [patch]
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Get authenticated user from context
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context or not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Decode request body
	input, ok := DecodeRequestBody[dtos.RegisterRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate mandatory fields: Email or Phone number must be present if provided (though this logic seems to imply at least one must be present if updating?)
	// Actually, this check implies that if both are empty, it's an error.
	if input.Email == "" && input.Phonenumber == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Missing mandatory fields email or phone number",
				Code:        http.StatusBadRequest,
			},
			Message:   mandatory,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Validate Kenyan phone number format if provided
	if input.Phonenumber != "" {
		if !utils.IsValidKenyanPhone(input.Phonenumber) {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Invalid phone number format",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid phone number",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}

	// Update user in database
	user, err := models.FindByIdAndUpdate(models.DB, *input, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update user with ID " + authuser.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with updated user details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + authuser.ID + " details updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   user,
		Message:   "User details updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
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
// @Success      200      {object}  map[string]interface{} "User deleted successfully"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id} [DELETE]
func DeleteUserByAdmin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.delete")
	if !ok {
		return
	}

	// Extract user ID from path variables
	userID := mux.Vars(r)["user_id"]

	// Delete user from database
	err := models.DeleteUserByID(models.DB, userID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete user with ID " + userID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + userID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200      {object}  map[string]interface{} "User activated successfully"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id}/activate [PATCH]
func ActivateUserByAdmin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.update")
	if !ok {
		return
	}

	// Extract user ID from path variables
	userID := mux.Vars(r)["user_id"]

	// Activate user in database
	err := models.ActivateUserByID(models.DB, userID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to activate user with ID " + userID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + userID + " activated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User activated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200      {object}  map[string]interface{} "User deactivated successfully"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id}/de-activate [DELETE]
func DeactivateUserByAdmin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.update")
	if !ok {
		return
	}

	// Extract user ID from path variables
	userID := mux.Vars(r)["user_id"]

	// Deactivate user in database
	err := models.DeactivateUserByID(models.DB, userID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to deactivate user with ID " + userID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + userID + " deactivated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User deactivated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// GetUserDetails retrieves the profile details of the currently authenticated user.
//
// @Summary      View User Details
// @Description  Get the profile details of the currently authenticated user.
// @Tags         Users
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User details fetched successfully"
// @Failure      401  {object}  map[string]string      "Unauthorized"
// @Failure      500  {object}  map[string]string      "Internal server error"
// @Router       /api/user/me [get]
func GetUserDetails(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not authenticated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch user details from database
	user, err := models.GetUserByUserID(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch user with ID " + authUser.ID + " details",
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with user details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + authUser.ID + " details fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   user,
		Message:   "User details",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      201      {object}  map[string]interface{} "User address added successfully"
// @Failure      400      {object}  map[string]string      "Invalid request payload"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/user/profile/addresses [post]
func CreateAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context and not authenticated",
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not found in context",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Decode request body
	input, ok := DecodeRequestBody[dtos.UserAdress](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateStructAndRespond(input, w, r, requestSummary, start, "Users") {
		return
	}

	// Create address in database
	err := models.CreateUserAddress(models.DB, *input, authUser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create address for user with ID " + authUser.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Address for user with ID " + authUser.ID + " added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "User address added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      201      {object}  map[string]interface{} "User address added successfully"
// @Failure      400      {object}  map[string]string      "Invalid request payload"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/profile/addresses [post]
func AdminCreateAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.create")
	if !ok {
		return
	}

	// Decode request body
	req, ok := DecodeRequestBody[dtos.AdminUserAddress](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
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
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create address for user with ID " + req.UserID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Address for user with ID " + req.UserID + " added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "User address added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetUserAddress retrieves all addresses associated with the authenticated user.
//
// @Summary      Get user addresses
// @Description  Retrieve a list of addresses for the currently authenticated user.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User addresses fetched successfully"
// @Failure      401  {object}  map[string]string      "Unauthorized"
// @Failure      500  {object}  map[string]string      "Internal server error"
// @Router       /api/user/profile/addresses [get]
func GetUserAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context and not authenticated",
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not found in context",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch addresses from database
	addresses, err := models.GetUserAddresses(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch addresses for user with ID " + authUser.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with list of addresses
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Addresses for user with ID " + authUser.ID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   addresses,
		Message:   "User address fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200         {object}  map[string]interface{} "User address updated successfully"
// @Failure      400         {object}  map[string]string      "Invalid request payload"
// @Failure      401         {object}  map[string]string      "Unauthorized"
// @Failure      500         {object}  map[string]string      "Internal server error"
// @Router       /api/user/profile/addresses/{address_id} [patch]
func UpdateAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Extract address ID from path variables
	addressID := mux.Vars(r)["address_id"]

	// Decode request body
	input, ok := DecodeRequestBody[dtos.UserAdress](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateStructAndRespond(input, w, r, requestSummary, start, "Users") {
		return
	}

	// Update address in database
	err := models.UpdateUserAddress(models.DB, addressID, authUser.ID, input)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update address for user with ID " + authUser.ID,
				Code:        http.StatusBadGateway,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + authUser.ID + " address updated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "User address updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func AdminUpdateAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.update")
	addressID := mux.Vars(r)["address_id"]
	req, ok := DecodeRequestBody[dtos.AdminUserAddress](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
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
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update address for user with ID " + req.UserID,
				Code:        http.StatusBadGateway,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + req.UserID + " address updated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "User address updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200         {object}  map[string]interface{} "User address deleted successfully"
// @Failure      401         {object}  map[string]string      "Unauthorized"
// @Failure      500         {object}  map[string]string      "Internal server error"
// @Router       /api/user/profile/addresses/{address_id} [delete]
func DeleteAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context and not validated",
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Extract address ID from path variables
	addressID := mux.Vars(r)["address_id"]

	// Delete address from database
	err := models.DeleteUserAddress(models.DB, addressID, authUser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete address for user with ID " + authUser.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + authUser.ID + " address deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User address deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200         {object}  map[string]interface{} "User address deleted successfully"
// @Failure      400         {object}  map[string]string      "Invalid request payload"
// @Failure      401         {object}  map[string]string      "Unauthorized"
// @Failure      500         {object}  map[string]string      "Internal server error"
// @Router       /api/admin/profile/addresses/{address_id} [delete]
func AdminDeleteAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.delete")
	if !ok {
		return
	}

	// Extract address ID from path variables
	addressID := mux.Vars(r)["address_id"]

	// Decode request body
	req, ok := DecodeRequestBody[dtos.AdminDeleteUserAddress](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}

	// Delete address from database
	err := models.DeleteUserAddress(models.DB, addressID, req.UserID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete address for user with ID " + req.UserID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + req.UserID + " address deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User address deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200      {object}  map[string]interface{} "User updated successfully"
// @Failure      400      {object}  map[string]string      "Invalid request payload"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id} [PATCH]
func UpdateUserByAdmin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.update")
	if !ok {
		return
	}

	// Extract user ID from path variables
	userID := mux.Vars(r)["user_id"]

	// Decode request body
	input, ok := DecodeRequestBody[dtos.RegisterRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate mandatory fields
	if (input.Email == "" || input.Phonenumber == "") && input.RoleID == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Missing mandatory fields email, phone number or role ID",
				Code:        http.StatusBadRequest,
			},
			Message:   mandatory,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Update user in database
	user, err := models.FindByIdAndUpdate(models.DB, *input, userID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update user with ID " + userID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with updated user details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: userWithID + userID + " details updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   user,
		Message:   "User details updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200    {object}  map[string]interface{} "Recommended products fetched successfully"
// @Failure      401    {object}  map[string]string      "Unauthorized"
// @Failure      404    {object}  map[string]string      "No recommendations found"
// @Failure      500    {object}  map[string]string      "Internal server error"
// @Router       /api/user/products/recommendations [get]
func GetUserBasedProductsRecommendations(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context and not validated",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Parse pagination params
	page := 1
	limit := 10
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("size")
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}

	// Get categories from purchase history
	purchaseCats, err := models.GetPurchasedCategories(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch purchased categories for user with ID " + authUser.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Get categories from wishlist
	wishlistCats, err := models.GetWishlistCategories(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch wishlist categories for user with ID " + authUser.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Merge categories
	categories := append(purchaseCats, wishlistCats...)

	// Fetch recommended products based on merged categories
	recommended, pagination, err := models.GetProductsByCategories(models.DB, categories, page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch recommended products",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with recommended products
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Recommended products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"products":   recommended,
			"pagination": pagination,
		},
		Message:   "Recommended products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// AddSubscriber adds a new subscriber to the mailing list.
//
// @Summary      Create a subscriber
// @Description  Add a new email to the subscriber list.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        subscriber  body      dtos.Subscriber  true  "Subscriber Details"
// @Success      201         {object}  map[string]interface{} "Subscriber created successfully"
// @Failure      400         {object}  map[string]string      "Invalid request payload"
// @Failure      404         {object}  map[string]string      "Resource not found"
// @Failure      500         {object}  map[string]string      "Internal server error"
// @Router       /api/subscribe [post]
func AddSubscriber(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	input, ok := DecodeRequestBody[dtos.Subscriber](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateStructAndRespond(input, w, r, requestSummary, start, "Users") {
		return
	}

	// Create subscriber in database
	err := models.CreateSubscribers(models.DB, input.Email)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create subscriber",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Subscriber added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Subscriber added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
