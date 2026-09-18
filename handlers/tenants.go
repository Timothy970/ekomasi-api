package handlers

import (
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TenantRequest is the DTO for creating or updating a tenant.
type TenantRequest struct {
	Name      string `json:"name"`
	Domain    string `json:"domain"`       // canonical slug reference
	Slogan    string `json:"slogan"`
	Logo      string `json:"logo"`
	AppLogo   string `json:"app_logo"`
	AdminLogo string `json:"admin_logo"`
	// Initial URL registrations (optional; managed via tenant_urls endpoints post-create)
	StorefrontURL string `json:"storefront_url"`
	AdminURL      string `json:"admin_url"`
}

// GetActiveTenantGinHandler returns the tenant resolved for the current request.
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

// CreateTenantGinHandler creates a new tenant. Superadmin only.
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

	if req.Name == "" || req.Domain == "" {
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

	storefrontURL := req.StorefrontURL
	if storefrontURL == "" {
		storefrontURL = req.Domain
	}

	err := models.CreateTenant(models.DB, models.CreateTenantParams{
		Name:          req.Name,
		Domain:        req.Domain,
		Slogan:        req.Slogan,
		Logo:          req.Logo,
		AppLogo:       req.AppLogo,
		AdminLogo:     req.AdminLogo,
		StorefrontURL: storefrontURL,
		AdminURL:      req.AdminURL,
	})
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
