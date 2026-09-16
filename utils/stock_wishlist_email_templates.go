package utils

import (
	"bytes"
	"ekomasi_backend/dtos"
	"fmt"
	"html/template"
	"strings"
	"time"
)

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
