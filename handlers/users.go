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
// @Success      201   {object}  map[string]any "User created successfully"
// @Failure      400   {object}  map[string]string      "Invalid request payload or missing mandatory fields"
// @Failure      401   {object}  map[string]string      "Unauthorized"
// @Failure      500   {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users [post]
func AddUser(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "users.create")
	if !ok {
		return
	}

	// Decode the request body into the RegisterRequest struct
	input, ok := DecodeRequestBody[dtos.RegisterRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate mandatory field: RoleID
	if input.RoleID == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Missing mandatory field role ID",
				Code:        http.StatusBadRequest,
			},
			Message:   "Role ID is mandatory",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Validate mandatory fields: Email or Phone number must be present
	if input.Email == "" && input.Phonenumber == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Missing mandatory fields, email or phone number",
				Code:        http.StatusBadRequest,
			},
			Message:   "Missing mandatory fields, email or phone number",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if input.Phonenumber != "" {
		if !utils.IsValidKenyanPhone(input.Phonenumber) {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Invalid phone number format",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid phone number format",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}

	// Create the user in the database
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	user, err := models.CreateUser(models.DB, *input, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create user",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with the created user details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "User created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   user,
		Message:   "User added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200    {object}  map[string]any "Users fetched successfully"
// @Failure      400    {object}  map[string]string      "Invalid request parameters"
// @Failure      401    {object}  map[string]string      "Unauthorized"
// @Failure      500    {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users [GET]
func GetAllUsers(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "")
	if !ok {
		return
	}

	// Parse pagination parameters
	page, limit := parsePagination(c.Query("page"), c.Query("size"))

	// Get filter parameters: query (name, phone, email) and role
	q := c.Query("q")
	role := c.Query("role")
	offset := (page - 1) * limit
	isAdmin := c.Query("isAdmin")

	// Fetch users from database with pagination and filters
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	users, meta, err := models.GetAllUsersWithPagination(models.DB, limit, offset, q, role, isAdmin, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch users: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Construct response with users and pagination metadata
	response := map[string]any{
		"users":      users,
		"pagination": meta,
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Users fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Users",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200      {object}  map[string]any "User fetched successfully"
// @Failure      401      {object}  map[string]string      "Unauthorized"
// @Failure      404      {object}  map[string]string      "User not found"
// @Failure      500      {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users/{user_id} [GET]
func GetUserByID(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "")
	if !ok {
		return
	}

	// Extract user ID from path variables
	userID := c.Param("user_id")

	// Fetch user details from database
	user, err := models.GetUserByUserID(models.DB, userID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch user with ID " + userID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
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
			Description: userWithID + userID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   user,
		Message:   "User fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200   {object}  map[string]any "User updated successfully"
// @Failure      400   {object}  map[string]string      "Invalid request payload or validation error"
// @Failure      401   {object}  map[string]string      "Unauthorized"
// @Failure      404   {object}  map[string]string      "User not found"
// @Failure      500   {object}  map[string]string      "Internal server error"
// @Router       /api/user/me [patch]
