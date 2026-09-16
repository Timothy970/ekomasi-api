package utils

import (
	"bytes"
	"ekomasi_backend/dtos"
	"fmt"
	"strconv"

	"github.com/jung-kurt/gofpdf"
)

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
	value(fmt.Sprintf("%.2f", *inv.BuyingPrice)) // Purchase cost formatted

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
