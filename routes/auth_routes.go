package routes

import (
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupAuthGinRoutes configures all authentication-related routes using native Gin router groups
func SetupAuthGinRoutes(api *gin.RouterGroup) {
	auth := api.Group("/auth")

	// User authentication with Redis rate limiting on sensitive endpoints
	auth.POST("/signup", handlers.RegisterHandler)
	auth.POST("/whatsapp/signup", handlers.WhatsAppLoginHandler)
	auth.POST("/whatsappwehbook/signup", handlers.WhatsAppWebhookHandler)
	api.GET("/verify-whatsapp", handlers.VerifyWhatsAppHandler)
	auth.POST("/signin", middleware.GinRateLimiter(5, 60, "login"), handlers.LoginHandler)
	auth.POST("/admin/signin", middleware.GinRateLimiter(5, 60, "admin_login"), handlers.AdminLoginHandler)

	// OTP management with Redis rate limiting
	auth.POST("/resend-otp", middleware.GinRateLimiter(3, 60, "otp_resend"), handlers.ResendOptHandler)
	auth.POST("/verify-otp", handlers.VerifySignupOTPHandler)

	// Token management
	auth.POST("/refresh-token", middleware.GinAuthenticateToken(), handlers.RefreshTokenHandler)
	auth.GET("/decode-token", handlers.DecodeTokenHandler)
	auth.POST("/logout", middleware.GinAuthenticateToken(), handlers.LogoutHandler)
}
