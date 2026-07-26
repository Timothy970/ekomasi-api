// Package models provides data access layer for the Ekomasi e-commerce platform.
//
// This file handles charges management including:
//   - Charge CRUD operations (shipping, handling, tax, etc.)
//   - Product-charge associations (linking charges to specific products)
//   - Validation helpers for charge existence
//
// Charges are flexible fees that can be applied to products (e.g., handling fees,
// environmental charges, premium shipping fees).
package models

import (
	"ekomasi_backend/dtos"
	"database/sql"
	"errors"
	"strconv"

	"github.com/teris-io/shortid"
)

// isChargeThere validates that a charge exists by charge_id.
//
// This is an internal validation helper used by other charge functions.
// Returns an error (rather than bool) for easier use in validation chains.
//
// Parameters:
//   - id: The charge_id to validate
//
// Returns:
//   - error: nil if charge exists, "tax with ID {id} not found" error if not found,
//     or database error if query fails
func isChargeThere(db DBExecutor, id string) error {
	// Check if charge record exists in database
	exists, err := RecordExists(db, "charges", "charge_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("tax with ID " + id + " not found")
	}
	return nil
}

// AddCharge creates a new charge in the database.
//
// Charges are flexible fees that can be applied to products (e.g., handling fees,
// environmental charges, shipping surcharges, etc.).
//
// Parameters:
//   - input: dtos.Charge containing:
//   - Type: Charge name/type (e.g., "Handling Fee", "Environmental Charge")
//   - Value: Charge amount (numeric value)
//
// Returns:
//   - *dtos.Charge: Pointer to created charge with generated charge_id
//   - error: Database error if insertion fails
func AddCharge(db DBExecutor, input dtos.Charge) (*dtos.Charge, error) {
	//validate charge with that name or value is not already there
	err := validateCharge(db, input.Type, input.Value)
	if err != nil {
		return nil, err
	}
	// Generate unique charge ID using shortid for user-friendly identifiers
	chargeID, _ := shortid.Generate()

	// Insert new charge into database
	_, err = db.Exec(`
		INSERT INTO charges (charge_id, charge_name, charge_value)
		VALUES (?, ?, ?)`,
		chargeID, input.Type, input.Value,
	)
	if err != nil {
		return nil, err
	}

	// Return created charge DTO with generated ID
	return &dtos.Charge{
		ID:    chargeID,
		Type:  input.Type,
		Value: input.Value,
	}, nil
}

// helper function to validate that a charge with the same name or value does not already exist
// parameters - name: the charge name to validate
//
//	value: the charge value to validate
//
// returns - error if a charge with the same name or value already exists, nil otherwise
func validateCharge(db DBExecutor, name string, value float64) error {
	// Check if charge with the same name already exists
	exists, err := RecordExists(db, "charges", "charge_name = ?", name)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("charge with name " + name + " already exists")
	}
	// Check if charge with the same value already exists
	exists, err = RecordExists(db, "charges", "charge_value = ?", value)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("charge with value " + strconv.FormatFloat(value, 'f', -1, 64) + " already exists")
	}
	return nil
}

// UpdateCharge updates an existing charge's name and value.
//
// This function validates charge existence before updating. All fields are updated
// (no partial updates supported - consider adding dynamic field updates if needed).
//
// Parameters:
//   - id: The charge_id to update
//   - input: dtos.Charge containing:
//   - Type: New charge name/type
//   - Value: New charge amount
//
// Returns:
//   - *dtos.Charge: Pointer to updated charge with provided values
//   - error: "tax with ID {id} not found" if charge doesn't exist, or database error
func UpdateCharge(db DBExecutor, id string, input dtos.Charge) (*dtos.Charge, error) {
	// Validate charge exists before updating
	err := isChargeThere(db, id)
	if err != nil {
		return nil, err
	}
	//validate charge with that name or value is not already there
	err = validateCharge(db, input.Type, input.Value)
	if err != nil {
		return nil, err
	}
	// Update charge name and value
	_, err = db.Exec(`
		UPDATE charges
		SET charge_name = ?, charge_value = ?
		WHERE charge_id = ?`,
		input.Type, input.Value, id,
	)
	if err != nil {
		return nil, err
	}

	// Return updated charge DTO
	return &dtos.Charge{
		ID:    id,
		Type:  input.Type,
		Value: input.Value,
	}, nil
}

// GetChargeByID retrieves a single charge by its charge_id.
//
// This function validates charge existence before querying.
//
// Parameters:
//   - id: The charge_id to retrieve
//
// Returns:
//   - *dtos.Charge: Pointer to charge object with ID, Type (name), and Value (amount)
//   - error: "tax with ID {id} not found" if charge doesn't exist, or database error
func GetChargeByID(db DBExecutor, id string) (*dtos.Charge, error) {
	// Validate charge exists
	err := isChargeThere(db, id)
	if err != nil {
		return nil, err
	}

	// Query charge details
	row := db.QueryRow(`SELECT charge_id, charge_name, charge_value FROM charges WHERE charge_id = ?`, id)

	var c dtos.Charge
	// Scan charge data into DTO
	if err := row.Scan(&c.ID, &c.Type, &c.Value); err != nil {
		if err == sql.ErrNoRows {
			// Charge not found (unlikely after existence check)
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// GetAllCharges retrieves all charges from the database.
//
// This function fetches all available charges without pagination.
// Consider adding pagination if charge count becomes large.
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.Charge: Array of all charges with ID, Type (name), and Value (amount)
//   - error: Database error if query fails
func GetAllCharges(db DBExecutor) ([]dtos.Charge, error) {
	// Query all charges from database
	rows, err := db.Query(`SELECT charge_id, charge_name, charge_value FROM charges`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var charges []dtos.Charge
	// Iterate through result set and build charge array
	for rows.Next() {
		var c dtos.Charge
		if err := rows.Scan(&c.ID, &c.Type, &c.Value); err != nil {
			return nil, err
		}
		charges = append(charges, c)
	}
	return charges, nil
}

// DeleteCharge removes a charge from the database.
//
// This function validates charge existence before deletion.
//
// Parameters:
//   - id: The charge_id to delete
//
// Returns:
//   - error: "tax with ID {id} not found" if charge doesn't exist, or database error
//
// Important:
//   - No check for associated product_charges - deletion may fail if foreign key constraints exist
//   - Consider checking product_charges associations before deletion to prevent constraint violations
//   - May want to implement soft delete or cascade rules depending on business requirements
func DeleteCharge(db DBExecutor, id string) error {
	// Validate charge exists
	err := isChargeThere(db, id)
	if err != nil {
		return err
	}

	// Delete charge from database
	_, err = db.Exec(`DELETE FROM charges WHERE charge_id = ?`, id)
	return err
}

// AddChargeToProduct associates a charge with a product.
//
// This function creates a relationship between a product and a charge (e.g., linking
// a "Handling Fee" charge to specific products that require special handling).
//
// Parameters:
//   - input: dtos.AddChargeToProductRequest containing:
//   - ProductID: The product to associate the charge with
//   - ChargeID: The charge to apply to the product
//
// Returns:
//   - error: Validation error if product or charge doesn't exist, or database error.
//     Returns nil (no error) if association already exists (idempotent operation)
//
// Behavior:
//   - Validates both product and charge exist before creating association
//   - If association already exists, returns nil without error (idempotent)
//   - Generates unique product_charge_id for the association record
func AddChargeToProduct(db DBExecutor, input dtos.AddChargeToProductRequest) error {
	// Step 1: Validate product exists
	err := IsProductThere(db, input.ProductID)
	if err != nil {
		return err
	}

	// Generate unique ID for product-charge association
	productChargeID, _ := shortid.Generate()

	// Step 2: Validate charge exists
	err = isChargeThere(db, input.ChargeID)
	if err != nil {
		return err
	}

	// Step 3: Check if this charge is already associated with this product
	exists, err := isProductCharge(db, input.ChargeID, input.ProductID)
	if err != nil {
		return err
	}

	// If association already exists, return nil (idempotent operation)
	if exists {
		return nil
	} else {
		// Step 4: Create new product-charge association
		_, err = db.Exec(`
		INSERT INTO product_charges (product_charge_id,product_id, charge_id)
		VALUES (?, ?,?)`,
			productChargeID, input.ProductID, input.ChargeID,
		)
		return err
	}
}

// isProductCharge checks if a charge is already associated with a product.
//
// This is an internal helper function used to prevent duplicate product-charge associations.
//
// Parameters:
//   - chargeID: The charge_id to check
//   - productID: The product_id to check
//
// Returns:
//   - bool: true if association exists, false otherwise
//   - error: Database error if query fails
func isProductCharge(db DBExecutor, chargeID, productID string) (bool, error) {
	// Check if product-charge association exists in database
	exists, err := RecordExists(db, "product_charges", "charge_id = ? and product_id = ?", chargeID, productID)
	if err != nil {
		return false, err
	}

	return exists, nil
}
