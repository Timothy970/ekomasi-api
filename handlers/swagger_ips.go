package handlers

import (
	"github.com/gin-gonic/gin"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"
)

// AddAllowedIPHandler adds a new IP address to the allowed list for Swagger access
func AddAllowedIPHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Admin permission check (placeholder, routes will use AuthenticateToken)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "System", "swagger.manage")
	if !ok {
		return
	}

	var req struct {
		IPAddress string `json:"ip_address"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "Invalid request body",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid request body",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	if req.IPAddress == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "IP address is required",
				Code:        http.StatusBadRequest,
			},
			Message:   "IP address is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	if err := models.AddAllowedIP(models.DB, req.IPAddress); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "Failed to add allowed IP",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "System",
			Description: "Allowed IP added successfully",
			Code:        http.StatusCreated,
		},
		Message:   "Allowed IP added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary,
	})
}

// ListAllowedIPsHandler retrieves all allowed IP addresses for Swagger access
func ListAllowedIPsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	ips, err := models.GetAllowedIPs(models.DB)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "Failed to list allowed IPs",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "System",
			Description: "Allowed IPs retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   ips,
		Message:   "Allowed IPs",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary,
	})
}

// DeleteAllowedIPHandler removes an IP address from the allowed list
func DeleteAllowedIPHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	ip := c.Param("ip")
	if ip == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "IP address is required",
				Code:        http.StatusBadRequest,
			},
			Message:   "IP address is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	if err := models.DeleteAllowedIP(models.DB, ip); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "System",
				Description: "Failed to delete allowed IP",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "System",
			Description: "Allowed IP deleted successfully",
			Code:        http.StatusOK,
		},
		Message:   "Allowed IP deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary,
	})
}
