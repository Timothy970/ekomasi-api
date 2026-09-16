package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/golang-jwt/jwt/v5"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
)

func RefreshTokenHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	req, err := decodeLoginRequest(c.Request)
	if err != nil {
		respondBadRequest(c, "Invalid request", start, c.Request, requestSummary)
		return
	}
	// Validate request payload
	if err := validateLoginRequest(req); err != nil {
		respondBadRequest(c, err.Error(), start, c.Request, requestSummary)
		return
	}

	// Check for rate limiting/jail
	identifier := getIdentifier(req)
	isAllowed, retryAfter, err := utils.CheckRateLimit(c.Request.Context(), "login:"+identifier, maxLoginAttempts, jailDuration)
	if err != nil {
		log.Printf("Rate limit error: %v", err)
	}
	if !isAllowed {
		respondTooManyAttempts(c, start, c.Request, requestSummary, retryAfter)
		return
	}

	// Check if user exists
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	user, err := fetchUser(req.Email, req.Phone, tenantID)
	if err != nil {
		log.Printf("ERR:::::::::::%v", err)
		handleFailedLogin(c, identifier, start, c.Request, requestSummary)
		return
	}
	if user == nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: noUserFound,
				Code:        http.StatusBadRequest,
			},
			Message:   "User not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Generate new tokens
	// Generate token that expires after 7 days
	tokenExpirationTime := 7 * 24 * time.Hour
	// for admin the token expiration time is 1 day
	if user.Role == "admin" || user.Role == "superadmin" {
		tokenExpirationTime = 24 * time.Hour
	}
	token, err := generateToken(user, "auth", tokenExpirationTime)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Token generation failed",
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
	refreshToken, _ := generateToken(user, "refresh_token", 12*time.Hour)
	// Update last login timestamp
	models.UpdateLastLogin(models.DB, user.ID)

	// Return success response
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "Token refreshed and sent successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"token":                    token,
			"token_expires_in":         int(tokenExpirationTime.Seconds()),
			"refresh_token":            refreshToken,
			"refresh_token_expires_in": int(12 * time.Hour.Seconds()),
		},
		Message:   "Token refreshed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})

}

// LogoutHandler invalidates the current user's token.
// Adds token to blacklist in Redis until original expiration.
//
// Flow:
// 1. Extracts token
// 2. Gets token expiration
// 3. Adds to blacklist
//
// @Summary      Logout
// @Description  Invalidate current session token
// @Tags         Auth
// @Success      200   {object}  dtos.GenericResponse
// @Security     BearerAuth
// @Router       /api/auth/logout [post]
func LogoutHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	token := utils.ExtractToken(c.GetHeader("Authorization"))

	// Parse token to get expiration time
	claims := &dtos.CustomClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return getJWTSecret(), nil
	})

	if err == nil {
		// Get expiration time correctly from jwt.NumericDate
		expTime := claims.ExpiresAt.Time
		remainingTime := time.Until(expTime)

		if remainingTime > 0 {
			// Store in Redis with remaining TTL
			err := dtos.Redis.Set(c.Request.Context(), "blacklist:"+token, "1", remainingTime).Err()
			if err != nil {
				log.Printf("Failed to blacklist token: %v", err)
			}
		}
	}

	// Respond with success
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "User with id" + claims.UserID + " logged out successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User logged out successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ViewProfileHandler handles viewing user profile.
// func ViewProfileHandler(c *gin.Context) {
// 	// Extract user information from context (set by middleware)
// 	username := c.Request.Context().Value("username").(string)

// 	// Retrieve user profile from database (replace with database logic)
// 	user := dtos.User{Username: username, Email: "test@example.com", Role: "customer"}

// 	// Respond with user profile
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(user.Profile)
// }

/*
ResendOptHandler resends OTP to user.
Used when original OTP expires or wasn't received.

Flow:
1. Validates request
2. Verifies user exists
3. Generates new OTP
4. Sends via email/SMS
*/
func ResendOptHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.ResendOTP](c, requestSummary, start)
	if !ok {
		return
	}

	// Fetch user by email or phone
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	user, err := fetchUser(req.Email, req.Phone, tenantID)
	if err != nil {
		log.Printf("ERR:::::::::::%v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Error fetching user for OTP resend",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	if user == nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "User with email or phone " + req.Email + req.Phone + " doesn't exist for OTP resend",
				Code:        http.StatusNotFound,
			},
			Message:   "User doesn't exist",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Rate limiting for OTP resend
	identifier := user.ID
	isAllowed, retryAfter, err := utils.CheckRateLimit(c.Request.Context(), "resend_otp:"+identifier, 3, 5*time.Minute)
	if err != nil {
		log.Printf("Rate limit error: %v", err)
	}
	if !isAllowed {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "OTP resend requested too soon for user with ID " + user.ID,
				Code:        http.StatusTooManyRequests,
			},
			Message:   fmt.Sprintf("Please wait %d seconds before requesting a new OTP", retryAfter),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Generate new OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Println("Failed to generate OTP:", err)
		respondInternalError(c, "Failed to generate OTP", start, c.Request, requestSummary)
		return
	}

	// Store OTP in Redis (valid for 5 minutes)
	if err := StoreOTPInRedis(user.ID, otp, 5*time.Minute); err != nil {
		log.Println("Failed to store OTP:", err)
		respondInternalError(c, "Failed to store OTP", start, c.Request, requestSummary)
		return
	}

	// Send via email
	if req.Email != "" {
		htmlBody := utils.GenerateOTPEmailHTML(otp)
		notification.SendEmail(user.Email, subject, htmlBody)
	}

	// Send via SMS
	if user.Phone != "" {
		notification.SendSmsMessages(user.Phone, fmt.Sprintf(message, otp))
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "Resend OTP generated and resent successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"expires_in_seconds": 60},
		Message:   "OTP resent successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// Helper functions

// generateToken creates a new JWT token with user claims
func generateToken(user *dtos.User, tokenType string, expiresIn time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"id":           user.ID,
		"email":        user.Email,
		"first_name":   user.FirstName,
		"last_name":    user.LastName,
		"role":         user.Role,
		"phone_number": user.Phone,
		"type":         tokenType,
		"exp":          time.Now().Add(expiresIn).Unix(),
		"permissions":  user.Permissions,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// retrieve secret at runtime; this will pick up whatever the environment
	// currently holds (loaded by godotenv in main.init).
	secret := getJWTSecret()
	return token.SignedString(secret)
}

// StoreOTPInRedis saves OTP in Redis with expiration
