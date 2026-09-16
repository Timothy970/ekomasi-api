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
	"fmt"
	"strings"
)

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
