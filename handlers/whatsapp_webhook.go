package handlers

import (
	"bytes"
	"ekomasi_backend/config"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type MetaWebhookPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					From string `json:"from"`
					ID   string `json:"id"`
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

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

	// Parse incoming message event asynchronously
	var payload MetaWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err == nil {
		for _, entry := range payload.Entry {
			for _, change := range entry.Changes {
				for _, msg := range change.Value.Messages {
					if msg.From != "" {
						fromPhone := msg.From
						flowID := os.Getenv("WHATSAPP_FLOW_ID")
						log.Printf("[WhatsApp Webhook] Received message '%s' from %s", msg.Text.Body, fromPhone)

						if flowID != "" {
							go func(phone string, fid string) {
								err := notification.SendWhatsAppFlowMessage(phone, fid, phone, "Open Store & Menu", "SIGN_IN")
								if err != nil {
									log.Printf("[WhatsApp Webhook] Error sending flow message: %v", err)
								}
							}(fromPhone, flowID)
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "EVENT_RECEIVED"})
}

// RegisterWhatsAppPublicKeyHandler reads public.pem and uploads it to Meta Graph API
func RegisterWhatsAppPublicKeyHandler(c *gin.Context) {
	cfg := config.Get()
	phoneID := cfg.WhatsAppCloud.PhoneNumberID
	token := cfg.WhatsAppCloud.AccessToken

	if phoneID == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "WHATSAPP_PHONE_NUMBER_ID or WHATSAPP_ACCESS_TOKEN is missing in config"})
		return
	}

	pubKeyBytes, err := os.ReadFile("public.pem")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read public.pem: %v", err)})
		return
	}

	metaURL := fmt.Sprintf("https://graph.facebook.com/v20.0/%s/whatsapp_business_encryption", phoneID)
	data := url.Values{}
	data.Set("business_public_key", string(pubKeyBytes))

	req, err := http.NewRequest("POST", metaURL, strings.NewReader(data.Encode()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create Meta request: %v", err)})
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send request to Meta: %v", err)})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from Meta"})
		return
	}

	var jsonResult map[string]interface{}
	if err := json.Unmarshal(respBody, &jsonResult); err != nil {
		c.JSON(resp.StatusCode, gin.H{"raw_response": string(respBody)})
		return
	}

	c.JSON(resp.StatusCode, jsonResult)
}

