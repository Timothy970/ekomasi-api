// Package models provides data access functions for the Adenzo e-commerce backend.
//
// This file contains warranty management operations including:
//   - Warranty type CRUD (create, read, update, delete)
//   - Product warranty associations
//   - Warranty period and expiry date tracking
//   - Manufacturing date management
//
// Warranty types define reusable warranty categories (e.g., "1 Year Manufacturer",
// "Extended Warranty") that can be assigned to products with specific dates.
package models

import (
	"adenzo_backend/dtos"
	"fmt"
	"log"
	"strings"

	"github.com/teris-io/shortid"
)

// CreateWarrantType creates a new warranty type in the system.
//
// Warranty types are reusable templates that define warranty categories
// (e.g., "1 Year Limited", "3 Year Extended") with descriptions.
//
// Parameters:
//   - wt: dtos.CreateWarrantyTypeRequest containing:
//   - Name: Warranty type name (e.g., "2 Year Manufacturer Warranty")
//   - Description: Detailed warranty terms and coverage
//
// Returns:
//   - error: Database error or nil on success
func CreateWarrantType(db DBExecutor, wt dtos.CreateWarrantyTypeRequest) error {
	//validate warranty type with the same name is not already there
	err := isWarrantyTypeNameUnique(db, wt.Name, "")
	if err != nil {
		return err
	}
	// Generate unique warranty type ID
	warrantyID, _ := shortid.Generate()

	// Insert warranty type record
	query := `INSERT INTO warranty_types (warranty_type_id, name, description) VALUES (?, ?, ?)`
	_, err = db.Exec(query, warrantyID, wt.Name, wt.Description)
	return err
}

// helper function to validate that a warranty type with the same name does not already exist
// parameters - name: the warranty type name to validate
// returns - error if a warranty type with the same name already exists, nil otherwise
func isWarrantyTypeNameUnique(db DBExecutor, name string, warrantyID string) error {
	// Build condition to check warranty type name uniqueness
	condition := "LOWER(name) = LOWER(?)"
	args := []interface{}{strings.ToLower(name)}

	// Exclude current record if warrantyID is provided (for updates)
	if warrantyID != "" {
		condition += " AND warranty_type_id != ?"
		args = append(args, warrantyID)
	}

	// Check if warranty type with the same name already exists
	exists, err := RecordExists(db, "warranty_types", condition, args...)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("warranty type name already exists")
	}
	return nil
}

// GetAllWarrantTypes retrieves all warranty types in the system.
//
// This function fetches all available warranty type templates with their
// names and descriptions.
//
// Returns:
//   - []dtos.WarrantyType: Array of all warranty types
//   - error: Database error or nil on success
func GetAllWarrantTypes(db DBExecutor) ([]dtos.WarrantyType, error) {
	// Retrieve all warranty types
	query := `SELECT warranty_type_id, name, description FROM warranty_types`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process each warranty type
	var warranties []dtos.WarrantyType
	for rows.Next() {
		var wt dtos.WarrantyType

		err := rows.Scan(&wt.WarrantyTypeID, &wt.Name, &wt.Description)
		if err != nil {
			return nil, err
		}
		warranties = append(warranties, wt)
	}
	return warranties, nil
}

// isWarrantyTypeThere validates if a warranty type exists.
//
// This is a helper function to ensure warranty type validity before
// performing operations like update, delete, or product association.
//
// Parameters:
//   - warrantyID: string - The warranty type ID to validate
//
// Returns:
//   - error: "warranty type does not exist" if not found, database error, or nil if exists
func isWarrantyTypeThere(db DBExecutor, warrantyID string) error {
	// Check warranty type existence
	exists, err := RecordExists(db, "warranty_types", "warranty_type_id = ?", warrantyID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("warranty type with ID %s does not exist", warrantyID)
	}
	return nil
}

// UpdateWarrantType updates an existing warranty type's information.
//
// This function validates warranty type existence and updates the name
// and description.
//
// Parameters:
//   - warrantyID: string - The warranty type ID to update
//   - wt: dtos.WarrantyType containing:
//   - Name: Updated warranty type name
//   - Description: Updated warranty terms and coverage
//
// Returns:
//   - error: "warranty type does not exist", database error, or nil on success
func UpdateWarrantType(db DBExecutor, warrantyID string, wt dtos.WarrantyType) error {
	// Validate warranty type exists
	if err := isWarrantyTypeThere(db, warrantyID); err != nil {
		return err
	}
	// Validate warranty type name uniqueness (if name is being updated)
	err := isWarrantyTypeNameUnique(db, wt.Name, warrantyID)
	if err != nil {
		return err
	}
	// Update warranty type information
	query := `UPDATE warranty_types SET name = ?, description = ? WHERE warranty_type_id = ?`
	_, err = db.Exec(query, wt.Name, wt.Description, warrantyID)
	return err
}

// DeleteWarrantType permanently removes a warranty type from the system.
//
// Parameters:
//   - warrantyID: string - The warranty type ID to delete
//
// Returns:
//   - error: "warranty type does not exist", database error, or nil on success
func DeleteWarrantType(db DBExecutor, warrantyID string) error {
	// Validate warranty type exists
	if err := isWarrantyTypeThere(db, warrantyID); err != nil {
		return err
	}

	// Delete warranty type record
	query := `DELETE FROM warranty_types WHERE warranty_type_id = ?`
	_, err := db.Exec(query, warrantyID)
	return err
}

// AddProductWarranties assigns a warranty to a product.
//
// This function implements a replace-on-add strategy: if the product already
// has warranties, the new warranty is added first, then old warranties are
// removed. This ensures the product always has warranty coverage during the
// transition.
//
// Workflow:
// 1. Validate product exists
// 2. Validate warranty type exists
// 3. Retrieve existing warranty IDs for the product
// 4. Insert new warranty record
// 5. Remove old warranties (if any)
//
// Parameters:
//   - pw: dtos.AddProductWarrantiesRequest containing:
//   - ProductID: The product to assign warranty to
//   - WarrantyTypeID: The warranty type template to use
//   - WarrantyPeriod: Warranty duration (e.g., "1 year", "24 months")
//   - ManufacturingDate: Product manufacturing date
//   - ExpiryDate: Warranty expiration date
//
// Returns:
//   - error: "product not found", "warranty type does not exist", database error, or nil on success
func AddProductWarranties(db DBExecutor, pw dtos.AddProductWarrantiesRequest) error {
	// Validate product exists
	err := IsProductThere(db, pw.ProductID)
	if err != nil {
		return err
	}

	// Validate warranty type exists
	err = isWarrantyTypeThere(db, pw.WarrantyTypeID)
	if err != nil {
		log.Printf("warranty type check error: %s with id %s", err, pw.WarrantyTypeID)
		return err
	}

	// Generate unique warranty ID
	warrantyID, _ := shortid.Generate()

	// Get existing warranty IDs for this product
	var warrantyIDS []string
	warrantyIDS, err = GetProductWarrantyIDs(db, pw.ProductID)
	if err != nil {
		return err
	}

	// Insert new warranty record
	query := `INSERT INTO product_warranties (warranty_id, product_id, warranty_type_id, warranty_period, manufacturing_date, expiry_date) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, warrantyID, pw.ProductID, pw.WarrantyTypeID, pw.WarrantyPeriod, pw.ManufacturingDate, pw.ExpiryDate)
	if err != nil {
		return err
	}

	// Remove old warranties (ensures product has only one active warranty)
	for _, id := range warrantyIDS {
		err = RemoveProductWarranty(db, id)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetProductWarrantyIDs retrieves all warranty IDs associated with a product.
//
// This helper function is used to identify existing warranties before
// adding new warranties or during warranty management operations.
//
// Parameters:
//   - productID: string - The product ID to get warranties for
//
// Returns:
//   - []string: Array of warranty IDs (empty if product has no warranties)
//   - error: Database error or nil on success
func GetProductWarrantyIDs(db DBExecutor, productID string) ([]string, error) {
	// Retrieve all warranty IDs for the product
	query := `SELECT warranty_id FROM product_warranties WHERE product_id = ?`
	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect warranty IDs
	var warrantyIDs []string
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		warrantyIDs = append(warrantyIDs, id)
	}
	return warrantyIDs, nil
}

// RemoveProductWarranty removes a warranty assignment from a product.
//
// This function deletes the warranty record but does not delete the
// warranty type template, which remains available for other products.
//
// Parameters:
//   - warrantyID: string - The warranty assignment ID to remove
//
// Returns:
//   - error: Database error or nil on success
func RemoveProductWarranty(db DBExecutor, warrantyID string) error {
	// Delete warranty assignment record
	query := `DELETE FROM product_warranties WHERE warranty_id = ?`
	_, err := db.Exec(query, warrantyID)
	return err
}
