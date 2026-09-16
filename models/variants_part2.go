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
	"ekomasi_backend/dtos"
	"errors"
	"fmt"
)

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
