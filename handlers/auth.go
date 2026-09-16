package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/utils"
)

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

	tempData, _ := json.Marshal(map[string]any{
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
// @Success      200   {object}  map[string]any "Verification successful with auth tokens"
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      404   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Router       /api/auth/verify-otp [post]
