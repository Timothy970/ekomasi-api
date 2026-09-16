// Package models provides data access functions for the Ekomasi e-commerce inventory management system.
//
// This file handles inventory operations including:
//   - Inventory listing with filtering (category, stock level, warehouse, search)
//   - Inventory CRUD operations (create, read, update, delete)
//   - Batch tracking with images, expiry dates, and manufacturing dates
//   - Quality inspection recording with inspector details and images
//   - Inventory condition and handling notes
//   - Inventory turnover ratio calculations for performance analysis
//   - Stock history tracking with pagination
//   - Stock summary reporting across warehouses
//   - Supplier information retrieval
package models

import (
	"ekomasi_backend/dtos"
	"fmt"
	"math"
	"time"

	"github.com/teris-io/shortid"
)

// StoreHandlingNotes records handling notes and condition for an inventory batch.
//
// This function stores notes about how inventory should be handled along with
// the condition status of the batch (e.g., "good", "damaged", "needs inspection").
//
// Parameters:
//   - req: dtos.InventoryCondition containing:
//   - BatchID: The batch_id (validated for existence)
//   - HandlingNotes: Instructions or observations about handling
//   - ConditionID: The condition_id (validated for existence)
//
// Returns:
//   - error: "batch not found" if batch doesn't exist,
//     "condition not found" if condition doesn't exist,
//     or database error
func StoreHandlingNotes(db DBExecutor, req dtos.InventoryCondition) error {
	// Validate batch exists
	err := isBatchThere(db, req.BatchID)
	if err != nil {
		return err
	}

	// Generate unique handling note ID
	notesID, _ := shortid.Generate()

	// Insert handling notes record
	_, err = db.Exec(`
		INSERT INTO inventory_handling_notes (handling_note_id, batch_id, handling_notes)
		VALUES (?, ?, ?)`,
		notesID, req.BatchID, req.HandlingNotes)
	return err
}

// StoreInventoryTracking creates a new inventory record with supplier and warehouse tracking.
//
// This function creates inventory with supplier linkage and automatically updates
// the product's total stock quantity across all warehouses.
//
// Parameters:
//   - req: dtos.InventoryTracking containing:
//   - ProductID: The product_id (validated for existence)
//   - SupplierID: The supplier_id (validated for existence)
//   - StoreID: The warehouse_id (validated for existence)
//   - Quantity: Initial stock quantity
//   - LowStockThreshold: Threshold for low stock alerts
//
// Returns:
//   - string: Generated inventory_id
//   - error: "product not found" if product doesn't exist,
//     "supplier not found" if supplier doesn't exist,
//     "warehouse not found" if warehouse doesn't exist,
//     or database error
//
// Side Effects:
//   - Updates product's total stock_quantity by adding the new inventory quantity
func StoreInventoryTracking(db DBExecutor, req dtos.InventoryTracking) (string, error) {
	// Validate product exists
	err := IsProductThere(db, req.ProductID)
	if err != nil {
		return "", err
	}

	// Validate supplier exists
	if req.SupplierID != nil {
		err = isSupplierThere(db, *req.SupplierID)
		if err != nil {
			return "", err
		}
	}
	// Validate warehouse exists
	exists, err := RecordExists(db, "warehouses", "warehouse_id = ?", req.StoreID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("warehouse not found")
	}

	// Generate unique inventory ID
	inventoryID, _ := shortid.Generate()

	// Insert inventory record with supplier and warehouse tracking
	_, err = db.Exec(`
		INSERT INTO inventory (inventory_id, product_id, quantity, low_stock_threshold, warehouse_id, supplier_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		inventoryID, req.ProductID, req.Quantity, req.LowStockThreshold, req.StoreID, req.SupplierID)
	if err != nil {
		return "", err
	}

	// Update product's total stock quantity across all warehouses
	err = UpdateProductTotalQuantity(db, req.ProductID, req.Quantity)
	if err != nil {
		return "", err
	}

	return inventoryID, nil
}

// UpdateProductTotalQuantity increments a product's total stock quantity.
//
// This is an internal helper function that adds to the product's aggregate
// stock_quantity field when new inventory is added.
//
// Parameters:
//   - productID: The product_id to update
//   - quantityToAdd: Amount to add to current stock_quantity
//
// Returns:
//   - error: Database error if update fails
func UpdateProductTotalQuantity(db DBExecutor, productID string, quantityToAdd int) error {
	// Increment product's total stock quantity
	_, err := db.Exec(`
		UPDATE products
		SET stock_quantity = stock_quantity + ?
		WHERE product_id = ?`,
		quantityToAdd, productID)
	return err
}

// GetInventoryStockSummary retrieves aggregated stock metrics for an inventory item.
//
// This function calculates total stock across warehouses, minimum threshold, and
// total sales for a specific inventory entry with optional warehouse filtering.
//
// Parameters:
//   - inventoryID: The inventory_id to summarize (validated for existence)
//   - storeID: Optional warehouse_id filter (empty string for all warehouses)
//
// Returns:
//   - *dtos.InventoryStockSummary: Pointer to summary with:
//   - TotalStock: Sum of quantity across filtered warehouses
//   - MinimumThreshold: Lowest low_stock_threshold value
//   - TotalSales: Sum of quantities sold from order_items
//   - error: "inventory not found" if inventory doesn't exist or database error
func GetInventoryStockSummary(db DBExecutor, inventoryID, storeID string) (*dtos.InventoryStockSummary, error) {
	// Validate inventory exists
	if err := isInventoryThere(db, inventoryID); err != nil {
		return nil, err
	}

	// Struct to hold results
	var summary dtos.InventoryStockSummary

	// Query to aggregate stock across warehouses and get minimum threshold
	query := `
		SELECT 
			product_id,
			SUM(quantity) AS total_stock,
			MIN(low_stock_threshold) AS min_threshold
		FROM inventory
		WHERE inventory_id = ?
	`

	args := []interface{}{inventoryID}

	// Add optional warehouse filter
	if storeID != "" {
		query += " AND warehouse_id = ?"
		args = append(args, storeID)
	}

	query += " GROUP BY product_id"

	// Execute stock summary query
	row := db.QueryRow(query, args...)
	var productID string
	if err := row.Scan(&productID, &summary.TotalStock, &summary.MinimumThreshold); err != nil {
		return nil, err
	}

	// Query total sales from order_items
	salesQuery := `
		SELECT COALESCE(SUM(quantity), 0)
		FROM order_items
		WHERE product_id = ?
	`
	salesArgs := []interface{}{productID}

	// Add optional warehouse filter for sales
	if storeID != "" {
		salesQuery += " AND store_id = ?"
		salesArgs = append(salesArgs, storeID)
	}

	// Execute sales count query
	if err := db.QueryRow(salesQuery, salesArgs...).Scan(&summary.TotalSales); err != nil {
		return nil, err
	}

	return &summary, nil
}

// GetInventoryStockHistory retrieves paginated inventory quantity change history.
//
// This function fetches historical inventory records for a product with calculated
// values (description with quantity and monetary amount based on buying price).
//
// Parameters:
//   - inventoryID: The inventory_id to get history for (validated for existence)
//   - page: Page number (1-indexed)
//   - size: Items per page
//
// Returns:
//   - *[]dtos.InventoryStockHistory: Pointer to array of history records with:
//   - Description: Formatted as "ProductName × Quantity"
//   - Amount: Calculated as Quantity × BuyingPrice
//   - Date: last_updated timestamp
//   - *dtos.PaginationMeta: Pagination metadata (page, size, total, prev/next flags)
//   - error: "inventory not found" if inventory doesn't exist,
//     "failed to fetch product_id" if product lookup fails,
//     "failed to fetch product details" if product info unavailable,
//     or database error
func GetInventoryStockHistory(db DBExecutor, inventoryID string, page, size int) (*[]dtos.InventoryStockHistory, *dtos.PaginationMeta, error) {
	// Validate inventory exists
	if err := isInventoryThere(db, inventoryID); err != nil {
		return nil, nil, err
	}

	// Calculate pagination offset
	offset := (page - 1) * size

	// Step 1: Get product_id for the given inventory
	var productID string
	productQuery := `SELECT product_id FROM inventory WHERE inventory_id = ?`
	if err := db.QueryRow(productQuery, inventoryID).Scan(&productID); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch product_id: %v", err)
	}

	// Step 2: Fetch product name and buying price for calculations
	var productName string
	var buyingPrice float64
	productInfoQuery := `SELECT name, buying_price FROM products WHERE product_id = ?`
	if err := db.QueryRow(productInfoQuery, productID).Scan(&productName, &buyingPrice); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch product details: %v", err)
	}

	// Step 3: Count total records for pagination metadata
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM inventory WHERE product_id = ?`
	if err := db.QueryRow(countQuery, productID).Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to count inventory history: %v", err)
	}

	// Step 4: Fetch paginated inventory history ordered by most recent
	query := `
		SELECT quantity, last_updated
		FROM inventory
		WHERE product_id = ?
		ORDER BY last_updated DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.Query(query, productID, size, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch inventory history: %v", err)
	}
	defer rows.Close()

	var history []dtos.InventoryStockHistory

	// Iterate through history records
	for rows.Next() {
		var record dtos.InventoryStockHistory
		var quantity int
		var lastUpdated time.Time

		// Scan quantity and timestamp
		if err := rows.Scan(&quantity, &lastUpdated); err != nil {
			return nil, nil, err
		}

		// Build description with product name and quantity
		record.Description = fmt.Sprintf("%s × %d", productName, quantity)

		// Calculate monetary amount (quantity × buying price)
		record.Amount = float64(quantity) * buyingPrice

		// Set timestamp
		record.Date = lastUpdated

		history = append(history, record)
	}

	// Step 5: Build pagination metadata
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalCount,
		TotalPages: int(math.Ceil(float64(totalCount) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page < int(math.Ceil(float64(totalCount)/float64(size))),
	}

	return &history, meta, nil
}

// helper function to update product buying price and selling price
func UpdateProductPrices(db DBExecutor, productID string, buyingPrice, sellingPrice float64) error {
	_, err := db.Exec(`
		UPDATE products
		SET buying_price = ?, price = ?
		WHERE product_id = ?`,
		buyingPrice, sellingPrice, productID)
	return err
}

// helper to update variant selection stock quantity
func UpdateVariantQuantities(db DBExecutor, variantQuantities []dtos.VariantQuantity) error {
	query := `
		UPDATE product_variant_combinations
		SET stock_quantity = stock_quantity + ?
		WHERE sku = ?`

	for _, vq := range variantQuantities {
		if _, err := db.Exec(query, vq.Quantity, vq.SKU); err != nil {
			return err
		}
	}
	return nil
}
