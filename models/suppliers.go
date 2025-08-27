package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"math"

	"github.com/teris-io/shortid"
)

// Create Supplier
func CreateSupplier(s dtos.Supplier) error {
	supplierID, _ := shortid.Generate()

	_, err := DB.Exec(`
		INSERT INTO suppliers (supplier_id, name, contact_email, contact_phone, extra_details)
		VALUES (?, ?, ?, ?, ?)`,
		supplierID, s.Name, s.ContactEmail, s.ContactPhone, s.ExtraDetails,
	)
	return err
}

// List Suppliers with Pagination
func ListSuppliers(page, size int) ([]dtos.Supplier, dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	// Count total
	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM suppliers`).Scan(&total)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	rows, err := DB.Query(`
		SELECT supplier_id, name, contact_email, contact_phone, extra_details
		FROM suppliers
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	var suppliers []dtos.Supplier
	for rows.Next() {
		var s dtos.Supplier
		if err := rows.Scan(&s.SupplierID, &s.Name, &s.ContactEmail, &s.ContactPhone, &s.ExtraDetails); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		suppliers = append(suppliers, s)
	}

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

// Get Supplier Details
func GetSupplierByID(id string) (dtos.Supplier, error) {
	var s dtos.Supplier
	err := DB.QueryRow(`
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
func isSupplierThere(id string) error {
	exists, err := RecordExists("suppliers", "supplier_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("supplier not found")
	}
	return nil
}

// Update Supplier
func UpdateSupplier(s dtos.Supplier, id string) error {
	err := isSupplierThere(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
		UPDATE suppliers
		SET name = ?, contact_email = ?, contact_phone = ?, extra_details = ?
		WHERE supplier_id = ?`,
		s.Name, s.ContactEmail, s.ContactPhone, s.ExtraDetails, id,
	)
	return err
}

// Delete Supplier
func DeleteSupplier(id string) error {
	err := isSupplierThere(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM suppliers WHERE supplier_id = ?`, id)
	return err
}
