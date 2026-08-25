package routes

import (
	"ekomasi_backend/handlers"

	"github.com/gin-gonic/gin"
)

// SetupWhatsAppGinRoutes registers WhatsApp Business Cloud API & WhatsApp Flow routes
func SetupWhatsAppGinRoutes(rg *gin.RouterGroup) {
	wa := rg.Group("/v1/whatsapp")
	{
		// WhatsApp Flow Data Endpoint (Encrypted requests from Meta)
		wa.POST("/flow-endpoint", handlers.WhatsAppFlowDataEndpoint)

		// Upload Public Key to Meta Graph API for Flow Encryption
		wa.POST("/register-public-key", handlers.RegisterWhatsAppPublicKeyHandler)

		// WhatsApp Webhook Handlers
		wa.GET("/webhook", handlers.VerifyWhatsAppWebhook)
		wa.POST("/webhook", handlers.HandleWhatsAppWebhookEvent)
	}
}
