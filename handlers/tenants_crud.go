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

// UpdateTenantHandler updates tenant settings (name, slogan, logos).
// Accessible by superadmins and tenant admins.
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

	err = models.UpdateTenant(
		models.DB, tenantID,
		req.Name, req.Slogan, req.Logo, req.AppLogo, req.AdminLogo,
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

// GetAllTenantsHandler lists all tenants. Superadmin only.
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

// ─── Tenant URL Management Handlers ─────────────────────────────────────────

// TenantURLRequest is the DTO for adding or updating a tenant URL.
type TenantURLRequest struct {
	URL       string `json:"url" binding:"required"`
	URLType   string `json:"url_type"` // "storefront" | "admin"
	IsPrimary bool   `json:"is_primary"`
}

// GetTenantURLsGinHandler lists all URLs registered for a tenant. Superadmin only.
func GetTenantURLsGinHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok || (authuser.Role != "superadmin" && authuser.Role != "admin") {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Unauthorized",
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
				Module:      "TenantURLs",
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

	urls, err := models.GetTenantURLs(models.DB, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Failed to fetch tenant URLs",
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
			Module:      "TenantURLs",
			Description: "Tenant URLs retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   urls,
		Message:   "Tenant URLs",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// AddTenantURLGinHandler registers a new URL for a tenant.
func AddTenantURLGinHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok || (authuser.Role != "superadmin" && authuser.Role != "admin") {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Unauthorized",
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
				Module:      "TenantURLs",
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

	var req TenantURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Invalid request body",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	if req.URLType == "" {
		req.URLType = "storefront"
	}

	if err := models.AddTenantURL(models.DB, tenantID, req.URL, req.URLType, req.IsPrimary); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Failed to add tenant URL",
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
			Module:      "TenantURLs",
			Description: "Tenant URL added successfully",
			Code:        http.StatusCreated,
		},
		Message:   "URL added to tenant",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// DeleteTenantURLGinHandler removes a URL from a tenant by URL record ID.
func DeleteTenantURLGinHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok || (authuser.Role != "superadmin" && authuser.Role != "admin") {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Unauthorized",
				Code:        http.StatusForbidden,
			},
			Message:   "Access denied",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	urlIDStr := c.Param("url_id")
	urlID, err := strconv.Atoi(urlIDStr)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Invalid URL ID",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid URL ID",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	if err := models.DeleteTenantURL(models.DB, urlID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Failed to delete tenant URL",
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
			Module:      "TenantURLs",
			Description: "Tenant URL removed successfully",
			Code:        http.StatusOK,
		},
		Message:   "URL removed from tenant",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// SetTenantURLPrimaryGinHandler sets a specific URL as primary for its url_type.
func SetTenantURLPrimaryGinHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok || (authuser.Role != "superadmin" && authuser.Role != "admin") {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Unauthorized",
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
				Module:      "TenantURLs",
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

	urlIDStr := c.Param("url_id")
	urlID, err := strconv.Atoi(urlIDStr)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Invalid URL ID",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid URL ID",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	var req TenantURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Invalid request body",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	if req.URLType == "" {
		req.URLType = "storefront"
	}

	if err := models.SetTenantURLPrimary(models.DB, urlID, tenantID, req.URLType); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "TenantURLs",
				Description: "Failed to update primary URL",
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
			Module:      "TenantURLs",
			Description: "Primary URL updated successfully",
			Code:        http.StatusOK,
		},
		Message:   "Primary URL updated",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
