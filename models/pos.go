// Package models provides data access functions for the Adenzo backend application.
//
// This file contains Point of Sale (POS) specific functionality including:
//   - Product scanning and lookup by SKU (barcode scanning)
//   - Receipt PDF generation for transactions
//   - Complete product detail retrieval with images, warranties, and variants
//   - Receipt formatting with itemized details, discounts, and totals
package models

import (
	"adenzo_backend/dtos"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

// GetProductThroughScanning retrieves complete product details by SKU for POS barcode scanning.
//
// This function is optimized for point-of-sale operations where products are scanned
// using barcode readers. It retrieves the product along with all associated data
// including images, warranties, variants, category, and any active deal discounts.
//
// Parameters:
//   - sku: string - The Stock Keeping Unit (barcode) to look up
//
// Returns:
//   - *dtos.Product: Complete product details including:
//   - Basic info: ID, Name, Description, SKU, Price, CategoryID, CategoryName, Tag
//   - Stock: StockQuantity
//   - Specifications: Weight, Dimensions, Manufacturer, WeightLimit
//   - Deal info: Discount, DiscountType (if product is in active deal)
//   - Images: Array of product images
//   - Warranty: Product warranty details
//   - ProductVariants: Available variants (size, color, etc.)
//   - Timestamps: CreatedAt, LastUpdated
//   - error: sql.ErrNoRows if SKU not found, or database error
func GetProductThroughScanning(db DBExecutor, sku string) (*dtos.Product, error) {
	// Query product with LEFT JOINs to include optional data (category, deals, specifications)
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, c.name, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE p.sku = ?
	`

	// Scan product data from database
	var p dtos.Product
	err := db.QueryRow(query, sku).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&p.CategoryName, &p.Tag, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
	)
	if err != nil {
		return nil, err
	}

	// Fetch associated product images
	images, err := fetchProductImages(db, p.ID)
	if err != nil {
		return nil, err
	}
	p.Images = images

	// Fetch product warranties
	warranties, err := FetchProductWarranties(db, p.ID)
	if err != nil {
		return nil, err
	}
	p.Warranty = &warranties

	// Fetch product variants (sizes, colors, etc.)
	variants, err := getProductVariants(db, p.ID)
	if err != nil {
		return nil, err
	}
	p.ProductVariants = variants

	return &p, nil
}

// GenerateReceiptPDF creates a printable PDF receipt for a completed order.
//
// This function generates a formatted A5-sized receipt containing order details,
// itemized product list with quantities and prices, discount calculations, and totals.
// The PDF is designed for thermal or standard receipt printers.
//
// Parameters:
//   - receipt: dtos.Order - Order details containing:
//   - OrderID: Unique order identifier
//   - CreatedAt: Order timestamp
//   - OrderStatus: Order status (nullable pointer)
//   - PaymentMethod: Payment method used
//   - Items: Array of order items with Name, StockQuantity, Price
//   - TotalAmount: Final amount paid (after discounts)
//   - TotalDiscount: Total discount applied
//
// Returns:
//   - gofpdf.Pdf: PDF document object ready for output (use pdf.Output() to save/send)
//   - error: PDF generation error or nil on success
func GenerateReceiptPDF(receipt dtos.Order) (gofpdf.Pdf, error) {
	// Initialize PDF in portrait mode, A5 size (148x210mm) - typical receipt size
	pdf := gofpdf.New("P", "mm", "A5", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)

	// Header section - Receipt title and order metadata
	pdf.Cell(0, 10, "=========== RECEIPT ===========")
	pdf.Ln(12) // Line break with 12mm spacing
	pdf.Cell(0, 10, fmt.Sprintf("Order ID      : %s", receipt.OrderID))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Date          : %s", receipt.CreatedAt))
	pdf.Ln(8)

	// Handle nullable order status
	status := ""
	if receipt.OrderStatus != nil {
		status = *receipt.OrderStatus
	}
	pdf.Cell(0, 10, fmt.Sprintf("Status        : %s", status))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Payment Method: %s", receipt.PaymentMethod))
	pdf.Ln(12)

	// Table Header - Create bordered cells with column headers
	pdf.SetFont("Arial", "B", 12)                              // Bold font for headers
	pdf.CellFormat(80, 10, "Item", "1", 0, "L", false, 0, "")  // 80mm wide, left-aligned
	pdf.CellFormat(20, 10, "Qty", "1", 0, "C", false, 0, "")   // 20mm wide, center-aligned
	pdf.CellFormat(30, 10, "Unit", "1", 0, "R", false, 0, "")  // 30mm wide, right-aligned
	pdf.CellFormat(30, 10, "Total", "1", 1, "R", false, 0, "") // 30mm wide, right-aligned, new line

	// Table Rows - Iterate through order items
	pdf.SetFont("Arial", "", 12) // Regular font for data rows
	for _, item := range receipt.Items {
		// Calculate line total (unit price × quantity)
		total := item.Price * float64(item.StockQuantity)

		// Create bordered cells for each column
		pdf.CellFormat(80, 10, item.Name, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 10, fmt.Sprintf("%d", item.StockQuantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("%.2f", item.Price), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("%.2f", total), "1", 1, "R", false, 0, "") // Last cell includes new line
	}

	// Totals Section - Calculate and display subtotal, discount, and grand total
	pdf.Ln(5) // Add 5mm spacing before totals

	// Calculate subtotal (grand total + discount = amount before discount)
	subtotal := receipt.TotalAmount + receipt.TotalDiscount
	pdf.CellFormat(130, 10, "Subtotal:", "0", 0, "R", false, 0, "")
	pdf.CellFormat(30, 10, fmt.Sprintf("%.2f", subtotal), "0", 1, "R", false, 0, "")

	// Show discount line only if discount was applied
	if receipt.TotalDiscount > 0 {
		pdf.CellFormat(130, 10, "Discount:", "0", 0, "R", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("-%.2f", receipt.TotalDiscount), "0", 1, "R", false, 0, "")
	}

	// Grand Total - Display final amount with top border ("T") for emphasis
	pdf.SetFont("Arial", "B", 12) // Bold font for grand total
	pdf.CellFormat(130, 10, "Grand Total:", "T", 0, "R", false, 0, "")
	pdf.CellFormat(30, 10, fmt.Sprintf("%.2f", receipt.TotalAmount), "T", 1, "R", false, 0, "")

	// Footer - Thank you message
	pdf.Ln(12) // Add 12mm spacing
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 10, "Thank you for your purchase!")

	// Return completed PDF document
	return pdf, nil
}
