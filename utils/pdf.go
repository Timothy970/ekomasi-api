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

type InvoiceData struct {
	Order dtos.AdminOrder
}

func GenerateInvoicePDF(order dtos.AdminOrder) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 10, 15)
	pdf.AddPage()
	// Get the directory where this file (pdf.go) is located
	_, filename, _, _ := runtime.Caller(0)
	utilsDir := filepath.Dir(filename)
	logoPath := filepath.Join(utilsDir, "logo.png")
	// -----------------------------
	// HEADER
	// -----------------------------
	pdf.SetFont("Arial", "B", 20)
	orderStatus := order.OrderStatus
	if orderStatus == "" {
		orderStatus = "PENDING"
	}
	pdf.CellFormat(190, 10, orderStatus, "", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(190, 6, order.OrderID, "", 1, "R", false, 0, "")

	// Left – Logo + website (adjust logo path)
	pdf.ImageOptions(
		logoPath,
		15, 20, 20, 0,
		false,
		gofpdf.ImageOptions{ImageType: "PNG"},
		0,
		"",
	)

	pdf.SetXY(40, 22)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 5, "www.adenzo.ac.ke")

	// -----------------------------
	// ORDER DETAILS (RIGHT SIDE)
	// -----------------------------
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

	// -----------------------------
	// BILLING + DUE DATE
	// -----------------------------
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(40, 5, "Billed to")

	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	if order.User != nil {
		if order.User.FirstName != "" || order.User.LastName != "" {
			pdf.Cell(40, 5, fmt.Sprintf("%s %s", order.User.FirstName, order.User.LastName))
			pdf.Ln(5)
		}
		if order.User.Email != "" {
			pdf.Cell(40, 5, order.User.Email)
			pdf.Ln(5)
		}
		if order.User.Phone != "" {
			pdf.Cell(40, 5, order.User.Phone)
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

	// -----------------------------
	// ITEM TABLE HEADER
	// -----------------------------
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(245, 245, 245)
	pdf.CellFormat(10, 8, "#", "1", 0, "L", true, 0, "")
	pdf.CellFormat(50, 8, "Image", "1", 0, "L", true, 0, "")
	pdf.CellFormat(80, 8, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Unit Price", "1", 1, "R", true, 0, "")

	// -----------------------------
	// ITEMS
	// -----------------------------
	pdf.SetFont("Arial", "", 10)
	for i, item := range order.Items {
		pdf.CellFormat(10, 14, fmt.Sprintf("%d", i+1), "1", 0, "L", false, 0, "")
		// image if exists
		if len(item.Images) > 0 && item.Images[0].URL != "" {
			img := item.Images[0].URL
			pdf.CellFormat(50, 14, "", "1", 0, "L", false, 0, "")
			// Insert small thumbnail from URL
			resp, err := http.Get(img)
			if err == nil && resp.StatusCode == 200 {
				defer resp.Body.Close()
				// Detect image type from Content-Type header
				contentType := resp.Header.Get("Content-Type")
				imageType := "PNG" // default
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
				pdf.RegisterImageOptionsReader(img, gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}, resp.Body)
				pdf.ImageOptions(img, pdf.GetX()-48, pdf.GetY()+2, 10, 10, false,
					gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}, 0, "")
			}
		} else {
			pdf.CellFormat(50, 14, "--", "1", 0, "L", false, 0, "")
		}
		shortDesc := EllipseText(item.Description, 52)
		pdf.CellFormat(80, 14, shortDesc, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 14, "1", "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 14, fmt.Sprintf("%.2f", item.Price), "1", 1, "R", false, 0, "")
	}

	// -----------------------------
	// SUMMARY
	// -----------------------------
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

	// -----------------------------
	// NOTES
	// -----------------------------
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(40, 8, "Notes")
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(190, 5, "Thank you for shopping with us!", "", "L", false)

	// Return as bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func EllipseText(text string, max int) string {
	if len(text) <= max {
		return text
	}
	if max <= 3 {
		return text[:max]
	}
	return text[:max-3] + "..."
}

func GenerateInventoryPDF(inv dtos.Inventory) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// ------------------------------
	// Styles
	// ------------------------------
	header := func(title string) {
		pdf.SetFont("Arial", "B", 14)
		pdf.SetTextColor(20, 20, 20)
		pdf.Cell(0, 10, title)
		pdf.Ln(12)
	}

	label := func(text string) {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(40, 6, text, "", 0, "L", false, 0, "")
	}

	value := func(text string) {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(120, 6, text, "", 0, "L", false, 0, "")
		pdf.Ln(7)
	}

	sectionBox := func() {
		pdf.Ln(4)
		pdf.SetDrawColor(230, 230, 230)
		pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
		pdf.Ln(6)
	}

	// ------------------------------
	// Header Info — Placed On (Top Right)
	// ------------------------------
	pdf.SetFont("Arial", "", 9)
	pdf.SetXY(150, 10)
	pdf.Cell(40, 5, "Placed on: "+inv.PlacedOn)
	pdf.Ln(10)

	// ------------------------------
	// BASIC INFO
	// ------------------------------
	header("Basic Info")

	label("Product Name:")
	value(inv.Name)

	label("Product ID:")
	value(inv.ProductID)

	label("Batch:")
	value(ptrToStr(inv.BatchNumber))

	label("Quantity:")
	value(strconv.Itoa(inv.Quantity) + " units")

	label("Category:")
	value(inv.CategoryName)

	sectionBox()

	// ------------------------------
	// SUPPLIER INFORMATION
	// ------------------------------
	header("Supplier Information")

	label("Name:")
	value(inv.SupplierInfo.Name)

	label("Contact:")
	value(inv.SupplierInfo.ContactPhone)

	label("Email:")
	value(inv.SupplierInfo.ContactEmail)

	label("Buying Price:")
	value(fmt.Sprintf("%.2f", inv.BuyingPrice))

	sectionBox()

	// ------------------------------
	// STOCK SUMMARY
	// ------------------------------
	header("Stock Summary")

	label("Total Stock:")
	value(strconv.Itoa(inv.StockQuantity))

	label("Minimum Threshold:")
	value(strconv.Itoa(inv.LowStockThreshold))

	sectionBox()

	// ------------------------------
	// ADDITIONAL INFO
	// ------------------------------
	header("Additional Info")

	label("Manufacturing Date:")
	value(ptrToStr(inv.ManufacturingDate))

	label("Expiry Date:")
	value(ptrToStr(inv.ExpiryDate))

	label("Warranty:")
	value(ptrToStr(inv.Warranty))

	// ------------------------------
	// Return PDF Bytes
	// ------------------------------
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
