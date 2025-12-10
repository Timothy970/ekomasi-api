package models

import (
	"adenzo_backend/dtos"
	"fmt"
	"log"
	"strings"

	"github.com/teris-io/shortid"
)

// Get bundles
func GetBundleProducts(limit, page int) ([]dtos.GetBundleRequest, *dtos.PaginationMeta, error) {
	// ----- Count total bundles -----
	var total int
	countQuery := `
		SELECT COUNT(*) 
		FROM products 
		WHERE product_type = 'bundle'
	`
	if err := DB.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, nil, err
	}

	offset := (page - 1) * limit
	totalPages := (total + limit - 1) / limit

	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	// ----- Fetch bundles -----
	query := `
		SELECT 
			product_id, name, description, sku, tag, price, stock_quantity,
			created_at, last_updated_at
		FROM products
		WHERE product_type = 'bundle'
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := DB.Query(query, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// ----- Scan rows -----
	var bundles []dtos.GetBundleRequest

	for rows.Next() {
		var bundle dtos.GetBundleRequest

		if err := rows.Scan(
			&bundle.ID, &bundle.Name, &bundle.Description, &bundle.SKU, &bundle.Tag,
			&bundle.Price, &bundle.StockQuantity, &bundle.CreatedAt, &bundle.LastUpdated,
		); err != nil {
			return nil, nil, err
		}

		// Fetch bundle products
		products, err := getProductsForBundle(bundle.ID)
		if err != nil {
			return nil, nil, err
		}
		bundle.Products = products

		// Fetch images
		images, err := fetchProductImages(bundle.ID)
		if err != nil {
			return nil, nil, err
		}
		bundle.Images = images

		bundles = append(bundles, bundle)
	}

	return bundles, pagination, nil
}

func GetBundleByIDProducts(bundleID string) ([]dtos.GetBundleRequest, error) {
	// ----- Fetch bundles -----
	query := `
		SELECT 
			product_id, name, description, sku, tag, price, stock_quantity,
			created_at, last_updated_at
		FROM products
		WHERE product_type = 'bundle' AND product_id = ?
		ORDER BY created_at DESC
	`

	rows, err := DB.Query(query, bundleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// ----- Scan rows -----
	var bundles []dtos.GetBundleRequest

	for rows.Next() {
		var bundle dtos.GetBundleRequest

		if err := rows.Scan(
			&bundle.ID, &bundle.Name, &bundle.Description, &bundle.SKU, &bundle.Tag,
			&bundle.Price, &bundle.StockQuantity, &bundle.CreatedAt, &bundle.LastUpdated,
		); err != nil {
			return nil, err
		}

		// Fetch bundle products
		products, err := getProductsForBundle(bundle.ID)
		if err != nil {
			return nil, err
		}
		bundle.Products = products

		// Fetch images
		images, err := fetchProductImages(bundle.ID)
		if err != nil {
			return nil, err
		}
		bundle.Images = images

		bundles = append(bundles, bundle)
	}

	return bundles, nil
}

func getProductsForBundle(bundleID string) ([]dtos.Product, error) {
	query := `
		SELECT product_id, quantity
		FROM bundle_products
		WHERE bundle_id = ?
	`
	rows, err := DB.Query(query, bundleID)
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

		// Reuse your existing reusable product function
		product, err := GetProductByID(productID)
		if err != nil {
			// Skip missing products instead of failing the entire bundle
			log.Printf("warning: failed to fetch product %s for bundle %s: %v", productID, bundleID, err)
			continue
		}

		// Override product quantity with bundle_products.quantity
		product.StockQuantity = quantity
		products = append(products, *product)
	}

	return products, nil
}

// create bundle
func CreateBundle(req dtos.Bundle, userID string) error {
	for _, product := range req.Products {
		err := IsProductThere(product.ProductID)
		if err != nil {
			return err
		}
	}
	productID, _ := shortid.Generate()

	_, err := DB.Exec(`
		INSERT INTO products (product_id, name, description, sku, price, stock_quantity, created_by_id, buying_price, search_vector, product_type)
		VALUES (?,?,?,?,?,?,?,?,?, 'bundle')
	`, productID, req.Name, req.Description, "BUNDLE-"+productID, req.Price, req.StockQuantity, userID, req.CompareAtPrice, req.Name)
	if err != nil {
		return err
	}
	//add bundle image
	err = InsertProductImage(productID, req.Image, "gallery", true)
	if err != nil {
		return err
	}
	//add products to bundle
	if len(req.Products) > 0 {
		err = AddProductsToBundle(req.Products, productID)
		if err != nil {
			return err
		}
	}
	return nil
}

// update bundle
func UpdateBundle(req dtos.Bundle, bundleID string) error {
	err := IsProductThere(bundleID)
	if err != nil {
		return err
	}
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
	if len(updates) == 0 {
		return nil // Nothing to update
	}

	query += " " + strings.Join(updates, ", ") + " WHERE product_id = ?"
	args = append(args, bundleID)

	if _, err := DB.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update bundle: %v", err)
	}
	if req.Image != "" {
		// Update bundle image
		imageQuery := `UPDATE product_images SET url = ? WHERE product_id = ?`
		_, err := DB.Exec(imageQuery, req.Image, bundleID)
		if err != nil {
			return fmt.Errorf("failed to update bundle image: %v", err)
		}
	}
	//delete existing products in bundle and add new ones
	if len(req.Products) > 0 {
		deleteQuery := `DELETE FROM bundle_products WHERE bundle_id = ?`
		_, err := DB.Exec(deleteQuery, bundleID)
		if err != nil {
			log.Printf("Delete bundle_products err::%s", err)
			return err
		}
		err = AddProductsToBundle(req.Products, bundleID)
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete bundle
func DeleteBundle(bundleID string) error {
	err := IsProductThere(bundleID)
	if err != nil {
		return err
	}
	query := `DELETE FROM products WHERE product_id = ?`
	_, err = DB.Exec(query, bundleID)
	if err != nil {
		return err
	}

	return nil
}

func AddProductsToBundle(req []dtos.BundleProducts, bundleID string) error {
	// check if bundle exists
	err := IsProductThere(bundleID)
	if err != nil {
		return fmt.Errorf("%s", nobundle)
	}

	checkQuery := `SELECT COUNT(1) FROM bundle_products WHERE bundle_id = ? AND product_id = ?`
	insertQuery := `INSERT INTO bundle_products (bundle_product_id, bundle_id, product_id, quantity) VALUES (?, ?, ?, ?)`

	for _, product := range req {
		// Check if this product already exists in the bundle
		var count int
		if err := DB.QueryRow(checkQuery, bundleID, product.ProductID).Scan(&count); err != nil {
			return err
		}

		if count > 0 {
			// Skip adding this product as it already exists in the bundle
			continue
		}

		// Generate bundle_product_id
		bundleProductID, _ := shortid.Generate()

		// Insert product into bundle
		if _, err := DB.Exec(insertQuery, bundleProductID, bundleID, product.ProductID, product.Quantity); err != nil {
			return err
		}
	}

	return nil
}

func RemoveProductsFromBundle(req dtos.AddProductsToBundle, bundleID string) error {
	err := IsProductThere(bundleID)
	if err != nil {
		return err
	}
	if len(req.ProductIDs) == 0 {
		return fmt.Errorf("no products provided")
	}

	deleteQuery := `DELETE FROM bundle_products WHERE bundle_id = ? AND product_id = ?`

	for _, productID := range req.ProductIDs {
		if _, err := DB.Exec(deleteQuery, bundleID, productID); err != nil {
			return fmt.Errorf("failed to remove product %s from bundle: %v", productID, err)
		}
	}

	return nil
}
