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
	"math"
	"strings"
)

// - []dtos.Product: Found products
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
func handleAllVariants(_ DBExecutor, variants []dtos.Variant, variantMap map[string]*dtos.VariantWithProducts) error {
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
//   - []any: Interface slice
func makeInterfaceSlice(strings []string) []any {
	args := make([]any, len(strings))
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
