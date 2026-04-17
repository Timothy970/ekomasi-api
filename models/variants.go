// Package models provides data access functions for the Adenzo e-commerce backend.
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
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"
	"math"
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

// HoldProductVariants retrieves all variant IDs associated with a product.
//
// This function is typically used to temporarily hold variant references
// before performing batch operations.
//
// Parameters:
//   - productID: string - The product to get variants for
//
// Returns:
//   - []string: Array of variant IDs
//   - error: Database error or nil on success
func HoldProductVariants(db DBExecutor, productID string) error {
	// Fetch all variant IDs for the product
	rows, err := db.Query(`
		SELECT variant_id FROM product_variants
		WHERE product_id = ?`, productID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Collect variant IDs
	var variantIDs []string
	for rows.Next() {
		var variantID string
		err := rows.Scan(&variantID)
		if err != nil {
			return err
		}
		variantIDs = append(variantIDs, variantID)
	}
	for _, variantID := range variantIDs {
		err := RemoveHeldProductVariants(db, variantID)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteVariantSelectionsByProductID(db DBExecutor, productID string) error {
	query := `
	DELETE FROM product_variant_combinations
	WHERE product_id = ?
	`

	result, err := db.Exec(query, productID)
	if err != nil {
		return fmt.Errorf("deleting variant selections for product %s: %w", productID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	// Optional: useful for logging/debugging
	if rowsAffected == 0 {
		// not necessarily an error, but good to know
		return nil
	}

	return nil
}

// RemoveHeldProductVariants removes all product-variant associations for a variant.
//
// This function deletes all products associated with the specified variant.
// Used for batch operations after holding variant references.
//
// Parameters:
//   - variantID: string - The variant to remove product associations for
//
// Returns:
//   - error: Database error or nil on success
func RemoveHeldProductVariants(db DBExecutor, variantID string) error {
	// Delete all product-variant associations for this variant
	_, err := db.Exec(`DELETE FROM product_variants WHERE variant_id = ?`, variantID)
	return err
}

// isProductWithVariant checks if a product has a specific variant.
//
// Parameters:
//   - variantID: string - The variant to check
//   - productID: string - The product to check
//
// Returns:
//   - bool: true if association exists, false otherwise
//   - error: Database error or nil on success
func isProductWithVariant(db DBExecutor, variantID, productID string) (bool, error) {
	// Check if product-variant association exists
	exists, err := RecordExists(db, "product_variants", "variant_id = ? and product_id = ?", variantID, productID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	} else {
		return true, nil
	}

}

// RemoveProductVariant removes a specific variant from a product.
//
// This function validates both product and variant existence before
// removing the association.
//
// Parameters:
//   - productID: string - The product to remove variant from
//   - variantID: string - The variant to remove
//
// Returns:
//   - error: "product not found", "variant not found",
//     "no such product variant mapping found", database error, or nil on success
func RemoveProductVariant(db DBExecutor, productID, variantID string) error {
	// Validate product exists
	err := IsProductThere(db, productID)
	if err != nil {
		return err
	}
	// Validate variant exists
	err = variantexists(db, variantID)
	if err != nil {
		return err
	}

	// Delete product-variant association
	result, err := db.Exec(`
		DELETE FROM product_variants
		WHERE product_id = ? AND variant_id = ?`, productID, variantID,
	)
	if err != nil {
		return err
	}

	// Verify deletion occurred
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no such product variant mapping found")
	}
	return nil
}

// ListProductVariants retrieves all variants for a specific product.
//
// This function fetches variant associations including additional price
// and stock quantity for each variant.
//
// Parameters:
//   - productID: string - The product to list variants for
//
// Returns:
//   - []dtos.ProductVariantResponse: Array of product-variant data
//   - error: Database error or nil on success
func ListProductVariants(db DBExecutor, productID string) ([]dtos.ProductVariantResponse, error) {
	// Fetch all variants for the product
	rows, err := db.Query(`
        SELECT variant_id, product_id, additional_price, stock_quantity
        FROM product_variants
        WHERE product_id = ?`, productID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process each variant
	var pv []dtos.ProductVariantResponse
	for rows.Next() {
		var item dtos.ProductVariantResponse
		if err := rows.Scan(&item.VariantID, &item.ProductID, &item.AdditionalPrice, &item.StockQuantity); err != nil {
			return nil, err
		}
		pv = append(pv, item)
	}
	return pv, nil
}

// GetVariantsWithProductsPaginated retrieves variants with their associated products.
//
// This function performs complex filtering and pagination:
// 1. Filters variants by IDs and/or names
// 2. Retrieves paginated products for those variants
// 3. Associates products with their variants
// 4. Handles special "All" variant name for fetching all products
//
// Parameters:
//   - variants: []dtos.Variant - Array of variant filters (by ID or name)
//   - page: int - Page number (1-based)
//   - limit: int - Products per page
//
// Returns:
//   - []*dtos.VariantWithProducts: Variants with their paginated products
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: "no variants provided", database error, or nil on success
func GetVariantsWithProductsPaginated(db DBExecutor, variants []dtos.Variant, page, limit int) ([]*dtos.VariantWithProducts, *dtos.PaginationMeta, error) {
	if len(variants) == 0 {
		return nil, nil, errors.New("no variants provided")
	}

	// Extract unique variant IDs and names from input
	variantIDs, variantNames := extractVariantFilters(variants)

	// Query variants and build variant map
	variantResults, variantMap, err := executeVariantQuery(db, variantIDs, variantNames)
	if err != nil {
		return nil, nil, err
	}
	if len(variantResults) == 0 {
		return nil, nil, nil
	}

	// Count total products across all variants for pagination
	resultVariantIDs := extractVariantIDs(variantResults)
	total, err := countTotalProducts(db, resultVariantIDs)
	if err != nil {
		return nil, nil, err
	}

	// Fetch paginated products with variant mappings
	products, productVariantMap, err := fetchProductsByVariantsPaginated(resultVariantIDs, limit, (page-1)*limit)
	if err != nil {
		return nil, nil, err
	}

	// Associate products with their variants
	associateProductsWithVariants(variantMap, products, productVariantMap)

	// Handle special "All" variant name (fetches all products for that variant)
	if err := handleAllVariants(db, variants, variantMap); err != nil {
		return nil, nil, err
	}

	pagination := createPaginationMeta(page, limit, total)
	return variantResults, pagination, nil
}

// extractVariantFilters extracts unique variant IDs and names from input.
//
// This helper deduplicates variant filters and separates IDs from names.
//
// Parameters:
//   - variants: []dtos.Variant - Input variant filters
//
// Returns:
//   - []string: Unique variant IDs
//   - []string: Unique variant names (excluding "All")
func extractVariantFilters(variants []dtos.Variant) ([]string, []string) {
	variantIDMap := make(map[string]bool)
	variantNameMap := make(map[string]bool)
	var variantIDs, variantNames []string

	for _, variant := range variants {
		// Collect unique variant IDs
		if variant.VariantID != "" && !variantIDMap[variant.VariantID] {
			variantIDs = append(variantIDs, variant.VariantID)
			variantIDMap[variant.VariantID] = true
		}
		// Collect unique variant names (excluding "All")
		if isVariantNameValid(variant.Name) && !variantNameMap[variant.Name] {
			variantNames = append(variantNames, variant.Name)
			variantNameMap[variant.Name] = true
		}
	}

	return variantIDs, variantNames
}

// isVariantNameValid checks if variant name is valid for filtering.
//
// "All" is treated specially and excluded from name filters.
//
// Parameters:
//   - name: string - Variant name to validate
//
// Returns:
//   - bool: true if valid for filtering, false otherwise
func isVariantNameValid(name string) bool {
	return name != "" && name != "All"
}

// executeVariantQuery fetches variants matching the filters.
//
// Parameters:
//   - variantIDs: []string - Variant IDs to filter by
//   - variantNames: []string - Variant names to filter by
//
// Returns:
//   - []*dtos.VariantWithProducts: Array of variants
//   - map[string]*dtos.VariantWithProducts: Map of variant ID to variant
//   - error: Database error or nil on success
func executeVariantQuery(db DBExecutor, variantIDs, variantNames []string) ([]*dtos.VariantWithProducts, map[string]*dtos.VariantWithProducts, error) {
	// Build SQL query with IN clauses
	query, args := buildVariantQuery(variantIDs, variantNames)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	return scanVariantRows(rows)
}

// buildVariantQuery constructs SQL query with dynamic filters.
//
// Parameters:
//   - variantIDs: []string - Variant IDs to include
//   - variantNames: []string - Variant names to include (case-insensitive)
//
// Returns:
//   - string: SQL query
//   - []interface{}: Query arguments
func buildVariantQuery(variantIDs, variantNames []string) (string, []interface{}) {
	// Base query joins variants with product_variants
	query := `
        SELECT v.variant_id, v.variant_type, v.name, v.hex_code,
               pv.additional_price, pv.stock_quantity
        FROM variants v
        INNER JOIN product_variants pv ON v.variant_id = pv.variant_id
        WHERE 1=1
    `
	var args []interface{}

	// Add IN clause for variant IDs if provided
	query, args = addInClause(query, args, "v.variant_id", variantIDs)
	// Add IN clause for variant names (case-insensitive) if provided
	query, args = addInClause(query, args, "LOWER(v.name)", transformToLower(variantNames))

	return query, args
}

// addInClause adds an IN clause to the SQL query.
//
// Parameters:
//   - query: string - Current SQL query
//   - args: []interface{} - Current query arguments
//   - field: string - Field name for IN clause
//   - values: []string - Values for IN clause
//
// Returns:
//   - string: Updated SQL query
//   - []interface{}: Updated query arguments
func addInClause(query string, args []interface{}, field string, values []string) (string, []interface{}) {
	if len(values) == 0 {
		return query, args
	}

	// Build placeholders: ?,?,?
	placeholders := "?" + strings.Repeat(",?", len(values)-1)
	query += fmt.Sprintf(" AND %s IN (%s)", field, placeholders)

	// Append values to arguments
	for _, value := range values {
		args = append(args, value)
	}

	return query, args
}

// transformToLower converts string array to lowercase.
//
// Used for case-insensitive variant name matching.
//
// Parameters:
//   - names: []string - Names to convert
//
// Returns:
//   - []string: Lowercase names
func transformToLower(names []string) []string {
	result := make([]string, len(names))
	for i, name := range names {
		result[i] = strings.ToLower(name)
	}
	return result
}

// scanVariantRows scans variant query results and builds variant map.
//
// Creates both an array and a map for efficient lookups.
//
// Parameters:
//   - rows: *sql.Rows - Query result rows
//
// Returns:
//   - []*dtos.VariantWithProducts: Array of variants
//   - map[string]*dtos.VariantWithProducts: Map of variant ID to variant
//   - error: Scan error or nil on success
func scanVariantRows(rows *sql.Rows) ([]*dtos.VariantWithProducts, map[string]*dtos.VariantWithProducts, error) {
	var variantResults []*dtos.VariantWithProducts
	variantMap := make(map[string]*dtos.VariantWithProducts)

	for rows.Next() {
		var variant dtos.VariantWithProducts
		// Scan variant fields from joined query
		err := rows.Scan(&variant.VariantID, &variant.VariantType, &variant.Name, &variant.HexCode,
			&variant.AdditionalPrice, &variant.StockQuantity)
		if err != nil {
			return nil, nil, err
		}
		// Add to both map and array
		variantMap[variant.VariantID] = &variant
		variantResults = append(variantResults, &variant)
	}

	return variantResults, variantMap, nil
}

// extractVariantIDs extracts variant IDs from variant array.
//
// Parameters:
//   - variants: []*dtos.VariantWithProducts - Variants to extract IDs from
//
// Returns:
//   - []string: Array of variant IDs
func extractVariantIDs(variants []*dtos.VariantWithProducts) []string {
	ids := make([]string, len(variants))
	for i, v := range variants {
		ids[i] = v.VariantID
	}
	return ids
}

// countTotalProducts counts total products for pagination.
//
// Counts unique products across all specified variants.
//
// Parameters:
//   - variantIDs: []string - Variant IDs to count products for
//
// Returns:
//   - int: Total unique product count
//   - error: Database error or nil on success
func countTotalProducts(db DBExecutor, variantIDs []string) (int, error) {
	if len(variantIDs) == 0 {
		return 0, nil
	}

	// Build query to count distinct products with these variants
	query := "SELECT COUNT(DISTINCT p.product_id) FROM products p " +
		"INNER JOIN product_variants pv ON p.product_id = pv.product_id " +
		"WHERE pv.variant_id IN (?" + strings.Repeat(",?", len(variantIDs)-1) + ")"

	args := makeInterfaceSlice(variantIDs)

	var total int
	err := db.QueryRow(query, args...).Scan(&total)
	return total, err
}

// associateProductsWithVariants maps products to their variants.
//
// Updates each variant's Products array with associated products.
//
// Parameters:
//   - variantMap: map[string]*dtos.VariantWithProducts - Variant lookup map
//   - products: []dtos.Product - Products to associate
//   - productVariantMap: map[string][]string - Map of product ID to variant IDs
func associateProductsWithVariants(variantMap map[string]*dtos.VariantWithProducts, products []dtos.Product, productVariantMap map[string][]string) {
	// Create product lookup map for efficient access
	productMap := createProductMap(products)

	for variantID, productIDs := range productVariantMap {
		if variant, exists := variantMap[variantID]; exists {
			variant.Products = findProductsByIDs(productMap, productIDs)
		}
	}
}

// createProductMap creates product lookup map.
//
// Parameters:
//   - products: []dtos.Product - Products to map
//
// Returns:
//   - map[string]dtos.Product: Map of product ID to product
func createProductMap(products []dtos.Product) map[string]dtos.Product {
	productMap := make(map[string]dtos.Product)
	for _, product := range products {
		productMap[product.ID] = product
	}
	return productMap
}

// findProductsByIDs retrieves products by ID array.
//
// Looks up products from map and returns only those that exist.
//
// Parameters:
//   - productMap: map[string]dtos.Product - Product lookup map
//   - productIDs: []string - Product IDs to find
//
// Returns:
//   - []dtos.Product: Found products
func findProductsByIDs(productMap map[string]dtos.Product, productIDs []string) []dtos.Product {
	var result []dtos.Product
	for _, productID := range productIDs {
		if product, exists := productMap[productID]; exists {
			result = append(result, product)
		}
	}
	return result
}

// handleAllVariants handles special "All" variant name.
//
// When variant name is "All", fetches all products for that variant
// type instead of paginated results.
//
// Parameters:
//   - variants: []dtos.Variant - Input variants to check for "All"
//   - variantMap: map[string]*dtos.VariantWithProducts - Variant map to update
//
// Returns:
//   - error: Database error or nil on success
func handleAllVariants(db DBExecutor, variants []dtos.Variant, variantMap map[string]*dtos.VariantWithProducts) error {
	for _, variant := range variants {
		// Check for "All" variant with valid ID
		if variant.Name == "All" && variant.VariantID != "" {
			if existingVariant, exists := variantMap[variant.VariantID]; exists {
				// Fetch all products for this variant (no pagination)
				allProducts, err := fetchAllProductsByVariant(variant.VariantID)
				if err != nil {
					return err
				}
				existingVariant.Products = allProducts
			}
		}
	}
	return nil
}

// createPaginationMeta builds pagination metadata.
//
// Calculates total pages and navigation flags.
//
// Parameters:
//   - page: int - Current page (1-based)
//   - limit: int - Items per page
//   - total: int - Total item count
//
// Returns:
//   - *dtos.PaginationMeta: Pagination metadata
func createPaginationMeta(page, limit, total int) *dtos.PaginationMeta {
	// Calculate total pages using ceiling division
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
}

// makeInterfaceSlice converts string slice to interface slice.
//
// Used for SQL variadic arguments.
//
// Parameters:
//   - strings: []string - Strings to convert
//
// Returns:
//   - []interface{}: Interface slice
func makeInterfaceSlice(strings []string) []interface{} {
	args := make([]interface{}, len(strings))
	for i, s := range strings {
		args[i] = s
	}
	return args
}

// fetchProductsByVariantsPaginated fetches paginated products with variant mappings.
//
// This refactored function:
// 1. Retrieves product-variant mappings
// 2. Gets unique product IDs
// 3. Fetches paginated product details
//
// Parameters:
//   - variantIDs: []string - Variant IDs to fetch products for
//   - limit: int - Number of products to fetch
//   - offset: int - Number of products to skip
//
// Returns:
//   - []dtos.Product: Array of products
//   - map[string][]string: Map of variant ID to product IDs
//   - error: "no variant IDs provided", database error, or nil on success
func fetchProductsByVariantsPaginated(variantIDs []string, limit, offset int) ([]dtos.Product, map[string][]string, error) {
	if len(variantIDs) == 0 {
		return nil, nil, errors.New("no variant IDs provided")
	}

	// First, get product-variant mappings and unique product IDs
	productVariantMap, productIDs, err := fetchProductVariantMappings(variantIDs)
	if err != nil {
		return nil, nil, err
	}

	if len(productIDs) == 0 {
		return nil, productVariantMap, nil
	}

	// Fetch product details with pagination
	products, err := fetchPaginatedProducts(productIDs, limit, offset)
	if err != nil {
		return nil, nil, err
	}

	return products, productVariantMap, nil
}

// fetchProductVariantMappings retrieves product-variant associations.
//
// Returns both a mapping of variant IDs to product IDs and a list of
// unique product IDs.
//
// Parameters:
//   - variantIDs: []string - Variant IDs to fetch mappings for
//
// Returns:
//   - map[string][]string: Map of variant ID to array of product IDs
//   - []string: Unique product IDs
//   - error: Database error or nil on success
func fetchProductVariantMappings(variantIDs []string) (map[string][]string, []string, error) {
	// Build query with IN clause
	query := buildMappingQuery(variantIDs)
	args := makeInterfaceSlice(variantIDs)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	return scanProductVariantMappings(rows)
}

// buildMappingQuery constructs query for product-variant mappings.
//
// Parameters:
//   - variantIDs: []string - Variant IDs for IN clause
//
// Returns:
//   - string: SQL query with placeholders
func buildMappingQuery(variantIDs []string) string {
	return `
        SELECT pv.variant_id, pv.product_id
        FROM product_variants pv
        WHERE pv.variant_id IN (?` + strings.Repeat(",?", len(variantIDs)-1) + `)
        ORDER BY pv.product_id`
}

// scanProductVariantMappings scans mapping rows and builds data structures.
//
// Tracks unique product IDs to avoid duplicates in the product ID list.
//
// Parameters:
//   - rows: *sql.Rows - Query result rows
//
// Returns:
//   - map[string][]string: Map of variant ID to product IDs
//   - []string: Unique product IDs in order of first appearance
//   - error: Scan error or nil on success
func scanProductVariantMappings(rows *sql.Rows) (map[string][]string, []string, error) {
	productVariantMap := make(map[string][]string)
	allProductIDs := make(map[string]bool) // Track uniqueness
	var productIDs []string

	for rows.Next() {
		var variantID, productID string
		if err := rows.Scan(&variantID, &productID); err != nil {
			return nil, nil, err
		}

		// Add product to variant's product list
		productVariantMap[variantID] = append(productVariantMap[variantID], productID)
		// Track unique product IDs
		if !allProductIDs[productID] {
			allProductIDs[productID] = true
			productIDs = append(productIDs, productID)
		}
	}

	return productVariantMap, productIDs, nil
}

// fetchPaginatedProducts fetches product details with pagination.
//
// Parameters:
//   - productIDs: []string - Product IDs to fetch
//   - limit: int - Number of products to return
//   - offset: int - Number of products to skip
//
// Returns:
//   - []dtos.Product: Array of products with images
//   - error: Database error or nil on success
func fetchPaginatedProducts(productIDs []string, limit, offset int) ([]dtos.Product, error) {
	// Build query with IN clause and pagination
	query := buildProductQueryVariants(productIDs)
	args := makeInterfaceSlice(productIDs)
	args = append(args, limit, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan products from rows
	products, err := scanProducts(rows)
	if err != nil {
		return nil, err
	}

	// Batch fetch images for all products
	return fetchProductImagesBatch(products)
}

// buildProductQueryVariants constructs product query with pagination.
//
// Parameters:
//   - productIDs: []string - Product IDs for IN clause
//
// Returns:
//   - string: SQL query with placeholders for IDs, limit, and offset
func buildProductQueryVariants(productIDs []string) string {
	return `
        SELECT p.product_id, p.name, p.description, p.price, p.category_id,
               p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
        FROM products p
        WHERE p.product_id IN (?` + strings.Repeat(",?", len(productIDs)-1) + `)
        ORDER BY p.product_id
        LIMIT ? OFFSET ?`
}

// scanProducts scans product rows into array.
//
// Parameters:
//   - rows: *sql.Rows - Query result rows
//
// Returns:
//   - []dtos.Product: Array of products
//   - error: Scan error or nil on success
func scanProducts(rows *sql.Rows) ([]dtos.Product, error) {
	var products []dtos.Product

	for rows.Next() {
		var p dtos.Product
		// Scan all product fields
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}

// fetchProductImagesBatch fetches images for multiple products.
//
// Iterates through products and fetches images for each.
//
// Parameters:
//   - products: []dtos.Product - Products to fetch images for
//
// Returns:
//   - []dtos.Product: Products with images populated
//   - error: Database error or nil on success
func fetchProductImagesBatch(products []dtos.Product) ([]dtos.Product, error) {
	for i := range products {
		// Fetch images for each product
		images, err := fetchProductImages(DB, products[i].ID)
		if err != nil {
			return nil, err
		}
		products[i].Images = images
	}
	return products, nil
}

// fetchAllProductsByVariant fetches all products for a variant.
//
// Used by handleAllVariants for "All" variant name. Fetches complete
// product set without pagination.
//
// Parameters:
//   - variantID: string - Variant ID to fetch all products for
//
// Returns:
//   - []dtos.Product: All products with this variant
//   - error: Database error or nil on success
func fetchAllProductsByVariant(variantID string) ([]dtos.Product, error) {
	// Query products with this variant (no pagination)
	rows, err := DB.Query(`
        SELECT p.product_id, p.name, p.description, p.price, p.category_id,
               p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
        FROM products p
        INNER JOIN product_variants pv ON p.product_id = pv.product_id
        WHERE pv.variant_id = ?`, variantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.Product
	for rows.Next() {
		var p dtos.Product
		// Scan product fields
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated)
		if err != nil {
			return nil, err
		}

		// Fetch images for this product
		images, err := fetchProductImages(DB, p.ID)
		if err != nil {
			return nil, err
		}
		p.Images = images

		products = append(products, p)
	}

	return products, nil
}
