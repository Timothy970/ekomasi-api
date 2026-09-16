// Package models provides data access functions for the Ekomasi e-commerce platform.
//
// This file contains functions for managing suppliers:
//   - Create, retrieve, update, delete supplier records
//   - List suppliers with pagination
//   - Supplier existence validation
//   - Contact information management (email, phone)
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"math"

	"github.com/teris-io/shortid"
)

// CreateSupplier creates a new supplier in the system.
//
// This function generates a unique supplier ID and inserts the supplier
// with their contact information and additional details.
//
// Parameters:
//   - s: dtos.Supplier containing:
//   - Name: Supplier company name
//   - ContactEmail: Email address for supplier communication
//   - ContactPhone: Phone number for supplier contact
//   - ExtraDetails: Additional supplier information/notes
//
// Returns:
//   - error: Database error or nil on success
func CreateSupplier(db DBExecutor, s dtos.Supplier) error {
	// Generate unique supplier ID
	supplierID, _ := shortid.Generate()

	// Insert new supplier record
	_, err := db.Exec(`
		INSERT INTO suppliers (supplier_id, name, contact_email, contact_phone, extra_details)
		VALUES (?, ?, ?, ?, ?)`,
		supplierID, s.Name, s.ContactEmail, s.ContactPhone, s.ExtraDetails,
	)
	return err
}

// ListSuppliers retrieves all suppliers with pagination.
//
// This function returns a paginated list of suppliers with metadata
// including total count and navigation information.
//
// Parameters:
//   - page: int - Page number (1-based)
//   - size: int - Number of items per page
//
// Returns:
//   - []dtos.Supplier: Array of suppliers with:
//   - SupplierID: Unique identifier
//   - Name: Supplier company name
//   - ContactEmail: Email address
//   - ContactPhone: Phone number
//   - ExtraDetails: Additional notes
//   - dtos.PaginationMeta: Pagination info (page, size, totals, navigation flags)
//   - error: Database error or nil on success
func ListSuppliers(db DBExecutor, page, size int) ([]dtos.Supplier, dtos.PaginationMeta, error) {
	// Calculate offset for pagination
	offset := (page - 1) * size

	// Get total count of suppliers
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM suppliers`).Scan(&total)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Query paginated suppliers
	rows, err := db.Query(`
		SELECT supplier_id, name, contact_email, contact_phone, extra_details
		FROM suppliers
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	// Process results
	var suppliers []dtos.Supplier
	for rows.Next() {
		var s dtos.Supplier
		if err := rows.Scan(&s.SupplierID, &s.Name, &s.ContactEmail, &s.ContactPhone, &s.ExtraDetails); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		suppliers = append(suppliers, s)
	}

	// Build pagination metadata
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return suppliers, meta, nil
}

// GetSupplierByID retrieves a single supplier by their ID.
//
// Parameters:
//   - id: string - The unique supplier ID to retrieve
//
// Returns:
//   - dtos.Supplier: Supplier details including:
//   - SupplierID: Unique identifier
//   - Name: Supplier company name
//   - ContactEmail: Email address
//   - ContactPhone: Phone number
//   - ExtraDetails: Additional notes
//   - error: "supplier doesn't exist", database error, or nil on success
func GetSupplierByID(db DBExecutor, id string) (dtos.Supplier, error) {
	var s dtos.Supplier

	// Query supplier by ID
	err := db.QueryRow(`
		SELECT supplier_id, name, contact_email, contact_phone, extra_details
		FROM suppliers
		WHERE supplier_id = ?`, id).
		Scan(&s.SupplierID, &s.Name, &s.ContactEmail, &s.ContactPhone, &s.ExtraDetails)

	if err != nil {
		if err == sql.ErrNoRows {
			return dtos.Supplier{}, errors.New("supplier doesn't exist")
		} else {
			return dtos.Supplier{}, err
		}
	}

	return s, err
}

// isSupplierThere validates that a supplier exists in the database.
//
// This helper function is used by update and delete operations
// to ensure the supplier exists before performing operations.
//
// Parameters:
//   - id: string - The unique supplier ID to check
//
// Returns:
//   - error: "supplier not found", database error, or nil if supplier exists
func isSupplierThere(db DBExecutor, id string) error {
	// Check supplier existence
	exists, err := RecordExists(db, "suppliers", "supplier_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("supplier not found")
	}
	return nil
}

// UpdateSupplier updates an existing supplier's information.
//
// This function validates supplier existence before updating
// all contact information and details.
//
// Parameters:
//   - s: dtos.Supplier containing updated:
//   - Name: New supplier company name
//   - ContactEmail: New email address
//   - ContactPhone: New phone number
//   - ExtraDetails: Updated additional notes
//   - id: string - The supplier ID to update
//
// Returns:
//   - error: "supplier not found", database error, or nil on success
func UpdateSupplier(db DBExecutor, s dtos.Supplier, id string) error {
	// Validate supplier exists
	err := isSupplierThere(db, id)
	if err != nil {
		return err
	}

	// Update supplier information
	_, err = db.Exec(`
		UPDATE suppliers
		SET name = ?, contact_email = ?, contact_phone = ?, extra_details = ?
		WHERE supplier_id = ?`,
		s.Name, s.ContactEmail, s.ContactPhone, s.ExtraDetails, id,
	)
	return err
}

// DeleteSupplier permanently removes a supplier from the system.
//
// This function validates supplier existence before deletion.
//
// Parameters:
//   - id: string - The supplier ID to delete
//
// Returns:
//   - error: "supplier not found", database error, or nil on success
func DeleteSupplier(db DBExecutor, id string) error {
	// Validate supplier exists
	err := isSupplierThere(db, id)
	if err != nil {
		return err
	}

	// Delete supplier record
	_, err = db.Exec(`DELETE FROM suppliers WHERE supplier_id = ?`, id)
	return err
}
