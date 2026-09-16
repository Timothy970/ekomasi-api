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
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
)

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
		c.JSON(http.StatusOK, map[string]any{
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
	ValidateStructAndRespond(req any, w http.ResponseWriter, r *http.Request, requestSummary string, start time.Time) bool
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
