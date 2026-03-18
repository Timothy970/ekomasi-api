// Package models provides data access functions for the Adenzo e-commerce inventory management system.
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
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

var noinventory = "inventory not found"

// ListInventory retrieves paginated inventory with multi-criteria filtering.
//
// This function fetches inventory records with product details, enriched with
// images and supplier information. Supports filtering by category, stock level,
// warehouse, and search term.
//
// Parameters:
//   - page: Page number (1-indexed)
//   - size: Items per page
//   - categoryID: Filter by category (empty string for all)
//   - stock: Stock level filter - "in" (quantity > 0), "out" (quantity = 0), "low" (≤ threshold), or "" for all
//   - storeID: Filter by warehouse_id (empty string for all)
//   - search: Search term matching product name, category name, or inventory_id (empty for no search)
//
// Returns:
//   - []dtos.Inventory: Array of inventory items with product details, images, and supplier info
//   - *dtos.PaginationMeta: Pagination metadata (page, size, total, prev/next flags)
//   - error: Database error if query fails
//
// Stock Level Filters:
//   - "in": Products with stock available (quantity > 0)
//   - "out": Out of stock products (quantity = 0)
//   - "low": Low stock products (quantity ≤ low_stock_threshold)
func ListInventory(db DBExecutor, page, size int, categoryID, stock, storeID, search string) ([]dtos.Inventory, *dtos.PaginationMeta, error) {
	// Calculate pagination offset
	offset := (page - 1) * size

	// Base COUNT query with joins to products, categories, and warehouses
	baseCount := `
		SELECT COUNT(*) 
		FROM inventory inv
		JOIN products prd ON inv.product_id = prd.product_id
		JOIN categories cat ON prd.category_id = cat.category_id
		JOIN warehouses whse ON inv.warehouse_id = whse.warehouse_id
	`

	// Build dynamic filter conditions
	var filters []string
	var args []interface{}

	// Filter by category if provided
	if categoryID != "" {
		filters = append(filters, "prd.category_id = ?")
		args = append(args, categoryID)
	}

	// Filter by stock level status
	if stock != "" {
		switch strings.ToLower(stock) {
		case "in":
			// In stock: quantity greater than zero
			filters = append(filters, "inv.quantity > 0")
		case "out":
			// Out of stock: quantity equals zero
			filters = append(filters, "inv.quantity = 0")
		case "low":
			// Low stock: at or below threshold
			filters = append(filters, "inv.quantity <= inv.low_stock_threshold")
		}
	}

	// Filter by warehouse if provided
	if storeID != "" {
		filters = append(filters, "inv.warehouse_id = ?")
		args = append(args, storeID)
	}

	// Add search filter across product name, category name, and inventory ID
	if search != "" {
		filters = append(filters, "(prd.name LIKE ? OR cat.name LIKE ? OR inv.inventory_id LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Apply filters to COUNT query
	countSQL := baseCount
	if len(filters) > 0 {
		countSQL += " WHERE " + strings.Join(filters, " AND ")
	}

	// Get total count for pagination
	var totalItems int
	if err := db.QueryRow(countSQL, args...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Build SELECT query with same filters
	selectSQL := `
		SELECT 
			inv.inventory_id, inv.warehouse_id, prd.product_id, inv.variant_id,
			inv.quantity, inv.low_stock_threshold, prd.name, prd.description,
			prd.sku, prd.tag, prd.price, prd.category_id, 
			cat.name, prd.stock_quantity, prd.search_vector, whse.name, prd.buying_price
		FROM inventory inv
		JOIN products prd ON inv.product_id = prd.product_id
		JOIN categories cat ON prd.category_id = cat.category_id
		JOIN warehouses whse ON inv.warehouse_id = whse.warehouse_id
	`

	// Apply same filters to SELECT query
	if len(filters) > 0 {
		selectSQL += " WHERE " + strings.Join(filters, " AND ")
	}

	// Order by most recently updated, with pagination
	selectSQL += " ORDER BY inv.last_updated DESC LIMIT ? OFFSET ?"

	// Execute query with pagination parameters
	rows, err := db.Query(selectSQL, append(args, size, offset)...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var inventories []dtos.Inventory

	// Iterate through result rows
	for rows.Next() {
		var inv dtos.Inventory
		var buyingPrice sql.NullFloat64
		var lowStockThreshold sql.NullInt64

		// Scan row into inventory struct
		err := rows.Scan(
			&inv.InventoryID, &inv.StoreID, &inv.ProductID, &inv.VariantID,
			&inv.Quantity, &lowStockThreshold, &inv.Name, &inv.Description,
			&inv.SKU, &inv.Tag, &inv.Price, &inv.CategoryID,
			&inv.CategoryName, &inv.StockQuantity, &inv.SearchVector, &inv.Store, &buyingPrice,
		)
		if err != nil {
			return nil, nil, err
		}

		// Convert nullable buying price
		if buyingPrice.Valid {
			inv.BuyingPrice = &buyingPrice.Float64
		}
		// Convert nullable low stock threshold
		if lowStockThreshold.Valid {
			threshold := int(lowStockThreshold.Int64)
			inv.LowStockThreshold = threshold
		}

		// Enrich with product images
		inv.Images, _ = fetchProductImages(db, inv.ProductID)

		// Enrich with supplier information
		inv.SupplierInfo, _ = fetchSupplierByInventoryID(db, inv.InventoryID)

		inventories = append(inventories, inv)
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalItems) / float64(size)))

	// Build pagination metadata
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return inventories, &meta, nil
}

// CreateInventory creates a new inventory record for a product variant.
//
// This function validates product and variant existence before creating the
// inventory entry with quantity and low stock threshold.
//
// Parameters:
//   - inv: dtos.CreateInventoryRequest containing:
//   - ProductID: The product_id (validated for existence)
//   - VariantID: The variant_id (validated for existence)
//   - Quantity: Initial stock quantity
//   - LowStockThreshold: Threshold for low stock alerts
//
// Returns:
//   - error: "product not found" if product doesn't exist,
//     "variant not found" if variant doesn't exist,
//     or database error
func CreateInventory(db DBExecutor, inv dtos.CreateInventoryRequest) error {
	// Validate product exists
	err := IsProductThere(db, inv.ProductID)
	if err != nil {
		return err
	}

	// Validate variant exists
	err = isVariantThere(db, inv.VariantID)
	if err != nil {
		return err
	}

	// Generate unique inventory ID
	inventoryID, _ := shortid.Generate()

	// Insert inventory record
	_, err = db.Exec(`
		INSERT INTO inventory (inventory_id, product_id, variant_id, quantity, low_stock_threshold)
		VALUES (?, ?, ?, ?, ?)`,
		inventoryID, inv.ProductID, inv.VariantID, inv.Quantity, inv.LowStockThreshold)
	return err
}

// GetInventory retrieves detailed inventory information with comprehensive enrichment.
//
// This function fetches a single inventory record with extensive details including
// product info, batch details, inspection records, handling notes, warranties,
// product images, and supplier information.
//
// Parameters:
//   - inventoryID: The inventory_id to retrieve
//
// Returns:
//   - *dtos.SingleInventory: Pointer to inventory with enriched data:
//   - Basic Info: InventoryID, StoreID, ProductID, Quantity, Threshold
//   - Product: Name, Description, SKU, Price, Category, Stock
//   - Batch: BatchNumber, ExpiryDate, ManufacturingDate, BatchImages
//   - Inspection: InspectionDate, Inspector (user object), InspectionNotes, InspectionImages
//   - Handling: HandlingNotes, ConditionID
//   - Pricing: BuyingPrice, UnitCost
//   - Enrichment: Product Images, Supplier Info, Warranty
//   - error: "inventory not found" if inventory doesn't exist or database error
//
// JSON Deserialization:
//   - InspectionImages: JSON array of image URLs
//   - BatchImages: JSON array of image URLs
func GetInventory(db DBExecutor, inventoryID string) (*dtos.SingleInventory, error) {
	// Validate inventory exists
	err := isInventoryThere(db, inventoryID)
	if err != nil {
		return nil, err
	}

	// Comprehensive query with multiple LEFT JOINs for enrichment
	query := `
		SELECT 
			inv.inventory_id, inv.warehouse_id, prd.product_id, inv.variant_id, inv.quantity, inv.low_stock_threshold,
			prd.name, prd.description, prd.sku, prd.tag, prd.price,
			prd.category_id, cat.name, prd.stock_quantity, prd.search_vector, invbatch.batch_number, invbatch.expiry_date, invbatch.manufacturing_date, prdWarranty.warranty_period, inv.last_updated, prd.buying_price, bacthinsp.inspection_date, bacthinsp.inspector_id, bacthinsp.inspection_notes, bacthinsp.images, invbatch.images, invhandlingnotes.handling_notes, invhandlingnotes.condition_id, whse.name
		FROM inventory inv
		JOIN products prd ON inv.product_id = prd.product_id
		JOIN categories cat ON prd.category_id = cat.category_id
		LEFT JOIN inventory_batches invbatch ON inv.inventory_id = invbatch.inventory_id
		LEFT JOIN product_warranties prdWarranty ON prd.product_id = prdWarranty.product_id
		LEFT JOIN batch_inspections bacthinsp ON invbatch.batch_id = bacthinsp.batch_id
		LEFT JOIN inventory_handling_notes invhandlingnotes ON invbatch.batch_id = invhandlingnotes.batch_id
		JOIN warehouses whse ON inv.warehouse_id = whse.warehouse_id
		WHERE inv.inventory_id = ?
		ORDER BY inv.last_updated DESC
	`

	row := db.QueryRow(query, inventoryID)

	var inv dtos.SingleInventory
	var inspectionDate, inspectorID, inspectionImagesJSON, batchImagesJSON sql.NullString
	var buyingPrice sql.NullFloat64
	var lowStockThreshold sql.NullInt64

	// Scan all fields including nullable JSON columns
	if err := row.Scan(
		&inv.InventoryID, &inv.StoreID, &inv.ProductID, &inv.VariantID, &inv.Quantity, &lowStockThreshold,
		&inv.Name, &inv.Description, &inv.SKU, &inv.Tag, &inv.Price,
		&inv.CategoryID, &inv.CategoryName, &inv.StockQuantity, &inv.SearchVector, &inv.BatchNumber, &inv.ExpiryDate, &inv.ManufacturingDate, &inv.Warranty, &inv.PlacedOn, &buyingPrice, &inspectionDate, &inspectorID, &inv.InspectionNotes, &inspectionImagesJSON, &batchImagesJSON, &inv.HandlingNotes, &inv.ConditionID, &inv.Store,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New(noinventory)
		}
		return nil, err
	}

	// Process nullable fields and enrich inventory data
	if err := enrichInventoryData(db, &inv, inspectionDate, inspectorID, inspectionImagesJSON, batchImagesJSON, buyingPrice, lowStockThreshold, inventoryID); err != nil {
		return nil, err
	}

	return &inv, nil
}

// enrichInventoryData processes nullable fields and enriches inventory with additional data.
func enrichInventoryData(db DBExecutor, inv *dtos.SingleInventory, inspectionDate, inspectorID, inspectionImagesJSON, batchImagesJSON sql.NullString, buyingPrice sql.NullFloat64, lowStockThreshold sql.NullInt64, inventoryID string) error {
	// Convert nullable inspection date
	if inspectionDate.Valid {
		inspDate := StringToTime(inspectionDate.String)
		inv.InspectionDate = &inspDate
	}

	// Convert nullable low stock threshold
	if lowStockThreshold.Valid {
		threshold := int(lowStockThreshold.Int64)
		inv.LowStockThreshold = threshold
	}

	// Fetch inspector user details if present
	if inspectorID.Valid {
		inspector, err := GetUserByUserID(db, inspectorID.String)
		if err != nil {
			return err
		}
		inv.Inspector = inspector
	}

	// Unmarshal JSON images
	unmarshalJSONImages(inspectionImagesJSON, &inv.InspectionImages)
	unmarshalJSONImages(batchImagesJSON, &inv.BatchImages)

	// Convert nullable buying price and set unit cost
	if buyingPrice.Valid {
		inv.BuyingPrice = &buyingPrice.Float64
		inv.UnitCost = &buyingPrice.Float64
	}

	// Enrich with product images
	if imgs, err := fetchProductImages(db, inv.ProductID); err == nil {
		inv.Images = imgs
	}

	// Enrich with supplier information
	inv.SupplierInfo, _ = fetchSupplierByInventoryID(db, inventoryID)
	return nil
}

// unmarshalJSONImages unmarshals a JSON string into a string slice pointer.
func unmarshalJSONImages(jsonData sql.NullString, target **[]string) {
	if jsonData.Valid {
		var imgs []string
		if err := json.Unmarshal([]byte(jsonData.String), &imgs); err == nil {
			*target = &imgs
		}
	}
}

// fetchSupplierByInventoryID retrieves supplier details for an inventory record.
//
// This is an internal helper function that fetches supplier contact information
// associated with a specific inventory entry.
//
// Parameters:
//   - inventoryID: The inventory_id to get supplier for
//
// Returns:
//   - dtos.Supplier: Supplier with ID, name, email, phone, and extra details
//   - error: Database error or sql.ErrNoRows if no supplier found
func fetchSupplierByInventoryID(db DBExecutor, inventoryID string) (dtos.Supplier, error) {
	var supplier dtos.Supplier

	// Query supplier via inventory JOIN
	query := `
		SELECT s.supplier_id, s.name, s.contact_email, s.contact_phone, s.extra_details
		FROM suppliers s
		JOIN inventory inv ON s.supplier_id = inv.supplier_id
		WHERE inv.inventory_id = ?
	`

	// Fetch supplier details
	err := db.QueryRow(query, inventoryID).Scan(&supplier.SupplierID, &supplier.Name, &supplier.ContactEmail, &supplier.ContactPhone, &supplier.ExtraDetails)
	return supplier, err
}

// UpdateInventory updates inventory quantity and/or low stock threshold.
//
// This function performs partial updates - only provided fields are updated.
// At least one parameter must be non-nil.
//
// Parameters:
//   - inventoryID: The inventory_id to update
//   - quantity: Pointer to new quantity (nil to skip update)
//   - threshold: Pointer to new low_stock_threshold (nil to skip update)
//
// Returns:
//   - error: "inventory not found" if inventory doesn't exist or database error
func UpdateInventory(db DBExecutor, inventoryID string, quantity, threshold *int) error {
	// Validate inventory exists
	err := isInventoryThere(db, inventoryID)
	if err != nil {
		return err
	}

	// Build dynamic UPDATE query
	query := "UPDATE inventory SET "
	args := []interface{}{}

	// Add quantity if provided
	if quantity != nil {
		query += "quantity = ?, "
		args = append(args, *quantity)
	}

	// Add threshold if provided
	if threshold != nil {
		query += "low_stock_threshold = ?, "
		args = append(args, *threshold)
	}

	// Remove trailing comma and add WHERE clause
	query = query[:len(query)-2]
	query += " WHERE inventory_id = ?"
	args = append(args, inventoryID)

	// Execute update
	_, err = db.Exec(query, args...)
	return err
}

// DeleteInventory permanently removes an inventory record.
//
// Parameters:
//   - inventoryID: The inventory_id to delete
//
// Returns:
//   - error: "inventory not found" if inventory doesn't exist or database error
func DeleteInventory(db DBExecutor, inventoryID string) error {
	// Validate inventory exists
	exists, err := RecordExists(db, "inventory", "inventory_id = ?", inventoryID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noinventory)
	}

	// Delete inventory record
	_, err = db.Exec(`DELETE FROM inventory WHERE inventory_id = ?`, inventoryID)
	return err
}

// getPeriodExpr returns SQL expression for time period grouping.
//
// This is an internal helper function that generates SQL expressions for
// grouping data by different time periods in inventory turnover calculations.
//
// Parameters:
//   - groupBy: Period type - "week", "month", "quarter", "year", or "" (defaults to daily)
//
// Returns:
//   - string: SQL expression for grouping:
//   - "week": YEARWEEK(o.created_at)
//   - "month": DATE_FORMAT(o.created_at, '%Y-%m')
//   - "quarter": CONCAT(YEAR(o.created_at), '-Q', QUARTER(o.created_at))
//   - "year": YEAR(o.created_at)
//   - default: DATE_FORMAT(o.created_at, '%Y-%m-%d') (daily)
func getPeriodExpr(groupBy string) string {
	switch strings.ToLower(groupBy) {
	case "week":
		return "YEARWEEK(o.created_at)"
	case "month":
		return "DATE_FORMAT(o.created_at, '%Y-%m')"
	case "quarter":
		return "CONCAT(YEAR(o.created_at), '-Q', QUARTER(o.created_at))"
	case "year":
		return "YEAR(o.created_at)"
	default:
		return "DATE_FORMAT(o.created_at, '%Y-%m-%d')" // daily fallback
	}
}

// GetInventoryTurnover calculates inventory turnover ratios for all products.
//
// This function computes inventory turnover metrics by comparing cost of goods sold (COGS)
// against average inventory value, grouped by time period. Higher turnover indicates
// faster inventory movement.
//
// Parameters:
//   - start: Start date for analysis period
//   - end: End date for analysis period
//   - groupBy: Time grouping - "week", "month", "quarter", "year", or "" for daily
//
// Returns:
//   - []dtos.InventoryTurnoverItem: Array of turnover metrics with:
//   - ProductID: Product identifier (nullable)
//   - CategoryID: Category identifier (nullable)
//   - AvgInventory: Average inventory value (unit_cost × quantity)
//   - COGS: Cost of goods sold (unit_price × quantity)
//   - TurnoverRatio: COGS / AvgInventory (0 if AvgInventory is 0)
//   - error: Database error if query fails
//
// Formula:
//   - TurnoverRatio = COGS / Average Inventory Value
//   - Higher ratio = inventory sells/turns over more frequently
func GetInventoryTurnover(db DBExecutor, start, end time.Time, groupBy string) ([]dtos.InventoryTurnoverItem, error) {
	// Get period grouping SQL expression
	periodExpr := getPeriodExpr(groupBy)

	// Build query with dynamic period grouping
	query := fmt.Sprintf(`
        SELECT 
            oi.product_id,
            p.category_id,
            AVG(poi.unit_cost * poi.quantity) AS avg_inventory,
            SUM(oi.unit_price * oi.quantity) AS cogs
        FROM orders o
        JOIN order_items oi ON o.order_id = oi.order_id
        JOIN products p ON oi.product_id = p.product_id
        LEFT JOIN purchase_order_items poi ON poi.product_id = oi.product_id
        LEFT JOIN purchase_orders po ON poi.po_id = po.po_id
        WHERE o.created_at BETWEEN ? AND ?
        GROUP BY %s, oi.product_id, p.category_id
    `, periodExpr)

	// Execute query with date range
	rows, err := db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []dtos.InventoryTurnoverItem{}

	// Iterate through results
	for rows.Next() {
		var item dtos.InventoryTurnoverItem
		var avgInv, cogs sql.NullFloat64
		var productID sql.NullString
		var categoryID sql.NullString

		// Scan row with nullable fields
		if err := rows.Scan(&productID, &categoryID, &avgInv, &cogs); err != nil {
			return nil, err
		}

		// Convert nullable product and category IDs
		if productID.Valid {
			item.ProductID = &productID.String
		}
		if categoryID.Valid {
			item.CategoryID = &categoryID.String
		}

		// Set inventory and COGS values
		item.AvgInventory = avgInv.Float64
		item.COGS = cogs.Float64

		// Calculate turnover ratio (avoid division by zero)
		if item.AvgInventory > 0 {
			item.TurnoverRatio = item.COGS / item.AvgInventory
		}

		results = append(results, item)
	}
	return results, nil
}

// GetInventoryTurnoverByProduct calculates inventory turnover ratio for a specific product.
//
// This function computes turnover metrics for a single product, comparing COGS against
// average inventory value over the specified period and time grouping.
//
// Parameters:
//   - productID: The product_id to analyze (validated for existence)
//   - start: Start date for analysis period
//   - end: End date for analysis period
//   - groupBy: Time grouping - "week", "month", "quarter", "year", or "" for daily
//
// Returns:
//   - []dtos.InventoryTurnoverItem: Array of turnover metrics for the product with:
//   - ProductID: The specified product identifier
//   - AvgInventory: Average inventory value
//   - COGS: Cost of goods sold
//   - TurnoverRatio: COGS / AvgInventory
//   - error: "product not found" if product doesn't exist or database error
func GetInventoryTurnoverByProduct(db DBExecutor, productID string, start, end time.Time, groupBy string) ([]dtos.InventoryTurnoverItem, error) {
	// Validate product exists
	err := IsProductThere(db, productID)
	if err != nil {
		return nil, err
	}

	// Get period grouping SQL expression
	periodExpr := getPeriodExpr(groupBy)

	// Build query with product filter
	query := fmt.Sprintf(`
        SELECT 
            oi.product_id,
            AVG(poi.unit_cost * poi.quantity) AS avg_inventory,
            SUM(oi.unit_price * oi.quantity) AS cogs
        FROM orders o
        JOIN order_items oi ON o.order_id = oi.order_id
        LEFT JOIN purchase_order_items poi ON poi.product_id = oi.product_id
        LEFT JOIN purchase_orders po ON poi.po_id = po.po_id
        WHERE o.created_at BETWEEN ? AND ? AND oi.product_id = ?
        GROUP BY %s, oi.product_id
    `, periodExpr)

	// Execute query with date range and product filter
	rows, err := db.Query(query, start, end, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []dtos.InventoryTurnoverItem{}

	// Iterate through results
	for rows.Next() {
		var item dtos.InventoryTurnoverItem
		var avgInv, cogs sql.NullFloat64
		var productID sql.NullString

		// Scan row with nullable fields
		if err := rows.Scan(&productID, &avgInv, &cogs); err != nil {
			return nil, err
		}

		// Convert nullable product ID
		if productID.Valid {
			item.ProductID = &productID.String
		}

		// Set inventory and COGS values
		item.AvgInventory = avgInv.Float64
		item.COGS = cogs.Float64

		// Calculate turnover ratio (avoid division by zero)
		if item.AvgInventory > 0 {
			item.TurnoverRatio = item.COGS / item.AvgInventory
		}

		results = append(results, item)
	}
	return results, nil
}

// StoreBatchDetails creates a new inventory batch with tracking information.
//
// This function stores batch-level details for inventory tracking including batch
// number, images, expiry date, and manufacturing date. Supports quality control
// and traceability requirements.
//
// Parameters:
//   - req: dtos.Batch containing:
//   - InventoryID: The inventory_id (validated for existence)
//   - BatchNumber: Unique batch identifier
//   - Images: Optional array of image URLs (marshaled to JSON)
//   - ExpiryDate: Product expiry date
//   - ManufacturingDate: Product manufacturing date
//
// Returns:
//   - string: Generated batch_id
//   - error: "inventory not found" if inventory doesn't exist,
//     JSON marshaling error,
//     or database error
func StoreBatchDetails(db DBExecutor, req dtos.Batch) (string, error) {
	// Validate inventory exists
	err := isInventoryThere(db, req.InventoryID)
	if err != nil {
		return "", err
	}

	// Generate unique batch ID
	batchID, _ := shortid.Generate()
	var imagesData []byte

	// Marshal images array to JSON if provided
	if req.Images != nil {
		imagesData, err = json.Marshal(req.Images)
		if err != nil {
			fmt.Println("Error:", err)
			return "", err
		}
	}

	// Insert batch record with JSON images
	_, err = db.Exec(`
		INSERT INTO inventory_batches (batch_id, inventory_id, batch_number, images, expiry_date, manufacturing_date)
		VALUES (?, ?, ?, ?, ?, ?)`,
		batchID, req.InventoryID, req.BatchNumber, imagesData, req.ExpiryDate, req.ManufacturingDate)
	if err != nil {
		log.Printf("Error inserting batch details: %v", err)
	}
	return batchID, err
}

// isInventoryThere validates that an inventory record exists.
//
// This is an internal helper function for inventory existence checks.
//
// Parameters:
//   - inventoryID: The inventory_id to validate
//
// Returns:
//   - error: "inventory not found" if inventory doesn't exist or database error
func isInventoryThere(db DBExecutor, inventoryID string) error {
	// Check inventory existence
	exists, err := RecordExists(db, "inventory", "inventory_id = ?", inventoryID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("inventory not found")
	}
	return nil
}

// isBatchThere validates that a batch record exists.
//
// This is an internal helper function for batch existence checks.
//
// Parameters:
//   - batchID: The batch_id to validate
//
// Returns:
//   - error: "batch not found" if batch doesn't exist or database error
func isBatchThere(db DBExecutor, batchID string) error {
	// Check batch existence
	exists, err := RecordExists(db, "inventory_batches", "batch_id = ?", batchID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("batch not found")
	}
	return nil
}

// StoreInspectionDetails records a quality inspection for an inventory batch.
//
// This function creates an inspection record with inspector details, date,
// notes, and supporting images for quality control tracking.
//
// Parameters:
//   - req: dtos.Inspection containing:
//   - BatchID: The batch_id being inspected (validated for existence)
//   - InspectorID: The user_id of inspector (validated for existence)
//   - InspectionDate: Date of inspection
//   - InspectionNotes: Inspection findings and observations
//   - Images: Optional array of inspection image URLs (marshaled to JSON)
//
// Returns:
//   - error: "batch not found" if batch doesn't exist,
//     "inspector not found" if inspector user doesn't exist,
//     JSON marshaling error,
//     or database error
func StoreInspectionDetails(db DBExecutor, req dtos.Inspection) error {
	// Validate batch exists
	err := isBatchThere(db, req.BatchID)
	if err != nil {
		return err
	}

	// Validate inspector (user) exists
	err = isUserThere(db, req.InspectorID)
	if err != nil {
		// Provide specific error message for inspector
		if err.Error() == "user not found" {
			return fmt.Errorf("inspector not found")
		}
		return err
	}

	var imagesData []byte

	// Marshal images array to JSON if provided
	if req.Images != nil {
		imagesData, err = json.Marshal(req.Images)
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}
	}

	// Generate unique inspection ID
	inspectionID, _ := shortid.Generate()

	// Insert inspection record with JSON images
	_, err = db.Exec(`
		INSERT INTO batch_inspections (inspection_id, batch_id, inspection_date, inspector_id, inspection_notes, images)
		VALUES (?, ?, ?, ?, ?, ?)`,
		inspectionID, req.BatchID, req.InspectionDate, req.InspectorID, req.InspectionNotes, imagesData)
	return err
}

// isConditionThere validates that a batch condition record exists.
//
// This is an internal helper function for condition existence checks.
//
// Parameters:
//   - conditionID: The condition_id to validate
//
// Returns:
//   - error: "condition not found" if condition doesn't exist or database error
func isConditionThere(db DBExecutor, conditionID string) error {
	// Check condition existence
	exists, err := RecordExists(db, "batch_conditions", "condition_id = ?", conditionID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("condition not found")
	}
	return nil
}

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
