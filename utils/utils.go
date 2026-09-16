package utils

import (
	"crypto/rand"
	"ekomasi_backend/dtos"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type PaginatedResponse struct {
	Data       []dtos.CategoryWithProducts `json:"data"`
	Pagination PaginationMeta              `json:"pagination"`
}

// PaginationMeta contains pagination state and navigation information.
//
// Provides clients with:
//   - Current page position
//   - Total items and pages
//   - Navigation flags (has previous/next)
//   - Optional previous/next page numbers
//
// Fields:
//   - Page: Current page number (1-based)
//   - Size: Items per page
//   - TotalItems: Total number of items across all pages
//   - TotalPages: Total number of pages
//   - HasPrev: true if previous page exists
//   - HasNext: true if next page exists
//   - PrevPage: Previous page number (omitted if HasPrev is false)
//   - NextPage: Next page number (omitted if HasNext is false)
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

// ExtractToken extracts the JWT token from the Authorization header.
//
// Removes the "Bearer " prefix from the Authorization header value.
// Returns empty string if header doesn't start with "Bearer ".
//
// Parameters:
//   - authHeader: string - Authorization header value (e.g., "Bearer eyJhbGc...")
//
// Returns:
//   - string: JWT token without "Bearer " prefix, or empty string
func ExtractToken(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

// helper funcfion to convert to string
func ToString(value any) string {
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

// GenerateResetPasswordToken generates a cryptographically secure random token.
//
// Creates a URL-safe base64-encoded token of specified byte length.
// Used for password reset links and secure verification tokens.
//
// Parameters:
//   - n: int - Number of random bytes to generate
//
// Returns:
//   - string: Base64-encoded token
//   - error: Random number generation error
func GenerateResetPasswordToken(n int) (string, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// NowInNairobi returns the current time in Africa/Nairobi timezone (EAT - UTC+3).
//
// Loads the Africa/Nairobi timezone and returns current time in that zone.
// Fatal error if timezone cannot be loaded.
//
// Returns:
//   - time.Time: Current time in Nairobi timezone
func NowInNairobi() time.Time {
	loc, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		log.Fatal("Failed to load Nairobi timezone:", err)
	}
	return time.Now().In(loc)
}

// PaginateCategories paginates category data and returns page slice with metadata.
//
// This function:
// 1. Calculates start and end indices based on page and size
// 2. Handles boundary conditions (start >= total, end > total)
// 3. Extracts data slice for current page
// 4. Calculates total pages using ceiling division
// 5. Generates pagination metadata with navigation flags
//
// Parameters:
//   - data: []dtos.CategoryWithProducts - Full category list to paginate
//   - page: int - Current page number (1-based)
//   - size: int - Number of items per page
//
// Returns:
//   - PaginatedResponse: Data slice for current page with pagination metadata
func PaginateCategories(data []dtos.CategoryWithProducts, page, size int) PaginatedResponse {
	// Calculate total items
	total := len(data)
	// Calculate slice boundaries
	start := (page - 1) * size
	end := start + size

	// Handle boundary conditions
	if start >= total {
		start = total // Empty page if beyond data
	}
	if end > total {
		end = total // Cap at data length
	}

	// Extract data slice for current page
	paginatedData := data[start:end]
	// Calculate total pages (ceiling division)
	totalPages := int(math.Ceil(float64(total) / float64(size)))

	// Build pagination metadata
	meta := PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,          // Previous page exists
		HasNext:    page < totalPages, // Next page exists
	}

	// Add optional prev/next page numbers
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

// GenerateOTP generates a random 4-digit one-time password.
//
// Creates cryptographically secure random number between 0000-9999.
// Pads with leading zeros to ensure 4-digit format.
//
// Returns:
//   - string: 4-digit OTP (e.g., "0042", "5831")
//   - error: Random number generation error
func GenerateOTP() (string, error) {
	// Generate random number from 0 to 9999
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}
	// Format with leading zeros to ensure 4 digits
	return fmt.Sprintf("%04d", n.Int64()), nil
}

// GenerateOTPEmailHTML returns styled HTML content for OTP verification email.
//
// Creates responsive email template with:
//   - Centered layout with white card on gray background
//   - Large, bold OTP display
//   - 5-minute expiry notice
//   - Security warning message
//   - Ekomasi branding
//
// Parameters:
//   - otp: string - 4-digit OTP to display
//
// Returns:
//   - string: Complete HTML email body
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
      <p style="font-size: 14px; color: #aaa;">— The Ekomasi Team</p>
    </div>
  </body>
</html>
`, otp)
}

// GetCurrentFuncName returns the name of the calling function.
//
// Uses runtime package to inspect call stack and extract function name.
// Returns "unknown" if call stack cannot be inspected.
// Strips package path to return only function name.
//
// Returns:
//   - string: Function name without package path, or "unknown"
func GetCurrentFuncName() string {
	// Get program counter at caller's position
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}

	// Get function from program counter
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	// Extract function name without package path
	fullName := fn.Name()
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1] // Return last part (function name)
}

// GenerateSecureTokenBase64 generates a cryptographically secure URL-safe base64 token.
//
// This function:
// 1. Validates token length is positive
// 2. Calculates required byte length (base64 encodes 3 bytes to 4 chars)
// 3. Generates random bytes using crypto/rand
// 4. Encodes to URL-safe base64 without padding
// 5. Trims to exact length if needed
//
// Token Properties:
//   - Cryptographically secure randomness
//   - URL-safe characters (no +, /, =)
//   - No padding characters
//   - Specified exact length
//
// Parameters:
//   - length: int - Desired token length in characters
//
// Returns:
//   - string: URL-safe base64 token
//   - error: Invalid length or random generation error
func GenerateSecureTokenBase64(length int) (string, error) {
	// Validate length
	if length <= 0 {
		return "", errors.New("token length must be positive")
	}

	// Calculate byte length needed for base64 encoding
	// Base64 encodes 3 bytes into 4 characters
	byteLength := (length * 3) / 4
	if (length*3)%4 != 0 {
		byteLength++ // Round up for partial bytes
	}

	// Generate cryptographically secure random bytes
	bytes := make([]byte, byteLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to URL-safe base64 without padding
	token := base64.RawURLEncoding.EncodeToString(bytes)

	// Trim to exact length if needed
	if len(token) > length {
		token = token[:length]
	}

	return token, nil
}

// IsValidKenyanPhone validates Kenyan phone number formats.
//
// Accepts these formats:
//   - +2547XXXXXXXX (international with 07)
//   - +2541XXXXXXXX (international with 01)
//   - 07XXXXXXXX (local mobile)
//   - 01XXXXXXXX (local landline)
//
// Parameters:
//   - phone: string - Phone number to validate
//
// Returns:
//   - bool: true if valid Kenyan format, false otherwise
