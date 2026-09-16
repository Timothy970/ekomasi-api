package handlers

import (
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// UpdateTenantHandler updates tenant settings. Accessible by superadmins and tenant admins.
func UpdateTenantHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Unauthenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not authenticated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	tenantIDStr := c.Param("id")
	tenantID, err := strconv.Atoi(tenantIDStr)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Invalid tenant ID format",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid tenant ID",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	if authuser.Role != "superadmin" && authuser.Role != "admin" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Access denied to update tenant",
				Code:        http.StatusForbidden,
			},
			Message:   "Access denied",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	var req TenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Failed to decode tenant request",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	if req.Domain == "" {
		if req.AppDomain != "" {
			req.Domain = req.AppDomain
		} else {
			req.Domain = req.AdminDomain
		}
	}

	err = models.UpdateTenant(
		models.DB, tenantID, req.Name, req.Domain, req.AppDomain, req.AdminDomain, req.Slogan, req.Logo, req.Color,
		req.AppLogo, req.AppPrimaryColor, req.AppSecondaryColor, req.AppTertiaryColor,
		req.AdminLogo, req.AdminPrimaryColor, req.AdminSecondaryColor, req.AdminTertiaryColor,
	)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Failed to update tenant in database",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Tenants",
			Description: "Tenant updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Tenant updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetAllTenantsHandler lists all tenants in the system. Superadmin only.
func GetAllTenantsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok || authuser.Role != "superadmin" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Unauthorized to list tenants",
				Code:        http.StatusForbidden,
			},
			Message:   "Only superadmins can manage tenants",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	tenants, err := models.GetAllTenants(models.DB)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Failed to query tenants from DB",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Tenants",
			Description: "Tenants list retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   tenants,
		Message:   "All tenants",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetTenantByIDHandler gets tenant details by ID. Superadmin only.
func GetTenantByIDHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok || authuser.Role != "superadmin" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Unauthorized to get tenant details",
				Code:        http.StatusForbidden,
			},
			Message:   "Access denied",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	tenantIDStr := c.Param("id")
	tenantID, err := strconv.Atoi(tenantIDStr)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Invalid tenant ID",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid tenant ID",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	tenant, err := models.GetTenantByID(models.DB, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: fmt.Sprintf("Failed to find tenant with ID %d", tenantID),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Tenants",
			Description: fmt.Sprintf("Tenant details for ID %d retrieved successfully", tenantID),
			Code:        http.StatusOK,
		},
		Payload:   tenant,
		Message:   "Tenant details",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// DeleteTenantHandler deletes a tenant. Superadmin only.
func DeleteTenantHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok || authuser.Role != "superadmin" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Unauthorized to delete tenant",
				Code:        http.StatusForbidden,
			},
			Message:   "Only superadmins can manage tenants",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	tenantIDStr := c.Param("id")
	tenantID, err := strconv.Atoi(tenantIDStr)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Invalid tenant ID",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid tenant ID",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	if tenantID == 1 {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Cannot delete the default tenant",
				Code:        http.StatusBadRequest,
			},
			Message:   "Cannot delete the default tenant",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	err = models.DeleteTenant(models.DB, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: fmt.Sprintf("Failed to delete tenant with ID %d", tenantID),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Tenants",
			Description: "Tenant deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Tenant deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
