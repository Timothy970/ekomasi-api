// Package handlers provides HTTP request handlers for WhatsApp-based authentication.
// This file implements passwordless login via WhatsApp, using verification tokens sent
// through WhatsApp messages. Users receive a secure link to verify their identity and
// authenticate without traditional passwords. Essential for mobile-first authentication.
package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
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
// @Success      200    {object}  map[string]interface{}     "Verification message sent successfully"
// @Failure      400    {object}  dtos.ErrorResponse       "Invalid request or phone number"
// @Failure      429    {object}  dtos.ErrorResponse       "Too many login attempts"
// @Failure      500    {object}  dtos.ErrorResponse       "Internal server error"
// @Router       /api/auth/whatsapp/login [post]
func WhatsAppLoginHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode and parse JSON request body with phone number
	req, ok := DecodeRequestBody[dtos.WhatsappLogin](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}

	// Validate phone number format and required fields
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Auth") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Check if user has been rate-limited due to too many failed attempts
	isAllowed, retryAfter, err := utils.CheckRateLimit(c.Request.Context(), "login:"+req.Phone, 5, 15*time.Minute)
	if err != nil {
		log.Printf("Rate limit error: %v", err)
	}
	if !isAllowed {
		// User is temporarily blocked, send 429 Too Many Requests response
		respondTooManyAttempts(c, start, c.Request, requestSummary, retryAfter)
		return
	}

	// Attempt to fetch existing user by phone number
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	user, err := models.GetUserByPhone(models.DB, req.Phone, tenantID)
	if err != nil {
		log.Printf("ERR:::::::::::%v", err)
		// User fetch failed, increment failed login attempts
		handleFailedLogin(c, req.Phone, start, c.Request, requestSummary)
		return
	}
	// Prepare user registration data for auto-creation
	var request = dtos.RegisterRequest{
		Phonenumber: req.Phone,
	}
	// If user doesn't exist, create new account automatically (frictionless onboarding)
	if user == nil {
		user, err = models.CreateUser(models.DB, request, tenantID)
		if err != nil {
			// User creation failed
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Auth",
					Description: "Error creating user",
					Code:        http.StatusInternalServerError,
				},
				Message:   "Error creating user",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}

	// Generate cryptographically secure verification token (32 bytes, base64 encoded)
	verificationToken, err := utils.GenerateSecureTokenBase64(32)
	if err != nil {
		log.Println("Failed to generate verification token:", err)
		// Token generation failed, return 500 error
		respondInternalError(c, "Failed to generate verification token", start, c.Request, requestSummary)
		return
	}

	// Store verification token in Redis with user ID, expires in 5 minutes
	if err := StoreVerificationTokenInRedis(user.ID, verificationToken, 5*time.Minute); err != nil {
		log.Println("Failed to store verification token:", err)
		// Redis storage failed, return 500 error
		respondInternalError(c, "Failed to store verification token", start, c.Request, requestSummary)
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
		respondInternalError(c, "Failed to send verification message", start, c.Request, requestSummary)
		return
	}

	// Clear any previous failed login attempts counter for this phone
	clearLoginAttempts(req.Phone)

	// Return success response indicating message was sent
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "WhatsApp verification message sent",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "WhatsApp verification message sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200    {object}  map[string]interface{}     "Verification successful with auth token"
// @Failure      400    {object}  dtos.ErrorResponse       "Missing verification token"
// @Failure      401    {object}  dtos.ErrorResponse       "Invalid or expired token"
// @Failure      500    {object}  dtos.ErrorResponse       "Token generation failed"
// @Router       /api/auth/whatsapp/verify [get]
func VerifyWhatsAppHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract verification token from query parameters
	token := c.Query("token")
	if token == "" {
		// Token is missing from request, return 400 Bad Request
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Verification token is required when verifying WhatsApp login",
				Code:        http.StatusBadRequest,
			},
			Message:   "Verification token is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Retrieve user ID from Redis using the token (token is deleted after retrieval)
	userID, err := GetUserIDFromVerificationToken(token)
	if err != nil {
		// Token not found in Redis or expired (5 minute expiration)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Invalid or expired verification token when retrieving user ID",
				Code:        http.StatusUnauthorized,
			},
			Message:   "Invalid or expired verification token",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Fetch complete user record from database using user ID
	user, err := models.GetUserByUserID(models.DB, userID)
	if err != nil {
		// User not found (should not happen if token was valid)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "No user found for the provided verification token",
				Code:        http.StatusUnauthorized,
			},
			Message:   noUserFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Failed to generate auth token after WhatsApp verification",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Token generation failed",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
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
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
