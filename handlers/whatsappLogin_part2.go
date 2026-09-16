// Package handlers provides HTTP request handlers for WhatsApp-based authentication.
// This file implements passwordless login via WhatsApp, using verification tokens sent
// through WhatsApp messages. Users receive a secure link to verify their identity and
// authenticate without traditional passwords. Essential for mobile-first authentication.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/utils"
)

// StoreVerificationTokenInRedis stores the verification token with user ID in Redis.
// Token is used as a one-time verification link sent via WhatsApp.
// Key format: "whatsapp_verify:{token}" -> userID
func StoreVerificationTokenInRedis(userID, token string, expiration time.Duration) error {
	ctx := context.Background()
	// Store token with user ID, auto-expires after specified duration (typically 5 minutes)
	err := Redis.Set(ctx, fmt.Sprintf("%s:%s", w, token), userID, expiration).Err()
	return err
}

// GetUserIDFromVerificationToken retrieves the user ID for a verification token.
// Token is consumed (deleted) after retrieval to prevent reuse.
// Returns error if token is invalid, expired, or already used.
func GetUserIDFromVerificationToken(token string) (string, error) {
	ctx := context.Background()
	// Retrieve user ID from Redis using token
	userID, err := Redis.Get(ctx, fmt.Sprintf("%s:%s", w, token)).Result()
	if err != nil {
		// Token not found (expired, invalid, or already used)
		return "", err
	}

	// Delete token immediately after use to prevent replay attacks
	Redis.Del(ctx, fmt.Sprintf("%s:%s", w, token))

	return userID, nil
}

// WhatsAppWebhookHandler receives webhook notifications from WhatsApp Business API.
// Processes incoming messages and interactive events from WhatsApp.
// Used for handling user-initiated login requests from WhatsApp chat.
//
// @Summary      WhatsApp webhook
// @Description  Receive and process WhatsApp Business API webhook notifications
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        webhook  body      dtos.WhatsAppWebhook     true  "WhatsApp webhook payload"
// @Success      200      {object}  map[string]interface{}     "Webhook processed successfully"
// @Failure      400      {object}  dtos.ErrorResponse       "Invalid webhook payload"
// @Router       /api/auth/whatsapp/webhook [post]
func WhatsAppWebhookHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Process webhook payload and handle incoming messages
	if !processWebhookRequest(c, requestSummary, start) {
		// Webhook processing failed, error response already sent
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "Webhook processed",
		},
		Payload:   nil,
		Message:   "Webhook processed",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// processWebhookRequest parses and processes WhatsApp webhook payload.
// Iterates through all entries and changes to extract messages.
// Returns false if webhook parsing fails.
func processWebhookRequest(c *gin.Context, requestSummary string, start time.Time) bool {
	// Decode WhatsApp webhook payload from request body
	webhook, ok := DecodeRequestBody[dtos.WhatsAppWebhook](c, requestSummary, start)
	if !ok {
		// Webhook payload parsing failed
		return false
	}

	// Process all webhook entries (may contain multiple events)
	for _, entry := range webhook.Entry {
		// Process all changes within each entry
		for _, change := range entry.Changes {
			// Process message-related changes
			if !processMessageChange(change, c.Request) {
				// Skip non-message changes
				continue
			}
		}
	}
	return true
}

// processMessageChange handles message-related webhook changes.
// Filters for "messages" field type and processes text messages.
// Returns false if change is not message-related.
func processMessageChange(change dtos.Change, r *http.Request) bool {
	// Only process message-related changes
	if change.Field != "messages" {
		return false
	}

	// Process all messages in this change
	for _, msg := range change.Value.Messages {
		// Process text messages only (skip images, videos, etc.)
		if !processTextMessage(msg, r) {
			// Skip non-text messages
			continue
		}
	}
	return true
}

// processTextMessage handles text messages from WhatsApp users.
// Checks for login-related messages and triggers login flow.
// Returns false if message is not a text message.
func processTextMessage(msg dtos.Message, r *http.Request) bool {
	// Only process text messages
	if msg.Type != "text" {
		return false
	}

	log.Printf("Received message from %s: %s", msg.From, msg.Text.Body)

	// Check if message is a login request
	if strings.HasPrefix(msg.Text.Body, "Login to MyApp") {
		// Initiate login flow for the sender
		handleLoginRequest(msg.From, r)
	}
	return true
}

// handleLoginRequest processes login requests initiated from WhatsApp chat.
// Extracts phone number and triggers WhatsAppLoginHandler internally.
func handleLoginRequest(sender string, r *http.Request) {
	// Extract phone number from WhatsApp sender ID
	phone := strings.TrimPrefix(sender, "whatsapp:")
	loginReq := dtos.WhatsappLogin{Phone: phone}

	// Create HTTP request for internal login handler call
	loginR, err := createLoginRequest(loginReq, r)
	if err != nil {
		log.Printf("Error creating login request: %v", err)
		return
	}

	// Call WhatsAppLoginHandler internally to send verification token
	loginW := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(loginW)
	c.Request = loginR
	WhatsAppLoginHandler(c)

	if loginW.Code != http.StatusOK {
		log.Printf("Login handler returned non-OK status: %d", loginW.Code)
	}
}

// createLoginRequest constructs an HTTP request for internal login handler call.
// Marshals login data to JSON and creates POST request with proper headers.
func createLoginRequest(loginReq dtos.WhatsappLogin, r *http.Request) (*http.Request, error) {
	// Convert login request to JSON
	jsonBody, err := json.Marshal(loginReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling login request: %w", err)
	}

	// Create POST request for internal handler
	loginR, err := http.NewRequest("POST", "/whatsapp/login", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set request body and headers
	loginR.Body = io.NopCloser(bytes.NewReader(jsonBody))
	loginR.Header.Set("Content-Type", "application/json")
	// Preserve original request context (tracing, timeout, etc.)
	return loginR.WithContext(r.Context()), nil
}

// GenerateWhatsAppDeeplink creates a URL to open WhatsApp with predefined message.
// Generates "wa.me" link that opens WhatsApp chat with pre-filled message.
// Used for creating "Login with WhatsApp" buttons that launch WhatsApp app.
func GenerateWhatsAppDeeplink(message string) string {
	// Get WhatsApp business number from environment variables
	sender, err := strconv.Atoi(os.Getenv("WHATSAPPSENDER"))
	if err != nil {
		log.Printf("invalid sender: %v", err)
	}

	// URL-encode the message text
	encodedMsg := url.QueryEscape(message)
	// Create WhatsApp deep link (wa.me format)
	return fmt.Sprintf("https://wa.me/%d?text=%s", sender, encodedMsg)
}

// StoreWhatsAppSessionData stores the login session data in Redis.
// Maintains session state during multi-step WhatsApp authentication flow.
// Key format: "whatsapp_session:{sessionID}" -> {user_id, token}
func StoreWhatsAppSessionData(sessionID, userID, token string) error {
	ctx := context.Background()
	// Construct session data with user ID and token
	data := map[string]string{
		"user_id": userID,
		"token":   token,
	}
	// Serialize to JSON for Redis storage
	jsonData, _ := json.Marshal(data)

	// Store session data with 10 minute expiration
	return Redis.Set(ctx,
		fmt.Sprintf("%s:%s", session, sessionID),
		jsonData,
		10*time.Minute).Err()
}

// GetWhatsAppSessionData retrieves session data from Redis.
// Returns user ID and token associated with the session.
// Returns error if session expired or doesn't exist.
func GetWhatsAppSessionData(sessionID string) (userID, token string, err error) {
	ctx := context.Background()
	// Retrieve session data from Redis
	data, err := Redis.Get(ctx, fmt.Sprintf("%s:%s", session, sessionID)).Result()
	if err != nil {
		// Session not found or expired
		return "", "", err
	}

	// Deserialize JSON data
	var sessionData map[string]string
	if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
		return "", "", err
	}

	return sessionData["user_id"], sessionData["token"], nil
}

// DeleteWhatsAppSessionData cleans up session data after verification.
// Removes session from Redis to prevent reuse and free memory.
func DeleteWhatsAppSessionData(sessionID string) error {
	ctx := context.Background()
	// Delete session key from Redis
	return Redis.Del(ctx, fmt.Sprintf("%s:%s", session, sessionID)).Err()
}
