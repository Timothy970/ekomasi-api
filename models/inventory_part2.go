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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
