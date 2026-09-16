package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/middleware"
	"ekomasi_backend/utils"
)

func LoginHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	req, err := decodeLoginRequest(c.Request)
	if err != nil {
		respondBadRequest(c, "Invalid Payload", start, c.Request, requestSummary)
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
		log.Printf("%v", err)
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
			Message:   noUserFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if user.Status != "active" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "User account is not active",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User account is not active",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Generate OTP (hardcoded for testing)
	// otp := "2025"
	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Println("Failed to generate:", err)
		return
	}

	// Store OTP in Redis
	if err := StoreOTPInRedis(user.ID, otp, 5*time.Minute); err != nil {
		log.Println("Failed to store:", err)
		respondInternalError(c, "Failed to generate OTP", start, c.Request, requestSummary)
		return
	}

	// Clear failed login attempts
	clearLoginAttempts(identifier)
	dispatchOTP(user, otp, *req)

	// Respond with success
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "OTP sent successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "OTP sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// AdminLoginHandler processes admin login requests.
// It enforces role-based access control, allowing only non-customer roles.
//
// @Summary      Admin Login
// @Description  Authenticate admin user and send OTP
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dtos.LoginRequest  true  "Login credentials"
// @Success      200   {object}  dtos.GenericResponse
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Failure      429   {object}  dtos.ErrorResponse
// @Router       /api/admin/login [post]
func AdminLoginHandler(c *gin.Context) {
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
		log.Printf("%v", err)
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
			Message:   noUserFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if user.Status != "active" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "User account is not active",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User account is not active",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Enforce role-based access (non-customers only)
	if user.Role != "" && strings.ToLower(user.Role) == "customer" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Unauthorized access",
				Code:        http.StatusUnauthorized,
			},
			Message:   "Unauthorized access",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Generate OTP (hardcoded for testing)
	otp := "2025"
	//
	// otp, err := utils.GenerateOTP()
	// if err != nil {
	// 	log.Println("Failed to generate one time password:", err)
	// 	return
	// }

	// Store OTP in Redis
	if err := StoreOTPInRedis(user.ID, otp, 5*time.Minute); err != nil {
		log.Println("Failed to store one time password:", err)
		respondInternalError(c, "Failed to store OTP", start, c.Request, requestSummary)
		return
	}

	// Clear failed login attempts
	clearLoginAttempts(identifier)
	dispatchOTP(user, otp, *req)

	// Respond with success
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "OTP sent successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "OTP sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// RefreshTokenHandler handles token refresh requests.
// Issues new access and refresh tokens if the refresh token is valid.
//
// Flow:
// 1. Validates refresh token
// 2. Verifies user still exists
// 3. Generates new token pair
// 4. Updates last login time
//
// @Summary      Refresh Token
// @Description  Refresh access and refresh tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  dtos.LoginRequest  true  "Refresh token"
// @Success      200   {object}  map[string]any "Verification successful with auth tokens"
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Router       /api/auth/refresh-token [post]
