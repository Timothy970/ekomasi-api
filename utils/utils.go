package utils

import (
	"adenzo_backend/dtos"
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
// GenerateOrderConfirmationHTML generates a purple-themed order confirmation email
func GenerateOrderConfirmationHTML(data dtos.OrderEmailData) string {
	// Calculate totals if not provided
	if data.Subtotal == 0 {
		data.Subtotal = calculateSubtotal(data.OrderItems)
	}
	if data.TotalAmount == 0 {
		data.TotalAmount = data.Subtotal + data.ShippingFee - data.Discount
	}

	template := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Order Confirmation</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            background-color: #f8f9fa;
        }
        
        .email-container {
            max-width: 800px;
            margin: 0 auto;
            background: #ffffff;
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        
        .header {
            background: linear-gradient(135deg, #8B5FBF 0%, #6A3093 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .header h1 {
            font-size: 28px;
            font-weight: 600;
            margin-bottom: 8px;
        }
        
        .header p {
            font-size: 16px;
            opacity: 0.9;
        }
        
        .content {
            padding: 30px;
        }
        
        .order-info {
            background: #f8f5ff;
            border-left: 4px solid #8B5FBF;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 25px;
        }
        
        .order-info h2 {
            color: #6A3093;
            font-size: 18px;
            margin-bottom: 15px;
        }
        
        .info-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }
        
        .info-item {
            margin-bottom: 8px;
        }
        
        .info-label {
            font-weight: 600;
            color: #6A3093;
            font-size: 14px;
        }
        
        .info-value {
            color: #555;
            font-size: 14px;
        }
        
        .order-items {
            margin: 25px 0;
        }
        
        .items-table {
            width: 100%;
            border-collapse: collapse;
            background: #fff;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
        }
        
        .items-table th {
            background: #8B5FBF;
            color: white;
            padding: 15px;
            text-align: left;
            font-weight: 600;
            font-size: 14px;
        }
        
        .items-table td {
            padding: 15px;
            border-bottom: 1px solid #eee;
        }
        
        .items-table tr:last-child td {
            border-bottom: none;
        }
        
        .items-table tr:hover {
            background: #f8f5ff;
        }
        
        .product-name {
            font-weight: 600;
            color: #333;
        }
        
        .quantity {
            text-align: center;
            color: #666;
        }
        
        .price {
            text-align: right;
            color: #6A3093;
            font-weight: 600;
        }
        
        .totals {
            background: #f8f5ff;
            padding: 20px;
            border-radius: 8px;
            margin: 25px 0;
        }
        
        .total-row {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid #e9ecef;
        }
        
        .total-row:last-child {
            border-bottom: none;
            font-weight: 700;
            font-size: 18px;
            color: #6A3093;
        }
        
        .total-label {
            color: #555;
        }
        
        .total-value {
            font-weight: 600;
        }
        
        .delivery-address {
            background: #f8f5ff;
            padding: 20px;
            border-radius: 8px;
            margin: 25px 0;
        }
        
        .delivery-address h3 {
            color: #6A3093;
            margin-bottom: 10px;
            font-size: 16px;
        }
        
        .footer {
            background: #f8f9fa;
            padding: 25px;
            text-align: center;
            color: #666;
            font-size: 14px;
        }
        
        .footer a {
            color: #8B5FBF;
            text-decoration: none;
        }
        
        .thank-you {
            text-align: center;
            margin: 25px 0;
            color: #6A3093;
            font-size: 18px;
            font-weight: 600;
        }
        
        @media (max-width: 600px) {
            .content {
                padding: 20px;
            }
            
            .info-grid {
                grid-template-columns: 1fr;
            }
            
            .items-table th,
            .items-table td {
                padding: 10px;
            }
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
                    <div class="info-item">
                        <div class="info-label">Order ID</div>
                        <div class="info-value">{{.OrderID}}</div>
                    </div>
                    <div class="info-item">
                        <div class="info-label">Order Date</div>
                        <div class="info-value">{{.OrderDate}}</div>
                    </div>
                </div>
            </div>
            
            <div class="order-items">
                <h2 style="color: #6A3093; margin-bottom: 15px;">Order Items</h2>
                <table class="items-table">
                    <thead>
                        <tr>
                            <th>Product</th>
                            <th style="text-align: center;">Qty</th>
                            <th style="text-align: right;">Price</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .OrderItems}}
                        <tr>
                            <td class="product-name">{{.ProductName}}</td>
                            <td class="quantity">{{.Quantity}}</td>
                            <td class="price">KES {{printf "%.2f" .UnitPrice}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
            
            <div class="totals">
                <div class="total-row">
                    <span class="total-label">Subtotal</span>
                    <span class="total-value">KES {{printf "%.2f" .Subtotal}}</span>
                </div>
                <div class="total-row">
                    <span class="total-label">Shipping</span>
                    <span class="total-value">KES {{printf "%.2f" .ShippingFee}}</span>
                </div>
                <div class="total-row">
                    <span class="total-label">Discount</span>
                    <span class="total-value">-KES {{printf "%.2f" .Discount}}</span>
                </div>
                <div class="total-row">
                    <span class="total-label">Total Amount</span>
                    <span class="total-value">KES {{printf "%.2f" .TotalAmount}}</span>
                </div>
            </div>
            
            <div class="delivery-address">
                <h3>Delivery Address</h3>
                <p>{{.DeliveryAddress}}</p>
            </div>
        </div>
        
        <div class="footer">
            <p>If you have any questions, please contact our <a href="mailto:support@example.com">customer support</a>.</p>
            <p>© 2025 Your Company Name. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`

	// Simple template execution (you might want to use html/template for production)
	html := strings.ReplaceAll(template, "{{.CustomerName}}", data.CustomerName)
	html = strings.ReplaceAll(html, "{{.OrderID}}", data.OrderID)
	html = strings.ReplaceAll(html, "{{.OrderDate}}", data.OrderDate)
	html = strings.ReplaceAll(html, "{{.DeliveryAddress}}", data.DeliveryAddress)

	// Format currency values
	html = strings.ReplaceAll(html, "{{printf \"%.2f\" .Subtotal}}", fmt.Sprintf("%.2f", data.Subtotal))
	html = strings.ReplaceAll(html, "{{printf \"%.2f\" .ShippingFee}}", fmt.Sprintf("%.2f", data.ShippingFee))
	html = strings.ReplaceAll(html, "{{printf \"%.2f\" .Discount}}", fmt.Sprintf("%.2f", data.Discount))
	html = strings.ReplaceAll(html, "{{printf \"%.2f\" .TotalAmount}}", fmt.Sprintf("%.2f", data.TotalAmount))

	// Generate order items rows
	itemsHTML := ""
	for _, item := range data.OrderItems {
		itemsHTML += fmt.Sprintf(`
                        <tr>
                            <td class="product-name">%s</td>
                            <td class="quantity">%d</td>
                            <td class="price">$%.2f</td>
                        </tr>`,
			item.ProductName, item.Quantity, item.UnitPrice)
	}
	html = strings.ReplaceAll(html, "{{range .OrderItems}}\n                        <tr>\n                            <td class=\"product-name\">{{.ProductName}}</td>\n                            <td class=\"quantity\">{{.Quantity}}</td>\n                            <td class=\"price\">${{printf \"%.2f\" .UnitPrice}}</td>\n                        </tr>\n                        {{end}}", itemsHTML)

	return html
}

// calculateSubtotal calculates the total from order items
func calculateSubtotal(items []dtos.OrderNotificationItemRequest) float64 {
	var subtotal float64
	for _, item := range items {
		subtotal += float64(item.Quantity) * item.UnitPrice
	}
	return subtotal
}

// Alternative version using html/template for better security and maintainability
func GenerateOrderConfirmationHTMLAdvanced(data dtos.OrderEmailData) (string, error) {
	tmpl := template.Must(template.New("email").Parse(emailTemplate))

	var buf strings.Builder
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Template string for the advanced version
const emailTemplate = `...` // (same template string as above)
