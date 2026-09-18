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
	tenantContextKey   contextKey = "tenantRecord"
)

// defaultFallbackTenant returns a minimal Tenant when database lookup fails entirely.
// Name and logo are intentionally generic; actual branding comes from tenant record.
func defaultFallbackTenant() *models.Tenant {
	return &models.Tenant{
		ID:     1,
		Name:   "Store",
		Domain: "localhost",
	}
}

// GinTenantMiddleware resolves the tenant from the incoming request and injects
// it into both the Gin context and the request context.
// Resolution order: X-Tenant-Domain header → ?domain query param → Host header.
// Domain matching is done via a precise tenant_urls JOIN (no LIKE).
func GinTenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		r := c.Request

		domain := r.Header.Get("X-Tenant-Domain")
		if domain == "" {
			domain = c.Query("domain")
		}
		if domain == "" {
			domain = r.Host
		}

		tenant := resolveTenant(domain)

		c.Set("tenant_id", tenant.ID)
		c.Set("tenantRecord", tenant)
		ctx := context.WithValue(r.Context(), tenantIDContextKey, tenant.ID)
		ctx = context.WithValue(ctx, tenantContextKey, tenant)
		c.Request = r.WithContext(ctx)

		c.Next()
	}
}

// TenantMiddleware resolves the tenant from the request and injects it into context.
// Identical resolution logic to GinTenantMiddleware, for use with stdlib http.Handler.
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := r.Header.Get("X-Tenant-Domain")
		if domain == "" {
			domain = r.URL.Query().Get("domain")
		}
		if domain == "" {
			domain = r.Host
		}

		tenant := resolveTenant(domain)

		ctx := context.WithValue(r.Context(), tenantIDContextKey, tenant.ID)
		ctx = context.WithValue(ctx, tenantContextKey, tenant)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// resolveTenant encapsulates shared tenant resolution logic.
// It tries the full domain first, then the host without port.
// Falls back to tenant ID=1 if not found, and to a minimal default if DB fails.
func resolveTenant(domain string) *models.Tenant {
	cleanDomain := domain
	if idx := strings.Index(domain, ":"); idx != -1 {
		cleanDomain = domain[:idx]
	}

	// Try exact match (full domain including port, e.g. "localhost:3000")
	tenant, err := models.GetTenantByDomain(models.DB, domain)
	if err != nil && cleanDomain != domain {
		// Try without port
		tenant, err = models.GetTenantByDomain(models.DB, cleanDomain)
	}

	// GetTenantByDomain already falls back to ID=1 on ErrNoRows;
	// only use the hardcoded fallback if the DB call fails altogether.
	if err != nil || tenant == nil {
		if t, dbErr := models.GetTenantByID(models.DB, 1); dbErr == nil {
			return t
		}
		return defaultFallbackTenant()
	}

	return tenant
}

// TenantIDFromContext retrieves the tenant ID from request context.
func TenantIDFromContext(ctx context.Context) int {
	if val := ctx.Value(tenantIDContextKey); val != nil {
		if id, ok := val.(int); ok {
			return id
		}
	}
	return 1 // default tenant
}

// TenantFromContext retrieves the full tenant record from request context.
func TenantFromContext(ctx context.Context) *models.Tenant {
	if val := ctx.Value(tenantContextKey); val != nil {
		if t, ok := val.(*models.Tenant); ok {
			return t
		}
	}
	return nil
}
