package routes

import (
	"github.com/gin-gonic/gin"
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"
)

// SetupPaymentGinRoutes configures all payment-related routes using native Gin router groups
func SetupPaymentGinRoutes(api *gin.RouterGroup) {
	payment := api.Group("/payment")

	// M-Pesa payment routes
	payment.POST("/callback", handlers.HandleMpesaCallback)
	payment.GET("/balance", handlers.HandleMpesaBalance)
	payment.GET("/return", handlers.HandleMpesaMoneyReturn)
	payment.POST("/return/callback", handlers.HandleMpesaReturnCallback)
	payment.POST("/balance/callback", handlers.HandleMpesaBalanceCallback)
	payment.POST("/pay", middleware.GinRateLimiter(3, 60, "stkpush"), handlers.HandleMpesaPayment)
	payment.POST("/mpesa/register-url", handlers.RegisterMpesaRoutesHandler)

	// Card Payment Gateway routes (Paystack, Flutterwave, Stripe)
	payment.GET("/gateways", handlers.GetPaymentGatewayConfigsHandler)
	payment.POST("/gateways/config", middleware.GinAuthenticateToken(), handlers.ConfigurePaymentGatewayHandler)
	payment.POST("/card/initialize", handlers.InitializeCardPaymentHandler)
	payment.GET("/card/verify", handlers.VerifyCardPaymentHandler)

	// Customer Wallet & Store Credit routes
	wallet := api.Group("/wallet")
	wallet.GET("", handlers.GetWalletBalanceHandler)
	wallet.POST("/deposit", handlers.TopupWalletMpesaHandler)
	wallet.POST("/pay", handlers.PayWithWalletOrSplitHandler)
	wallet.GET("/transactions", handlers.GetWalletBalanceHandler)

	// Admin Wallet & Store Credit Management
	adminWallet := api.Group("/admin/wallet")
	adminWallet.POST("/store-credit/issue", handlers.TopupWalletMpesaHandler)
}
