// Package models provides data access functions for the Ekomasi e-commerce backend.
//
// This file contains product variant management operations including:
//   - Variant CRUD (create, read, update, delete)
//   - Product-variant associations
//   - Variant filtering and grouping by type
//   - Paginated product retrieval by variants
//   - Stock quantity and additional price management
//
// Variants represent product options like colors, sizes, materials, etc.
// Each variant has a type (e.g., "color", "size"), name, and optional hex code.
// Products can have multiple variants with variant-specific pricing and stock.
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"strings"

	"github.com/teris-io/shortid"
)

// CreateVariant creates a new variant in the system.
//
// Variants represent product options (e.g., "Red" color, "Large" size).
// Hex codes are typically used for color variants.
//
// Parameters:
//   - req: dtos.VariantRequest containing:
//   - VariantType: Category of variant (e.g., "color", "size", "material")
//   - Name: Variant name (e.g., "Red", "Large", "Cotton")
//   - HexCode: Optional hex color code for color variants
//
// Returns:
//   - string: Generated variant ID
//   - error: Database error or nil on success
func CreateVariant(db DBExecutor, req dtos.VariantRequest) (string, error) {
	//validate name of that variant type is not already there
	err := validateVariantName(db, req.VariantType, req.Name)
	if err != nil {
		return "", err
	}
	// Generate unique variant ID
	id, _ := shortid.Generate()

	// Insert variant record
	_, err = db.Exec(`
        INSERT INTO variants (variant_id, variant_type, name, hex_code)
        VALUES (?, ?, ?, ?)`,
		id, req.VariantType, req.Name, req.HexCode,
	)
	return id, err
}

// helper function to validate if a variant name already exists for a given variant type
// parameters:
//   - variantType: string - The type of the variant (e.g., "color", "size")
//   - name: string - The name of the variant to validate
//
// returns:
//   - error: "variant name already exists for this variant type", database error, or nil if valid
func validateVariantName(db DBExecutor, variantType, name string) error {

	exists, err := RecordExists(db, "variants", "LOWER(variant_type) = LOWER(?) AND LOWER(name) = LOWER(?)", strings.ToLower(variantType), strings.ToLower(name))
	if err != nil {
		return err
	}
	if exists {
		return errors.New("variant with this name already exists")
	}
	return nil
}

// GetVariant retrieves a specific variant by ID.
//
// Parameters:
//   - id: string - The variant ID to retrieve
//
// Returns:
//   - *dtos.VariantResponse: Variant data with type, name, and hex code
//   - error: "variant not found", database error, or nil on success
func GetVariant(db DBExecutor, id string) (*dtos.VariantResponse, error) {
	// Validate variant exists
	err := variantexists(db, id)
	if err != nil {
		return nil, err
	}

	var v dtos.VariantResponse
	// Retrieve variant details
	err = db.QueryRow(`
        SELECT variant_id, variant_type, name, hex_code
        FROM variants
        WHERE variant_id = ?`, id,
	).Scan(&v.VariantID, &v.VariantType, &v.Name, &v.HexCode)
	if err == sql.ErrNoRows {
		return nil, errors.New("variant not found")
	}
	return &v, err
}

// ListVariants retrieves all variants grouped by variant type.
//
// This function fetches all variants and organizes them by type
// (e.g., all "color" variants together, all "size" variants together)
// in insertion order for deterministic output.
//
// Returns:
//   - []dtos.GroupedVariants: Array of variant groups, each containing:
//   - VariantType: The type name (e.g., "color", "size")
//   - Variants: Array of variants of that type
//   - error: Database error or nil on success
func ListVariants(db DBExecutor) ([]dtos.GroupedVariants, error) {
	// Fetch all variants ordered by type then name
	rows, err := db.Query(`
        SELECT variant_id, variant_type, name, hex_code
        FROM variants
        ORDER BY lower(variant_type), name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group variants by type (case-insensitive) while preserving insertion order
	groupMap := make(map[string][]dtos.VariantResponse)
	order := []string{} // Track first appearance of each variant type

	for rows.Next() {
		var v dtos.VariantResponse
		if err := rows.Scan(&v.VariantID, &v.VariantType, &v.Name, &v.HexCode); err != nil {
			return nil, err
		}
		// Use lowercase variant type as key for case-insensitive grouping
		key := strings.ToLower(v.VariantType)
		// Track first occurrence of this variant type
		if _, exists := groupMap[key]; !exists {
			order = append(order, key)
		}
		groupMap[key] = append(groupMap[key], v)
	}

	// Build grouped slice in deterministic order
	var grouped []dtos.GroupedVariants
	for _, t := range order {
		grouped = append(grouped, dtos.GroupedVariants{
			VariantType: t,
			Variants:    groupMap[t],
		})
	}

	return grouped, nil
}

// variantexists validates if a variant exists in the system.
//
// This helper function is used before update/delete operations.
//
// Parameters:
//   - id: string - The variant ID to validate
//
// Returns:
//   - error: "variant not found" if not exists, database error, or nil if exists
func variantexists(db DBExecutor, id string) error {
	exists, err := RecordExists(db, "variants", "variant_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("variant not found")
	}
	return nil
}

// UpdateVariantByID updates an existing variant's information.
//
// Parameters:
//   - id: string - The variant ID to update
//   - req: dtos.VariantRequest containing updated variant data
//
// Returns:
//   - error: "variant not found", database error, or nil on success
func UpdateVariantByID(db DBExecutor, id string, req dtos.VariantRequest) error {
	// Validate variant exists
	err := variantexists(db, id)
	if err != nil {
		return err
	}
	//validate name of that variant type is not already there
	err = validateVariantName(db, req.VariantType, req.Name)
	if err != nil {
		return err
	}

	// Update variant fields
	_, err = db.Exec(`
        UPDATE variants
        SET variant_type = ?, name = ?, hex_code = ?
        WHERE variant_id = ?`,
		req.VariantType, req.Name, req.HexCode, id,
	)
	return err
}

// DeleteVariantByID permanently removes a variant.
//
// Parameters:
//   - id: string - The variant ID to delete
//
// Returns:
//   - error: "variant not found", database error, or nil on success
func DeleteVariantByID(db DBExecutor, id string) error {
	// Validate variant exists
	err := variantexists(db, id)
	if err != nil {
		return err
	}

	// Delete variant record
	_, err = db.Exec(`DELETE FROM variants WHERE variant_id = ?`, id)
	return err
}

// AddProductVariant associates a variant with a product.
//
// This function creates a product-variant mapping with optional additional
// price and stock quantity. If the product already has this variant, the
// function returns successfully without error (idempotent).
//
// Parameters:
//   - variantID: string - The variant to associate
//   - req: dtos.ProductVariantRequest containing:
//   - ProductID: The product to add variant to
//   - AdditionalPrice: Optional price adjustment for this variant
//   - StockQuantity: Stock available for this variant
//
// Returns:
//   - error: "variant not found", "product not found", database error, or nil on success
//
// adding boolean for checking association between product and variant and to be true by default
func AddProductVariant(db DBExecutor, variantID string, req dtos.ProductVariantRequest) error {
	// Generate unique product-variant ID
	pvID, _ := shortid.Generate()

	// Validate variant exists
	if err := variantexists(db, variantID); err != nil {
		return err
	}
	// Validate product exists
	if err := IsProductThere(db, req.ProductID); err != nil {
		return err
	}

	// Check if product-variant association already exists
	exists, err := isProductWithVariant(db, variantID, req.ProductID)
	if err != nil {
		return err
	}

	// Handle optional additional price
	additionalPrice := 0.0
	if req.AdditionalPrice != nil {
		additionalPrice = *req.AdditionalPrice
	}

	// Skip insertion if association already exists (idempotent)
	if exists {
		return nil

	} else {
		// Insert new product-variant record
		_, err = db.Exec(`
			INSERT INTO product_variants (product_variants_id, variant_id, product_id, additional_price, stock_quantity)
			VALUES (?, ?, ?, ?, ?)`,
			pvID, variantID, req.ProductID, additionalPrice, req.StockQuantity,
		)
	}

	return err
}
