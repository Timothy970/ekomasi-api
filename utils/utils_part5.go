package utils

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
			<p>© {{.CurrentYear}} Ekomasi. All rights reserved.</p>
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
