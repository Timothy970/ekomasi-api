// Package models provides data access functions for the Ekomasi e-commerce backend.
//
// This file contains warehouse management operations including:
//   - Warehouse CRUD (create, read, update, delete)
//   - Warehouse listing with pagination
//   - Warehouse details management
//
// Warehouses represent physical storage locations for inventory management.
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"math"

	"github.com/teris-io/shortid"
)

// CreateWarehouse creates a new warehouse in the system.
//
// This function generates a unique warehouse ID and inserts the warehouse
// record with name, location, and additional details.
//
// Parameters:
//   - req: dtos.CreateWarehouseRequest containing:
//   - Name: Warehouse name
//   - Location: Warehouse physical location/address
//   - WarehouseDetails: Additional warehouse information
//
// Returns:
//   - string: The generated warehouse ID
//   - error: Database error or nil on success
func CreateWarehouse(req dtos.CreateWarehouseRequest) (string, error) {
	// Generate unique warehouse ID
	id, _ := shortid.Generate()

	// Insert warehouse record
	query := `INSERT INTO warehouses (warehouse_id, name, location, warehouse_details) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, id, req.Name, req.Location, req.WarehouseDetails)
	if err != nil {
		return "", err
	}
	return id, nil
}

// ListWarehouses retrieves all warehouses with pagination.
//
// This function fetches paginated warehouse records with complete details
// including name, location, and warehouse-specific information.
//
// Parameters:
//   - page: int - Page number (1-based)
//   - size: int - Number of items per page
//
// Returns:
//   - []dtos.Warehouse: Array of warehouse records
//   - dtos.PaginationMeta: Pagination metadata (page, size, total, has prev/next)
//   - error: Database error or nil on success
func ListWarehouses(page, size int) ([]dtos.Warehouse, dtos.PaginationMeta, error) {
	// Calculate offset for pagination
	offset := (page - 1) * size

	// Count total warehouses
	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM warehouses`).Scan(&total)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Fetch paginated warehouse records (COALESCE handles NULL warehouse_details)
	query := `SELECT warehouse_id, name, location, COALESCE(warehouse_details, '') as warehouse_details  FROM warehouses LIMIT ? OFFSET ?`
	rows, err := DB.Query(query, size, offset)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	// Process each warehouse record
	var warehouses []dtos.Warehouse
	for rows.Next() {
		var w dtos.Warehouse
		if err := rows.Scan(&w.WarehouseID, &w.Name, &w.Location, &w.WarehouseDetails); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		warehouses = append(warehouses, w)
	}

	// Build pagination metadata
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))), // Ceiling division
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return warehouses, meta, nil
}

// GetWarehouseByID retrieves a specific warehouse by its ID.
//
// This function fetches complete warehouse information including name,
// location, and additional details.
//
// Parameters:
//   - id: string - The unique warehouse ID
//
// Returns:
//   - *dtos.Warehouse: Warehouse data, or nil if not found
//   - error: Database error or nil on success/not found
func GetWarehouseByID(id string) (*dtos.Warehouse, error) {
	var w dtos.Warehouse

	// Retrieve warehouse by ID
	query := `SELECT warehouse_id, name, location, warehouse_details FROM warehouses WHERE warehouse_id = ?`
	err := DB.QueryRow(query, id).Scan(&w.WarehouseID, &w.Name, &w.Location, &w.WarehouseDetails)

	// Return nil if warehouse not found (not an error)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// UpdateWarehouse updates an existing warehouse's information.
//
// This function validates warehouse existence and updates name, location,
// and additional details.
//
// Parameters:
//   - id: string - The warehouse ID to update
//   - req: dtos.UpdateWarehouseRequest containing:
//   - Name: Updated warehouse name
//   - Location: Updated warehouse location/address
//   - WarehouseDetails: Updated additional information
//
// Returns:
//   - error: "warehouse not found", database error, or nil on success
func UpdateWarehouse(id string, req dtos.UpdateWarehouseRequest) error {
	// Validate warehouse exists
	exists, err := RecordExists(DB, "warehouses", "warehouse_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("warehouse not found")
	}

	// Update warehouse information
	query := `UPDATE warehouses SET name = ?, location = ?, warehouse_details = ? WHERE warehouse_id = ?`
	_, err = DB.Exec(query, req.Name, req.Location, req.WarehouseDetails, id)
	return err
}

// DeleteWarehouse permanently removes a warehouse from the system.
//
// This function validates warehouse existence before deletion. Note that
// deleting a warehouse with associated inventory may cause referential
// integrity issues depending on database constraints.
//
// Parameters:
//   - id: string - The warehouse ID to delete
//
// Returns:
//   - error: "warehouse not found", database error, or nil on success
func DeleteWarehouse(id string) error {
	// Validate warehouse exists
	exists, err := RecordExists(DB, "warehouses", "warehouse_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("warehouse not found")
	}

	// Delete warehouse record
	query := `DELETE FROM warehouses WHERE warehouse_id = ?`
	_, err = DB.Exec(query, id)
	return err
}
