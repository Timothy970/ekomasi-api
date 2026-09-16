package utils

import (
	"fmt"
	"os"
	"strings"
)

func GenerateOrderAssignmentEmailContent(orderID, riderName, customerName, deliveryAddress, orderDate string) string {
	frontEndUrl := os.Getenv("FRONT_END_BASE_URL")
	if frontEndUrl == "" {
		frontEndUrl = "https://ekomasi.com" // fallback
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
			<p>Need help? <a href="mailto:support@ekomasi.com">Contact Support</a></p>
			<p>© 2026 Ekomasi. All rights reserved.</p>
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
