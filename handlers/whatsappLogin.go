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

	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/utils"
)

// noUserFound is the standard error message for missing user accounts
var noUserFound = "User not found"

// w is the Redis key prefix for WhatsApp verification tokens
var w = "whatsapp_verify"

// session is the Redis key prefix for WhatsApp session data
var session = "whatsapp_session"

// WhatsAppLoginHandler initiates passwordless login via WhatsApp.
// Sends a secure verification token to the user's WhatsApp number.
// Creates new user accounts automatically if they don't exist (frictionless onboarding).
//
// @Summary      WhatsApp login
// @Description  Initiate passwordless login by sending verification token via WhatsApp
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        login  body      dtos.WhatsappLogin       true  "Phone number for WhatsApp login"
// @Success      200    {object}  dtos.SuccessResponse     "Verification message sent successfully"
// @Failure      400    {object}  dtos.ErrorResponse       "Invalid request or phone number"
// @Failure      429    {object}  dtos.ErrorResponse       "Too many login attempts"
// @Failure      500    {object}  dtos.ErrorResponse       "Internal server error"
// @Router       /api/auth/whatsapp/login [post]
func WhatsAppLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode and parse JSON request body with phone number
	req, ok := DecodeRequestBody[dtos.WhatsappLogin](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}

	// Validate phone number format and required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Auth") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Check if user has been rate-limited due to too many failed attempts
	if jailed, _ := isUserJailed(req.Phone); jailed {
		// User is temporarily blocked, send 429 Too Many Requests response
		respondTooManyAttempts(w, start, r, requestSummary)
		return
	}

	// Attempt to fetch existing user by phone number
	// user, err := fetchUser("", req.Phone)
	user, err := models.GetUserByPhone(models.DB, req.Phone)
	if err != nil {
		log.Printf("ERR:::::::::::%v", err)
		// User fetch failed, increment failed login attempts
		handleFailedLogin(w, req.Phone, start, r, requestSummary)
		return
	}
	// Prepare user registration data for auto-creation
	var request = dtos.RegisterRequest{
		Phonenumber: req.Phone,
	}
	// If user doesn't exist, create new account automatically (frictionless onboarding)
	if user == nil {
		user, err = models.CreateUser(models.DB, request)
		if err != nil {
			// User creation failed
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Auth",
					Description: "Error creating user",
					Code:        http.StatusInternalServerError,
				},
				Message:   "Error creating user",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}

	// Generate cryptographically secure verification token (32 bytes, base64 encoded)
	verificationToken, err := utils.GenerateSecureTokenBase64(32)
	if err != nil {
		log.Println("Failed to generate verification token:", err)
		// Token generation failed, return 500 error
		respondInternalError(w, "Failed to generate verification token", start, r, requestSummary)
		return
	}

	// Store verification token in Redis with user ID, expires in 5 minutes
	if err := StoreVerificationTokenInRedis(user.ID, verificationToken, 5*time.Minute); err != nil {
		log.Println("Failed to store verification token:", err)
		// Redis storage failed, return 500 error
		respondInternalError(w, "Failed to store verification token", start, r, requestSummary)
		return
	}

	// Create a verification URL with the token

	// Convert phone string to integer for WhatsApp API
	phoneInt, err := strconv.Atoi(user.Phone)
	if err != nil {
		log.Println("Invalid phone number:", err)
	}
	// Send WhatsApp message with verification link to user's phone
	if err := notification.SendWhatsappMessages(phoneInt, verificationToken, "Auth"); err != nil {
		log.Println("Failed to send WhatsApp message:", err)
		// WhatsApp API call failed, return 500 error
		respondInternalError(w, "Failed to send verification message", start, r, requestSummary)
		return
	}

	// Clear any previous failed login attempts counter for this phone
	clearLoginAttempts(req.Phone)

	// Return success response indicating message was sent
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "WhatsApp verification message sent",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "WhatsApp verification message sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// VerifyWhatsAppHandler completes the WhatsApp login flow by verifying the token.
// Users click the verification link sent via WhatsApp, which contains the token.
// Returns authentication token upon successful verification.
//
// @Summary      Verify WhatsApp login
// @Description  Complete WhatsApp login by verifying token and returning auth token
// @Tags         Authentication
// @Produce      json
// @Param        token  query     string                   true  "Verification token from WhatsApp message"
// @Success      200    {object}  dtos.SuccessResponse     "Verification successful with auth token"
// @Failure      400    {object}  dtos.ErrorResponse       "Missing verification token"
// @Failure      401    {object}  dtos.ErrorResponse       "Invalid or expired token"
// @Failure      500    {object}  dtos.ErrorResponse       "Token generation failed"
// @Router       /api/auth/whatsapp/verify [get]
func VerifyWhatsAppHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract verification token from query parameters
	token := r.URL.Query().Get("token")
	if token == "" {
		// Token is missing from request, return 400 Bad Request
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Verification token is required when verifying WhatsApp login",
				Code:        http.StatusBadRequest,
			},
			Message:   "Verification token is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Retrieve user ID from Redis using the token (token is deleted after retrieval)
	userID, err := GetUserIDFromVerificationToken(token)
	if err != nil {
		// Token not found in Redis or expired (5 minute expiration)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Invalid or expired verification token when retrieving user ID",
				Code:        http.StatusUnauthorized,
			},
			Message:   "Invalid or expired verification token",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Fetch complete user record from database using user ID
	user, err := models.GetUserByUserID(models.DB, userID)
	if err != nil {
		// User not found (should not happen if token was valid)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "No user found for the provided verification token",
				Code:        http.StatusUnauthorized,
			},
			Message:   noUserFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Construct user DTO with essential information for token generation
	var User dtos.User
	User.ID = user.ID
	User.FirstName = user.FirstName
	User.LastName = user.LastName
	User.Email = user.Email
	User.Role = user.Role
	User.Phone = user.Phone
	// Generate JWT authentication token valid for 1 hour
	authToken, err := generateToken(&User, "auth", time.Hour)
	if err != nil {
		// JWT token generation failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Failed to generate auth token after WhatsApp verification",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Token generation failed",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Update user's last login timestamp for analytics and security tracking
	err = models.UpdateLastLogin(models.DB, user.ID)
	if err != nil {
		log.Printf("Error updating last login for user %s: %v", user.ID, err)
	}

	// Return success response with JWT token and expiration time (1 hour = 3600 seconds)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "WhatsApp verification successful",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"token":      authToken,
			"expires_in": 3600,
		},
		Message:   "WhatsApp verification successful",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

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
// @Success      200      {object}  dtos.SuccessResponse     "Webhook processed successfully"
// @Failure      400      {object}  dtos.ErrorResponse       "Invalid webhook payload"
// @Router       /api/auth/whatsapp/webhook [post]
func WhatsAppWebhookHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Process webhook payload and handle incoming messages
	if !processWebhookRequest(w, r, requestSummary, start) {
		// Webhook processing failed, error response already sent
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "Webhook processed",
		},
		Payload:   nil,
		Message:   "Webhook processed",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// processWebhookRequest parses and processes WhatsApp webhook payload.
// Iterates through all entries and changes to extract messages.
// Returns false if webhook parsing fails.
func processWebhookRequest(w http.ResponseWriter, r *http.Request, requestSummary string, start time.Time) bool {
	// Decode WhatsApp webhook payload from request body
	webhook, ok := DecodeRequestBody[dtos.WhatsAppWebhook](r, w, requestSummary, start)
	if !ok {
		// Webhook payload parsing failed
		return false
	}

	// Process all webhook entries (may contain multiple events)
	for _, entry := range webhook.Entry {
		// Process all changes within each entry
		for _, change := range entry.Changes {
			// Process message-related changes
			if !processMessageChange(change, r) {
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
	WhatsAppLoginHandler(loginW, loginR)

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
