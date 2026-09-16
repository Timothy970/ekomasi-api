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
)

// - []dtos.Product: Array of products
// - error: Scan error or nil on success
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
