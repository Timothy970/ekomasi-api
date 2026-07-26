package handlers

import (
	"bytes"
	"ekomasi_backend/config"
	"ekomasi_backend/utils"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// VerifyWhatsAppWebhook handles Meta Webhook verification GET requests
func VerifyWhatsAppWebhook(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	cfg := config.Get()
	expectedToken := cfg.WhatsAppCloud.WebhookVerifyToken

	if mode == "subscribe" && (expectedToken == "" || token == expectedToken) {
		log.Println("[WhatsApp Webhook] Verification successful")
		c.String(http.StatusOK, challenge)
		return
	}

	log.Printf("[WhatsApp Webhook] Verification failed. Received token: %s", token)
	c.JSON(http.StatusForbidden, gin.H{"error": "Verification failed"})
}

// HandleWhatsAppWebhookEvent receives incoming WhatsApp message status and payload events
func HandleWhatsAppWebhookEvent(c *gin.Context) {
	cfg := config.Get()
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	// Restore body for downstream reading
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	signature := c.GetHeader("X-Hub-Signature-256")
	if !utils.VerifyWhatsAppWebhookSignature(bodyBytes, signature, cfg.WhatsAppCloud.AppSecret) {
		log.Println("[WhatsApp Webhook] Invalid HMAC signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	log.Printf("[WhatsApp Webhook] Event received: %s", string(bodyBytes))
	c.JSON(http.StatusOK, gin.H{"status": "EVENT_RECEIVED"})
}
