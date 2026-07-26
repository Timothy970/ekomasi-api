package services

import (
	"context"
	"database/sql"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"strings"

	"github.com/teris-io/shortid"
)

// ValidateAndProcessBulkImport performs enterprise-grade bulk product import validation and execution
func ValidateAndProcessBulkImport(db models.DBExecutor, tenantID string, products []dtos.BulkUploadProduct, mode string, validateOnly bool, userID string) (*dtos.BulkImportDiagnosticResult, error) {
	if mode == "" {
		mode = "upsert"
	}
	mode = strings.ToLower(mode)

	result := &dtos.BulkImportDiagnosticResult{
		TotalRows:    len(products),
		Mode:         mode,
		ValidateOnly: validateOnly,
		Errors:       []dtos.BulkValidationRowError{},
		CreatedSKUs:  []string{},
		UpdatedSKUs:  []string{},
	}

	// 1. Pre-fetch existing categories for tenant for Category Name & Slug resolution
	categoryMap := make(map[string]string) // normalized name/slug/uuid -> category UUID
	var catNamesList []string
	catQuery := `SELECT id, name, COALESCE(slug, '') FROM categories WHERE tenant_id = ?`
	catRows, err := db.QueryContext(context.Background(), catQuery, tenantID)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var id, name, slug string
			if err := catRows.Scan(&id, &name, &slug); err == nil {
				categoryMap[strings.ToLower(id)] = id
				categoryMap[strings.ToLower(name)] = id
				catNamesList = append(catNamesList, name)
				if slug != "" {
					categoryMap[strings.ToLower(slug)] = id
					catNamesList = append(catNamesList, slug)
				}
			}
		}
	}

	// 2. Pre-fetch existing SKUs for tenant
	skuMap := make(map[string]string) // lowercase SKU -> product ID
	skuQuery := `SELECT id, LOWER(sku) FROM products WHERE tenant_id = ?`
	skuRows, err := db.QueryContext(context.Background(), skuQuery, tenantID)
	if err == nil {
		defer skuRows.Close()
		for skuRows.Next() {
			var id, sku string
			if err := skuRows.Scan(&id, &sku); err == nil {
				skuMap[sku] = id
			}
		}
	}

	validProducts := []dtos.BulkUploadProduct{}

	// 3. Row-by-Row Granular Validation
	for idx, p := range products {
		rowNum := idx + 2 // Row 1 is CSV Header
		hasError := false

		// Check Name
		if strings.TrimSpace(p.Name) == "" {
			result.Errors = append(result.Errors, dtos.BulkValidationRowError{
				RowNumber:    rowNum,
				SKU:          p.SKU,
				ProductName:  p.Name,
				Field:        "name",
				Value:        p.Name,
				ErrorMessage: "Product name is required",
			})
			hasError = true
		}

		// Check SKU
		cleanSKU := strings.TrimSpace(p.SKU)
		if cleanSKU == "" {
			result.Errors = append(result.Errors, dtos.BulkValidationRowError{
				RowNumber:    rowNum,
				SKU:          cleanSKU,
				ProductName:  p.Name,
				Field:        "sku",
				Value:        cleanSKU,
				ErrorMessage: "SKU is required",
			})
			hasError = true
		}

		// Check Price
		if p.Price < 0 {
			result.Errors = append(result.Errors, dtos.BulkValidationRowError{
				RowNumber:    rowNum,
				SKU:          cleanSKU,
				ProductName:  p.Name,
				Field:        "price",
				Value:        fmt.Sprintf("%.2f", p.Price),
				ErrorMessage: "Price must be greater than or equal to 0",
			})
			hasError = true
		}

		// Check Stock Quantity
		if p.StockQuantity < 0 {
			result.Errors = append(result.Errors, dtos.BulkValidationRowError{
				RowNumber:    rowNum,
				SKU:          cleanSKU,
				ProductName:  p.Name,
				Field:        "stock_quantity",
				Value:        fmt.Sprintf("%d", p.StockQuantity),
				ErrorMessage: "Stock quantity cannot be negative",
			})
			hasError = true
		}

		// Category Name / UUID Auto-Resolution with "Did You Mean?" Fuzzy Match Suggestion
		rawCat := strings.ToLower(strings.TrimSpace(p.CategoryID))
		resolvedCatID, catFound := categoryMap[rawCat]
		if !catFound && rawCat != "" {
			errMsg := fmt.Sprintf("Category '%s' not found for this tenant", p.CategoryID)
			if bestMatch, ok := utils.FindBestFuzzyMatch(rawCat, catNamesList, 3); ok {
				errMsg += fmt.Sprintf(". Did you mean '%s'?", bestMatch)
			}
			result.Errors = append(result.Errors, dtos.BulkValidationRowError{
				RowNumber:    rowNum,
				SKU:          cleanSKU,
				ProductName:  p.Name,
				Field:        "category",
				Value:        p.CategoryID,
				ErrorMessage: errMsg,
			})
			hasError = true
		} else if catFound {
			p.CategoryID = resolvedCatID
		}

		// Mode Validation
		lowerSKU := strings.ToLower(cleanSKU)
		_, exists := skuMap[lowerSKU]

		if mode == "create_only" && exists {
			result.Errors = append(result.Errors, dtos.BulkValidationRowError{
				RowNumber:    rowNum,
				SKU:          cleanSKU,
				ProductName:  p.Name,
				Field:        "sku",
				Value:        cleanSKU,
				ErrorMessage: fmt.Sprintf("SKU '%s' already exists (mode: create_only)", cleanSKU),
			})
			hasError = true
		} else if mode == "update_stock" && !exists {
			result.Errors = append(result.Errors, dtos.BulkValidationRowError{
				RowNumber:    rowNum,
				SKU:          cleanSKU,
				ProductName:  p.Name,
				Field:        "sku",
				Value:        cleanSKU,
				ErrorMessage: fmt.Sprintf("SKU '%s' not found for update (mode: update_stock)", cleanSKU),
			})
			hasError = true
		}

		if hasError {
			result.InvalidRowsCount++
		} else {
			result.ValidRowsCount++
			validProducts = append(validProducts, p)
		}
	}

	// 4. Return Dry Run Results if validateOnly = true
	if validateOnly {
		return result, nil
	}

	// 5. Database Batch Execution inside Transaction
	_, ok := db.(*sql.Tx)
	var localTx *sql.Tx

	if !ok {
		localTx, err = models.DB.Begin()
		if err != nil {
			return nil, fmt.Errorf("failed to start bulk import transaction: %w", err)
		}
		defer localTx.Rollback()
		db = localTx
	}

	for _, p := range validProducts {
		cleanSKU := strings.TrimSpace(p.SKU)
		lowerSKU := strings.ToLower(cleanSKU)
		existingID, exists := skuMap[lowerSKU]

		if exists && (mode == "upsert" || mode == "update_stock") {
			// Update existing product price, stock & details
			updateQuery := `
				UPDATE products
				SET price = ?, stock_quantity = ?, name = COALESCE(NULLIF(?, ''), name),
				    description = COALESCE(NULLIF(?, ''), description), updated_at = CURRENT_TIMESTAMP
				WHERE id = ? AND tenant_id = ?
			`
			if _, err := db.ExecContext(context.Background(), updateQuery, p.Price, p.StockQuantity, p.Name, p.Description, existingID, tenantID); err != nil {
				return nil, fmt.Errorf("failed to update SKU '%s': %w", cleanSKU, err)
			}
			result.UpdatedSKUs = append(result.UpdatedSKUs, cleanSKU)
			result.ProcessedCount++
		} else if !exists && (mode == "upsert" || mode == "create_only") {
			// Insert new product into bulk_products staging / main products table
			newID, _ := shortid.Generate()
			insertQuery := `
				INSERT INTO bulk_products (
					id, name, description, sku, price, sub_category_id,
					stock_quantity, tag, low_stock_quantity_warning, barcode,
					buying_price, weight, weight_limit, dimensions, brand, manufacturer, warranty_period,
					created_by_id
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`
			var weight, weightLimit float64
			if p.Weight != nil {
				weight = *p.Weight
			}
			if p.WeightLimit != nil {
				weightLimit = *p.WeightLimit
			}
			var warranty int
			if p.WarrantyPeriod != nil {
				warranty = *p.WarrantyPeriod
			}
			var brand, manufacturer, tag, dimensions string
			if p.Brand != nil {
				brand = *p.Brand
			}
			if p.Manufacturer != nil {
				manufacturer = *p.Manufacturer
			}
			if p.Tag != nil {
				tag = *p.Tag
			}
			if p.Dimensions != nil {
				dimensions = *p.Dimensions
			}

			if _, err := db.ExecContext(context.Background(), insertQuery,
				newID, p.Name, p.Description, p.SKU, p.Price, p.CategoryID,
				p.StockQuantity, tag, p.LowStockQuantityWarning, p.Barcode,
				p.BuyingPrice, weight, weightLimit, dimensions, brand, manufacturer, warranty,
				userID,
			); err != nil {
				return nil, fmt.Errorf("failed to insert SKU '%s': %w", cleanSKU, err)
			}
			existingID = newID
			result.CreatedSKUs = append(result.CreatedSKUs, cleanSKU)
			result.ProcessedCount++
		}

		// Insert Combination & Variant Offset if Combination details provided
		if p.CombinationName != "" || p.CombinationSKU != "" {
			combSKU := p.CombinationSKU
			if combSKU == "" {
				combSKU = cleanSKU + "-" + strings.ReplaceAll(p.CombinationName, " ", "-")
			}
			_, _ = models.InsertCombination(db, dtos.Combination{
				ProductID:       existingID,
				Name:            p.CombinationName,
				SKU:             combSKU,
				AdditionalPrice: p.AdditionalPrice,
			})
		}
	}

	if localTx != nil {
		if err := localTx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit bulk import transaction: %w", err)
		}
	}

	return result, nil
}
