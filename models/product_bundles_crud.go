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
	"fmt"
	"log"
	"strings"

	"github.com/teris-io/shortid"
)

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
	args := []any{}
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
	if req.CompareAtPrice != nil {
		updates = append(updates, "buying_price = ?")
		args = append(args, *req.CompareAtPrice)
	}

	if req.StockQuantity != 0 {
		updates = append(updates, "stock_quantity = ?")
		args = append(args, req.StockQuantity)
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
