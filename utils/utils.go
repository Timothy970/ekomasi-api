// Package utils provides general utility functions for the Adenzo e-commerce platform.
//
// This file contains a wide range of utilities:
//   - Pagination helpers for category listings
//   - Token generation (OTP, reset tokens, secure tokens)
//   - Email HTML templates (OTP, orders, wishlists, carts, vouchers, low stock alerts)
//   - SMS message templates
//   - Phone number validation (Kenyan format)
//   - Type conversion utilities
//   - Pointer helpers for optional fields
//   - Time zone handling (Nairobi)
//   - Function name extraction for logging
//
// Email Templates:
//   - OTP verification emails
//   - Order confirmation and receipts
//   - Wishlist sharing and reminders
//   - Cart abandonment reminders
//   - E-voucher notifications
//   - Low stock alerts
//
// Token Generation:
//   - OTP: 4-digit codes for verification
//   - Reset tokens: Base64 encoded random bytes
//   - Secure tokens: URL-safe base64 tokens
//
// Validation:
//   - Kenyan phone numbers (07/01/2547/2541 formats)
package utils

import (
	"adenzo_backend/dtos"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"math/big"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PaginatedResponse wraps category data with pagination metadata.
//
// Used for returning paginated category listings with pagination controls.
//
// Fields:
//   - Data: Category items for current page
//   - Pagination: Metadata about pagination state
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
//   - Adenzo branding
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
      <p style="font-size: 14px; color: #aaa;">— The Adenzo Team</p>
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
func IsValidKenyanPhone(phone string) bool {
	// Regex for Kenyan phone numbers (07/01 local, +2547/+2541 international)
	re := regexp.MustCompile(`^(?:\+?2547\d{8}|\+?2541\d{8}|07\d{8}|01\d{8})$`)
	return re.MatchString(phone)
}

// StringPtr returns pointer to string, or nil if string is empty.
//
// Useful for optional database fields that need nil for empty values.
//
// Parameters:
//   - s: string - String value
//
// Returns:
//   - *string: Pointer to string, or nil if empty
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// IntPtr returns pointer to int.
//
// Useful for optional integer fields in structs.
//
// Parameters:
//   - i: int - Integer value
//
// Returns:
//   - *int: Pointer to integer
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns pointer to bool.
//
// Useful for optional boolean fields in structs.
//
// Parameters:
//   - b: bool - Boolean value
//
// Returns:
//   - *bool: Pointer to boolean
func BoolPtr(b bool) *bool {
	return &b
}

// CartReminderEmail generates HTML email for cart abandonment reminders.
//
// Creates responsive email template with:
//   - Purple gradient theme matching brand colors
//   - Prominent call-to-action button
//   - Cart link for completing purchase
//   - Support contact information
//   - Mobile-responsive design
//
// Parameters:
//   - cartLink: string - URL to user's shopping cart
//   - supportEmail: string - Support email address
//   - phone: string - Support phone number
//
// Returns:
//   - string: Complete HTML email body
func CartReminderEmail(cartLink, supportEmail, phone string) string {
	return `
	<!DOCTYPE html>
	<html lang="en">
	<head>
	  <meta charset="UTF-8">
	  <title>Cart Reminder</title>
	  <style>
	    body {
	      font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
	      background-color: #f8f6fb;
	      margin: 0;
	      padding: 0;
	      color: #333;
	    }
	    .container {
	      max-width: 600px;
	      margin: 30px auto;
	      background: #ffffff;
	      border-radius: 16px;
	      box-shadow: 0 4px 12px rgba(150, 120, 200, 0.15);
	      overflow: hidden;
	    }
	    .header {
	      background-color: #d9b3ff;
	      padding: 20px;
	      text-align: center;
	      color: #4a148c;
	      font-size: 24px;
	      font-weight: bold;
	    }
	    .content {
	      padding: 25px;
	      font-size: 16px;
	      line-height: 1.6;
	    }
	    .cta {
	      display: inline-block;
	      background: #b388ff;
	      color: #fff !important;
	      padding: 14px 28px;
	      margin: 20px 0;
	      border-radius: 8px;
	      font-size: 16px;
	      text-decoration: none;
	      font-weight: bold;
	      box-shadow: 0 3px 8px rgba(150, 120, 200, 0.25);
	      transition: background 0.3s ease;
	    }
	    .cta:hover {
	      background: #9c6cff;
	    }
	    .footer {
	      background: #f3e8ff;
	      padding: 15px;
	      text-align: center;
	      font-size: 14px;
	      color: #555;
	      border-top: 1px solid #e1c4ff;
	    }
	  </style>
	</head>
	<body>
	  <div class="container">
	    <div class="header">
	      🛒 You Forgot Something!
	    </div>
	    <div class="content">
	      <p>Hi ,</p>
	      <p>It looks like you left a few things in your shopping cart. We've saved them for you in case you'd like to come back and complete your purchase.</p>
	      <p>Ready to make them yours?</p>
	      <a href="` + cartLink + `" class="cta">Complete Your Order</a>
	      <p>If you have any questions or ran into an issue, don't hesitate to contact our support team at <a href="mailto:` + supportEmail + `">` + supportEmail + `</a> or call us at <strong>` + phone + `</strong>.</p>
	      <p>Thanks,<br>The Adenzo Team</p>
	    </div>
	    <div class="footer">
	      &copy; 2025 Adenzo. All rights reserved.
	    </div>
	  </div>
	</body>
	</html>
	`
}

// WishlistReminderEmail generates HTML email for wishlist reminders.
//
// Creates responsive email template with:
//   - Purple gradient theme
//   - Personalized greeting with customer name
//   - Wishlist link with call-to-action
//   - Support contact information
//   - Urgency messaging (items may not be available forever)
//
// Parameters:
//   - customerName: string - Customer's name for personalization
//   - wishlistLink: string - URL to customer's wishlist
//   - supportEmail: string - Support email address
//   - phone: string - Support phone number
//
// Returns:
//   - string: Complete HTML email body
func WishlistReminderEmail(customerName, wishlistLink, supportEmail, phone string) string {
	return `
	<!DOCTYPE html>
	<html lang="en">
	<head>
	  <meta charset="UTF-8">
	  <title>Wishlist Reminder</title>
	  <style>
	    body {
	      font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
	      background-color: #f8f6fb;
	      margin: 0;
	      padding: 0;
	      color: #333;
	    }
	    .container {
	      max-width: 600px;
	      margin: 30px auto;
	      background: #ffffff;
	      border-radius: 16px;
	      box-shadow: 0 4px 12px rgba(150, 120, 200, 0.15);
	      overflow: hidden;
	    }
	    .header {
	      background-color: #d9b3ff;
	      padding: 20px;
	      text-align: center;
	      color: #4a148c;
	      font-size: 24px;
	      font-weight: bold;
	    }
	    .content {
	      padding: 25px;
	      font-size: 16px;
	      line-height: 1.6;
	    }
	    .cta {
	      display: inline-block;
	      background: #b388ff;
	      color: #fff !important;
	      padding: 14px 28px;
	      margin: 20px 0;
	      border-radius: 8px;
	      font-size: 16px;
	      text-decoration: none;
	      font-weight: bold;
	      box-shadow: 0 3px 8px rgba(150, 120, 200, 0.25);
	      transition: background 0.3s ease;
	    }
	    .cta:hover {
	      background: #9c6cff;
	    }
	    .footer {
	      background: #f3e8ff;
	      padding: 15px;
	      text-align: center;
	      font-size: 14px;
	      color: #555;
	      border-top: 1px solid #e1c4ff;
	    }
	  </style>
	</head>
	<body>
	  <div class="container">
	    <div class="header">
	      💜 Your Wishlist Awaits!
	    </div>
	    <div class="content">
	      <p>Hi ` + customerName + `,</p>
	      <p>We noticed you saved some items in your wishlist. They're still waiting for you — and they might not be available forever!</p>
	      <p>Why not treat yourself today?</p>
	      <a href="` + wishlistLink + `" class="cta">View My Wishlist</a>
	      <p>If you need any assistance, reach out to our support team at <a href="mailto:` + supportEmail + `">` + supportEmail + `</a> or call us at <strong>` + phone + `</strong>.</p>
	      <p>Happy Shopping,<br>The Adenzo Team</p>
	    </div>
	    <div class="footer">
	      &copy; 2025 Adenzo. All rights reserved.
	    </div>
	  </div>
	</body>
	</html>`
}

// WishlistReminderSMS generates plain text SMS for wishlist reminders.
//
// Creates short, friendly SMS message with:
//   - Personalized greeting
//   - Wishlist link
//   - Emojis for visual appeal
//   - Brand signature
//
// Parameters:
//   - customerName: string - Customer's name
//   - wishlistLink: string - URL to wishlist (shortened recommended for SMS)
//
// Returns:
//   - string: Plain text SMS message
func WishlistReminderSMS(customerName, wishlistLink string) string {
	return fmt.Sprintf(
		"Hi %s, your wishlist is waiting 💜. Don’t miss out on your favorite items! Check it here 👉 %s. – The Adenzo Team",
		customerName, wishlistLink,
	)
}

// SendVoucherEmail sends an e-voucher email with HTML and plain-text fallback
func SendVoucherEmail(data dtos.VoucherEmailInfo) (string, string) {
	t, err := time.Parse(time.RFC3339, data.ExpiryDate)
	if err != nil {
		log.Fatal(err)
	}

	formatted := t.Format("2006 January 02 15:04")
	subject := fmt.Sprintf("🎁 You’ve received a KES %v e-voucher!", data.Amount)
	frontEndUrl := os.Getenv("FRONT_END_BASE_URL")
	// HTML body (simplified placeholder replacement)
	htmlBody := fmt.Sprintf(`
	<!doctype html>
	<html>
	<body style="font-family:sans-serif; background:#f4f4f6; padding:20px;">
	  <div style="max-width:600px; margin:auto; background:#6b3aa6; color:#fff; padding:20px; border-radius:12px;">
	    <h2 style="margin:0;">E-Voucher</h2>
	    <p>Hello %s,</p>
	    <p>You have received an e-voucher worth KES <strong style="font-size:20px;">%v</strong> from %s.</p>

	    <div style="background:#fff; color:#6b3aa6; padding:10px; margin:20px 0; border-radius:8px; text-align:center; font-weight:bold;">
	      %s
	    </div>

	    <p style="margin:0;">Expires: <strong>%s</strong></p>
	    <p style="margin:0;">Message: %s</p>

	    <p style="margin:20px 0;">
	      <a href="%s" style="display:inline-block; background:#9b59b6; color:#fff; padding:12px 20px; text-decoration:none; border-radius:6px;">
	        Redeem Your Voucher
	      </a>
	    </p>
	  </div>
	</body>
	</html>
	`, data.ToName, data.Amount, data.FromName, data.Code, formatted, data.PersonalizedMsg, frontEndUrl+"dashboard/giftcards/redeem")
	return subject, htmlBody
}

// GenerateOrderConfirmationHTML generates styled HTML email for order confirmation or receipt.
//
// This function:
// 1. Calculates totals if not provided (subtotal, total amount)
// 2. Sets color scheme based on order type
// 3. Builds order items table HTML
// 4. Replaces template placeholders with actual data
// 5. Returns complete HTML email
//
// Order Types:
//   - "order_confirmation": Purple gradient theme
//   - "order_receipt": Pink gradient theme
//
// Email Sections:
//   - Header with order type indicator
//   - Customer greeting
//   - Order details (ID, date)
//   - Items table (product, quantity, price)
//   - Totals (subtotal, shipping, discount, total)
//   - Delivery address
//
// Parameters:
//   - data: dtos.OrderEmailData - Order information including:
//   - CustomerName, OrderID, OrderDate
//   - OrderItems: Product details
//   - Subtotal, ShippingFee, Discount, TotalAmount
//   - DeliveryAddress
//   - orderType: string - "order_confirmation" or "order_receipt"
//
// Returns:
//   - string: Complete HTML email body
func GenerateOrderConfirmationHTML(data dtos.OrderEmailData, orderType string) string {
	// Calculate totals if not provided
	if data.Subtotal == 0 {
		data.Subtotal = calculateSubtotal(data.OrderItems)
	}
	if data.TotalAmount == 0 {
		data.TotalAmount = data.Subtotal + data.ShippingFee - data.Discount
	}

	// Dynamic colors and text based on order type
	bgGradient := "linear-gradient(135deg, #8B5FBF 0%, #6A3093 100%)" // Purple for confirmation
	sectionBG := "#f8f5ff"                                            // Light purple background
	headerTitle := "🎉 Order Confirmed!"
	headerSubtitle := "Thank you for your purchase"

	if strings.ToLower(orderType) == "order_receipt" {
		bgGradient = "linear-gradient(135deg, #fd90c5ff 0%, #ff6fa3 100%)" // pink gradient
		sectionBG = "#fff1f7"                                              // very light pink
		headerTitle = "🧾 Order Receipt"
		headerSubtitle = "Here is your purchase receipt"
	}

	const template = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{HeaderTitle}}</title>
<style>
	body {
		font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
		line-height: 1.6;
		color: #333;
		background-color: #f8f9fa;
		margin: 0;
		padding: 0;
	}
	.email-container {
		max-width: 800px;
		margin: 0 auto;
		background: #fff;
		border-radius: 12px;
		box-shadow: 0 4px 6px rgba(0,0,0,0.1);
		overflow: hidden;
	}
	.header {
		background: {{BGGradient}};
		color: white;
		text-align: center;
		padding: 30px;
	}
	.header h1 { font-size: 28px; margin-bottom: 8px; }
	.content { padding: 30px; }
	.order-info, .totals, .delivery-address {
		background: {{SectionBG}};
		padding: 20px;
		border-radius: 8px;
		margin: 25px 0;
	}
	.order-info h2, .delivery-address h3 { color: #6A3093; }
	.info-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 15px;
	}
	.info-label { font-weight: 600; color: #6A3093; font-size: 14px; }
	.info-value { font-size: 14px; color: #555; }
	.items-table {
		width: 100%;
		border-collapse: collapse;
		box-shadow: 0 2px 4px rgba(0,0,0,0.05);
	}
	.items-table th {
		background: #8B5FBF;
		color: white;
		padding: 15px;
		text-align: left;
	}
	.items-table td { padding: 15px; border-bottom: 1px solid #eee; }
	.items-table tr:hover { background: #f8f5ff; }
	.price { text-align: right; color: #6A3093; font-weight: 600; }
	.totals .total-row {
		display: flex;
		justify-content: space-between;
		padding: 8px 0;
		border-bottom: 1px solid #e9ecef;
	}
	.totals .total-row:last-child {
		font-weight: 700;
		font-size: 18px;
		color: #6A3093;
		border-bottom: none;
	}
	.footer {
		text-align: center;
		padding: 25px;
		background: #f8f9fa;
		font-size: 14px;
		color: #666;
	}
	.footer a { color: #8B5FBF; text-decoration: none; }
	.thank-you {
		text-align: center;
		color: #6A3093;
		font-size: 18px;
		font-weight: 600;
		margin: 25px 0;
	}
	@media (max-width: 600px) {
		.info-grid { grid-template-columns: 1fr; }
	}
</style>
</head>
<body>
	<div class="email-container">
		<div class="header">
			<h1>{{HeaderTitle}}</h1>
			<p>{{HeaderSubtitle}}</p>
		</div>

		<div class="content">
			<div class="thank-you">
				Thank you for shopping with us, {{.CustomerName}}!
			</div>

			<div class="order-info">
				<h2>Order Details</h2>
				<div class="info-grid">
					<div>
						<div class="info-label">Order ID</div>
						<div class="info-value">{{.OrderID}}</div>
					</div>

					<div>
						<div class="info-label">Order Date</div>
						<div class="info-value">{{.OrderDate}}</div>
					</div>
				</div>
			</div>

			<h2 style="color:#6A3093;margin-bottom:15px;">Order Items</h2>
			<table class="items-table">
				<thead>
					<tr>
						<th>Product</th>
						<th style="text-align:center;">Qty</th>
						<th style="text-align:right;">Price</th>
					</tr>
				</thead>
				<tbody>
					{{ItemsHTML}}
				</tbody>
			</table>

			<div class="totals">
				<div class="total-row"><span>Subtotal</span><span>KES {{Subtotal}}</span></div>
				<div class="total-row"><span>Shipping</span><span>KES {{ShippingFee}}</span></div>
				<div class="total-row"><span>Discount</span><span>-KES {{Discount}}</span></div>
				<div class="total-row"><span>Total Amount</span><span>KES {{TotalAmount}}</span></div>
			</div>

			<div class="delivery-address">
				<h3>Delivery Address</h3>
				<p>{{.DeliveryAddress}}</p>
			</div>
		</div>
	</div>
</body>
</html>`

	// === Build items HTML ===
	var itemsHTML strings.Builder
	for _, item := range data.OrderItems {
		itemsHTML.WriteString(fmt.Sprintf(`
			<tr>
				<td>%s</td>
				<td style="text-align:center;">%d</td>
				<td style="text-align:right;">KES %.2f</td>
			</tr>`,
			item.ProductName, item.Quantity, item.UnitPrice))
	}

	// === Replace marker variables ===
	html := template
	html = strings.ReplaceAll(html, "{{HeaderTitle}}", headerTitle)
	html = strings.ReplaceAll(html, "{{HeaderSubtitle}}", headerSubtitle)
	html = strings.ReplaceAll(html, "{{BGGradient}}", bgGradient)
	html = strings.ReplaceAll(html, "{{SectionBG}}", sectionBG)
	html = strings.ReplaceAll(html, "{{ItemsHTML}}", itemsHTML.String())

	html = strings.ReplaceAll(html, "{{Subtotal}}", fmt.Sprintf("%.2f", data.Subtotal))
	html = strings.ReplaceAll(html, "{{ShippingFee}}", fmt.Sprintf("%.2f", data.ShippingFee))
	html = strings.ReplaceAll(html, "{{Discount}}", fmt.Sprintf("%.2f", data.Discount))
	html = strings.ReplaceAll(html, "{{TotalAmount}}", fmt.Sprintf("%.2f", data.TotalAmount))

	// basic replacements for fields
	html = strings.ReplaceAll(html, "{{.CustomerName}}", data.CustomerName)
	html = strings.ReplaceAll(html, "{{.OrderID}}", data.OrderID)
	html = strings.ReplaceAll(html, "{{.OrderDate}}", data.OrderDate)
	html = strings.ReplaceAll(html, "{{.DeliveryAddress}}", data.DeliveryAddress)

	return html
}

// GenerateLowStockAlertHTML generates HTML email for low stock inventory alerts.
//
// This function:
// 1. Builds email template with red gradient theme
// 2. Injects store name and alert date
// 3. Builds product table rows with images
// 4. Replaces template placeholders with generated HTML
// 5. Returns complete email with product details
//
// Email Contents:
//   - Red gradient header (alert urgency)
//   - Alert date and store name
//   - Products table (image, name, SKU, quantity)
//   - Restock reminder message
//
// Parameters:
//   - data: dtos.LowStockEmailData - Alert information including:
//   - StoreName: Store identifier
//   - AlertDate: When alert was generated
//   - Products: Array of low stock products with images
//
// Returns:
//   - string: Complete HTML email body
func GenerateLowStockAlertHTML(data dtos.LowStockEmailData) string {
	const template = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Low Stock Alert</title>
<style>
    body {
        font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        background-color: #f8f9fa;
        margin: 0; padding: 0;
        color: #333;
    }
    .email-container {
        max-width: 900px;
        margin: 0 auto;
        background: #fff;
        border-radius: 12px;
        box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        overflow: hidden;
    }
    .header {
        background: linear-gradient(135deg, #D7263D 0%, #8A0F23 100%);
        color: white;
        text-align: center;
        padding: 30px;
    }
    .header h1 { margin: 0; font-size: 28px; }
    .content { padding: 30px; }
    .alert-box {
        background: #fff4f4;
        border-left: 6px solid #D7263D;
        padding: 20px;
        border-radius: 8px;
        margin-bottom: 30px;
    }
    .alert-text {
        color: #8A0F23;
        font-size: 18px;
        font-weight: 600;
    }
    table {
        width: 100%;
        border-collapse: collapse;
        margin-top: 20px;
        box-shadow: 0 2px 4px rgba(0,0,0,0.05);
    }
    th {
        background: #D7263D;
        color: white;
        padding: 14px;
        text-align: left;
    }
    td {
        padding: 14px;
        border-bottom: 1px solid #eee;
        vertical-align: middle;
    }
    .product-img {
        width: 55px;
        height: 55px;
        border-radius: 6px;
        object-fit: cover;
        border: 1px solid #ddd;
    }
    tr:hover { background: #fff4f4; }
    .footer {
        text-align: center;
        padding: 20px;
        background: #f8f9fa;
        font-size: 14px;
        color: #666;
    }
</style>
</head>
<body>
    <div class="email-container">
        <div class="header">
            <h1>⚠️ Low Stock Alert</h1>
            <p>{{.StoreName}} — {{.AlertDate}}</p>
        </div>

        <div class="content">

            <div class="alert-box">
                <div class="alert-text">
                    The following products are running low on stock:
                </div>
            </div>

            <table>
                <thead>
                    <tr>
                        <th>Image</th>
                        <th>Product</th>
                        <th>SKU</th>
                        <th style="text-align:center;">Stock Available</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Products}}
                    <tr>
                        <td><img src="{{.ImageURL}}" class="product-img"/></td>
                        <td>{{.Name}}</td>
                        <td>{{.SKU}}</td>
                        <td style="text-align:center;">{{.StockQuantity}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>

        </div>

        <div class="footer">
            <p>Please restock soon to avoid stockouts.</p>
            <p>© 2025 Your Company Name. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`

	// Inject store name and date
	html := strings.NewReplacer(
		"{{.StoreName}}", data.StoreName,
		"{{.AlertDate}}", data.AlertDate,
	).Replace(template)

	// Build rows
	var itemsHTML strings.Builder
	for _, p := range data.Products {
		imageURL := ""
		if len(p.Images) > 0 {
			imageURL = p.Images[0].URL // First product URL
		}

		itemsHTML.WriteString(fmt.Sprintf(`
                    <tr>
                        <td><img src="%s" class="product-img"/></td>
                        <td>%s</td>
                        <td>%s</td>
                        <td style="text-align:center;">%d</td>
                    </tr>`,
			imageURL, p.Name, p.SKU, p.StockQuantity))
	}

	// Replace exact block (must match template indentation exactly)
	block := `
                    {{range .Products}}
                    <tr>
                        <td><img src="{{.ImageURL}}" class="product-img"/></td>
                        <td>{{.Name}}</td>
                        <td>{{.SKU}}</td>
                        <td style="text-align:center;">{{.StockQuantity}}</td>
                    </tr>
                    {{end}}`

	html = strings.Replace(html, block, itemsHTML.String(), 1)

	return html
}

// calculateSubtotal computes the subtotal from order items.
//
// Multiplies quantity by unit price for each item and sums the results.
//
// Parameters:
//   - items: []dtos.OrderNotificationItemRequest - Array of order items
//
// Returns:
//   - float64: Total subtotal before shipping and discounts
func calculateSubtotal(items []dtos.OrderNotificationItemRequest) float64 {
	var subtotal float64
	// Sum up (quantity * unit price) for each item
	for _, item := range items {
		subtotal += float64(item.Quantity) * item.UnitPrice
	}
	return subtotal
}

// WishlistItem represents a product item in a wishlist email.
//
// Fields:
//   - Title: Product name
//   - ImageURL: Product image URL
//   - Price: Formatted price string (e.g., "KES 2,999")
//   - ProductURL: Link to product page
type WishlistItem struct {
	Title      string
	ImageURL   string
	Price      string // e.g. "$29.99"
	ProductURL string
}

// wishlistTemplateData holds data for wishlist email template execution.
//
// Internal type used by GenerateWishlistEmailHTML for template rendering.
//
// Fields:
//   - SenderName: Person sharing wishlist
//   - SenderDetails: Contact info for sender
//   - PersonalNote: Custom message from sender
//   - ShareURL: Link to full wishlist
//   - Items: Products to display in email
//   - ShowCount: Number of items shown
//   - GeneratedAt: Formatted generation date
//   - HasMore: true if more items exist beyond preview
//   - CurrentYear: For copyright footer
type wishlistTemplateData struct {
	SenderName    string
	SenderDetails string
	PersonalNote  string
	ShareURL      string
	Items         []WishlistItem
	ShowCount     int
	GeneratedAt   string
	HasMore       bool
	CurrentYear   int
}

// GenerateWishlistEmailHTML generates HTML email for wishlist sharing.
//
// This function:
// 1. Limits preview to first 3 items (maxPreview)
// 2. Sets HasMore flag if more items exist
// 3. Prepares template data with current year
// 4. Executes HTML template with data
// 5. Returns complete HTML email
//
// Email Features:
//   - Purple gradient theme
//   - Sender name and details
//   - Personal note from sender
//   - Product preview (max 3 items)
//   - "View Full Wishlist" button
//   - Responsive design
//
// Parameters:
//   - senderName: string - Name of wishlist sender
//   - senderDetails: string - Contact info for sender
//   - personalMessage: string - Custom message
//   - shareURL: string - Link to full wishlist
//   - items: []WishlistItem - Products to display
//
// Returns:
//   - string: Complete HTML email body
//   - error: Template execution error
func GenerateWishlistEmailHTML(senderName, senderDetails, personalMessage, shareURL string, items []WishlistItem) (string, error) {
	const maxPreview = 3 // Show max 3 items in email preview
	// Determine how many items to show
	showCount := len(items)
	hasMore := false
	if len(items) > maxPreview {
		showCount = maxPreview // Limit to 3 items
		hasMore = true         // Flag for "more items" message
	}
	currentYear := time.Now().Year()
	// Prepare template data
	data := wishlistTemplateData{
		SenderName:    senderName,
		SenderDetails: senderDetails,
		PersonalNote:  personalMessage,
		ShareURL:      shareURL,
		Items:         items[:showCount], // Slice to max preview
		ShowCount:     showCount,
		HasMore:       hasMore,
		GeneratedAt:   time.Now().Format("January 2, 2006"),
		CurrentYear:   currentYear,
	}

	// Parse and execute template
	tpl := template.Must(template.New("wishlistEmail").Parse(emailTemplate))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// emailTemplate is the HTML template for wishlist sharing emails.
//
// Contains responsive HTML with Go template placeholders:
//   - {{.SenderName}}, {{.SenderDetails}}, {{.PersonalNote}}
//   - {{.ShareURL}}, {{.GeneratedAt}}, {{.CurrentYear}}
//   - {{range .Items}} for product iteration
//   - {{if .HasMore}} for conditional sections
const emailTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.SenderName}}'s Wishlist</title>
<style>
	body {
		font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
		line-height: 1.6;
		color: #333;
		background-color: #f8f9fa;
		margin: 0;
		padding: 0;
	}
	.email-container {
		max-width: 800px;
		margin: 0 auto;
		background: #fff;
		border-radius: 12px;
		box-shadow: 0 4px 6px rgba(0,0,0,0.1);
		overflow: hidden;
	}
	.header {
		background: linear-gradient(135deg, #8B5FBF 0%, #6A3093 100%);
		color: white;
		text-align: center;
		padding: 30px;
	}
	.header h1 { font-size: 28px; margin-bottom: 8px; }
	.header .sender-contact {
		font-size: 14px;
		opacity: 0.9;
		margin-top: 8px;
	}
	.content { padding: 30px; }
	.wishlist-info {
		background: #f8f5ff;
		padding: 20px;
		border-radius: 8px;
		margin: 25px 0;
	}
	.wishlist-info h2 { color: #6A3093; margin-top: 0; }
	.personal-note {
		font-style: italic;
		color: #555;
		margin-top: 15px;
		padding: 15px;
		background: #fff;
		border-left: 4px solid #8B5FBF;
		border-radius: 4px;
	}
	.items-table {
		width: 100%;
		border-collapse: collapse;
		box-shadow: 0 2px 4px rgba(0,0,0,0.05);
		margin: 25px 0;
	}
	.items-table th {
		background: #8B5FBF;
		color: white;
		padding: 15px;
		text-align: left;
	}
	.items-table td { 
		padding: 15px; 
		border-bottom: 1px solid #eee;
		vertical-align: middle;
	}
	.items-table tr:hover { background: #f8f5ff; }
	.item-image {
		width: 80px;
		height: 80px;
		border-radius: 8px;
		object-fit: cover;
	}
	.no-image {
		width: 80px;
		height: 80px;
		background: linear-gradient(135deg, #fae8ff, #e9d5ff);
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #a855f7;
		font-size: 12px;
		text-align: center;
	}
	.price { 
		text-align: right; 
		color: #6A3093; 
		font-weight: 600;
		font-size: 16px;
	}
	.view-button-container {
		text-align: center;
		margin: 30px 0;
	}
	.view-button {
		display: inline-block;
		background: linear-gradient(135deg, #8B5FBF 0%, #6A3093 100%);
		color: white;
		padding: 15px 40px;
		border-radius: 8px;
		text-decoration: none;
		font-weight: 600;
		font-size: 16px;
		box-shadow: 0 4px 12px rgba(106, 48, 147, 0.3);
	}
	.more-items {
		text-align: center;
		color: #666;
		font-style: italic;
		margin: 20px 0;
	}
	.footer {
		text-align: center;
		padding: 25px;
		background: #f8f9fa;
		font-size: 14px;
		color: #666;
	}
	.footer a { color: #8B5FBF; text-decoration: none; }
	.thank-you {
		text-align: center;
		color: #6A3093;
		font-size: 18px;
		font-weight: 600;
		margin: 25px 0;
	}
	@media (max-width: 600px) {
		.item-image, .no-image { width: 60px; height: 60px; }
	}
</style>
</head>
<body>
	<div class="email-container">
		<div class="header">
			<h1>💜 Wishlist from {{.SenderName}}</h1>
			{{if .SenderDetails}}
			<div class="sender-contact">{{.SenderDetails}}</div>
			{{end}}
		</div>

		<div class="content">
			<div class="thank-you">
				Would you love these picks?
			</div>

			{{if .PersonalNote}}
			<div class="personal-note">
				"{{.PersonalNote}}"
			</div>
			{{end}}

			<h2 style="color:#6A3093;margin-bottom:15px;">Wishlist Items</h2>
			<table class="items-table">
				<thead>
					<tr>
						<th style="width:100px;">Image</th>
						<th>Product</th>
						<th style="text-align:right;">Price</th>
					</tr>
				</thead>
				<tbody>
					{{range .Items}}
					<tr>
						<td>
							{{if .ImageURL}}
							<a href="{{.ProductURL}}" target="_blank">
								<img src="{{.ImageURL}}" alt="{{.Title}}" class="item-image">
							</a>
							{{else}}
							<div class="no-image">No image</div>
							{{end}}
						</td>
						<td>
							<a href="{{.ProductURL}}" target="_blank" style="color:#333;text-decoration:none;font-weight:600;">{{.Title}}</a>
						</td>
						<td class="price">{{.Price}}</td>
					</tr>
					{{end}}
				</tbody>
			</table>

			{{if .HasMore}}
			<div class="more-items">
				Plus more amazing items waiting for you...
			</div>
			{{end}}

			<div class="view-button-container">
				<a href="{{.ShareURL}}" class="view-button" target="_blank">View Full Wishlist</a>
			</div>
		</div>

		<div class="footer">
			<p>Shared on {{.GeneratedAt}} • <a href="{{.ShareURL}}" target="_blank">Open in browser</a></p>
			<p>© {{.CurrentYear}} Adenzo. All rights reserved.</p>
		</div>
	</div>
</body>
</html>`

// GenerateOrderAssignmentEmailContent generates HTML email for order assignment notifications.
//
// This function creates an email to notify riders/delivery personnel when they are
// assigned to deliver an order. The email includes order details and a link to view
// the full order information.
//
// Parameters:
//   - orderID: string - Unique order identifier
//   - riderName: string - Name of the assigned rider/delivery person
//   - customerName: string - Name of the customer who placed the order
//   - deliveryAddress: string - Delivery destination address
//   - orderDate: string - Date when order was placed
//
// Returns:
//   - string: Complete HTML email body with order assignment details
func GenerateOrderAssignmentEmailContent(orderID, riderName, customerName, deliveryAddress, orderDate string) string {
	frontEndUrl := os.Getenv("FRONT_END_BASE_URL")
	if frontEndUrl == "" {
		frontEndUrl = "https://adenzo.com" // fallback
	}
	orderDetailsURL := fmt.Sprintf("%s/orders/details/%s", frontEndUrl, orderID)

	const template = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Order Assignment</title>
<style>
	body {
		font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
		line-height: 1.6;
		color: #333;
		background-color: #f8f9fa;
		margin: 0;
		padding: 0;
	}
	.email-container {
		max-width: 800px;
		margin: 0 auto;
		background: #fff;
		border-radius: 12px;
		box-shadow: 0 4px 6px rgba(0,0,0,0.1);
		overflow: hidden;
	}
	.header {
		background: linear-gradient(135deg, #4F46E5 0%, #7C3AED 100%);
		color: white;
		text-align: center;
		padding: 30px;
	}
	.header h1 { font-size: 28px; margin-bottom: 8px; }
	.content { padding: 30px; }
	.order-info, .delivery-info {
		background: #EEF2FF;
		padding: 20px;
		border-radius: 8px;
		margin: 25px 0;
	}
	.order-info h2, .delivery-info h3 { color: #4F46E5; }
	.info-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 15px;
	}
	.info-label { font-weight: 600; color: #4F46E5; font-size: 14px; }
	.info-value { font-size: 14px; color: #555; }
	.greeting {
		text-align: center;
		color: #4F46E5;
		font-size: 20px;
		font-weight: 600;
		margin: 25px 0;
	}
	.message {
		background: #F0FDF4;
		border-left: 4px solid #10B981;
		padding: 15px 20px;
		margin: 25px 0;
		border-radius: 4px;
	}
	.message p {
		margin: 0;
		color: #166534;
		font-weight: 500;
	}
	.action-button {
		text-align: center;
		margin: 30px 0;
	}
	.action-button a {
		display: inline-block;
		background: linear-gradient(135deg, #4F46E5 0%, #7C3AED 100%);
		color: white;
		padding: 15px 40px;
		text-decoration: none;
		border-radius: 8px;
		font-weight: 600;
		font-size: 16px;
		box-shadow: 0 4px 6px rgba(79, 70, 229, 0.3);
		transition: transform 0.2s;
	}
	.action-button a:hover {
		transform: translateY(-2px);
		box-shadow: 0 6px 12px rgba(79, 70, 229, 0.4);
	}
	.delivery-info .address {
		background: white;
		padding: 15px;
		border-radius: 6px;
		margin-top: 10px;
		border: 1px solid #E0E7FF;
	}
	.footer {
		text-align: center;
		padding: 25px;
		background: #f8f9fa;
		font-size: 14px;
		color: #666;
	}
	.footer a { color: #4F46E5; text-decoration: none; }
	@media (max-width: 600px) {
		.info-grid { grid-template-columns: 1fr; }
	}
</style>
</head>
<body>
	<div class="email-container">
		<div class="header">
			<h1>🚚 New Order Assignment</h1>
			<p>You have been assigned a new delivery</p>
		</div>

		<div class="content">
			<div class="greeting">
				Hello {{RiderName}}!
			</div>

			<div class="message">
				<p>You have been assigned to deliver order {{OrderID}}. Please review the order details below and proceed with the delivery.</p>
			</div>

			<div class="order-info">
				<h2>Order Information</h2>
				<div class="info-grid">
					<div>
						<div class="info-label">Order ID</div>
						<div class="info-value">{{OrderID}}</div>
					</div>
					<div>
						<div class="info-label">Order Date</div>
						<div class="info-value">{{OrderDate}}</div>
					</div>
					<div>
						<div class="info-label">Customer Name</div>
						<div class="info-value">{{CustomerName}}</div>
					</div>
					<div>
						<div class="info-label">Assigned Rider</div>
						<div class="info-value">{{RiderName}}</div>
					</div>
				</div>
			</div>

			<div class="delivery-info">
				<h3>Delivery Address</h3>
				<div class="address">
					<p>{{DeliveryAddress}}</p>
				</div>
			</div>

			<div class="action-button">
				<a href="{{OrderDetailsURL}}" target="_blank">View Order Details</a>
			</div>

			<div style="text-align:center; color:#666; font-size:14px; margin-top:25px;">
				<p>Please ensure the order is delivered safely and on time.</p>
			</div>
		</div>

		<div class="footer">
			<p>Need help? <a href="mailto:support@adenzo.com">Contact Support</a></p>
			<p>© 2026 Adenzo. All rights reserved.</p>
		</div>
	</div>
</body>
</html>`

	// Replace placeholders
	html := template
	html = strings.ReplaceAll(html, "{{OrderID}}", orderID)
	html = strings.ReplaceAll(html, "{{RiderName}}", riderName)
	html = strings.ReplaceAll(html, "{{CustomerName}}", customerName)
	html = strings.ReplaceAll(html, "{{DeliveryAddress}}", deliveryAddress)
	html = strings.ReplaceAll(html, "{{OrderDate}}", orderDate)
	html = strings.ReplaceAll(html, "{{OrderDetailsURL}}", orderDetailsURL)

	return html
}
