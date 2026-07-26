package routes

import (
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupWalletGinRoutes registers Wallet & Store Credit API endpoints
func SetupWalletGinRoutes(rg *gin.RouterGroup) {
	wallet := rg.Group("/v1/wallet")
	{
		wallet.GET("/balance", handlers.GetWalletBalanceHandler)
		wallet.POST("/topup", middleware.RateLimiterMiddleware(3, 1*time.Minute), handlers.TopupWalletMpesaHandler)
		wallet.POST("/pay-split", handlers.PayWithWalletOrSplitHandler)
	}

	// Faceted Search Endpoint
	rg.GET("/v1/products/search", handlers.SearchProductsFacetedHandler)
}
