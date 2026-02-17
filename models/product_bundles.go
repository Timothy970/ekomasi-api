// Package models provides data access functions for the Adenzo e-commerce platform.
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
	"adenzo_backend/dtos"
	"fmt"
	"log"
	"strings"

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
			created_at, last_updated_at, sell_when_out_of_stock, buying_price
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
			&bundle.Price, &bundle.StockQuantity, &bundle.CreatedAt, &bundle.LastUpdated, &bundle.KeepSelling, &bundle.BuyingPrice,
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
			created_at, last_updated_at, sell_when_out_of_stock, buying_price
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
			&bundle.Price, &bundle.StockQuantity, &bundle.CreatedAt, &bundle.LastUpdated, &bundle.KeepSelling, &bundle.BuyingPrice,
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

		// Override stock quantity with bundle-specific quantity
		product.StockQuantity = quantity
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

// UpdateBundle updates an existing product bundle.
//
// Supports partial updates - only provided fields are updated.
// Updates bundle metadata, image, and/or product associations.
//
// Parameters:
//   - req: dtos.Bundle - Bundle update data (partial fields allowed)
//   - bundleID: string - The bundle to update
//
// Returns:
//   - error: "product not found", database error, or nil on success
func UpdateBundle(db DBExecutor, req dtos.Bundle, bundleID string) error {
	// Validate bundle exists
	if err := IsProductThere(db, bundleID); err != nil {
		return err
	}

	// Update bundle metadata
	if err := updateBundleMetadata(db, req, bundleID); err != nil {
		return err
	}

	// Update bundle image if provided
	if err := updateBundleImage(db, req.Image, bundleID); err != nil {
		return err
	}

	// Replace bundle products if provided
	if err := replaceBundleProducts(db, req.Products, bundleID); err != nil {
		return err
	}

	return nil
}

// updateBundleMetadata updates bundle fields based on provided data
func updateBundleMetadata(db DBExecutor, req dtos.Bundle, bundleID string) error {
	query := "UPDATE products SET"
	args := []interface{}{}
	updates := []string{}

	if req.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, req.Name)
	}
	if req.Description != "" {
		updates = append(updates, "description = ?")
		args = append(args, req.Description)
	}
	if req.Price != 0 {
		updates = append(updates, "price = ?")
		args = append(args, req.Price)
	}
	if req.KeepSelling != nil {
		updates = append(updates, "sell_when_out_of_stock = ?")
		args = append(args, *req.KeepSelling)
	}
	if req.CompareAtPrice != nil {
		updates = append(updates, "buying_price = ?")
		args = append(args, *req.CompareAtPrice)
	}

	// Skip update if no fields provided
	if len(updates) == 0 {
		return nil
	}

	query += " " + strings.Join(updates, ", ") + " WHERE product_id = ?"
	args = append(args, bundleID)

	if _, err := db.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update bundle: %v", err)
	}
	return nil
}

// updateBundleImage updates the bundle image if provided
func updateBundleImage(db DBExecutor, image, bundleID string) error {
	if image == "" {
		return nil
	}

	imageQuery := `UPDATE product_images SET url = ? WHERE product_id = ?`
	if _, err := db.Exec(imageQuery, image, bundleID); err != nil {
		return fmt.Errorf("failed to update bundle image: %v", err)
	}
	return nil
}

// replaceBundleProducts replaces all products in a bundle
func replaceBundleProducts(db DBExecutor, products []dtos.BundleProducts, bundleID string) error {
	if len(products) == 0 {
		return nil
	}

	// Delete all existing product associations
	deleteQuery := `DELETE FROM bundle_products WHERE bundle_id = ?`
	if _, err := db.Exec(deleteQuery, bundleID); err != nil {
		log.Printf("Delete bundle_products err::%s", err)
		return err
	}

	// Add new product associations
	return AddProductsToBundle(db, products, bundleID)
}

// DeleteBundle deletes a product bundle.
//
// Deletes the bundle product record. Cascade deletes will handle:
//   - bundle_products associations
//   - product_images
//   - Other related records via database constraints
//
// Parameters:
//   - bundleID: string - The bundle to delete
//
// Returns:
//   - error: "product not found", database error, or nil on success
func DeleteBundle(db DBExecutor, bundleID string) error {
	// Validate bundle exists
	err := IsProductThere(db, bundleID)
	if err != nil {
		return err
	}

	// Delete bundle (cascade deletes handle related records)
	query := `DELETE FROM products WHERE product_id = ?`
	_, err = db.Exec(query, bundleID)
	if err != nil {
		return err
	}

	return nil
}

// AddProductsToBundle adds products to a bundle.
//
// This function has idempotent behavior - if a product is already
// in the bundle, it skips adding it instead of failing.
//
// Parameters:
//   - req: []dtos.BundleProducts - Products to add with quantities
//   - bundleID: string - The bundle to add products to
//
// Returns:
//   - error: "product not found", database error, or nil on success
func AddProductsToBundle(db DBExecutor, req []dtos.BundleProducts, bundleID string) error {
	// Validate bundle exists
	err := IsProductThere(db, bundleID)
	if err != nil {
		return fmt.Errorf("%s", nobundle)
	}

	checkQuery := `SELECT COUNT(1) FROM bundle_products WHERE bundle_id = ? AND product_id = ?`
	insertQuery := `INSERT INTO bundle_products (bundle_product_id, bundle_id, product_id, quantity) VALUES (?, ?, ?, ?)`

	for _, product := range req {
		// Check if product already exists in bundle (idempotent behavior)
		var count int
		if err := db.QueryRow(checkQuery, bundleID, product.ProductID).Scan(&count); err != nil {
			return err
		}

		if count > 0 {
			// Skip adding - product already in bundle
			continue
		}

		// Generate unique bundle_product_id
		bundleProductID, _ := shortid.Generate()

		// Insert product into bundle with quantity
		if _, err := db.Exec(insertQuery, bundleProductID, bundleID, product.ProductID, product.Quantity); err != nil {
			return err
		}
	}

	return nil
}

// RemoveProductsFromBundle removes products from a bundle.
//
// Removes specified products from the bundle's product associations.
//
// Parameters:
//   - req: dtos.AddProductsToBundle - Contains array of product IDs to remove
//   - bundleID: string - The bundle to remove products from
//
// Returns:
//   - error: "product not found", "no products provided", database error, or nil on success
func RemoveProductsFromBundle(db DBExecutor, req dtos.AddProductsToBundle, bundleID string) error {
	// Validate bundle exists
	err := IsProductThere(db, bundleID)
	if err != nil {
		return err
	}

	// Validate product IDs provided
	if len(req.ProductIDs) == 0 {
		return fmt.Errorf("no products provided")
	}

	deleteQuery := `DELETE FROM bundle_products WHERE bundle_id = ? AND product_id = ?`

	// Remove each product from bundle
	for _, productID := range req.ProductIDs {
		if _, err := db.Exec(deleteQuery, bundleID, productID); err != nil {
			return fmt.Errorf("failed to remove product %s from bundle: %v", productID, err)
		}
	}

	return nil
}
