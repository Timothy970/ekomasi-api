// Package utils provides utility functions for the Ekomasi e-commerce platform.
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
	"ekomasi_backend/dtos"
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

// BulkUploadHeader defines a CSV header with its required status
type BulkUploadHeader struct {
	Name     string
	Required bool
}

// Bulk upload CSV file expected headers
var BulkUploadHeaders = []BulkUploadHeader{
	{Name: "name", Required: true},
	{Name: "description", Required: true},
	{Name: "sku", Required: true},
	{Name: "price", Required: true},
	{Name: "sub_category_id", Required: true},
	{Name: "stock_quantity", Required: false},
	{Name: "tag", Required: false},
	{Name: "low_stock_quantity_warning", Required: false},
	{Name: "barcode", Required: true},
	{Name: "buying_price", Required: true},
	{Name: "weight", Required: true},
	{Name: "weight_limit", Required: true},
	{Name: "dimensions", Required: true},
	{Name: "brand", Required: true},
	{Name: "manufacturer", Required: false},
	{Name: "warranty_period", Required: true},
	{Name: "combination_name", Required: false},
	{Name: "combination_sku", Required: false},
	{Name: "additional_price", Required: false},
	{Name: "option_1", Required: false},
	{Name: "option_2", Required: false},
	{Name: "option_3", Required: false},
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
	expectedHeaders := BulkUploadHeaders

	// Validate minimum required column count (first 16 mandatory fields)
	minRequiredCount := 16
	if len(headers) < minRequiredCount {
		return nil, fmt.Errorf("invalid CSV: missing required columns. Expected at least %d, got %d", minRequiredCount, len(headers))
	}

	// Normalize headers to lowercase for case-insensitive comparison
	for i := range headers {
		headers[i] = strings.ToLower(strings.TrimSpace(headers[i]))
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
			log.Printf("Error reading CSV row %d: %v", rowNumber, err)
			continue // Skip malformed rows
		}

		// Sanitize all field values by trimming whitespace
		for i := range record {
			record[i] = sanitize(record[i])
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
			continue
		}

		// Validate required fields (first 16 columns)
		skipRow := false
		for i := 0; i < minRequiredCount; i++ {
			header := expectedHeaders[i]
			if i < len(record) {
				if err := required(record[i], header.Name, !header.Required); err != nil {
					log.Printf("Row %d skipped: %v", rowNumber, err)
					skipRow = true
					break
				}
			}
		}
		if skipRow {
			continue
		}

		// Safe field accessor
		getField := func(idx int) string {
			if idx < len(record) {
				return record[idx]
			}
			return ""
		}

		// Convert string values to appropriate data types
		price, _ := strconv.ParseFloat(getField(3), 64)
		stockQty, _ := strconv.Atoi(getField(5))
		lowStockWarn, _ := strconv.Atoi(getField(7))
		barcode := getField(8)
		buyingPrice, _ := strconv.ParseFloat(getField(9), 64)
		weight, _ := strconv.ParseFloat(getField(10), 64)
		weightLimit, _ := strconv.ParseFloat(getField(11), 64)
		warrantyType, _ := strconv.Atoi(getField(15))
		additionalPrice, _ := strconv.ParseFloat(getField(18), 64)

		// Construct product struct from parsed values
		product := dtos.BulkUploadProduct{
			Name:                    getField(0),
			Description:             getField(1),
			SKU:                     getField(2),
			Price:                   price,
			CategoryID:              getField(4),
			StockQuantity:           stockQty,
			Tag:                     strToPtr(getField(6)),
			SearchVector:            getField(0),
			LowStockQuantityWarning: lowStockWarn,
			Barcode:                 barcode,
			BuyingPrice:             buyingPrice,
			Weight:                  &weight,
			WeightLimit:             &weightLimit,
			Dimensions:              strToPtr(getField(12)),
			Brand:                   strToPtr(getField(13)),
			Manufacturer:            strToPtr(getField(14)),
			WarrantyPeriod:          &warrantyType,
			CombinationName:         getField(16),
			CombinationSKU:          getField(17),
			AdditionalPrice:         additionalPrice,
			Option1:                 getField(19),
			Option2:                 getField(20),
			Option3:                 getField(21),
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
