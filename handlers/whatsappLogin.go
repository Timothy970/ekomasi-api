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

var noUserFound = "User not found"
var w = "whatsapp_verify"
var session = "whatsapp_session"

func WhatsAppLoginHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Decode request
	req, ok := DecodeRequestBody[dtos.WhatsappLogin](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	if jailed, _ := isUserJailed(req.Phone); jailed {
		respondTooManyAttempts(w, start, r, requestSummary)
		return
	}

	// Fetch user
	// user, err := fetchUser("", req.Phone)
	user, err := models.GetUserByPhone(req.Phone)
	if err != nil {
		log.Printf("ERR:::::::::::%v", err)
		handleFailedLogin(w, req.Phone, start, r, requestSummary)
		return
	}
	//will create a new user and return the user
	var request = dtos.RegisterRequest{
		Phonenumber: req.Phone,
	}
	if user == nil {
		user, err = models.CreateUser(request)
		if err != nil {
			log.Printf("create user error : %s", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Error creating user",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}

	// Generate a unique verification token (instead of OTP)
	verificationToken, err := utils.GenerateSecureTokenBase64(32)
	if err != nil {
		log.Println("Failed to generate verification token:", err)
		respondInternalError(w, "Failed to generate verification token", start, r, requestSummary)
		return
	}

	// Store the verification token with user ID
	if err := StoreVerificationTokenInRedis(user.ID, verificationToken, 5*time.Minute); err != nil {
		log.Println("Failed to store verification token:", err)
		respondInternalError(w, "Failed to store verification token", start, r, requestSummary)
		return
	}

	// Create a verification URL with the token

	// Send WhatsApp message with verification link
	phoneInt, err := strconv.Atoi(user.Phone)
	if err != nil {
		log.Println("Invalid phone number:", err)
	}
	if err := notification.SendWhatsappMessages(phoneInt, verificationToken, "Auth"); err != nil {
		log.Println("Failed to send WhatsApp message:", err)
		respondInternalError(w, "Failed to send verification message", start, r, requestSummary)
		return
	}

	clearLoginAttempts(req.Phone)

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "WhatsApp verification message sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
func VerifyWhatsAppHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Get token from query params
	token := r.URL.Query().Get("token")
	if token == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "Verification token is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Retrieve user ID from Redis using the token
	userID, err := GetUserIDFromVerificationToken(token)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   "Invalid or expired verification token",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Get user from database
	user, err := models.GetUserByUserID(userID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   noUserFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Generate authentication token
	authToken, err := generateToken(user, "auth", time.Hour)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Token generation failed",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Update last login timestamp
	models.UpdateLastLogin(user.ID)

	// Return success response (could redirect to app with token)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
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

// StoreVerificationTokenInRedis stores the verification token with user ID
func StoreVerificationTokenInRedis(userID, token string, expiration time.Duration) error {
	ctx := context.Background()
	err := Redis.Set(ctx, fmt.Sprintf("%s:%s", w, token), userID, expiration).Err()
	return err
}

// GetUserIDFromVerificationToken retrieves the user ID for a verification token
func GetUserIDFromVerificationToken(token string) (string, error) {
	ctx := context.Background()
	userID, err := Redis.Get(ctx, fmt.Sprintf("%s:%s", w, token)).Result()
	if err != nil {
		return "", err
	}

	// Delete the token after use
	Redis.Del(ctx, fmt.Sprintf("%s:%s", w, token))

	return userID, nil
}

func WhatsAppWebhookHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if !processWebhookRequest(w, r, requestSummary, start) {
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Message:   "Webhook processed",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func processWebhookRequest(w http.ResponseWriter, r *http.Request, requestSummary string, start time.Time) bool {
	webhook, ok := DecodeRequestBody[dtos.WhatsAppWebhook](r, w, requestSummary, start)
	if !ok {
		return false
	}

	for _, entry := range webhook.Entry {
		for _, change := range entry.Changes {
			if !processMessageChange(change, r) {
				continue
			}
		}
	}
	return true
}

func processMessageChange(change dtos.Change, r *http.Request) bool {
	if change.Field != "messages" {
		return false
	}

	for _, msg := range change.Value.Messages {
		if !processTextMessage(msg, r) {
			continue
		}
	}
	return true
}

func processTextMessage(msg dtos.Message, r *http.Request) bool {
	if msg.Type != "text" {
		return false
	}

	log.Printf("Received message from %s: %s", msg.From, msg.Text.Body)

	if strings.HasPrefix(msg.Text.Body, "Login to MyApp") {
		handleLoginRequest(msg.From, r)
	}
	return true
}

func handleLoginRequest(sender string, r *http.Request) {
	phone := strings.TrimPrefix(sender, "whatsapp:")
	loginReq := dtos.WhatsappLogin{Phone: phone}

	loginR, err := createLoginRequest(loginReq, r)
	if err != nil {
		log.Printf("Error creating login request: %v", err)
		return
	}

	loginW := httptest.NewRecorder()
	WhatsAppLoginHandler(loginW, loginR)

	if loginW.Code != http.StatusOK {
		log.Printf("Login handler returned non-OK status: %d", loginW.Code)
	}
}

func createLoginRequest(loginReq dtos.WhatsappLogin, r *http.Request) (*http.Request, error) {
	jsonBody, err := json.Marshal(loginReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling login request: %w", err)
	}

	loginR, err := http.NewRequest("POST", "/whatsapp/login", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	loginR.Body = io.NopCloser(bytes.NewReader(jsonBody))
	loginR.Header.Set("Content-Type", "application/json")
	return loginR.WithContext(r.Context()), nil
}

// GenerateWhatsAppDeeplink creates a URL to open WhatsApp with predefined message
func GenerateWhatsAppDeeplink(message string) string {
	sender, err := strconv.Atoi(os.Getenv("WHATSAPPSENDER"))
	if err != nil {
		log.Printf("invalid sender: %v", err)
	}

	encodedMsg := url.QueryEscape(message)
	return fmt.Sprintf("https://wa.me/%d?text=%s", sender, encodedMsg)
}

// StoreWhatsAppSessionData stores the login session data
func StoreWhatsAppSessionData(sessionID, userID, token string) error {
	ctx := context.Background()
	data := map[string]string{
		"user_id": userID,
		"token":   token,
	}
	jsonData, _ := json.Marshal(data)

	return Redis.Set(ctx,
		fmt.Sprintf("%s:%s", session, sessionID),
		jsonData,
		10*time.Minute).Err()
}

// GetWhatsAppSessionData retrieves session data
func GetWhatsAppSessionData(sessionID string) (userID, token string, err error) {
	ctx := context.Background()
	data, err := Redis.Get(ctx, fmt.Sprintf("%s:%s", session, sessionID)).Result()
	if err != nil {
		return "", "", err
	}

	var sessionData map[string]string
	if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
		return "", "", err
	}

	return sessionData["user_id"], sessionData["token"], nil
}

// DeleteWhatsAppSessionData cleans up after verification
func DeleteWhatsAppSessionData(sessionID string) error {
	ctx := context.Background()
	return Redis.Del(ctx, fmt.Sprintf("%s:%s", session, sessionID)).Err()
}
