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
	"strconv"
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

// JWT secret key for token signing
var jwtSecret = []byte("Q7wcj5g0cDNRxoknR5uu")

/*
RegisterHandler processes new user registration requests.
It validates input, checks for existing users, creates the account,
generates and sends OTP for verification.

Flow:
1. Validates request body
2. Checks if email/phone is provided
3. Validates phone number format if provided
4. Checks for existing user
5. Creates user in database
6. Generates and stores OTP
7. Sends OTP via email/SMS
*/
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.RegisterRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	// Validate request
	if req.Phonenumber == "" && req.Email == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "Phone number or email is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if req.Phonenumber != "" {
		if !utils.IsValidKenyanPhone(req.Phonenumber) {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusBadRequest,
				Message:   "Invalid phone number",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	// check if existing user
	if !CheckUserExistsByEmailOrPhone(w, r, *req, start, requestSummary) {
		return
	}

	user, err := models.CreateUser(*req)
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

	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Println("Failed to create OTP:", err)
		return
	}
	// Store OTP in Redis with a TTL 5 minutes
	if err := StoreOTPInRedis(user.ID, otp, 5*time.Minute); err != nil {
		log.Printf("Failed to store OTP in Redis:%s", err)
		return
	}
	handleSendingOtps(*req, otp, *user)
	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "User created, activation code sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func handleSendingOtps(req dtos.RegisterRequest, otp string, user dtos.User) {
	if req.Email != "" {
		htmlBody := utils.GenerateOTPEmailHTML(otp)
		notification.SendEmail(user.Email, subject, htmlBody)
	}
	if req.Phonenumber != "" {
		notification.SendSmsMessages(req.Phonenumber, fmt.Sprintf(message, otp))
		phoneInt, err := strconv.Atoi(req.Phonenumber)
		if err != nil {
			log.Println("Invalid phoner:", err)
		} else {
			log.Println("Phone as int:", phoneInt)
		}
		// notification.SendWhatsappOtpMessages(phoneInt, otp)
	}
}
func CheckUserExistsByEmailOrPhone(w http.ResponseWriter, r *http.Request, req dtos.RegisterRequest, start time.Time, requestSummary string) bool {
	if req.Email != "" {
		existingUser, err := models.GetUserByEmail(req.Email)
		if err != nil {
			log.Printf("GetUserByEmail error: %s", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Error checking email",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return false
		}
		if existingUser != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusConflict,
				Message:   "User with this email already exists",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return false
		}
	}

	if req.Phonenumber != "" {
		existingUser, err := models.GetUserByPhone(req.Phonenumber)
		if err != nil {
			log.Printf("GetUserByPhone error: %s", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Error checking phone number",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return false
		}
		if existingUser != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusConflict,
				Message:   "User with this phone number already exists",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return false
		}
	}

	return true
}

/*
VerifySignupOTPHandler verifies OTP during signup process.
It validates the OTP sent during registration and generates auth tokens
on successful verification.

Flow:
1. Validates request (email/phone + OTP)
2. Looks up user
3. Validates OTP
4. Generates auth tokens
5. Updates last login
6. Returns tokens
*/
func VerifySignupOTPHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.VerifyOTP](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate required fields
	if req.Email == "" && req.Phone == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   "Email or phone number is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	if req.OTP == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   "OTP cannot be empty",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Lookup user
	var user *dtos.User
	var err error

	if req.Email != "" {
		user, err = models.GetUserByEmail(req.Email)
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
	} else if req.Phone != "" {
		user, err = models.GetUserByPhone(req.Phone)
		if err != nil {
			log.Printf("GetUserByPhone error: %s", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Error checking phone number",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
	}

	// Validate OTP
	storedOTP, err := GetAndInvalidateOTP(user.ID)
	if err != nil {
		log.Printf("Error retrieving OTP: %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   "Invalid or expired OTP",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	if storedOTP != req.OTP {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   "Incorrect OTP",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Generate token
	token, err := generateToken(user, "auth", 5*time.Hour)
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
	refreshToken, _ := generateToken(user, "refresh_token", 12*time.Hour)
	// Update last login timestamp
	models.UpdateLastLogin(user.ID)

	// Return success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
		Payload: map[string]interface{}{
			"token":                    token,
			"token_expires_in":         3600,
			"refresh_token":            refreshToken,
			"refresh_token_expires_in": 3600 * 12,
		},
		Message:   "Verification successful",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

/*
LoginHandler processes user login requests.
It implements rate limiting and jail mechanism for failed attempts.
Sends OTP for two-factor authentication.

Flow:
1. Validates login request
2. Checks if user is jailed
3. Verifies user exists
4. Generates and sends OTP
5. Clears failed attempts on success
*/
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	req, err := decodeLoginRequest(r)
	if err != nil {
		respondBadRequest(w, "Invalid request", start, r, requestSummary)
		return
	}
	if err := validateLoginRequest(req); err != nil {
		respondBadRequest(w, err.Error(), start, r, requestSummary)
		return
	}

	identifier := getIdentifier(req)
	if jailed, _ := isUserJailed(identifier); jailed {
		respondTooManyAttempts(w, start, r, requestSummary)
		return
	}

	user, err := fetchUser(req.Email, req.Phone)
	if err != nil {
		log.Printf("%v", err)
		handleFailedLogin(w, identifier, start, r, requestSummary)
		return
	}
	if user == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   "User not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Println("Failed to generate OTP:", err)
		return
	}
	if err := StoreOTPInRedis(user.ID, otp, 5*time.Minute); err != nil {
		log.Println("Failed to store OTP:", err)
		respondInternalError(w, "Failed to store OTP", start, r, requestSummary)
		return
	}

	clearLoginAttempts(identifier)
	dispatchOTP(user, otp)

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "OTP sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

/*
RefreshTokenHandler handles token refresh requests.
Issues new access and refresh tokens if the refresh token is valid.

Flow:
1. Validates refresh token
2. Verifies user still exists
3. Generates new token pair
4. Updates last login time
*/
func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	req, err := decodeLoginRequest(r)
	if err != nil {
		respondBadRequest(w, "Invalid request", start, r, requestSummary)
		return
	}
	if err := validateLoginRequest(req); err != nil {
		respondBadRequest(w, err.Error(), start, r, requestSummary)
		return
	}

	identifier := getIdentifier(req)
	if jailed, _ := isUserJailed(identifier); jailed {
		respondTooManyAttempts(w, start, r, requestSummary)
		return
	}

	user, err := fetchUser(req.Email, req.Phone)
	if err != nil {
		log.Printf("ERR:::::::::::%v", err)
		handleFailedLogin(w, identifier, start, r, requestSummary)
		return
	}
	if user == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   "User not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// validate it a good refresh token
	// Generate token that expires after 7 days
	token, err := generateToken(user, "auth", 7*24*time.Hour)
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
	refreshToken, _ := generateToken(user, "refresh_token", 12*time.Hour)
	// Update last login timestamp
	models.UpdateLastLogin(user.ID)

	// Return success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
		Payload: map[string]any{
			"token":                    token,
			"token_expires_in":         3600 * 7 * 24,
			"refresh_token":            refreshToken,
			"refresh_token_expires_in": 3600 * 12,
		},
		Message:   "Token refreshed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})

}

/*
LogoutHandler invalidates the current user's token.
Adds token to blacklist in Redis until original expiration.

Flow:
1. Extracts token
2. Gets token expiration
3. Adds to blacklist
*/
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Logged out successfully",
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
	req, ok := DecodeRequestBody[dtos.ResendOTP](r, w, requestSummary, start)
	if !ok {
		return
	}
	user, err := fetchUser(req.Email, req.Phone)
	if err != nil {
		log.Printf("ERR:::::::::::%v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if user == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   "User does'nt exist",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Println("Failed to generate OTP:", err)
		return
	}
	if err := StoreOTPInRedis(user.ID, otp, 5*time.Minute); err != nil {
		log.Println("Failed to store OTP:", err)
		respondInternalError(w, "Failed to store OTP", start, r, requestSummary)
		return
	}
	if req.Email != "" {
		htmlBody := utils.GenerateOTPEmailHTML(otp)
		notification.SendEmail(user.Email, subject, htmlBody)
	}
	if user.Phone != "" {
		notification.SendSmsMessages(user.Phone, fmt.Sprintf(message, otp))
		phoneInt, err := strconv.Atoi(user.Phone)
		if err != nil {
			log.Println("Invalid phone number:", err)
		} else {
			log.Println("Phone as int:", phoneInt)
		}
		// notification.SendWhatsappMessages(phoneInt, otp, "Otp")
	}
	// Respond with success message
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "OTP resent successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
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

// GetAndInvalidateOTP retrieves and immediately invalidates an OTP
func GetAndInvalidateOTP(userID string) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", userID)

	// Get the OTP
	otp, err := Redis.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("failed to get OTP: %w", err)
	}

	// Invalidate the OTP
	if err := Redis.Del(ctx, key).Err(); err != nil {
		log.Printf("Warning: failed to delete OTP after retrieval for user %s: %v", userID, err)
	}

	return otp, nil
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

func fetchUser(email, phone string) (*dtos.User, error) {
	if email != "" {
		return models.GetUserByEmail(email)
	}
	return models.GetUserByPhone(phone)
}

// Error handling helpers
func handleFailedLogin(w http.ResponseWriter, identifier string, start time.Time, r *http.Request, raw string) {
	_ = incrementFailedLogin(identifier)
	attempts, _ := getFailedAttempts(identifier)
	if attempts >= maxLoginAttempts {
		_ = jailUser(identifier)
	}
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		Code:      http.StatusUnauthorized,
		Message:   "Invalid credentials",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   raw,
	})
}

func dispatchOTP(user *dtos.User, otp string) {
	message := fmt.Sprintf(message, otp)
	htmlBody := utils.GenerateOTPEmailHTML(otp)

	if user.Email != "" {
		notification.SendEmail(user.Email, "Adenzo, Here is your OTP", htmlBody)
	}
	if user.Phone != "" {
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
		Code:      http.StatusBadRequest,
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   raw,
	})
}

func respondTooManyAttempts(w http.ResponseWriter, start time.Time, r *http.Request, raw string) {
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		Code:      http.StatusTooManyRequests,
		Message:   "Too many failed attempts. Try again later.",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   raw,
	})
}

func respondInternalError(w http.ResponseWriter, msg string, start time.Time, r *http.Request, raw string) {
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		Code:      http.StatusInternalServerError,
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
