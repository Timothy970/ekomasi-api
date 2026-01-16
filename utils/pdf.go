// Package utils provides PDF generation utilities for the Adenzo e-commerce platform.
//
// This file contains PDF document generation functions:
//   - Invoice PDF generation from order data
//   - Inventory report PDF generation
//   - Text truncation utilities
//   - PDF styling and layout helpers
//
// Invoice PDF Features:
//   - Company logo and branding
//   - Order details (ID, date, amount, payment info)
//   - Customer billing information
//   - Itemized product table with images
//   - Order summary (subtotal, taxes, discount, shipping)
//   - Notes section
//
// Inventory PDF Features:
//   - Basic product information
//   - Supplier details with contact info
//   - Stock summary and thresholds
//   - Additional info (dates, warranty)
//   - Section-based layout with visual separators
package utils

import (
	"adenzo_backend/dtos"
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// InvoiceData holds order data for invoice generation.
//
// This struct wraps order information for PDF invoice creation.
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

	// -----------------------------
	// HEADER - Order Status and ID (Right Aligned)
	// -----------------------------
	pdf.SetFont("Arial", "B", 20)
	// Default to PENDING if status is empty
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
	pdf.Cell(40, 5, "www.adenzo.ac.ke")

	// -----------------------------
	// ORDER DETAILS (Right Side)
	// -----------------------------
	pdf.SetXY(110, 30)
	pdf.SetFont("Arial", "", 10)
	// Order number
	pdf.CellFormat(80, 5, fmt.Sprintf("Order #:  %s", order.OrderID), "", 1, "R", false, 0, "")
	pdf.SetX(110)
	// Order date formatted as DD/MM/YYYY
	pdf.CellFormat(80, 5, fmt.Sprintf("Order date: %s", order.CreatedAt.Format("02/01/2006")), "", 1, "R", false, 0, "")
	pdf.SetX(110)
	// Total amount in KES currency
	pdf.CellFormat(80, 5, fmt.Sprintf("Total amount: KES %.2f", order.TotalAmount), "", 1, "R", false, 0, "")
	pdf.SetX(110)
	// Payment method (e.g., M-Pesa, Card)
	pdf.CellFormat(80, 5, fmt.Sprintf("Payment method: %s", order.PaymentMethod), "", 1, "R", false, 0, "")
	pdf.SetX(110)
	// Payment status (e.g., Paid, Pending)
	pdf.CellFormat(80, 5, fmt.Sprintf("Payment status: %s", order.PaymentStatus), "", 1, "R", false, 0, "")

	pdf.Ln(25)

	// -----------------------------
	// BILLING INFORMATION
	// -----------------------------
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(40, 5, "Billed to")

	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	// Display customer information if available
	if order.User != nil {
		// Customer full name
		if order.User.FirstName != "" || order.User.LastName != "" {
			pdf.Cell(40, 5, fmt.Sprintf("%s %s", order.User.FirstName, order.User.LastName))
			pdf.Ln(5)
		}
		// Customer email
		if order.User.Email != "" {
			pdf.Cell(40, 5, order.User.Email)
			pdf.Ln(5)
		}
		// Customer phone
		if order.User.Phone != "" {
			pdf.Cell(40, 5, order.User.Phone)
			pdf.Ln(5)
		}
		// Country code
		pdf.Cell(40, 5, "KE")

	} else {
		// Default country if no user info
		pdf.Cell(40, 5, "KE")
	}

	pdf.Ln(10)
	// Due date section
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 5, "Due date")
	pdf.Ln(6)
	pdf.Cell(40, 5, time.Now().Format("02/01/2006"))
	pdf.Ln(10)

	// -----------------------------
	// ITEM TABLE HEADER
	// -----------------------------
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(245, 245, 245)                                // Light gray background
	pdf.CellFormat(10, 8, "#", "1", 0, "L", true, 0, "")           // Item number
	pdf.CellFormat(50, 8, "Image", "1", 0, "L", true, 0, "")       // Product image
	pdf.CellFormat(80, 8, "Description", "1", 0, "L", true, 0, "") // Product description
	pdf.CellFormat(20, 8, "Qty", "1", 0, "C", true, 0, "")         // Quantity
	pdf.CellFormat(30, 8, "Unit Price", "1", 1, "R", true, 0, "")  // Price per unit

	// -----------------------------
	// ITEMS - Populate Product Rows
	// -----------------------------
	pdf.SetFont("Arial", "", 10)
	for i, item := range order.Items {
		// Item number (1-based index)
		pdf.CellFormat(10, 14, fmt.Sprintf("%d", i+1), "1", 0, "L", false, 0, "")

		// Product image handling
		if len(item.Images) > 0 && item.Images[0].URL != "" {
			img := item.Images[0].URL
			pdf.CellFormat(50, 14, "", "1", 0, "L", false, 0, "")

			// Fetch and embed product image from URL
			resp, err := http.Get(img)
			if err == nil && resp.StatusCode == 200 {
				defer resp.Body.Close()

				// Detect image format from Content-Type header
				contentType := resp.Header.Get("Content-Type")
				imageType := "PNG" // Default to PNG
				switch contentType {
				case "image/jpeg", "image/jpg":
					imageType = "JPEG"
				case "image/png":
					imageType = "PNG"
				case "image/webp":
					imageType = "WEBP"
				case "image/avif":
					imageType = "AVIF"
				}

				// Register and insert image as thumbnail (10x10mm)
				pdf.RegisterImageOptionsReader(img, gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}, resp.Body)
				pdf.ImageOptions(img, pdf.GetX()-48, pdf.GetY()+2, 10, 10, false,
					gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}, 0, "")
			}
		} else {
			// No image available - display placeholder
			pdf.CellFormat(50, 14, "--", "1", 0, "L", false, 0, "")
		}

		// Product description (truncated to fit)
		shortDesc := EllipseText(item.Description, 52)
		pdf.CellFormat(80, 14, shortDesc, "1", 0, "L", false, 0, "")

		// Quantity (fixed at 1 for now)
		pdf.CellFormat(20, 14, "1", "1", 0, "C", false, 0, "")

		// Unit price formatted with 2 decimal places
		pdf.CellFormat(30, 14, fmt.Sprintf("%.2f", item.Price), "1", 1, "R", false, 0, "")
	}

	// -----------------------------
	// ORDER SUMMARY - Totals Section
	// -----------------------------
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)

	// Subtotal (items total before adjustments)
	pdf.Cell(140, 5, "Subtotal")
	pdf.CellFormat(40, 5, fmt.Sprintf("KES %.2f", order.SubTotal), "", 1, "R", false, 0, "")

	// Taxes (VAT or other applicable taxes)
	pdf.Cell(140, 5, "Taxes")
	pdf.CellFormat(40, 5, fmt.Sprintf("KES %.2f", order.EstimatedTax), "", 1, "R", false, 0, "")

	// Discount (promotional or coupon discounts)
	pdf.Cell(140, 5, "Discount")
	pdf.CellFormat(40, 5, fmt.Sprintf("KES %.2f", order.TotalDiscount), "", 1, "R", false, 0, "")

	// Shipping/Delivery charge (handle nil pointer)
	pdf.Cell(140, 5, "Shipping")
	shipping := float64(0)
	if order.DeliveryCharge != nil {
		shipping = *order.DeliveryCharge
	}
	pdf.CellFormat(40, 5, fmt.Sprintf("KES %.2f", shipping), "", 1, "R", false, 0, "")

	// Grand Total (bold emphasis)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(140, 8, "Total")
	pdf.CellFormat(40, 8, fmt.Sprintf("KES %.2f", order.TotalAmount), "", 1, "R", false, 0, "")

	// -----------------------------
	// NOTES - Thank You Message
	// -----------------------------
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(40, 8, "Notes")
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(190, 5, "Thank you for shopping with us!", "", "L", false)

	// -----------------------------
	// Generate and Return PDF Bytes
	// -----------------------------
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// EllipseText truncates text to maximum length with ellipsis.
//
// If text exceeds max length, it's truncated and "..." is appended.
// Used to fit long product descriptions in PDF table cells.
//
// Parameters:
//   - text: string - Original text to truncate
//   - max: int - Maximum character length (including ellipsis)
//
// Returns:
//   - string: Truncated text with "..." or original if within limit
func EllipseText(text string, max int) string {
	if len(text) <= max {
		return text // Text fits within limit
	}
	if max <= 3 {
		return text[:max] // Too short for ellipsis
	}
	// Truncate and add ellipsis
	return text[:max-3] + "..."
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
func GenerateInventoryPDF(inv dtos.SingleInventory) ([]byte, error) {
	// Initialize PDF document with portrait orientation, A4 size
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// ------------------------------
	// Reusable Styling Functions
	// ------------------------------

	// Section header styling (bold, large)
	header := func(title string) {
		pdf.SetFont("Arial", "B", 14)
		pdf.SetTextColor(20, 20, 20)
		pdf.Cell(0, 10, title)
		pdf.Ln(12)
	}

	// Field label styling (bold, left-aligned)
	label := func(text string) {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(40, 6, text, "", 0, "L", false, 0, "")
	}

	// Field value styling (regular, left-aligned)
	value := func(text string) {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(120, 6, text, "", 0, "L", false, 0, "")
		pdf.Ln(7)
	}

	// Visual section separator (horizontal line)
	sectionBox := func() {
		pdf.Ln(4)
		pdf.SetDrawColor(230, 230, 230) // Light gray line
		pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
		pdf.Ln(6)
	}

	// ------------------------------
	// Header - Placed On Date (Top Right)
	// ------------------------------
	pdf.SetFont("Arial", "", 9)
	pdf.SetXY(150, 10)
	pdf.Cell(40, 5, "Placed on: "+inv.PlacedOn)
	pdf.Ln(10)

	// ------------------------------
	// BASIC INFO Section
	// ------------------------------
	header("Basic Info")

	label("Product Name:")
	value(inv.Name)

	label("Product ID:")
	value(inv.ProductID)

	label("Batch:")
	value(ptrToStr(inv.BatchNumber)) // Handle nullable batch number

	label("Quantity:")
	value(strconv.Itoa(inv.Quantity) + " units") // Current quantity with units

	label("Category:")
	value(inv.CategoryName)

	sectionBox() // Visual separator

	// ------------------------------
	// SUPPLIER INFORMATION Section
	// ------------------------------
	header("Supplier Information")

	label("Name:")
	value(inv.SupplierInfo.Name)

	label("Contact:")
	value(inv.SupplierInfo.ContactPhone)

	label("Email:")
	value(inv.SupplierInfo.ContactEmail)

	label("Buying Price:")
	value(fmt.Sprintf("%.2f", inv.BuyingPrice)) // Purchase cost formatted

	sectionBox() // Visual separator

	// ------------------------------
	// STOCK SUMMARY Section
	// ------------------------------
	header("Stock Summary")

	label("Total Stock:")
	value(strconv.Itoa(inv.StockQuantity)) // Available stock count

	label("Minimum Threshold:")
	value(strconv.Itoa(inv.LowStockThreshold)) // Reorder level

	sectionBox() // Visual separator

	// ------------------------------
	// ADDITIONAL INFO Section
	// ------------------------------
	header("Additional Info")

	label("Manufacturing Date:")
	value(ptrToStr(inv.ManufacturingDate)) // Handle nullable date

	label("Expiry Date:")
	value(ptrToStr(inv.ExpiryDate)) // Handle nullable date

	label("Warranty:")
	value(ptrToStr(inv.Warranty)) // Handle nullable warranty info

	// ------------------------------
	// Generate and Return PDF Bytes
	// ------------------------------
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
