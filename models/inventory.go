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
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"math"
	"strings"

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
	var args []any

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
