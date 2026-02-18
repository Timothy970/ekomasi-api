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

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"

	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/utils"
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
var message = "Your verification code is %s. It will expire in 5 minutes. Adenzo."
var subject = "Adenzo, Here is your OTP"
var verificationRedisKey = "pending_signup:"

// JWT secret key for token signing
var jwtSecret = []byte("Q7wcj5g0cDNRxoknR5uu")

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
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.RegisterRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate request payload
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Auth") {
		return
	}

	// Validate required fields (email or phone)
	if req.Phonenumber == "" && req.Email == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Phone number or email is required for registration",
				Code:        http.StatusBadRequest,
			},
			Message:   "Phone number or email is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Validate phone number format
	if req.Phonenumber != "" {
		if !utils.IsValidKenyanPhone(req.Phonenumber) {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Auth",
					Description: "Invalid phone number format provided",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid phone number",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}

	// Check if user already exists
	if !CheckUserExistsByEmailOrPhone(w, r, *req, start, requestSummary) {
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
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Verify using the OTP sent within 10 minutes to complete registration.",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// getVerificationRedisKey returns the Redis key for verification based on email or phone.
func getVerificationRedisKey(email, phone string) string {
	if email != "" {
		return fmt.Sprintf("%s%s", verificationRedisKey, email)
	}
	return fmt.Sprintf("%s%s", verificationRedisKey, phone)
}
func CheckUserExistsByEmailOrPhone(w http.ResponseWriter, r *http.Request, req dtos.RegisterRequest, start time.Time, requestSummary string) bool {
	existingUser, err := fetchUser(req.Email, req.Phonenumber)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Failed to check if user exists using email or phone number: " + err.Error(),
				Code:        http.StatusConflict,
			},
			Message:   "Failed to check if user exists",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return false

	}
	if existingUser != nil {
		message := "User with this email already exists"
		if req.Email == "" {
			message = "User with this phone number  already exists"
		}
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "User with this phone number or email " + req.Phonenumber + " or " + req.Email + " already exists",
				Code:        http.StatusConflict,
			},
			Message:   message,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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
func VerifySignupOTPHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.VerifyOTP](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate required fields
	if req.Email == "" && req.Phone == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Email or phone number is required",
				Code:        http.StatusBadRequest,
			},
			Message:   "Email or phone number is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	var user *dtos.User
	var err error

	// Try fetching existing user
	user, err = fetchUser(req.Email, req.Phone)
	// If other error fetching user
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Error retrieving user for OTP verification: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Error retrieving user",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	log.Printf("found user::::%v", user)
	// If no user found → proceed to signup verification
	if user == nil {
		VerifySignUp(w, r, req, start, requestSummary)
		return
	}

	// Otherwise → user exists → handle sign-in verification
	verifySignIn(user, *req, w, r, start, requestSummary)
}

func verifySignIn(user *dtos.User, req dtos.VerifyOTP, w http.ResponseWriter, r *http.Request, start time.Time, requestSummary string) {
	//first validate otp
	if err := validateOtp(user.ID, req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Invalid or expired OTP",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//get refresh token and token
	token, refreshToken, err := issueTokens(user)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Token generation failed: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// update last login for user
	models.UpdateLastLogin(models.DB, user.ID)
	//invalidate otp after successful login
	InvalidateOTP(user.ID)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
		RawBody:   requestSummary})
}

func VerifySignUp(w http.ResponseWriter, r *http.Request, req *dtos.VerifyOTP, start time.Time, requestSummary string) {
	ctx := context.Background()
	//check if redis key for the email or phone exists for  a pending signup
	tempKey := getVerificationRedisKey(req.Email, req.Phone)
	val, err := Redis.Get(ctx, tempKey).Result()
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module: "Auth", Description: "Pending signup not found or expired", Code: http.StatusNotFound,
			},
			Message:   "Invalid or expired OTP",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Failed to read pending registration data",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Internal server error",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	if pending.Otp != req.OTP {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module: "Auth", Description: "Invalid OTP", Code: http.StatusBadRequest,
			},

			Message:   "Invalid OTP",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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
	user, err := models.CreateUser(models.DB, newUserReq)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Failed to create user after verification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Remove pending record from Redis
	Redis.Del(ctx, tempKey)

	token, refreshToken, err := issueTokens(user)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Token generation failed after signup verification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return

	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
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
	storedOTP, err := GetOTP(userID)
	if err != nil {
		log.Printf("Error retrieving OTP: %s", err)
		return fmt.Errorf("invalid or expired OTP")
	}
	if storedOTP != req.OTP {
		return errors.New("incorrect OTP")
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
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, err := decodeLoginRequest(r)
	if err != nil {
		respondBadRequest(w, "Invalid Payload", start, r, requestSummary)
		return
	}
	// Validate request payload
	if err := validateLoginRequest(req); err != nil {
		respondBadRequest(w, err.Error(), start, r, requestSummary)
		return
	}

	// Check for rate limiting/jail
	identifier := getIdentifier(req)
	if jailed, _ := isUserJailed(identifier); jailed {
		respondTooManyAttempts(w, start, r, requestSummary)
		return
	}

	// Check if user exists
	user, err := fetchUser(req.Email, req.Phone)
	if err != nil {
		log.Printf("%v", err)
		handleFailedLogin(w, identifier, start, r, requestSummary)
		return
	}
	if user == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: noUserFound,
				Code:        http.StatusBadRequest,
			},
			Message:   noUserFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if user.Status != "active" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "User account is not active",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User account is not active",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Generate OTP (hardcoded for testing)
	otp := "2025"
	// otp, err := utils.GenerateOTP()
	// if err != nil {
	// 	log.Println("Failed to generate:", err)
	// 	return
	// }

	// Store OTP in Redis
	if err := StoreOTPInRedis(user.ID, otp, 5*time.Minute); err != nil {
		log.Println("Failed to store:", err)
		respondInternalError(w, "Failed to generate OTP", start, r, requestSummary)
		return
	}

	// Clear failed login attempts
	clearLoginAttempts(identifier)
	// dispatchOTP(user, otp, *req)

	// Respond with success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "OTP sent successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "OTP sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func AdminLoginHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, err := decodeLoginRequest(r)
	if err != nil {
		respondBadRequest(w, "Invalid request", start, r, requestSummary)
		return
	}
	// Validate request payload
	if err := validateLoginRequest(req); err != nil {
		respondBadRequest(w, err.Error(), start, r, requestSummary)
		return
	}

	// Check for rate limiting/jail
	identifier := getIdentifier(req)
	if jailed, _ := isUserJailed(identifier); jailed {
		respondTooManyAttempts(w, start, r, requestSummary)
		return
	}

	// Check if user exists
	user, err := fetchUser(req.Email, req.Phone)
	if err != nil {
		log.Printf("%v", err)
		handleFailedLogin(w, identifier, start, r, requestSummary)
		return
	}
	if user == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: noUserFound,
				Code:        http.StatusBadRequest,
			},
			Message:   noUserFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if user.Status != "active" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "User account is not active",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User account is not active",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Enforce role-based access (non-customers only)
	if user.Role != "" && strings.ToLower(user.Role) == "customer" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Unauthorized access",
				Code:        http.StatusUnauthorized,
			},
			Message:   "Unauthorized access",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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
		respondInternalError(w, "Failed to store OTP", start, r, requestSummary)
		return
	}

	// Clear failed login attempts
	clearLoginAttempts(identifier)
	// dispatchOTP(user, otp, *req)

	// Respond with success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "OTP sent successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "OTP sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, err := decodeLoginRequest(r)
	if err != nil {
		respondBadRequest(w, "Invalid request", start, r, requestSummary)
		return
	}
	// Validate request payload
	if err := validateLoginRequest(req); err != nil {
		respondBadRequest(w, err.Error(), start, r, requestSummary)
		return
	}

	// Check for rate limiting/jail
	identifier := getIdentifier(req)
	if jailed, _ := isUserJailed(identifier); jailed {
		respondTooManyAttempts(w, start, r, requestSummary)
		return
	}

	// Check if user exists
	user, err := fetchUser(req.Email, req.Phone)
	if err != nil {
		log.Printf("ERR:::::::::::%v", err)
		handleFailedLogin(w, identifier, start, r, requestSummary)
		return
	}
	if user == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: noUserFound,
				Code:        http.StatusBadRequest,
			},
			Message:   "User not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Token generation failed",
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
	refreshToken, _ := generateToken(user, "refresh_token", 12*time.Hour)
	// Update last login timestamp
	models.UpdateLastLogin(models.DB, user.ID)

	// Return success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
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
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	token := utils.ExtractToken(r.Header.Get("Authorization"))

	// Parse token to get expiration time
	claims := &dtos.CustomClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err == nil {
		// Get expiration time correctly from jwt.NumericDate
		expTime := claims.ExpiresAt.Time
		remainingTime := time.Until(expTime)

		if remainingTime > 0 {
			// Store in Redis with remaining TTL
			err := dtos.Redis.Set(r.Context(), "blacklist:"+token, "1", remainingTime).Err()
			if err != nil {
				log.Printf("Failed to blacklist token: %v", err)
			}
		}
	}

	// Respond with success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "User with id" + claims.UserID + " logged out successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User logged out successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ViewProfileHandler handles viewing user profile.
// func ViewProfileHandler(w http.ResponseWriter, r *http.Request) {
// 	// Extract user information from context (set by middleware)
// 	username := r.Context().Value("username").(string)

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
func ResendOptHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.ResendOTP](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Fetch user by email or phone
	user, err := fetchUser(req.Email, req.Phone)
	if err != nil {
		log.Printf("ERR:::::::::::%v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "Error fetching user for OTP resend",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	if user == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "User with email or phone " + req.Email + req.Phone + " doesn't exist for OTP resend",
				Code:        http.StatusNotFound,
			},
			Message:   "User doesn't exist",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	ctx := context.Background()
	activityKey := fmt.Sprintf("otp_resend_activity:%s", user.ID)

	// Check if resend was requested recently
	if exists, _ := Redis.Exists(ctx, activityKey).Result(); exists > 0 {
		ttl, err := Redis.TTL(ctx, activityKey).Result()
		if err != nil {
			log.Printf("Failed to fetch TTL for user %s: %v", user.ID, err)
			ttl = 0
		}

		remainingSeconds := int(ttl.Seconds())
		if remainingSeconds < 0 {
			remainingSeconds = 0
		}

		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Auth",
				Description: "OTP resend requested too soon for user with ID " + user.ID,
				Code:        http.StatusTooManyRequests,
			},
			Message:   fmt.Sprintf("Please wait %d seconds before requesting a new OTP", remainingSeconds),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Generate new OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Println("Failed to generate OTP:", err)
		respondInternalError(w, "Failed to generate OTP", start, r, requestSummary)
		return
	}

	// Store OTP in Redis (valid for 5 minutes)
	if err := StoreOTPInRedis(user.ID, otp, 5*time.Minute); err != nil {
		log.Println("Failed to store OTP:", err)
		respondInternalError(w, "Failed to store OTP", start, r, requestSummary)
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

	// Save resend activity limit (60 seconds)
	Redis.Set(ctx, activityKey, time.Now().Unix(), 60*time.Second)

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "Resend OTP generated and resent successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]interface{}{"expires_in_seconds": 60},
		Message:   "OTP resent successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
	return token.SignedString(jwtSecret)
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

// DecodeTokenHandler decodes and returns JWT claims without validation
func DecodeTokenHandler(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenStr == "" {
		http.Error(w, `{"error":"Token is required"}`, http.StatusBadRequest)
		return
	}

	// Parse without validating expiration (optional)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to decode token: %v"}`, err), http.StatusBadRequest)
		return
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token_claims": claims,
		})
		return
	}

	http.Error(w, `{"error":"Invalid token claims"}`, http.StatusBadRequest)
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
func fetchUser(email, phone string) (*dtos.User, error) {
	if email != "" {
		return models.GetUserByEmail(models.DB, email)
	}
	return models.GetUserByPhone(models.DB, phone)
}

// Error handling helpers
func handleFailedLogin(w http.ResponseWriter, identifier string, start time.Time, r *http.Request, raw string) {
	_ = incrementFailedLogin(identifier)
	attempts, _ := getFailedAttempts(identifier)
	if attempts >= maxLoginAttempts {
		_ = jailUser(identifier)
	}
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "Invalid credentials provided during login",
			Code:        http.StatusBadRequest,
		},
		Message:   "Invalid credentials",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   raw,
	})
}

func dispatchOTP(user *dtos.User, otp string, req dtos.LoginRequest) {
	message := fmt.Sprintf(message, otp)
	htmlBody := utils.GenerateOTPEmailHTML(otp)
	log.Printf("Dispatching OTP to user with email/phone %s ", user.Email+user.Phone)
	if req.Email != "" {
		notification.SendEmail(user.Email, "Adenzo, Here is your OTP", htmlBody)
	}
	if req.Phone != "" {
		notification.SendSmsMessages(user.Phone, message)
		// if phoneInt, err := strconv.Atoi(user.Phone); err == nil {
		// 	// notification.SendWhatsappMessages(phoneInt, otp, "Otp")
		// } else {
		// 	log.Println("Invalid phone number:", err)
		// }
	}
}

// Response helpers
func respondBadRequest(w http.ResponseWriter, msg string, start time.Time, r *http.Request, raw string) {
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: msg,
			Code:        http.StatusBadRequest,
		},
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   raw,
	})
}

func respondTooManyAttempts(w http.ResponseWriter, start time.Time, r *http.Request, raw string) {
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: "Too many failed login attempts",
			Code:        http.StatusTooManyRequests,
		},
		Message:   "Too many failed attempts. Try again later.",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   raw,
	})
}

func respondInternalError(w http.ResponseWriter, msg string, start time.Time, r *http.Request, raw string) {
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Auth",
			Description: msg,
			Code:        http.StatusInternalServerError,
		},
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
