/*
Package handlers implements HTTP request handlers for authentication and authorization.
This file contains handlers for user authentication flows including registration,
login, logout and token management.
*/
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
)

// Authentication-related constants
const (
	// Maximum number of failed login attempts before user is jailed
	maxLoginAttempts = 5
	// Duration for which a user is jailed after max failed attempts
	jailDuration = 15 * time.Minute
	// Redis key prefix for tracking login attempts
	loginAttemptKeyPrefix = "login:attempts:"
	// Redis key prefix for tracking jailed users
	jailKeyPrefix = "login:jail:"
)

// Message templates for notifications
var message = "Your verification code is %s. It will expire in 5 minutes. Ekomasi."
var subject = "Ekomasi, Here is your OTP"
var verificationRedisKey = "pending_signup:"

// JWT secret key for token signing
// jwtSecret is obtained dynamically from the environment using getJWTSecret().
// We avoid initializing it at package load time because the .env file may be
// loaded later (in main.init), resulting in an empty value.
//
// getJWTSecret reads the value each time so it reflects whatever is currently
// set in the environment.

// no global jwtSecret variable defined here

// RegisterHandler processes new user registration requests.
// It validates input, checks for existing users, creates the account,
// generates and sends OTP for verification.
//
// Flow:
// 1. Validates request body
// 2. Checks if email/phone is provided
// 3. Validates phone number format if provided
// 4. Checks for existing user
// 5. Stores user in redis temporarily
// 6. Generates and stores OTP
// 7. Sends OTP via email/SMS
//
// @Summary      Register a new user
// @Description  Register a new user with email or phone number
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dtos.RegisterRequest  true  "Registration details"
// @Success      201   {object}  dtos.GenericResponse
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      409   {object}  dtos.ErrorResponse
// @Router       /api/auth/register [post]
func RegisterHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.RegisterRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate request payload
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Auth") {
		return
	}

	// Validate required fields (email or phone)
	if req.Phonenumber == "" && req.Email == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Phone number or email is required for registration",
				Code:        http.StatusBadRequest,
			},
			Message:   "Phone number or email is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Validate phone number format
	if req.Phonenumber != "" {
		if !utils.IsValidKenyanPhone(req.Phonenumber) {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Auth",
					Description: "Invalid phone number format provided",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid phone number",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}

	// Check if user already exists
	if !CheckUserExistsByEmailOrPhone(c, *req, start, requestSummary) {
		return
	}

	// Generate OTP (hardcoded for testing environment)
	otp := "2025"
	// otp, err := utils.GenerateOTP()
	// if err != nil {
	// 	log.Println("Failed to create OTP:", err)
	// 	return
	// }

	// Store temporary registration info in Redis for 10 minutes
	tempKey := getVerificationRedisKey(req.Email, req.Phonenumber)

	tempData, _ := json.Marshal(map[string]interface{}{
		"email":     req.Email,
		"phone":     req.Phonenumber,
		"firstname": req.Firstname,
		"lastname":  req.Lastname,
		"password":  req.Password,
		"otp":       otp,
	})

	// Rate limiting for registration
	identifier := req.Email
	if identifier == "" {
		identifier = req.Phonenumber
	}
	isAllowed, retryAfter, err := utils.CheckRateLimit(c.Request.Context(), "signup:"+identifier, 3, 1*time.Hour)
	if err != nil {
		log.Printf("Rate limit error: %v", err)
	}
	if !isAllowed {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Too many registration attempts for " + identifier,
				Code:        http.StatusTooManyRequests,
			},
			Message:   fmt.Sprintf("Too many registration attempts. Please try again in %d seconds.", retryAfter),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	if err := Redis.Set(context.Background(), tempKey, tempData, 10*time.Minute).Err(); err != nil {
		log.Printf("Failed to store temporary registration: %v", err)
		return
	}

	// Send OTP (disabled for testing)
	// if req.Email != "" {
	// 	htmlBody := utils.GenerateOTPEmailHTML(otp)
	// 	notification.SendEmail(req.Email, subject, htmlBody)
	// }
	// if req.Phonenumber != "" {
	// 	notification.SendSmsMessages(req.Phonenumber, fmt.Sprintf(message, otp))
	// }

	// Respond with success message
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Verify using the OTP sent within 10 minutes to complete registration.",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// getVerificationRedisKey returns the Redis key for verification based on email or phone.
func getVerificationRedisKey(email, phone string) string {
	if email != "" {
		return fmt.Sprintf("%s%s", verificationRedisKey, email)
	}
	return fmt.Sprintf("%s%s", verificationRedisKey, phone)
}
func CheckUserExistsByEmailOrPhone(c *gin.Context, req dtos.RegisterRequest, start time.Time, requestSummary string) bool {
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	existingUser, err := fetchUser(req.Email, req.Phonenumber, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Failed to check if user exists using email or phone number: " + err.Error(),
				Code:        http.StatusConflict,
			},
			Message:   "Failed to check if user exists",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return false

	}
	if existingUser != nil {
		message := "User with this email already exists"
		if req.Email == "" {
			message = "User with this phone number  already exists"
		}
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "User with this phone number or email " + req.Phonenumber + " or " + req.Email + " already exists",
				Code:        http.StatusConflict,
			},
			Message:   message,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return false
	}

	return true
}

// VerifySignupOTPHandler verifies OTP during signup process.
// It validates the OTP sent during registration and generates auth tokens
// on successful verification.
//
// Flow:
// 1. Validates request (email/phone + OTP)
// 2. Looks up user
// 3. Validates OTP
// 4. Generates auth tokens
// 5. Updates last login
// 6. Returns tokens
//
// @Summary      Verify signup OTP
// @Description  Verify OTP for new user registration
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dtos.VerifyOTP  true  "OTP verification details"
// @Success      200   {object}  map[string]interface{} "Verification successful with auth tokens"
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      404   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Router       /api/auth/verify-otp [post]
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
		Payload: map[string]interface{}{
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
		Payload: map[string]interface{}{
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
// @Success      200   {object}  map[string]interface{} "Verification successful with auth tokens"
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Router       /api/auth/refresh-token [post]
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
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
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
		Payload:   map[string]interface{}{"expires_in_seconds": 60},
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
func StoreOTPInRedis(userID string, otp string, ttl time.Duration) error {
	key := fmt.Sprintf("otp:%s", userID)
	err := Redis.Set(context.Background(), key, otp, ttl).Err()
	if err != nil {
		log.Printf("Failed to store OTP in Redis for user %s: %v", userID, err)
		return err
	}
	return nil
}

// getJWTSecret returns the JWT secret currently set in the environment.
// It is not cached at package load time because the environment might be
// modified (via godotenv.Load) after the package initialization. Any code
// needing to sign or verify tokens should call this helper to ensure they
// get the correct value. If the value is missing, we log a warning.
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("JWT_SECRET environment variable is not set")
	}
	return []byte(secret)
}

// Get OTP
func GetOTP(userID string) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", userID)

	// Get the OTP
	otp, err := Redis.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("failed to get OTP: %w", err)
	}
	return otp, nil
}

// Invalidate otp without retrieval
func InvalidateOTP(userID string) error {
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", userID)
	return Redis.Del(ctx, key).Err()
}

// AtomicVerifyOTP retrieves and deletes an OTP in a single atomic operation to prevent race conditions.
func AtomicVerifyOTP(userID string, providedOTP string) (bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", userID)

	// Lua script to get the value and delete the key only if it matches the provided OTP
	script := `
		local val = redis.call('GET', KEYS[1])
		if val and val == ARGV[1] then
			redis.call('DEL', KEYS[1])
			return val
		end
		return val
	`
	val, err := Redis.Eval(ctx, script, []string{key}, providedOTP).Result()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("failed to atomically verify OTP: %w", err)
	}

	if val == nil {
		return false, nil // OTP not found or already consumed
	}

	storedOTP, ok := val.(string)
	if !ok {
		return false, fmt.Errorf("unexpected OTP value type in Redis")
	}

	return storedOTP == providedOTP, nil
}

// DecodeTokenHandler decodes and returns JWT claims without validation
func DecodeTokenHandler(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenStr == "" {
		c.String(http.StatusBadRequest, `{"error":"Token is required"}`)
		return
	}

	// Parse without validating expiration (optional)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf(`{"error":"Failed to decode token: %v"}`, err))
		return
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		c.JSON(http.StatusOK, map[string]interface{}{
			"token_claims": claims,
		})
		return
	}

	c.String(http.StatusBadRequest, `{"error":"Invalid token claims"}`)
}

// Redis key management helpers
func getLoginKey(identifier string) string {
	return loginAttemptKeyPrefix + identifier
}

func getJailKey(identifier string) string {
	return jailKeyPrefix + identifier
}

// Rate limiting and jail mechanism helpers
func isUserJailed(identifier string) (bool, error) {
	key := getJailKey(identifier)
	count, err := Redis.Exists(context.Background(), key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func incrementFailedLogin(identifier string) error {
	key := getLoginKey(identifier)
	pipe := Redis.TxPipeline()
	pipe.Incr(context.Background(), key)
	pipe.Expire(context.Background(), key, jailDuration)
	_, err := pipe.Exec(context.Background())
	return err
}

func getFailedAttempts(identifier string) (int, error) {
	val, err := Redis.Get(context.Background(), getLoginKey(identifier)).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func jailUser(identifier string) error {
	return Redis.Set(context.Background(), getJailKey(identifier), "jailed", jailDuration).Err()
}

func clearLoginAttempts(identifier string) {
	Redis.Del(context.Background(), getLoginKey(identifier))
	Redis.Del(context.Background(), getJailKey(identifier))
}

// Request processing helpers
func decodeLoginRequest(r *http.Request) (*dtos.LoginRequest, error) {
	var req dtos.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return &req, err
}

func validateLoginRequest(req *dtos.LoginRequest) error {
	if req.Email == "" && req.Phone == "" {
		return errors.New("either email or phone number is required")
	}
	return nil
}

func getIdentifier(req *dtos.LoginRequest) string {
	if req.Email != "" {
		return req.Email
	}
	return req.Phone
}

// trying fetching user by email or phone
func fetchUser(email, phone string, tenantID int) (*dtos.User, error) {
	if email != "" {
		return models.GetUserByEmail(models.DB, email, tenantID)
	}
	return models.GetUserByPhone(models.DB, phone, tenantID)
}

// Error handling helpers
func handleFailedLogin(c *gin.Context, identifier string, start time.Time, r *http.Request, raw string) {
	// Logic moved to CheckRateLimit in handlers
	utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "Invalid credentials provided during login",
			Code:        http.StatusBadRequest,
		},
		Message:   "Invalid credentials",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   raw,
	})
}

func dispatchOTP(user *dtos.User, otp string, req dtos.LoginRequest) {
	// message := fmt.Sprintf(message, otp)
	// htmlBody := utils.GenerateOTPEmailHTML(otp)
	log.Printf("Dispatching OTP to user with email/phone %s |||||||||||||||||||||||||  OTP: %s ", user.Email+user.Phone, otp)
	// if req.Email != "" {
	// 	notification.SendEmail(user.Email, "Ekomasi, Here is your OTP", htmlBody)
	// }
	if req.Phone != "" {
		// notification.SendSmsMessages(user.Phone, message)
		// if phoneInt, err := strconv.Atoi(user.Phone); err == nil {
		// 	// notification.SendWhatsappMessages(phoneInt, otp, "Otp")
		// } else {
		// 	log.Println("Invalid phone number:", err)
		// }
		templateData := map[string]any{
			"1": otp,
			"2": "5",
		}
		notification.SendTemplateMessage(user.Phone, "otp", templateData)
	}
}

// Response helpers
func respondBadRequest(c *gin.Context, msg string, start time.Time, r *http.Request, raw string) {
	utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: msg,
			Code:        http.StatusBadRequest,
		},
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   raw,
	})
}

func respondTooManyAttempts(c *gin.Context, start time.Time, r *http.Request, raw string, retryAfter int) {
	utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "Too many failed login attempts",
			Code:        http.StatusTooManyRequests,
		},
		Message:   fmt.Sprintf("Too many failed attempts. Try again in %d seconds.", retryAfter),
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   raw,
	})
}

func respondInternalError(c *gin.Context, msg string, start time.Time, r *http.Request, raw string) {
	utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: msg,
			Code:        http.StatusInternalServerError,
		},
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   raw,
	})
}

// Dependency injection interfaces and initialization
type UtilsService interface {
	GetRequestSummary(r *http.Request) string
	ValidateStructAndRespond(req interface{}, w http.ResponseWriter, r *http.Request, requestSummary string, start time.Time) bool
	RespondWithError(w http.ResponseWriter, options utils.ErrorJSONResponseOptions)
	RespondWithJSON(w http.ResponseWriter, options utils.SuccessJSONResponseOptions)
	IsValidKenyanPhone(phone string) bool
	GenerateOTP() (string, error)
	GetCurrentFuncName() string
}

type ModelsService interface {
	CreateUser(input dtos.RegisterRequest) (*dtos.User, error)
}

var (
	utilsService  UtilsService
	modelsService ModelsService
)

// Initialize with default implementations
func InitServices(utilsSvc UtilsService, modelsSvc ModelsService) {
	utilsService = utilsSvc
	modelsService = modelsSvc
}

// AtomicGetAndDelete retrieves a value and deletes the key in a single atomic operation.
func AtomicGetAndDelete(ctx context.Context, key string) (string, error) {
	script := `
		local val = redis.call('GET', KEYS[1])
		if val then
			redis.call('DEL', KEYS[1])
		end
		return val
	`

	val, err := Redis.Eval(ctx, script, []string{key}).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}

	if val == nil {
		return "", nil
	}

	res, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("unexpected value type from Redis")
	}

	return res, nil
}
