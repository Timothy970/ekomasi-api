package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

func VerifySignupOTPHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.VerifyOTP](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate required fields
	if req.Email == "" && req.Phone == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Email or phone number is required",
				Code:        http.StatusBadRequest,
			},
			Message:   "Email or phone number is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	// Check for rate limiting/jail
	identifierRequest := dtos.LoginRequest{
		Email: req.Email,
		Phone: req.Phone}
	identifier := getIdentifier(&identifierRequest)
	isAllowed, retryAfter, err := utils.CheckRateLimit(c.Request.Context(), "verify_otp:"+identifier, 5, 30*time.Minute)
	if err != nil {
		log.Printf("Rate limit error: %v", err)
	}
	if !isAllowed {
		respondTooManyAttempts(c, start, c.Request, requestSummary, retryAfter)
		return
	}
	var user *dtos.User

	// Try fetching existing user
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	user, err = fetchUser(req.Email, req.Phone, tenantID)
	// If other error fetching user
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Error retrieving user for OTP verification: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Error retrieving user",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	log.Printf("found user::::%v", user)
	// If no user found → proceed to signup verification
	if user == nil {
		VerifySignUp(c, req, start, requestSummary)
		return
	}

	// Otherwise → user exists → handle sign-in verification
	verifySignIn(user, *req, c, start, requestSummary)
}

func verifySignIn(user *dtos.User, req dtos.VerifyOTP, c *gin.Context, start time.Time, requestSummary string) {
	//first validate otp
	if err := validateOtp(user.ID, req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Invalid or expired OTP",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//get refresh token and token
	token, refreshToken, err := issueTokens(user)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Token generation failed: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// update last login for user
	models.UpdateLastLogin(models.DB, user.ID)
	// OTP already invalidated by AtomicVerifyOTP via validateOtp
	// Clear rate limit attempts after successful verification
	identifier := user.Email
	if identifier == "" {
		identifier = user.Phone
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module: "Auth", Description: "Login verified", Code: http.StatusOK,
		},
		Payload: map[string]any{
			"token":                    token,
			"refresh_token":            refreshToken,
			"token_expires_in":         int((7 * 24 * time.Hour).Seconds()),
			"refresh_token_expires_in": int((12 * time.Hour).Seconds()),
		},
		Message:   "Sign-in verification successful",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func VerifySignUp(c *gin.Context, req *dtos.VerifyOTP, start time.Time, requestSummary string) {
	ctx := context.Background()
	//check if redis key for the email or phone exists for  a pending signup
	tempKey := getVerificationRedisKey(req.Email, req.Phone)
	// Atomic get and delete to prevent race conditions during signup
	val, err := AtomicGetAndDelete(ctx, tempKey)
	if err != nil || val == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module: "Auth", Description: "Pending signup not found or expired", Code: http.StatusNotFound,
			},
			Message:   "Invalid or expired OTP",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	var pending struct {
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
		Password  string `json:"password"`
		Otp       string `json:"otp"`
	}
	//get the user details from redis for the user pending verification
	if err := json.Unmarshal([]byte(val), &pending); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Failed to read pending registration data",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Internal server error",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	if pending.Otp != req.OTP {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module: "Auth", Description: "Invalid OTP", Code: http.StatusBadRequest,
			},

			Message:   "Invalid OTP",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Create user now and store in DB
	newUserReq := dtos.RegisterRequest{
		Email:       pending.Email,
		Phonenumber: pending.Phone,
		Firstname:   pending.Firstname,
		Lastname:    pending.Lastname,
		Password:    pending.Password,
	}
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	user, err := models.CreateUser(models.DB, newUserReq, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Failed to create user after verification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Clear rate limit attempts after successful verification
	identifier := req.Email
	if identifier == "" {
		identifier = req.Phone
	}

	token, refreshToken, err := issueTokens(user)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Token generation failed after signup verification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return

	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module: "Auth", Description: "User created and verified", Code: http.StatusOK,
		},
		Payload: map[string]any{
			"token":                    token,
			"refresh_token":            refreshToken,
			"token_expires_in":         int((7 * 24 * time.Hour).Seconds()),
			"refresh_token_expires_in": int((12 * time.Hour).Seconds()),
		},
		Message:   "Signup verification successful",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func issueTokens(user *dtos.User) (string, string, error) {
	tokenExpiration := 7 * 24 * time.Hour
	if user.Role == "admin" || user.Role == "superadmin" {
		tokenExpiration = 24 * time.Hour
	}

	token, err := generateToken(user, "auth", tokenExpiration)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := generateToken(user, "refresh_token", 12*time.Hour)
	if err != nil {
		return "", "", err
	}
	return token, refreshToken, nil
}

func validateOtp(userID string, req dtos.VerifyOTP) error {
	isValid, err := AtomicVerifyOTP(userID, req.OTP)
	if err != nil {
		return fmt.Errorf("OTP verification failed: %w", err)
	}
	if !isValid {
		return errors.New("invalid or expired OTP")
	}
	return nil
}

// LoginHandler processes user login requests.
// It implements rate limiting and jail mechanism for failed attempts.
// Sends OTP for two-factor authentication.
//
// Flow:
// 1. Validates login request
// 2. Checks if user is jailed
// 3. Verifies user exists
// 4. Generates and sends OTP
// 5. Clears failed attempts on success
//
// @Summary      User Login
// @Description  Authenticate user and send OTP
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dtos.LoginRequest  true  "Login credentials"
// @Success      200   {object}  dtos.GenericResponse
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      429   {object}  dtos.ErrorResponse
// @Router       /api/auth/login [post]
