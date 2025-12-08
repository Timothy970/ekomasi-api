package utils

import (
	"adenzo_backend/dtos"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strconv"
	"strings"
)

// sanitize trims whitespace and ensures empty strings become ""
func sanitize(value string) string {
	return strings.TrimSpace(value)
}

// required checks for empty fields except the allowed empty ones
func required(field, name string, allowEmpty bool) error {
	if !allowEmpty && field == "" {
		return fmt.Errorf("missing value for required field: %s", name)
	}
	return nil
}

// ParseProductsCSV reads and sanitizes a CSV file into a list of products.
func ParseProductsCSV(file multipart.File) ([]dtos.BulkUploadProduct, error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %v", err)
	}

	expectedHeaders := []string{
		"name", "description", "sku", "price", "category_id",
		"stock_quantity", "tag", "low_stock_quantity_warning",
		"sell_when_out_of_stock", "show_stock_quantity",
		"buying_price", "image",
	}

	// Check for missing or extra columns
	if len(headers) < len(expectedHeaders) {
		return nil, fmt.Errorf("invalid CSV: missing columns. Expected %d, got %d", len(expectedHeaders), len(headers))
	}

	// Normalize headers
	for i := range headers {
		headers[i] = strings.ToLower(strings.TrimSpace(headers[i]))
	}

	// Validate header names
	for i, header := range expectedHeaders {
		if headers[i] != header {
			return nil, fmt.Errorf("invalid CSV: expected header '%s', got '%s'", header, headers[i])
		}
	}

	var products []dtos.BulkUploadProduct
	rowNumber := 1 // starts from 1 (header)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		rowNumber++
		if err != nil {
			log.Printf("⚠️ Error reading CSV row %d: %v", rowNumber, err)
			continue
		}

		// Trim whitespace and cut trailing empty columns
		for i := range record {
			record[i] = sanitize(record[i])
		}
		if len(record) > len(expectedHeaders) {
			record = record[:len(expectedHeaders)]
		}

		// Skip completely empty lines
		allEmpty := true
		for _, val := range record {
			if val != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			log.Printf("Row %d skipped: empty line", rowNumber)
			continue
		}

		log.Printf("Processing row %d: %+v", rowNumber, record)

		// Required fields validation
		requiredFields := map[string]bool{
			"name": true, "description": true, "sku": true, "price": true,
			"category_id": true, "stock_quantity": true,
			"tag": false, "low_stock_quantity_warning": true,
			"sell_when_out_of_stock": true, "show_stock_quantity": true,
			"buying_price": true, "image": true,
		}

		skipRow := false
		for i, header := range expectedHeaders {
			if err := required(record[i], header, !requiredFields[header]); err != nil {
				log.Printf("Row %d skipped: %v", rowNumber, err)
				skipRow = true
				break
			}
		}
		if skipRow {
			continue
		}

		// Convert numeric and boolean values safely
		price, _ := strconv.ParseFloat(record[3], 64)
		stockQty, _ := strconv.Atoi(record[5])
		lowStockWarn, _ := strconv.Atoi(record[7])
		sellOut, _ := strconv.ParseBool(record[8])
		showStock, _ := strconv.ParseBool(record[9])
		buyingPrice, _ := strconv.ParseFloat(record[10], 64)

		product := dtos.BulkUploadProduct{
			Name:                    record[0],
			Description:             record[1],
			SKU:                     record[2],
			Price:                   price,
			CategoryID:              record[4],
			StockQuantity:           stockQty,
			Tag:                     record[6],
			SearchVector:            record[0], // optional search field
			LowStockQuantityWarning: lowStockWarn,
			SellWhenOutOfStock:      sellOut,
			ShowStockQuantity:       showStock,
			BuyingPrice:             buyingPrice,
			Image:                   record[11],
		}

		// Log the successfully parsed product
		log.Printf(" Row %d parsed successfully: %+v", rowNumber, product)

		products = append(products, product)
	}

	if len(products) == 0 {
		return nil, errors.New("no valid products found in CSV")
	}

	log.Printf("Successfully parsed %d valid products", len(products))
	return products, nil
}

func ExportInventoryCSV(w io.Writer, inv dtos.Inventory) error {
	writer := csv.NewWriter(w)

	// ============================
	// BASIC INFO SECTION
	// ============================
	writer.Write([]string{"BASIC INFO"})
	writer.Write([]string{
		"Inventory ID", "Product ID", "Batch Number",
		"Name", "Description", "SKU",
		"Quantity",
		"Price", "Buying Price",
		"Category ID", "Category Name",
		"Images",
	})

	images := []string{}
	for _, img := range inv.Images {
		images = append(images, img.URL)
	}
	writer.Write([]string{
		inv.InventoryID,
		inv.ProductID,
		ptrToStr(inv.BatchNumber),
		inv.Name,
		inv.Description,
		inv.SKU,
		strconv.Itoa(inv.Quantity),
		floatToStr(inv.Price),
		floatToStr(inv.BuyingPrice),
		inv.CategoryID,
		inv.CategoryName,
		strings.Join(images, ";"),
	})

	writer.Write([]string{}) // Empty row between sections

	// ============================
	// SUPPLIER INFO SECTION
	// ============================
	writer.Write([]string{"SUPPLIER INFO"})
	writer.Write([]string{
		"Inventory ID", "Supplier ID", "Supplier Name",
		"Contact Email", "Contact Phone",
	})

	s := inv.SupplierInfo
	writer.Write([]string{
		inv.InventoryID,
		s.SupplierID,
		s.Name,
		s.ContactEmail,
		s.ContactPhone,
	})

	writer.Write([]string{}) // Empty row between sections

	// ============================
	// ADDITIONAL INFO SECTION
	// ============================
	writer.Write([]string{"ADDITIONAL INFO"})
	writer.Write([]string{
		"Inventory ID",
		"Manufacturing Date", "Expiry Date", "Warranty",
		"Placed On", "Low Stock Threshold",
	})

	writer.Write([]string{
		inv.InventoryID,
		ptrToStr(inv.ManufacturingDate),
		ptrToStr(inv.ExpiryDate),
		ptrToStr(inv.Warranty),
		inv.PlacedOn,
		strconv.Itoa(inv.LowStockThreshold),
	})

	writer.Flush()
	return writer.Error()
}

// Helpers
func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func floatToStr(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
