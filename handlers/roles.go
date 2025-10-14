package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// CreateRoleHandler handles POST /api/roles
func CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.RoleRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	err := models.CreateRole(req.Name, req.Description)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Role created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// GetRolesHandler handles GET /api/roles?name=admin&start_date=2025-01-01&end_date=2025-12-31
func GetRolesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	name := r.URL.Query().Get("name")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Validate date format if provided
	if startDate != "" && endDate != "" {
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusBadRequest,
				Message:   "Invalid start_date format. Use YYYY-MM-DD",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusBadRequest,
				Message:   "Invalid end_date format. Use YYYY-MM-DD",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}

	roles, err := models.GetRoles(name, startDate, endDate)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   roles,
		Message:   "Role fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateRoleHandler handles PUT /api/roles/{name}
func UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.RoleRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	params := mux.Vars(r)
	roleID := params["role_id"]

	if err := models.UpdateRole(req.Name, req.Description, roleID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Role updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// DeleteRoleHandler handles DELETE /api/roles/{name}
func DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	params := mux.Vars(r)
	roleID := params["role_id"]

	if err := models.DeleteRole(roleID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Role deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
