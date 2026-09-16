package routes

import (
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupGinRoutes configures all application routes using the Gin framework (*gin.Engine)
func SetupGinRoutes(router *gin.Engine) {
	// Enable CORS & Tenant Resolution middleware globally on the Gin Engine
	router.Use(middleware.GinTenantMiddleware())

	// Create main /api Group
	api := router.Group("/api")

	// Public tenant resolution route
	api.GET("/tenant/active", handlers.GetActiveTenantHandler)

	// Setup all route groups
	SetupAuthGinRoutes(api)
	SetupHomeGinRoutes(api)
	SetupUserGinRoutes(api)
	SetupWishlistGinRoutes(api)
	SetupProductGinRoutes(api)
	SetupCartGinRoutes(api)
	SetupOrderGinRoutes(api)
	SetupPaymentGinRoutes(api)
	SetupAdminGinRoutes(api)
	SetupReportsGinRoutes(api)
	SetupMiscGinRoutes(api)
	SetupWhatsAppGinRoutes(api)
	SetupWalletGinRoutes(api)
}
