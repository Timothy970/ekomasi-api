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
	"fmt"
	"log"
	"time"

	"github.com/teris-io/shortid"
)

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
