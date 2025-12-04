package models

import (
	"adenzo_backend/dtos"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

func GetProductThroughScanning(sku string) (*dtos.Product, error) {
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

	var p dtos.Product
	err := DB.QueryRow(query, sku).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&p.CategoryName, &p.Tag, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
	)
	if err != nil {
		return nil, err
	}
	// Fetch product images
	images, err := fetchProductImages(p.ID)
	if err != nil {
		return nil, err
	}
	p.Images = images
	// Fetch product warranties
	warranties, err := FetchProductWarranties(p.ID)
	if err != nil {
		return nil, err
	}
	p.Warranty = &warranties
	// Fetch product variants
	variants, err := getProductVariants(p.ID)
	if err != nil {
		return nil, err
	}
	p.ProductVariants = variants
	return &p, nil
}

func GenerateReceiptPDF(receipt dtos.Order) (gofpdf.Pdf, error) {
	pdf := gofpdf.New("P", "mm", "A5", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)

	// Header
	pdf.Cell(0, 10, "=========== RECEIPT ===========")
	pdf.Ln(12)
	pdf.Cell(0, 10, fmt.Sprintf("Order ID      : %s", receipt.OrderID))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Date          : %s", receipt.CreatedAt))
	pdf.Ln(8)
	status := ""
	if receipt.OrderStatus != nil {
		status = *receipt.OrderStatus
	}
	pdf.Cell(0, 10, fmt.Sprintf("Status        : %s", status))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Payment Method: %s", receipt.PaymentMethod))
	pdf.Ln(12)

	// Table Header
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(80, 10, "Item", "1", 0, "L", false, 0, "")
	pdf.CellFormat(20, 10, "Qty", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 10, "Unit", "1", 0, "R", false, 0, "")
	pdf.CellFormat(30, 10, "Total", "1", 1, "R", false, 0, "")

	// Table Rows
	pdf.SetFont("Arial", "", 12)
	for _, item := range receipt.Items {
		total := item.Price * float64(item.StockQuantity)

		pdf.CellFormat(80, 10, item.Name, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 10, fmt.Sprintf("%d", item.StockQuantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("%.2f", item.Price), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("%.2f", total), "1", 1, "R", false, 0, "")
	}

	// Totals
	pdf.Ln(5)
	subtotal := receipt.TotalAmount + receipt.TotalDiscount
	pdf.CellFormat(130, 10, "Subtotal:", "0", 0, "R", false, 0, "")
	pdf.CellFormat(30, 10, fmt.Sprintf("%.2f", subtotal), "0", 1, "R", false, 0, "")

	if receipt.TotalDiscount > 0 {
		pdf.CellFormat(130, 10, "Discount:", "0", 0, "R", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("-%.2f", receipt.TotalDiscount), "0", 1, "R", false, 0, "")
	}

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(130, 10, "Grand Total:", "T", 0, "R", false, 0, "")
	pdf.CellFormat(30, 10, fmt.Sprintf("%.2f", receipt.TotalAmount), "T", 1, "R", false, 0, "")

	pdf.Ln(12)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 10, "Thank you for your purchase!")

	// Send PDF to response
	return pdf, nil
}
