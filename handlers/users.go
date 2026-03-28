package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/csv"

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
	if input.Phonenumber != "" {
		if !utils.IsValidKenyanPhone(input.Phonenumber) {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Invalid phone number format",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid phone number format",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
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
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	// Get filter parameters: query (name, phone, email) and role
	q := r.URL.Query().Get("q")
	role := r.URL.Query().Get("role")
	offset := (page - 1) * limit
	isAdmin := r.URL.Query().Get("isAdmin")

	// Fetch users from database with pagination and filters
	users, meta, err := models.GetAllUsersWithPagination(models.DB, limit, offset, q, role, isAdmin)
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

	// Validate mandatory fields: Email or Phone number must be present if provided
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
	// Rate limiting for OTP resend
	identifier := authuser.ID
	isAllowed, retryAfter, err := utils.CheckRateLimit(r.Context(), "resend_otp:"+identifier, 3, 5*time.Minute)
	if err != nil {
		log.Printf("Rate limit error: %v", err)
	}
	if !isAllowed {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "OTP resend requested too soon for user with ID " + authuser.ID,
				Code:        http.StatusTooManyRequests,
			},
			Message:   fmt.Sprintf("Please wait %d seconds before requesting a new OTP", retryAfter),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	currentUser, err := models.GetUserByUserID(models.DB, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch current user details",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Error retrieving user",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	emailChanged := input.Email != "" && input.Email != currentUser.Email
	phoneChanged := input.Phonenumber != "" && input.Phonenumber != currentUser.Phone

	// Validate uniqueness against other users first
	if emailChanged {
		if exists, _ := models.EmailExistsForOtherUser(models.DB, authuser.ID, input.Email); exists {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Email already exists for another user",
					Code:        http.StatusConflict,
				},
				Message:   "Email already exists",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	if phoneChanged {
		if exists, _ := models.PhoneExistsForOtherUser(models.DB, authuser.ID, input.Phonenumber); exists {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Phone number already exists for another user",
					Code:        http.StatusConflict,
				},
				Message:   "Phone number already exists",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}

	// Generate OTP
	// otp, err := utils.GenerateOTP()
	// if err != nil {
	// 	log.Printf("Failed to generate OTP for user update: %v", err)
	// 	return
	// }
	//since we  are testing using a hardcoded OTP, we can skip the generation step and directly use the hardcoded value
	otp := "2025"
	// Store OTP in Redis
	if err := StoreOTPInRedis(authuser.ID, otp, 10*time.Minute); err != nil {
		log.Printf("Failed to store OTP in Redis for user update: %v", err)
		return
	}

	// Store the pending update data in Redis
	tempKey := "pending_user_update:" + authuser.ID
	tempData, _ := json.Marshal(input)
	if err := Redis.Set(context.Background(), tempKey, tempData, 10*time.Minute).Err(); err != nil {
		log.Printf("Failed to store pending user update in Redis: %v", err)
		http.Error(w, "Failed to initiate user update", http.StatusInternalServerError)
		return
	}

	// Prepare a login request for dispatchOTP
	dispatchReq := dtos.LoginRequest{}
	if emailChanged {
		dispatchReq.Email = input.Email
	}
	if phoneChanged {
		dispatchReq.Phone = input.Phonenumber
	}

	// Create a temporary User object for dispatchOTP (using the NEW values)
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

	// Dispatch OTP to the NEW contact info
	//commenting out the actual dispatch for now since we are using a hardcoded OTP for testing. In production, this should be enabled to send real OTPs.
	// dispatchOTP(dispatchUser, otp, dispatchReq)

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Verification required for sensitive field update",
			Code:        http.StatusAccepted,
		},
		Message:   "Verification required. Please enter the OTP sent to your new email/phone to finalize the update.",
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

// DeactivateMyAccount deactivates the authenticated user's account.
//
// @Summary      Deactivate my account
// @Description  Deactivate the account of the currently authenticated user.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      dtos.RegisterRequest  true  "User Deactivation Details"
// @Success      201   {object}  map[string]interface{} "Account deactivated successfully"
// @Failure      400   {object}  map[string]string      "Invalid request payload or missing mandatory fields"
// @Failure      401   {object}  map[string]string      "Unauthorized"
// @Failure      500   {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users [delete]
func DeactivateMyAccount(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
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
	err := models.DeactivateUserByID(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to deactivate user account",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "User deactivated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User deactivated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteUser deletes the currently authenticated user's account.
//
// @Summary      Delete User
// @Description  Delete the account of the currently authenticated user.
// @Tags         Users
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User details fetched successfully"
// @Failure      401  {object}  map[string]string      "Unauthorized"
// @Failure      500  {object}  map[string]string      "Internal server error"
// @Router       /api/user/me [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
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
	err := models.DeleteUserByID(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete user with ID " + authUser.ID,
				Code:        http.StatusInternalServerError,
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
			Description: "User with ID " + authUser.ID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// VerifyUserUpdateHandler finalizes the user profile update after OTP verification.
//
// @Summary      Verify user update
// @Description  Verify the OTP sent to the new email/phone to finalize the profile update.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        otp  body      map[string]string  true  "OTP verification"
// @Success      200  {object}  map[string]interface{} "Profile updated successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      401  {object}  map[string]string      "Unauthorized or invalid OTP"
// @Failure      404  {object}  map[string]string      "Pending update not found"
// @Failure      500  {object}  map[string]string      "Internal server error"
// @Router       /api/user/me/verify-update [patch]
func VerifyUserUpdateHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Get authenticated user from context
	authuser, ok := middleware.UserFromContext(r.Context())
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

	// Decode request body
	req, ok := DecodeRequestBody[dtos.VerifyUserUpdate](r, w, requestSummary, start)
	if !ok {
		return
	}
	// Rate limiting for OTP verification
	identifier := authuser.ID
	isAllowed, retryAfter, err := utils.CheckRateLimit(r.Context(), "verify_user_update:"+identifier, 3, 5*time.Minute)
	if err != nil {
		log.Printf("Rate limit error: %v", err)
	}
	if !isAllowed {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "OTP verification requested too soon for user with ID " + authuser.ID,
				Code:        http.StatusTooManyRequests,
			},
			Message:   fmt.Sprintf("Please wait %d seconds before verifying OTP again", retryAfter),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Verify OTP atomically (checks and deletes in one step to prevent race conditions)
	isValid, err := AtomicVerifyOTP(authuser.ID, req.OTP)
	if err != nil || !isValid {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Invalid or expired OTP for user update",
				Code:        http.StatusUnauthorized,
			},
			Message:   "Invalid or expired OTP",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Retrieve pending update from Redis
	tempKey := "pending_user_update:" + authuser.ID
	val, err := Redis.Get(r.Context(), tempKey).Result()
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Pending update not found or expired",
				Code:        http.StatusNotFound,
			},
			Message:   "Pending update session expired. Please try updating your profile again.",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	var input dtos.RegisterRequest
	if err := json.Unmarshal([]byte(val), &input); err != nil {
		log.Printf("Failed to unmarshal pending user update: %v", err)
		return
	}

	// Apply update in database
	user, err := models.FindByIdAndUpdate(models.DB, input, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to apply profile update after verification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Cleanup Redis (OTP already invalidated by AtomicVerifyOTP)
	Redis.Del(r.Context(), tempKey)

	//clear rate limit for OTP verification
	utils.ClearRateLimit(r.Context(), "verify_user_update:"+identifier)
	// Respond with success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Profile successfully updated after OTP verification",
			Code:        http.StatusOK,
		},
		Payload:   user,
		Message:   "Profile details updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Handler to add a partner
// Summary: Add a partner
// Description: Add a new partner to the system. This endpoint is restricted to administrators.
// Tags: Admin
// Accept: json
// Produce: json
// Param: partner body dtos.Partner true "Partner Details"
// Success: 201 {object} map[string]interface{} "Partner added successfully"
// Failure: 400 {object} map[string]string "Invalid request payload"
// Failure: 401 {object} map[string]string "Unauthorized"
// Failure: 500 {object} map[string]string "Internal server error"
// Router: /api/admin/partners [post]
func AddPartner(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.create")
	if !ok {
		return
	}
	url, err := utils.ParseAndUploadFile(r, "image", 20)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to upload image : " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	req := dtos.Partner{
		Name:  r.FormValue("name"),
		Image: url,
	}
	// Create partner in database
	err = models.CreatePartner(models.DB, req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create partner",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Partner added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Partner added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// Handler to get all partners
// Summary: Get all partners
// Description: Retrieve a list of all partners in the system.
// Tags: Users
// Accept: json
// Produce: json
// Success: 200 {object} map[string]interface{} "Partners fetched successfully"
// Failure: 500 {object} map[string]string "Internal server error"
// Router: /api/partners [get]
func GetAllPartners(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	partners, err := models.GetAllPartners(models.DB)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch partners",
				Code:        http.StatusInternalServerError,
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
			Description: "Partners fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   partners,
		Message:   "Partners fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Handler to delete a partner
// Summary: Delete a partner
// Description: Delete a partner from the system by ID. This endpoint is restricted to administrators.
// Tags: Admin
// Accept: json
// Produce: json
// Param: partner_id path string true "Partner ID"
// Success: 200 {object} map[string]interface{} "Partner deleted successfully"
// Failure: 400 {object} map[string]string "Invalid partner ID"
// Failure: 401 {object} map[string]string "Unauthorized"
// Failure: 500 {object} map[string]string "Internal server error"
// Router: /api/admin/partners/{partner_id} [delete]
func DeletePartner(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	// Check if the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "users.delete")
	if !ok {
		return
	}
	partnerID := mux.Vars(r)["partner_id"]
	err := models.DeletePartnerByID(models.DB, partnerID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete partner with ID " + partnerID,
				Code:        http.StatusInternalServerError,
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
			Description: "Partner with ID " + partnerID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Partner deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func DownloadUsersCSVHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Check if the requesting user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", ""); !ok {
		return
	}

	// Parse filters (ignore pagination for CSV export)
	q := r.URL.Query().Get("q")
	role := r.URL.Query().Get("role")
	isAdmin := r.URL.Query().Get("isAdmin")

	// Fetch users - using a large limit for export
	users, _, err := models.GetAllUsersWithPagination(models.DB, 1000000, 0, q, role, isAdmin)
	if err != nil {
		log.Printf("Error fetching users for CSV: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch users for CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=users.csv")

	// Initialize CSV writer
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header row
	header := []string{"First Name", "Last Name", "Email", "Phone", "Role", "Status", "Date Joined", "Last Login"}
	if err := writer.Write(header); err != nil {
		log.Printf("Error writing CSV header: %v", err)
		return
	}

	// Write data rows
	for _, u := range users {
		row := []string{
			u.FirstName,
			u.LastName,
			u.Email,
			u.Phone,
			u.Role,
			u.Status,
			u.DateJoined,
			u.LastLogin,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Error writing CSV row: %v", err)
			return
		}
	}
}
