package utils

import (
	"bytes"
	"ekomasi_backend/dtos"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type InvoiceData struct {
	Order dtos.AdminOrder // Order details including items, customer, payment
}

// GenerateInvoicePDF creates a PDF invoice from order data.
//
// This function generates a professional invoice document with:
// 1. Header with logo, order status, and order ID
// 2. Order details (date, amount, payment method/status)
// 3. Customer billing information
// 4. Itemized product table with images
// 5. Order summary (subtotal, taxes, discount, shipping, total)
// 6. Notes section
//
// Parameters:
//   - order: dtos.AdminOrder - Order data to generate invoice from
//
// Returns:
//   - []byte: PDF document bytes
//   - error: PDF generation error or nil on success
func GenerateInvoicePDF(order dtos.AdminOrder) ([]byte, error) {
	// Initialize PDF document with portrait orientation, A4 size
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 10, 15) // Left, Top, Right margins
	pdf.AddPage()

	// Get logo path relative to this source file
	_, filename, _, _ := runtime.Caller(0)
	utilsDir := filepath.Dir(filename)
	logoPath := filepath.Join(utilsDir, "logo.png")

	renderInvoiceHeader(pdf, order, logoPath)
	renderBillingInfo(pdf, order.User)
	renderItemTable(pdf, order.Items)
	renderOrderSummary(pdf, order)

	// Generate and Return PDF Bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func renderInvoiceHeader(pdf *gofpdf.Fpdf, order dtos.AdminOrder, logoPath string) {
	// HEADER - Order Status and ID (Right Aligned)
	pdf.SetFont("Arial", "B", 20)
	orderStatus := order.OrderStatus
	if orderStatus == "" {
		orderStatus = "PENDING"
	}
	pdf.CellFormat(190, 10, orderStatus, "", 1, "R", false, 0, "")

	// Display Order ID below status
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(190, 6, order.OrderID, "", 1, "R", false, 0, "")

	// Left Side - Company Logo + Website
	pdf.ImageOptions(
		logoPath,
		15, 20, 20, 0, // x, y, width, height (0 = auto)
		false,
		gofpdf.ImageOptions{ImageType: "PNG"},
		0,
		"",
	)

	// Website URL next to logo
	pdf.SetXY(40, 22)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 5, "www.ekomasi.ac.ke")

	// ORDER DETAILS (Right Side)
	pdf.SetXY(110, 30)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(80, 5, fmt.Sprintf("Order #:  %s", order.OrderID), "", 1, "R", false, 0, "")
	pdf.SetX(110)
	pdf.CellFormat(80, 5, fmt.Sprintf("Order date: %s", order.CreatedAt.Format("02/01/2006")), "", 1, "R", false, 0, "")
	pdf.SetX(110)
	pdf.CellFormat(80, 5, fmt.Sprintf("Total amount: KES %.2f", order.TotalAmount), "", 1, "R", false, 0, "")
	pdf.SetX(110)
	pdf.CellFormat(80, 5, fmt.Sprintf("Payment method: %s", order.PaymentMethod), "", 1, "R", false, 0, "")
	pdf.SetX(110)
	pdf.CellFormat(80, 5, fmt.Sprintf("Payment status: %s", order.PaymentStatus), "", 1, "R", false, 0, "")

	pdf.Ln(25)
}

func renderBillingInfo(pdf *gofpdf.Fpdf, user *dtos.Users) {
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(40, 5, "Billed to")

	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	if user != nil {
		if user.FirstName != "" || user.LastName != "" {
			pdf.Cell(40, 5, fmt.Sprintf("%s %s", user.FirstName, user.LastName))
			pdf.Ln(5)
		}
		if user.Email != "" {
			pdf.Cell(40, 5, user.Email)
			pdf.Ln(5)
		}
		if user.Phone != "" {
			pdf.Cell(40, 5, user.Phone)
			pdf.Ln(5)
		}
		pdf.Cell(40, 5, "KE")
	} else {
		pdf.Cell(40, 5, "KE")
	}

	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 5, "Due date")
	pdf.Ln(6)
	pdf.Cell(40, 5, time.Now().Format("02/01/2006"))
	pdf.Ln(10)
}

func renderItemTable(pdf *gofpdf.Fpdf, items []dtos.OrderProduct) {
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(245, 245, 245)
	pdf.CellFormat(10, 8, "#", "1", 0, "L", true, 0, "")
	pdf.CellFormat(50, 8, "Image", "1", 0, "L", true, 0, "")
	pdf.CellFormat(80, 8, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Unit Price", "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	for i, item := range items {
		renderItemRow(pdf, i+1, item)
	}
}

func renderItemRow(pdf *gofpdf.Fpdf, index int, item dtos.OrderProduct) {
	pdf.CellFormat(10, 14, fmt.Sprintf("%d", index), "1", 0, "L", false, 0, "")

	if len(item.Images) > 0 && item.Images[0].URL != "" {
		img := item.Images[0].URL
		pdf.CellFormat(50, 14, "", "1", 0, "L", false, 0, "")
		renderProductImage(pdf, img)
	} else {
		pdf.CellFormat(50, 14, "--", "1", 0, "L", false, 0, "")
	}

	shortDesc := EllipseText(item.Description, 52)
	pdf.CellFormat(80, 14, shortDesc, "1", 0, "L", false, 0, "")
	pdf.CellFormat(20, 14, "1", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 14, fmt.Sprintf("%.2f", item.Price), "1", 1, "R", false, 0, "")
}

func renderProductImage(pdf *gofpdf.Fpdf, imgURL string) {
	resp, err := http.Get(imgURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	imageType := getImageFormatFromContentType(contentType)

	pdf.RegisterImageOptionsReader(imgURL, gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}, resp.Body)
	pdf.ImageOptions(imgURL, pdf.GetX()-48, pdf.GetY()+2, 10, 10, false,
		gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}, 0, "")
}

func renderOrderSummary(pdf *gofpdf.Fpdf, order dtos.AdminOrder) {
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)

	pdf.Cell(140, 5, "Subtotal")
	pdf.CellFormat(40, 5, fmt.Sprintf("KES %.2f", order.SubTotal), "", 1, "R", false, 0, "")

	pdf.Cell(140, 5, "Taxes")
	pdf.CellFormat(40, 5, fmt.Sprintf("KES %.2f", order.EstimatedTax), "", 1, "R", false, 0, "")

	pdf.Cell(140, 5, "Discount")
	pdf.CellFormat(40, 5, fmt.Sprintf("KES %.2f", order.TotalDiscount), "", 1, "R", false, 0, "")

	pdf.Cell(140, 5, "Shipping")
	shipping := float64(0)
	if order.DeliveryCharge != nil {
		shipping = *order.DeliveryCharge
	}
	pdf.CellFormat(40, 5, fmt.Sprintf("KES %.2f", shipping), "", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(140, 8, "Total")
	pdf.CellFormat(40, 8, fmt.Sprintf("KES %.2f", order.TotalAmount), "", 1, "R", false, 0, "")

	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(40, 8, "Notes")
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(190, 5, "Thank you for shopping with us!", "", "L", false)
}

// EllipseText truncates text to maximum length with ellipsis.
//
// If text exceeds max length, it's truncated and "..." is appended.
// Used to fit long product descriptions in PDF table cells.
//
// Parameters:
//   - text: string - Original text to truncate
//   - maxLen: int - Maximum character length (including ellipsis)
//
// Returns:
//   - string: Truncated text with "..." or original if within limit
func EllipseText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text // Text fits within limit
	}
	if maxLen <= 3 {
		return text[:maxLen] // Too short for ellipsis
	}
	// Truncate and add ellipsis
	return text[:maxLen-3] + "..."
}

// getImageFormatFromContentType returns the gofpdf image format string based on Content-Type header.
func getImageFormatFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return "JPEG"
	case "image/png":
		return "PNG"
	case "image/webp":
		return "WEBP"
	case "image/avif":
		return "AVIF"
	default:
		return "PNG"
	}
}

// GenerateInventoryPDF creates an inventory report PDF.
//
// This function generates a structured inventory document with:
// 1. Basic Info: Product details, quantity, category
// 2. Supplier Information: Supplier contact details and buying price
// 3. Stock Summary: Current stock and low stock threshold
// 4. Additional Info: Manufacturing date, expiry date, warranty
//
// Parameters:
//   - inv: dtos.SingleInventory - Inventory data to generate report from
//
// Returns:
//   - []byte: PDF document bytes
//   - error: PDF generation error or nil on success
