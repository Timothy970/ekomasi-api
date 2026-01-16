// Package utils provides utility functions for the Adenzo e-commerce platform.
//
// This file contains CSV processing utilities:
//   - Product bulk upload CSV parsing
//   - Inventory export to CSV format
//   - CSV validation and sanitization
//   - Type conversion helpers (string, float, pointer handling)
//   - Error reporting for invalid CSV data
//
// CSV Import Features:
//   - Header validation with expected column names
//   - Required field enforcement
//   - Data type conversion (string to int, float, bool)
//   - Row-level error handling (skip invalid, continue processing)
//   - Whitespace trimming and empty line skipping
//
// CSV Export Features:
//   - Multi-section CSV generation (Basic Info, Supplier Info, Additional Info)
//   - Null-safe pointer handling
//   - Number formatting
//   - Image URL aggregation
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

// sanitize trims whitespace from CSV field values.
//
// This helper ensures consistent data by removing leading and trailing
// whitespace from all CSV field values.
//
// Parameters:
//   - value: string - Raw CSV field value
//
// Returns:
//   - string: Trimmed value
func sanitize(value string) string {
	return strings.TrimSpace(value)
}

// required validates that required fields are not empty.
//
// This helper checks if a field value is empty and returns an error
// if the field is required (allowEmpty is false).
//
// Parameters:
//   - field: string - Field value to validate
//   - name: string - Field name for error messages
//   - allowEmpty: bool - Whether empty values are allowed
//
// Returns:
//   - error: Missing field error or nil if valid
func required(field, name string, allowEmpty bool) error {
	if !allowEmpty && field == "" {
		return fmt.Errorf("missing value for required field: %s", name)
	}
	return nil
}

// ParseProductsCSV reads and parses a CSV file into bulk upload products.
//
// This function performs comprehensive CSV validation and parsing:
// 1. Validates CSV headers match expected column names
// 2. Validates required fields for each row
// 3. Converts data types (string to int/float/bool)
// 4. Skips invalid rows and logs errors
// 5. Returns successfully parsed products
//
// Expected CSV columns (in order):
//
//	name, description, sku, price, category_id, stock_quantity, tag,
//	low_stock_quantity_warning, sell_when_out_of_stock, show_stock_quantity,
//	buying_price, image
//
// Parameters:
//   - file: multipart.File - Uploaded CSV file
//
// Returns:
//   - []dtos.BulkUploadProduct: Array of successfully parsed products
//   - error: Header validation error, "no valid products found", or nil on success
func ParseProductsCSV(file multipart.File) ([]dtos.BulkUploadProduct, error) {
	// Create CSV reader with automatic whitespace trimming
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read and validate CSV headers
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %v", err)
	}

	// Define expected header columns
	expectedHeaders := []string{
		"name", "description", "sku", "price", "category_id",
		"stock_quantity", "tag", "low_stock_quantity_warning",
		"sell_when_out_of_stock", "show_stock_quantity",
		"buying_price", "image",
	}

	// Validate minimum column count
	if len(headers) < len(expectedHeaders) {
		return nil, fmt.Errorf("invalid CSV: missing columns. Expected %d, got %d", len(expectedHeaders), len(headers))
	}

	// Normalize headers to lowercase for case-insensitive comparison
	for i := range headers {
		headers[i] = strings.ToLower(strings.TrimSpace(headers[i]))
	}

	// Validate each header name matches expected
	for i, header := range expectedHeaders {
		if headers[i] != header {
			return nil, fmt.Errorf("invalid CSV: expected header '%s', got '%s'", header, headers[i])
		}
	}

	var products []dtos.BulkUploadProduct
	rowNumber := 1 // Track row number for error logging (starts at 1 for header)

	// Process each CSV row
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file reached
		}
		rowNumber++
		if err != nil {
			log.Printf("⚠️ Error reading CSV row %d: %v", rowNumber, err)
			continue // Skip malformed rows
		}

		// Sanitize all field values by trimming whitespace
		for i := range record {
			record[i] = sanitize(record[i])
		}
		// Trim extra columns if CSV has more than expected
		if len(record) > len(expectedHeaders) {
			record = record[:len(expectedHeaders)]
		}

		// Skip completely empty rows
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

		// Define which fields are required (true) vs optional (false)
		requiredFields := map[string]bool{
			"name": true, "description": true, "sku": true, "price": true,
			"category_id": true, "stock_quantity": true,
			"tag": false, "low_stock_quantity_warning": true,
			"sell_when_out_of_stock": true, "show_stock_quantity": true,
			"buying_price": true, "image": true,
		}

		// Validate required fields
		skipRow := false
		for i, header := range expectedHeaders {
			if err := required(record[i], header, !requiredFields[header]); err != nil {
				log.Printf("Row %d skipped: %v", rowNumber, err)
				skipRow = true
				break
			}
		}
		if skipRow {
			continue // Skip row if validation failed
		}

		// Convert string values to appropriate data types
		price, _ := strconv.ParseFloat(record[3], 64)        // price
		stockQty, _ := strconv.Atoi(record[5])               // stock_quantity
		lowStockWarn, _ := strconv.Atoi(record[7])           // low_stock_quantity_warning
		sellOut, _ := strconv.ParseBool(record[8])           // sell_when_out_of_stock
		showStock, _ := strconv.ParseBool(record[9])         // show_stock_quantity
		buyingPrice, _ := strconv.ParseFloat(record[10], 64) // buying_price

		// Construct product struct from parsed values
		product := dtos.BulkUploadProduct{
			Name:                    record[0],    // name
			Description:             record[1],    // description
			SKU:                     record[2],    // sku
			Price:                   price,        // converted price
			CategoryID:              record[4],    // category_id
			StockQuantity:           stockQty,     // converted stock_quantity
			Tag:                     record[6],    // tag (optional)
			SearchVector:            record[0],    // use name for search indexing
			LowStockQuantityWarning: lowStockWarn, // converted low_stock_quantity_warning
			SellWhenOutOfStock:      sellOut,      // converted sell_when_out_of_stock
			ShowStockQuantity:       showStock,    // converted show_stock_quantity
			BuyingPrice:             buyingPrice,  // converted buying_price
			Image:                   record[11],   // image URL
		}

		// Log successful parsing
		log.Printf(" Row %d parsed successfully: %+v", rowNumber, product)

		products = append(products, product)
	}

	// Validate at least one product was successfully parsed
	if len(products) == 0 {
		return nil, errors.New("no valid products found in CSV")
	}

	log.Printf("Successfully parsed %d valid products", len(products))
	return products, nil
}

// ExportInventoryCSV exports inventory data to CSV format.
//
// This function generates a multi-section CSV with:
// 1. BASIC INFO: Product details, pricing, category, images
// 2. SUPPLIER INFO: Supplier contact information
// 3. ADDITIONAL INFO: Manufacturing dates, warranty, stock thresholds
//
// Parameters:
//   - w: io.Writer - Output writer for CSV data
//   - inv: dtos.SingleInventory - Inventory data to export
//
// Returns:
//   - error: CSV write error or nil on success
func ExportInventoryCSV(w io.Writer, inv dtos.SingleInventory) error {
	writer := csv.NewWriter(w)

	// ============================
	// BASIC INFO SECTION
	// ============================
	writer.Write([]string{"BASIC INFO"})
	// Write column headers for basic information
	writer.Write([]string{
		"Inventory ID", "Product ID", "Batch Number",
		"Name", "Description", "SKU",
		"Quantity",
		"Price", "Buying Price",
		"Category ID", "Category Name",
		"Images",
	})

	// Aggregate image URLs into semicolon-separated string
	images := []string{}
	for _, img := range inv.Images {
		images = append(images, img.URL)
	}
	// Write basic info data row
	writer.Write([]string{
		inv.InventoryID,
		inv.ProductID,
		ptrToStr(inv.BatchNumber), // Handle nullable batch number
		inv.Name,
		inv.Description,
		inv.SKU,
		strconv.Itoa(inv.Quantity),   // Convert int to string
		floatToStr(inv.Price),        // Format float with 2 decimals
		floatToStr(*inv.BuyingPrice), // Format buying price
		inv.CategoryID,
		inv.CategoryName,
		strings.Join(images, ";"), // Join image URLs
	})

	writer.Write([]string{}) // Empty row separator between sections

	// ============================
	// SUPPLIER INFO SECTION
	// ============================
	writer.Write([]string{"SUPPLIER INFO"})
	// Write column headers for supplier information
	writer.Write([]string{
		"Inventory ID", "Supplier ID", "Supplier Name",
		"Contact Email", "Contact Phone",
	})

	// Write supplier info data row
	s := inv.SupplierInfo
	writer.Write([]string{
		inv.InventoryID,
		s.SupplierID,
		s.Name,
		s.ContactEmail,
		s.ContactPhone,
	})

	writer.Write([]string{}) // Empty row separator between sections

	// ============================
	// ADDITIONAL INFO SECTION
	// ============================
	writer.Write([]string{"ADDITIONAL INFO"})
	// Write column headers for additional information
	writer.Write([]string{
		"Inventory ID",
		"Manufacturing Date", "Expiry Date", "Warranty",
		"Placed On", "Low Stock Threshold",
	})

	// Write additional info data row
	writer.Write([]string{
		inv.InventoryID,
		ptrToStr(inv.ManufacturingDate), // Handle nullable date
		ptrToStr(inv.ExpiryDate),        // Handle nullable date
		ptrToStr(inv.Warranty),          // Handle nullable warranty
		inv.PlacedOn,
		strconv.Itoa(inv.LowStockThreshold), // Convert threshold to string
	})

	// Flush buffer and return any write errors
	writer.Flush()
	return writer.Error()
}

// ptrToStr safely converts string pointer to string.
//
// Returns empty string if pointer is nil, otherwise returns the dereferenced value.
//
// Parameters:
//   - s: *string - Pointer to string (may be nil)
//
// Returns:
//   - string: Dereferenced value or empty string
func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// floatToStr converts float to formatted string.
//
// Formats float with 2 decimal places for CSV export.
//
// Parameters:
//   - f: float64 - Float value to format
//
// Returns:
//   - string: Formatted float string (e.g., "123.45")
func floatToStr(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
