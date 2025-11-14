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
	// added for numbers starting with 2541 or 01
	re := regexp.MustCompile(`^(?:\+?2547\d{8}|\+?2541\d{8}|07\d{8}|01\d{8})$`)
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

// Wishlist Reminder Email (returns HTML string)
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

// Wishlist Reminder SMS (plain text)
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
	`, data.ToName, data.Amount, data.FromName, data.Code, formatted, data.PersonalizedMsg, "https://uat.app.adenzo.co.ke")
	return subject, htmlBody
}

// Body for order placements Email
// GenerateOrderConfirmationHTML generates a styled HTML email for order confirmation.
func GenerateOrderConfirmationHTML(data dtos.OrderEmailData) string {
	// Calculate totals if not provided
	if data.Subtotal == 0 {
		data.Subtotal = calculateSubtotal(data.OrderItems)
	}
	if data.TotalAmount == 0 {
		data.TotalAmount = data.Subtotal + data.ShippingFee - data.Discount
	}

	const template = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Order Confirmation</title>
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
	.content { padding: 30px; }
	.order-info, .totals, .delivery-address {
		background: #f8f5ff;
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
			<h1>🎉 Order Confirmed!</h1>
			<p>Thank you for your purchase</p>
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
					{{range .OrderItems}}
					<tr>
						<td>{{.ProductName}}</td>
						<td style="text-align:center;">{{.Quantity}}</td>
						<td class="price">KES {{printf "%.2f" .UnitPrice}}</td>
					</tr>
					{{end}}
				</tbody>
			</table>

			<div class="totals">
				<div class="total-row"><span>Subtotal</span><span>KES {{printf "%.2f" .Subtotal}}</span></div>
				<div class="total-row"><span>Shipping</span><span>KES {{printf "%.2f" .ShippingFee}}</span></div>
				<div class="total-row"><span>Discount</span><span>-KES {{printf "%.2f" .Discount}}</span></div>
				<div class="total-row"><span>Total Amount</span><span>KES {{printf "%.2f" .TotalAmount}}</span></div>
			</div>

			<div class="delivery-address">
				<h3>Delivery Address</h3>
				<p>{{.DeliveryAddress}}</p>
			</div>
		</div>

		<div class="footer">
			<p>If you have any questions, contact our <a href="mailto:support@example.com">customer support</a>.</p>
			<p>© 2025 Your Company Name. All rights reserved.</p>
		</div>
	</div>
</body>
</html>`

	// Build HTML manually (for production, prefer html/template)
	html := strings.NewReplacer(
		"{{.CustomerName}}", data.CustomerName,
		"{{.OrderID}}", data.OrderID,
		"{{.OrderDate}}", data.OrderDate,
		"{{.DeliveryAddress}}", data.DeliveryAddress,
	).Replace(template)

	// Replace totals
	html = strings.NewReplacer(
		"{{printf \"%.2f\" .Subtotal}}", fmt.Sprintf("%.2f", data.Subtotal),
		"{{printf \"%.2f\" .ShippingFee}}", fmt.Sprintf("%.2f", data.ShippingFee),
		"{{printf \"%.2f\" .Discount}}", fmt.Sprintf("%.2f", data.Discount),
		"{{printf \"%.2f\" .TotalAmount}}", fmt.Sprintf("%.2f", data.TotalAmount),
	).Replace(html)

	// Build order items HTML
	var itemsHTML strings.Builder
	for _, item := range data.OrderItems {
		itemsHTML.WriteString(fmt.Sprintf(`
			<tr>
				<td>%s</td>
				<td style="text-align:center;">%d</td>
				<td class="price">KES %.2f</td>
			</tr>`,
			item.ProductName, item.Quantity, item.UnitPrice))
	}

	// Inject items into template
	html = strings.ReplaceAll(html,
		`{{range .OrderItems}}
					<tr>
						<td>{{.ProductName}}</td>
						<td style="text-align:center;">{{.Quantity}}</td>
						<td class="price">KES {{printf "%.2f" .UnitPrice}}</td>
					</tr>
					{{end}}`,
		itemsHTML.String(),
	)

	return html
}

// calculateSubtotal computes subtotal from order items.
func calculateSubtotal(items []dtos.OrderNotificationItemRequest) float64 {
	var subtotal float64
	for _, item := range items {
		subtotal += float64(item.Quantity) * item.UnitPrice
	}
	return subtotal
}

type WishlistItem struct {
	Title      string
	ImageURL   string
	Price      string // e.g. "$29.99"
	ProductURL string
}

type wishlistTemplateData struct {
	WishlistName string
	PersonalNote string
	ShareURL     string
	Items        []WishlistItem
	ShowCount    int
	GeneratedAt  string
	HasMore      bool
}

func GenerateWishlistEmailHTML(wishlistName, personalMessage, shareURL string, items []WishlistItem) (string, error) {
	const maxPreview = 3
	showCount := len(items)
	hasMore := false
	if len(items) > maxPreview {
		showCount = maxPreview
		hasMore = true
	}

	data := wishlistTemplateData{
		WishlistName: wishlistName,
		PersonalNote: personalMessage,
		ShareURL:     shareURL,
		Items:        items[:showCount],
		ShowCount:    showCount,
		HasMore:      hasMore,
		GeneratedAt:  time.Now().Format("January 2, 2006"),
	}

	tpl := template.Must(template.New("wishlistEmail").Parse(emailTemplate))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const emailTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.WishlistName}} — Shared Wishlist</title>
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
			<h1>{{.WishlistName}}</h1>
			<p>A special collection shared with you</p>
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
			<p>© 2025 Your Company Name. All rights reserved.</p>
		</div>
	</div>
</body>
</html>`
