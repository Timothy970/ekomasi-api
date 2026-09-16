// Package models provides data access functions for the Ekomasi e-commerce platform.
//
// This file contains product bundle management functionality:
//   - Bundle retrieval with pagination
//   - Bundle creation with product associations
//   - Bundle updates (metadata, images, products)
//   - Bundle deletion with cascade
//   - Product-bundle associations (add/remove)
//
// Product bundles allow:
//   - Grouping multiple products into a single bundle
//   - Custom bundle pricing (different from sum of individual products)
//   - Per-product quantities within bundles
//   - Bundle-specific images and descriptions
//   - Stock management for bundles
//   - SKU generation with "BUNDLE-" prefix
package models

import (
	"ekomasi_backend/dtos"
	"log"

	"github.com/teris-io/shortid"
)

// GetBundleProducts retrieves paginated product bundles.
//
// Fetches all bundles (products with product_type='bundle') with their
// associated products, quantities, and images.
//
// Parameters:
//   - limit: int - Number of bundles per page
//   - page: int - Page number (1-based)
//
// Returns:
//   - []dtos.GetBundleRequest: Array of bundles with products and images
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func GetBundleProducts(db DBExecutor, limit, page int) ([]dtos.GetBundleRequest, *dtos.PaginationMeta, error) {
	// ----- Count total bundles -----
	// Count total bundles for pagination
	var total int
	countQuery := `
		SELECT COUNT(*) 
		FROM products 
		WHERE product_type = 'bundle'
	`
	if err := db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, nil, err
	}

	// Calculate pagination values
	offset := (page - 1) * limit
	totalPages := (total + limit - 1) / limit // Ceiling division

	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	// Fetch bundles ordered by creation date (newest first)
	query := `
		SELECT 
			product_id, name, description, sku, tag, price, stock_quantity,
			created_at, last_updated_at, buying_price
		FROM products
		WHERE product_type = 'bundle'
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan bundle rows and populate associated data
	var bundles []dtos.GetBundleRequest

	for rows.Next() {
		var bundle dtos.GetBundleRequest

		// Scan bundle base fields
		if err := rows.Scan(
			&bundle.ID, &bundle.Name, &bundle.Description, &bundle.SKU, &bundle.Tag,
			&bundle.Price, &bundle.StockQuantity, &bundle.CreatedAt, &bundle.LastUpdated, &bundle.BuyingPrice,
		); err != nil {
			return nil, nil, err
		}

		// Fetch products included in this bundle
		products, err := getProductsForBundle(db, bundle.ID)
		if err != nil {
			return nil, nil, err
		}
		bundle.Products = products

		// Fetch bundle images
		images, err := fetchProductImages(db, bundle.ID)
		if err != nil {
			return nil, nil, err
		}
		bundle.Images = images

		bundles = append(bundles, bundle)
	}

	return bundles, pagination, nil
}

// GetBundleByIDProducts retrieves a specific bundle by ID.
//
// Fetches bundle details including associated products and images.
//
// Parameters:
//   - bundleID: string - The bundle product ID to retrieve
//
// Returns:
//   - *dtos.GetBundleRequest: Pointer to the bundle (or nil if not found)
//   - error: Database error or nil on success
func GetBundleByIDProducts(db DBExecutor, bundleID string) (*dtos.GetBundleRequest, error) {
	// Fetch specific bundle by ID
	query := `
		SELECT 
			product_id, name, description, sku, tag, price, stock_quantity,
			created_at, last_updated_at, buying_price
		FROM products
		WHERE product_type = 'bundle' AND product_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, bundleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan bundle rows and populate associated data
	var bundle dtos.GetBundleRequest

	for rows.Next() {
		// Scan bundle base fields
		if err := rows.Scan(
			&bundle.ID, &bundle.Name, &bundle.Description, &bundle.SKU, &bundle.Tag,
			&bundle.Price, &bundle.StockQuantity, &bundle.CreatedAt, &bundle.LastUpdated, &bundle.BuyingPrice,
		); err != nil {
			return nil, err
		}

		// Fetch products included in this bundle
		products, err := getProductsForBundle(db, bundle.ID)
		if err != nil {
			return nil, err
		}
		bundle.Products = products

		// Fetch bundle images
		images, err := fetchProductImages(db, bundle.ID)
		if err != nil {
			return nil, err
		}
		bundle.Images = images
	}

	return &bundle, nil
}

// getProductsForBundle retrieves all products associated with a bundle.
//
// Fetches products from bundle_products table and enriches with full
// product details. The quantity field is overridden with the bundle-specific
// quantity from bundle_products table.
//
// Parameters:
//   - bundleID: string - The bundle to fetch products for
//
// Returns:
//   - []dtos.Product: Array of products with bundle quantities
//   - error: Database error or nil on success
func getProductsForBundle(db DBExecutor, bundleID string) ([]dtos.Product, error) {
	// Query bundle product associations
	query := `
		SELECT product_id, quantity
		FROM bundle_products
		WHERE bundle_id = ?
	`
	rows, err := db.Query(query, bundleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.Product

	for rows.Next() {
		var productID string
		var quantity int
		if err := rows.Scan(&productID, &quantity); err != nil {
			return nil, err
		}

		// Fetch full product details using reusable function
		product, err := GetProductByID(db, productID)
		if err != nil {
			// Skip missing products instead of failing entire bundle
			// (handles cases where product was deleted but bundle_products entry remains)
			log.Printf("warning: failed to fetch product %s for bundle %s: %v", productID, bundleID, err)
			continue
		}

		product.BundleQuantity = quantity
		products = append(products, *product)
	}

	return products, nil
}

// CreateBundle creates a new product bundle.
//
// This function:
// 1. Validates all included products exist
// 2. Creates bundle product record with product_type='bundle'
// 3. Generates SKU with "BUNDLE-" prefix
// 4. Inserts bundle image
// 5. Associates products with the bundle
//
// Parameters:
//   - req: dtos.Bundle - Bundle data (name, description, price, products, image)
//   - userID: string - The user creating the bundle
//
// Returns:
//   - error: "product not found", database error, or nil on success
func CreateBundle(db DBExecutor, req dtos.Bundle, userID string) error {
	// Validate all products in bundle exist
	for _, product := range req.Products {
		err := IsProductThere(db, product.ProductID)
		if err != nil {
			return err
		}
	}

	// Generate unique bundle ID
	productID, _ := shortid.Generate()

	// Insert bundle as product with product_type='bundle'
	_, err := db.Exec(`
		INSERT INTO products (product_id, name, description, sku, price, stock_quantity, created_by_id, buying_price, search_vector, product_type)
		VALUES (?,?,?,?,?,?,?,?,?, 'bundle')
	`, productID, req.Name, req.Description, "BUNDLE-"+productID, req.Price, req.StockQuantity, userID, req.CompareAtPrice, req.Name)
	if err != nil {
		return err
	}

	// Add bundle image (set as primary)
	err = InsertProductImage(db, productID, req.Image, "gallery", true)
	if err != nil {
		return err
	}

	// Associate products with bundle
	if len(req.Products) > 0 {
		err = AddProductsToBundle(db, req.Products, productID)
		if err != nil {
			return err
		}
	}
	return nil
}
