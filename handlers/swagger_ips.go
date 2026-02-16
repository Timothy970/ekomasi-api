package handlers

import (
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// AddAllowedIPHandler adds a new IP address to the allowed list for Swagger access
func AddAllowedIPHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Admin permission check (placeholder, routes will use AuthenticateToken)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "System", "swagger.manage")
	if !ok {
		return
	}

	var req struct {
		IPAddress string `json:"ip_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "Invalid request body",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid request body",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	if req.IPAddress == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "IP address is required",
				Code:        http.StatusBadRequest,
			},
			Message:   "IP address is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	if err := models.AddAllowedIP(models.DB, req.IPAddress); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "Failed to add allowed IP",
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "System",
			Description: "Allowed IP added successfully",
			Code:        http.StatusCreated,
		},
		Message:   "Allowed IP added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// ListAllowedIPsHandler retrieves all allowed IP addresses for Swagger access
func ListAllowedIPsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	ips, err := models.GetAllowedIPs(models.DB)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "Failed to list allowed IPs",
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "System",
			Description: "Allowed IPs retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   ips,
		Message:   "Allowed IPs",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// DeleteAllowedIPHandler removes an IP address from the allowed list
func DeleteAllowedIPHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	ip := mux.Vars(r)["ip"]
	if ip == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "IP address is required",
				Code:        http.StatusBadRequest,
			},
			Message:   "IP address is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	if err := models.DeleteAllowedIP(models.DB, ip); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "Failed to delete allowed IP",
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "System",
			Description: "Allowed IP deleted successfully",
			Code:        http.StatusOK,
		},
		Message:   "Allowed IP deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
