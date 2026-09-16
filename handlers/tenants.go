package handlers

import (
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetActiveTenantGinHandler resolves and returns the settings for the currently active tenant using Gin Context
func GetActiveTenantGinHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	tenant := middleware.TenantFromContext(c.Request.Context())

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Tenants",
			Description: "Active tenant configuration retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   tenant,
		Message:   "Active tenant details",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// CreateTenantGinHandler creates a new tenant natively using Gin context. Restricted to superadmins.
func CreateTenantGinHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok || authuser.Role != "superadmin" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Unauthorized to create a tenant",
				Code:        http.StatusForbidden,
			},
			Message:   "Only superadmins can manage tenants",
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

	if req.Name == "" || (req.Domain == "" && req.AppDomain == "" && req.AdminDomain == "") {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Name and domain are required",
				Code:        http.StatusBadRequest,
			},
			Message:   "Name and domain are required fields",
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

	err := models.CreateTenant(
		models.DB, req.Name, req.Domain, req.AppDomain, req.AdminDomain, req.Slogan, req.Logo, req.Color,
		req.AppLogo, req.AppPrimaryColor, req.AppSecondaryColor, req.AppTertiaryColor,
		req.AdminLogo, req.AdminPrimaryColor, req.AdminSecondaryColor, req.AdminTertiaryColor,
	)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Failed to create tenant in database",
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
			Description: "Tenant created successfully",
			Code:        http.StatusCreated,
		},
		Message:   "Tenant created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

type TenantRequest struct {
	Name                string `json:"name"`
	Domain              string `json:"domain"`
	AppDomain           string `json:"app_domain"`
	AdminDomain         string `json:"admin_domain"`
	Slogan              string `json:"slogan"`
	Logo                string `json:"logo"`
	Color               string `json:"color"`
	AppLogo             string `json:"app_logo"`
	AppPrimaryColor     string `json:"app_primary_color"`
	AppSecondaryColor   string `json:"app_secondary_color"`
	AppTertiaryColor    string `json:"app_tertiary_color"`
	AdminLogo           string `json:"admin_logo"`
	AdminPrimaryColor   string `json:"admin_primary_color"`
	AdminSecondaryColor string `json:"admin_secondary_color"`
	AdminTertiaryColor  string `json:"admin_tertiary_color"`
}

// GetActiveTenantHandler resolves and returns the settings for the currently active tenant
func GetActiveTenantHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	tenant := middleware.TenantFromContext(c.Request.Context())

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Tenants",
			Description: "Active tenant configuration retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   tenant,
		Message:   "Active tenant details",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// CreateTenantHandler creates a new tenant. Restricted to superadmins.
func CreateTenantHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure the user is authenticated and is a superadmin
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok || authuser.Role != "superadmin" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Unauthorized to create a tenant",
				Code:        http.StatusForbidden,
			},
			Message:   "Only superadmins can manage tenants",
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

	if req.Name == "" || (req.Domain == "" && req.AppDomain == "" && req.AdminDomain == "") {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Name and domain are required",
				Code:        http.StatusBadRequest,
			},
			Message:   "Name and domain are required fields",
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

	err := models.CreateTenant(
		models.DB, req.Name, req.Domain, req.AppDomain, req.AdminDomain, req.Slogan, req.Logo, req.Color,
		req.AppLogo, req.AppPrimaryColor, req.AppSecondaryColor, req.AppTertiaryColor,
		req.AdminLogo, req.AdminPrimaryColor, req.AdminSecondaryColor, req.AdminTertiaryColor,
	)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Tenants",
				Description: "Failed to create tenant in database",
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
			Description: "Tenant created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Tenant created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
