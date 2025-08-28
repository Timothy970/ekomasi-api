package utils

import (
	"adenzo_backend/dtos"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type PaginatedResponse struct {
	Data       []dtos.CategoryWithProducts `json:"data"`
	Pagination PaginationMeta              `json:"pagination"`
}

type PaginationMeta struct {
	Page       int  `json:"page"`
	Size       int  `json:"size"`
	TotalItems int  `json:"total_items"`
	TotalPages int  `json:"total_pages"`
	HasPrev    bool `json:"has_prev"`
	HasNext    bool `json:"has_next"`
	PrevPage   int  `json:"prev_page,omitempty"`
	NextPage   int  `json:"next_page,omitempty"`
}

// func CheckPasswordHash(password, hash string) bool {
// 	log.Printf("password:::%s", password)
// 	log.Printf("hash:::%s", hash)
// 	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
// 	return err == nil
// }

func ExtractToken(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

// helper funcfion to convert to string
func ToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.Itoa(int(v))
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
func GenerateResetPasswordToken(n int) (string, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// NowInNairobi returns the current time in Africa/Nairobi timezone
func NowInNairobi() time.Time {
	loc, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		log.Fatal("Failed to load Nairobi timezone:", err)
	}
	return time.Now().In(loc)
}
func PaginateCategories(data []dtos.CategoryWithProducts, page, size int) PaginatedResponse {
	total := len(data)
	start := (page - 1) * size
	end := start + size

	if start >= total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedData := data[start:end]
	totalPages := int(math.Ceil(float64(total) / float64(size)))

	meta := PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	if meta.HasPrev {
		meta.PrevPage = page - 1
	}
	if meta.HasNext {
		meta.NextPage = page + 1
	}

	return PaginatedResponse{
		Data:       paginatedData,
		Pagination: meta,
	}
}

// GenerateOTP generates a random 4-digit OTP
func GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000)) // 0 to 999999
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil // pad with leading zeros
}

// GenerateOTPEmailHTML returns the HTML content for an OTP email
func GenerateOTPEmailHTML(otp string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
  <body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px;">
    <div style="max-width: 500px; margin: auto; background-color: white; padding: 30px; border-radius: 8px;">
      <h2 style="color: #333;">Hello,</h2>
      <p style="font-size: 16px; color: #555;">
        Your One-Time Password (OTP) is:
      </p>
      <div style="font-size: 24px; font-weight: bold; color: #000; margin: 20px 0;">
        %s
      </div>
      <p style="font-size: 14px; color: #888;">
        This code will expire in 5 minutes.
      </p>
      <p style="font-size: 14px; color: #888;">
        If you didn’t request this, please ignore this email or contact support immediately.
      </p>
      <hr style="margin: 30px 0;">
      <p style="font-size: 14px; color: #aaa;">— The Adenzo Team</p>
    </div>
  </body>
</html>
`, otp)
}

func GetCurrentFuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	// Trim package path if you just want the function name
	fullName := fn.Name()
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1]
}

func GenerateSecureTokenBase64(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("token length must be positive")
	}

	// Calculate how many bytes we need for the desired length
	// Base64 encodes 3 bytes into 4 characters
	byteLength := (length * 3) / 4
	if (length*3)%4 != 0 {
		byteLength++
	}

	// Generate random bytes
	bytes := make([]byte, byteLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to URL-safe base64 without padding
	token := base64.RawURLEncoding.EncodeToString(bytes)

	// Trim to exact length if needed (shouldn't be necessary with proper calculation)
	if len(token) > length {
		token = token[:length]
	}

	return token, nil
}
func IsValidKenyanPhone(phone string) bool {
	re := regexp.MustCompile(`^(?:2547\d{8}|07\d{8})$`)
	return re.MatchString(phone)
}
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func IntPtr(i int) *int {
	return &i
}

func BoolPtr(b bool) *bool {
	return &b
}
