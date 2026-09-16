package utils

import (
	"ekomasi_backend/dtos"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

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
