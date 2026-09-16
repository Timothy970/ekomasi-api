// Package models provides data access functions for the Ekomasi e-commerce platform.
//
// This file contains functions for managing stock transfers between warehouses:
//   - Create stock transfers with inventory updates
//   - Validate product availability and warehouse existence
//   - List transfers with pagination and search
//   - Retrieve, update stock transfer records
//   - Automatic inventory adjustments (deduct from source, add to destination)
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"math"

	"github.com/teris-io/shortid"
)

// CreateStockTransfer creates a new stock transfer between warehouses.
//
// This function performs several operations:
//  1. Validates product and warehouse existence
//  2. Checks product availability in source warehouse
//  3. Creates transfer record
//  4. Deducts quantity from source warehouse inventory
//  5. Adds quantity to destination warehouse inventory (creates record if needed)
//
// Parameters:
//   - st: dtos.StockTransferDTO containing:
//   - ProductID: Product being transferred
//   - VariantID: Optional variant ID
//   - FromWarehouseID: Source warehouse
//   - ToWarehouseID: Destination warehouse
//   - Quantity: Amount to transfer (must be available)
//   - TransferDetails: Optional notes/reason for transfer
//
// Returns:
//   - error: "product not found", "from/to warehouse not found",
//     "product does not exist in from warehouse",
//     "insufficient product quantity in from warehouse",
//     database error, or nil on success
func CreateStockTransfer(db DBExecutor, st dtos.StockTransferDTO) error {
	// Validate product exists
	err := IsProductThere(db, st.ProductID)
	if err != nil {
		return err
	}

	// Validate source warehouse exists
	err = isWarehouseThere(db, st.FromWarehouseID, "from")
	if err != nil {
		return err
	}

	// Validate destination warehouse exists
	err = isWarehouseThere(db, st.ToWarehouseID, "to")
	if err != nil {
		return err
	}

	// Validate product availability in source warehouse
	err = validateProductAndWarehouse(db, st)
	if err != nil {
		return err
	}

	// Generate unique transfer ID
	transferID, _ := shortid.Generate()

	// Insert transfer record
	query := `
		INSERT INTO stock_transfers 
		(transfer_id, product_id, variant_id, from_warehouse_id, to_warehouse_id, quantity, transfer_details, status) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, transferID, st.ProductID, st.VariantID, st.FromWarehouseID, st.ToWarehouseID, st.Quantity, st.TransferDetails, "Pending")

	return err
}

// validateProductAndWarehouse validates product availability in source warehouse.
//
// This helper function performs two critical checks:
//  1. Product exists in the source warehouse inventory
//  2. Sufficient quantity is available for transfer
//
// Parameters:
//   - st: dtos.StockTransferDTO containing ProductID, FromWarehouseID, and Quantity
//
// Returns:
//   - error: "product does not exist in from warehouse",
//     "insufficient product quantity in from warehouse",
//     database error, or nil if validation passes
func validateProductAndWarehouse(db DBExecutor, st dtos.StockTransferDTO) error {
	// Check if product exists in source warehouse
	query := `SELECT COUNT(*) FROM inventory WHERE product_id = ? AND warehouse_id = ?`
	var count int
	err := db.QueryRow(query, st.ProductID, st.FromWarehouseID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("product does not exist in from warehouse")
	}

	// Check if sufficient quantity is available in source warehouse
	var availableQty int
	err = db.QueryRow("SELECT quantity FROM inventory WHERE product_id = ? AND warehouse_id = ?", st.ProductID, st.FromWarehouseID).Scan(&availableQty)
	if err != nil {
		return err
	}
	if availableQty < st.Quantity {
		return errors.New("insufficient product quantity in from warehouse")
	}

	return nil
}

// isWarehouseThere validates warehouse existence with context-aware error messages.
//
// This helper function checks if a warehouse exists and provides specific
// error messages based on whether it's the source or destination warehouse.
//
// Parameters:
//   - id: string - The warehouse ID to validate
//   - from: string - Context indicator ("from" for source, "to" for destination, other for generic)
//
// Returns:
//   - error: "from warehouse not found", "to warehouse not found",
//     "warehouse not found", database error, or nil if warehouse exists
func isWarehouseThere(db DBExecutor, id, from string) error {
	// Check warehouse existence
	exists, err := RecordExists(db, "warehouses", "warehouse_id = ?", id)
	if err != nil {
		return err
	}

	if !exists {
		// Return context-specific error message
		switch from {
		case "from":
			return errors.New("from warehouse not found")
		case "to":
			return errors.New("to warehouse not found")
		default:
			return errors.New("warehouse not found")
		}
	}

	return nil
}

// ListStockTransfers retrieves stock transfers with pagination and search.
//
// This function joins across stock_transfers, products, and warehouses tables
// to provide enriched transfer data. Supports searching by product name,
// source warehouse name, or destination warehouse name.
//
// Parameters:
//   - page: int - Page number (1-based)
//   - size: int - Items per page
//   - searchParam: string - Optional search term to filter by product or warehouse names
//     (case-insensitive partial match)
//
// Returns:
//   - []dtos.StockTransferResponseDTO: Array of transfers with:
//   - TransferID, ProductID, VariantID
//   - FromWarehouseID, ToWarehouseID, Quantity
//   - TransferDate, TransferDetails
//   - ProductName, FromWarehouseName, ToWarehouseName (enriched data)
//   - *dtos.PaginationMeta: Pagination info (page, size, totals, navigation flags)
//   - error: Database error or nil on success
func ListStockTransfers(db DBExecutor, page, size int, searchParam string) ([]dtos.StockTransferResponseDTO, *dtos.PaginationMeta, error) {
	// Calculate offset for pagination
	offset := (page - 1) * size

	// Build count query with joins
	var countArgs []any
	countQuery := `
		SELECT COUNT(*)
		FROM stock_transfers st
		JOIN products p ON st.product_id = p.product_id
		JOIN warehouses fw ON st.from_warehouse_id = fw.warehouse_id
		JOIN warehouses tw ON st.to_warehouse_id = tw.warehouse_id
	`

	// Build select query with enriched data
	var selectArgs []any
	selectQuery := `
		SELECT 
			st.transfer_id,
			st.product_id,
			st.variant_id,
			st.from_warehouse_id,
			st.to_warehouse_id,
			st.quantity,
			st.transfer_date,
			st.transfer_details,
			st.status,
			p.name AS product_name,
			fw.name AS from_warehouse_name,
			tw.name AS to_warehouse_name
		FROM stock_transfers st
		JOIN products p ON st.product_id = p.product_id
		JOIN warehouses fw ON st.from_warehouse_id = fw.warehouse_id
		JOIN warehouses tw ON st.to_warehouse_id = tw.warehouse_id
	`

	// Add search filter if provided
	if searchParam != "" {
		searchLike := "%" + searchParam + "%"
		// Search across product name and warehouse names
		countQuery += `
			WHERE p.name LIKE ? OR fw.name LIKE ? OR tw.name LIKE ?
		`
		selectQuery += `
			WHERE p.name LIKE ? OR fw.name LIKE ? OR tw.name LIKE ?
		`
		// Add search parameters to both query argument slices
		countArgs = append(countArgs, searchLike, searchLike, searchLike)
		selectArgs = append(selectArgs, searchLike, searchLike, searchLike)
	}

	// Add ordering and pagination to select query
	selectQuery += `
		ORDER BY st.transfer_date DESC
		LIMIT ? OFFSET ?
	`
	selectArgs = append(selectArgs, size, offset)

	// Get total count for pagination
	var total int
	if err := db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, nil, err
	}

	// Execute select query
	rows, err := db.Query(selectQuery, selectArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Process results
	var transfers []dtos.StockTransferResponseDTO
	for rows.Next() {
		var st dtos.StockTransferResponseDTO
		if err := rows.Scan(
			&st.TransferID,
			&st.ProductID,
			&st.VariantID,
			&st.FromWarehouseID,
			&st.ToWarehouseID,
			&st.Quantity,
			&st.TransferDate,
			&st.TransferDetails,
			&st.Status,
			&st.ProductName,
			&st.FromWarehouseName,
			&st.ToWarehouseName,
		); err != nil {
			return nil, nil, err
		}
		transfers = append(transfers, st)
	}

	// Build pagination metadata
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page < int(math.Ceil(float64(total)/float64(size))),
	}

	return transfers, meta, nil
}

// GetStockTransferByID retrieves a single stock transfer by its ID.
//
// Parameters:
//   - id: string - The unique transfer ID to retrieve
//
// Returns:
//   - *dtos.StockTransferDTO: Transfer details including:
//   - TransferID, ProductID, VariantID
//   - FromWarehouseID, ToWarehouseID, Quantity
//   - TransferDate, TransferDetails
//   - error: "stock transfer not found", database error, or nil on success
func GetStockTransferByID(db DBExecutor, id string) (*dtos.StockTransferDTO, error) {
	var st dtos.StockTransferDTO

	// Query transfer by ID
	err := db.QueryRow(`
		SELECT transfer_id, product_id, variant_id, from_warehouse_id, to_warehouse_id, quantity, transfer_date, transfer_details
		FROM stock_transfers
		WHERE transfer_id = ?`, id).
		Scan(&st.TransferID, &st.ProductID, &st.VariantID, &st.FromWarehouseID, &st.ToWarehouseID, &st.Quantity, &st.TransferDate, &st.TransferDetails)

	if err == sql.ErrNoRows {
		return nil, errors.New("stock transfer not found")
	}

	return &st, err
}

// UpdateStockTransfer updates the quantity of an existing stock transfer.
//
// Parameters:
//   - quantity: int - New transfer quantity
//   - id: string - The transfer ID to update
//
// Returns:
//   - error: "stock transfer not found", database error, or nil on success
func UpdateStockTransfer(db DBExecutor, req dtos.StockTransferUpdateDTO, id string) error {
	// Validate transfer exists
	exists, err := RecordExists(db, "stock_transfers", "transfer_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("stock transfer not found")
	}
	// update transer staus
	query := `UPDATE stock_transfers SET status = ? WHERE transfer_id = ?`
	_, err = db.Exec(query, req.Status, id)
	if err != nil {
		return err
	}
	//if status is Recieved, then update inventory
	if req.Status == "Received" {
		// Deduct quantity from source warehouse inventory
		_, err = db.Exec(`
		UPDATE inventory SET quantity = quantity - ?
		WHERE product_id = ? AND warehouse_id = ? AND quantity >= ?`,
			req.Quantity, req.ProductID, req.FromWarehouseID, req.Quantity)

		// Get inventory settings from destination warehouse (if exists)
		toInventoryQuery := `SELECT low_stock_threshold, supplier_id FROM inventory WHERE product_id = ? AND warehouse_id = ?`
		var lowStockThreshold sql.NullInt64
		var supplier sql.NullString
		err = db.QueryRow(toInventoryQuery, req.ProductID, req.ToWarehouseID).Scan(&lowStockThreshold, &supplier)

		// Add quantity to destination warehouse (create new inventory record)
		inventoryID, _ := shortid.Generate()
		_, err = db.Exec(`INSERT INTO inventory (inventory_id, product_id, warehouse_id, quantity, low_stock_threshold, supplier_id) VALUES (?, ?, ?, ?, ?, ?)`, inventoryID, req.ProductID, req.ToWarehouseID, req.Quantity, lowStockThreshold, supplier)
		if err != nil {
			return err
		}
	}
	return nil
}
