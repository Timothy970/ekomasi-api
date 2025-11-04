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

// AddUser for Admins only
//
// @Summary      Create a user
// @Description  Create a user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User created successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/admin/users [post]
func AddUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users")
	if !ok {
		return
	}
	input, ok := DecodeRequestBody[dtos.RegisterRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
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
	user, err := models.CreateUser(*input)
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
	//send email to the created user
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

// GetAllUsers handles GET /users?page=1&limit=10
//
// @Summary      List all Users
// @Description  List all users
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "Users fetched successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/admin/users [GET]
func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users")
	if !ok {
		return
	}
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
	// support query by user name,phone,email
	q := r.URL.Query().Get("q")
	role := r.URL.Query().Get("role")
	offset := (page - 1) * limit
	users, meta, err := models.GetAllUsersWithPagination(limit, offset, q, role)
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

// @Summary      Get user by id
// @Description  Det user by id
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User fetched successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/admin/users/{user_id} [GET]
func GetUserByID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users")
	if !ok {
		return
	}
	userID := mux.Vars(r)["user_id"]
	user, err := models.GetUserByUserID(userID)
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

// UpdateUser
// @Summary      Update a user
// @Description  Update user details by ID
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User updated successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/user/{id} [put]
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
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
	input, ok := DecodeRequestBody[dtos.RegisterRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
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
	if !utils.ValidateStructAndRespond(input, w, r, requestSummary, start, "Users") {
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
				Message:   "Invalid phone number",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	user, err := models.FindByIdAndUpdate(*input, authuser.ID)
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

// DeleteUser handles DELETE api/user/{id}
// @Summary      Delete a user by an admin
// @Description  Delete user details by ID
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User deleted successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/admin/users/{user_id} [DELETE]
func DeleteUserByAdmin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users")
	if !ok {
		return
	}
	userID := mux.Vars(r)["user_id"]
	err := models.DeleteUserByID(userID)
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

// Activate user
// @Summary      Activate a user by an admin
// @Description  Activate user details by ID
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User activated successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/admin/users/{user_id}/activate [PATCH]
func ActivateUserByAdmin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users")
	if !ok {
		return
	}
	userID := mux.Vars(r)["user_id"]
	err := models.ActivateUserByID(userID)
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

// Deactivate user
// @Summary      Deactivate a user by an admin
// @Description  Deactivate user details by ID
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User activated successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/admin/users/{user_id}/de-activate [DELETE]
func DeactivateUserByAdmin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users")
	if !ok {
		return
	}
	userID := mux.Vars(r)["user_id"]
	err := models.DeactivateUserByID(userID)
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

// GetUserDetails handles GET api/users/me
// @Summary View User Details
// @Description Get user details
// @Tags Users
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/user/me [get]
func GetUserDetails(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
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
	user, err := models.GetUserByUserID(authUser.ID)

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

// Create user Address
//
// @Summary      Create user address
// @Description  Create User Address
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User address added successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/user/profile/addresses [post]
func CreateAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
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
	input, ok := DecodeRequestBody[dtos.UserAdress](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(input, w, r, requestSummary, start, "Users") {
		return
	}
	err := models.CreateUserAddress(*input, authUser.ID)
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
	//send email to the created user
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

// Create user Address
//
// @Summary      Get user address
// @Description  Get User Address
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User address fetched successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/user/profile/addresses [get]
func GetUserAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
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

	addresses, err := models.GetUserAddresses(authUser.ID)
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
	//send email to the created user
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

// Update user address
//
// @Summary      Update user address
// @Description  Update User Address
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User address updated successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/user/profile/addresses [get]
func UpdateAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
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
	addressID := mux.Vars(r)["address_id"]
	input, ok := DecodeRequestBody[dtos.UserAdress](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(input, w, r, requestSummary, start, "Users") {
		return
	}
	err := models.UpdateUserAddress(addressID, authUser.ID, input)
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
	//send email to the created user
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

// delete user address
//
// @Summary      Delete user address
// @Description  Delete User Address
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User address deleted successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/user/profile/addresses [delete]
func DeleteAddress(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
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
	addressID := mux.Vars(r)["address_id"]
	err := models.DeleteUserAddress(addressID, authUser.ID)
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
	//send email to the created user
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

// Update user Admin
// UpdateUser
// @Summary      Update a user by an admin
// @Description  Update user details by ID
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "User updated successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      "User not found"
// @Router       /api/admin/users/{user_id} [PATCH]
func UpdateUserByAdmin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users")
	if !ok {
		return
	}
	userID := mux.Vars(r)["user_id"]
	input, ok := DecodeRequestBody[dtos.RegisterRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
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
	user, err := models.FindByIdAndUpdate(*input, userID)
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

//get recommended user products based on purchase history and wishlist

func GetUserBasedProductsRecommendations(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
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
	// get categories from purchase history & wishlist
	purchaseCats, err := models.GetPurchasedCategories(authUser.ID)
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

	wishlistCats, err := models.GetWishlistCategories(authUser.ID)
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

	// merge categories
	categories := append(purchaseCats, wishlistCats...)

	// fetch recommended products
	recommended, pagination, err := models.GetProductsByCategories(categories, page, limit)
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

// Add subcribers
//
// @Summary      Create a subscriber
// @Description  Create a subscriber
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{} "subscriber created successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      404  {object}  map[string]string      ""
// @Router       /api/admin/users [post]
func AddSubscriber(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	input, ok := DecodeRequestBody[dtos.Subscriber](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(input, w, r, requestSummary, start, "Users") {
		return
	}
	err := models.CreateSubscribers(input.Email)
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
	//send email to the created user
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
