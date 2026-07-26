package middleware

import (
	"context"
	"ekomasi_backend/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	tenantIDContextKey contextKey = "tenantID"
	tenantContextKey   contextKey = "tenant"
)

// GinTenantMiddleware resolves the tenant from the request and injects it into Gin context and r.Context()
func GinTenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		r := c.Request
		// 1. Try X-Tenant-Domain header
		domain := r.Header.Get("X-Tenant-Domain")

		// 2. Try query parameter
		if domain == "" {
			domain = c.Query("domain")
		}

		// 3. Try Host header
		if domain == "" {
			domain = r.Host
		}

		// Strip port if present for comparison
		cleanDomain := domain
		if idx := strings.Index(domain, ":"); idx != -1 {
			cleanDomain = domain[:idx]
		}

		var tenant *models.Tenant
		var err error

		// Try database query with full domain
		tenant, err = models.GetTenantByDomain(models.DB, domain)
		if err != nil && cleanDomain != domain {
			tenant, err = models.GetTenantByDomain(models.DB, cleanDomain)
		}

		// Fallback to default tenant (id = 1) if not found
		if err != nil || tenant == nil {
			tenant, err = models.GetTenantByID(models.DB, 1)
			if err != nil {
				tenant = &models.Tenant{
					ID:     1,
					Name:   "Ekomasi Store",
					Domain: "localhost:3000",
					Logo:   "/images/ekomasi-logo.png",
					Color:  "#4f46e5",
				}
			}
		}

		// Inject into Gin context and Request Context
		c.Set("tenant_id", tenant.ID)
		c.Set("tenant", tenant)
		ctx := context.WithValue(r.Context(), tenantIDContextKey, tenant.ID)
		ctx = context.WithValue(ctx, tenantContextKey, tenant)
		c.Request = r.WithContext(ctx)

		c.Next()
	}
}

// TenantMiddleware resolves the tenant from the request and injects it into context
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Try X-Tenant-Domain header
		domain := r.Header.Get("X-Tenant-Domain")

		// 2. Try query parameter
		if domain == "" {
			domain = r.URL.Query().Get("domain")
		}

		// 3. Try Host header
		if domain == "" {
			domain = r.Host
		}

		// Strip port if present for comparison, but if we have local development with ports,
		// we can try checking both the full host (with port) and without port.
		cleanDomain := domain
		if idx := strings.Index(domain, ":"); idx != -1 {
			cleanDomain = domain[:idx]
		}

		var tenant *models.Tenant
		var err error

		// Try database query with full domain (e.g. "localhost:3000" or "tenant1.ekomasi.com")
		tenant, err = models.GetTenantByDomain(models.DB, domain)
		if err != nil && cleanDomain != domain {
			// If full domain failed and clean domain is different, try clean domain
			tenant, err = models.GetTenantByDomain(models.DB, cleanDomain)
		}

		// If not found or error, fall back to default tenant (id = 1)
		if err != nil || tenant == nil {
			tenant, err = models.GetTenantByID(models.DB, 1)
			if err != nil {
				// Safety fallback if database doesn't even have tenant 1
				tenant = &models.Tenant{
					ID:     1,
					Name:   "Ekomasi Store",
					Domain: "localhost:3000",
					Logo:   "/images/ekomasi-logo.png",
					Color:  "#4f46e5",
				}
			}
		}

		// Inject into context
		ctx := context.WithValue(r.Context(), tenantIDContextKey, tenant.ID)
		ctx = context.WithValue(ctx, tenantContextKey, tenant)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// TenantIDFromContext retrieves the tenant ID from request context
func TenantIDFromContext(ctx context.Context) int {
	if val := ctx.Value(tenantIDContextKey); val != nil {
		if id, ok := val.(int); ok {
			return id
		}
	}
	return 1 // default tenant
}

// TenantFromContext retrieves the tenant metadata from request context
func TenantFromContext(ctx context.Context) *models.Tenant {
	if val := ctx.Value(tenantContextKey); val != nil {
		if t, ok := val.(*models.Tenant); ok {
			return t
		}
	}
	return nil
}
