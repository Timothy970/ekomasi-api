package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"

	"github.com/teris-io/shortid"
)

func UpdateProductByID(db DBExecutor, productID string, input dtos.CreateProduct) (*dtos.CreateProduct, error) {
	// Validate product exists
	errr := IsProductThere(db, productID)
	if errr != nil {
		return nil, errr
	}

	// Check if SKU exists in another product (exclude current product from check)
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM products
			WHERE sku = ? AND product_id != ?
		)`, input.SKU, productID,
	).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf("failed to check SKU uniqueness: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("duplicate SKU")
	}

	// Update product (last_updated_at auto-updated by CURRENT_TIMESTAMP)
	_, err = db.Exec(`
		UPDATE products
		SET name = ?, description = ?, sku = ?, stock_quantity = ?, search_vector = ?, last_updated_at = CURRENT_TIMESTAMP, tag = ?, low_stock_quantity_warning = ?, barcode = ?
		WHERE product_id = ?`,
		input.Name, input.Description, input.SKU, input.StockQuantity, input.SearchVector, input.Tag, input.LowStockAlert, input.Barcode,
		productID,
	)

	if err != nil {
		return nil, err
	}

	// Return updated product summary
	return &dtos.CreateProduct{
		ID:            productID,
		Name:          input.Name,
		Description:   input.Description,
		SKU:           input.SKU,
		CategoryID:    input.CategoryID,
		StockQuantity: input.StockQuantity,
		SearchVector:  input.SearchVector,
		Tag:           input.Tag,
		LowStockAlert: input.LowStockAlert,
		Barcode:       input.Barcode,
	}, nil
}

// DeleteProductByID permanently removes a product with safety checks.
//
// This function performs validation to prevent deletion of products that are:
//  1. Part of active (uncollected) orders
//  2. Included in product bundles
//
// Warning: This is a hard delete operation that removes the product record.
// Consider implementing soft delete (is_deleted flag) for audit trail.
//
// Parameters:
//   - productID: string - The product_id to delete
//
// Returns:
//   - error: "product not found", "cannot delete product; it's used in active orders",
//     "cannot delete product; it's part of a bundle", database error, or nil on success
func DeleteProductByID(db DBExecutor, productID string) error {
	// Validate product exists
	err := IsProductThere(db, productID)
	if err != nil {
		return err
	}

	// Check if product is used in any uncollected orders
	query := `
		SELECT COUNT(*) 
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE oi.product_id = ? AND o.status != 'collected'
	`
	var count int
	err = db.QueryRow(query, productID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check product usage in orders: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete product; it's used in active orders")
	}

	// Check if product is part of any bundles
	var bundleCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM bundle_products WHERE product_id = ?`, productID).Scan(&bundleCount)
	if err != nil {
		return fmt.Errorf("failed to check product usage in bundles: %w", err)
	}
	if bundleCount > 0 {
		return fmt.Errorf("cannot delete product; it's part of a bundle")
	}

	// Delete product (CASCADE should handle related records)
	_, err = db.Exec("DELETE FROM products WHERE product_id = ?", productID)
	return err
}

// InsertProductImage adds an image to a product's gallery.
//
// Parameters:
//   - productID: string - The product to add the image to
//   - imageURL: string - URL or path to the image
//   - fileType: string - Image type/category (e.g., "gallery", "thumbnail")
//   - isPrimary: bool - Whether this is the primary product image
//
// Returns:
//   - error: Database error or nil on success
func InsertProductImage(db DBExecutor, productID, imageURL, fileType string, isPrimary bool) error {
	// Generate unique image ID
	imageID, _ := shortid.Generate()

	// Insert image record
	query := `INSERT INTO product_images (image_id, product_id, url, is_primary, type) VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, imageID, productID, imageURL, isPrimary, fileType)
	return err
}

// DeleteProductImage removes an image from a product's gallery.
//
// Parameters:
//   - imageID: string - The image_id to delete
//
// Returns:
//   - error: Database error or nil on success
func DeleteProductImage(db DBExecutor, imageID string) error {
	_, err := db.Exec("DELETE FROM product_images WHERE image_id = ?", imageID)
	return err
}

// GetProductImages retrieves all images for a product.
//
// Parameters:
//   - productID: string - The product_id to fetch images for
//
// Returns:
//   - []dtos.Image: Array of images containing:
//   - ImageID, URL, IsPrimary, Type
//   - error: Database error or nil on success
func GetProductImages(db DBExecutor, productID string) ([]dtos.Image, error) {
	query := `SELECT image_id, product_id, url, is_primary, type FROM product_images WHERE product_id = ?`
	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan image rows
	var images []dtos.Image
	for rows.Next() {
		var img dtos.Image
		var productID string // Temporary variable for unused product_id column
		err := rows.Scan(&img.ImageID, &productID, &img.URL, &img.IsPrimary, &img.Type)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}

// GetRelatedProducts retrieves products from the same category, excluding a specific product.
//
// This function is used to show "related products" or "you may also like" recommendations
// based on category similarity.
//
// Parameters:
//   - categoryID: string - The category_id to find related products in
//   - excludeProductID: string - Product ID to exclude from results (typically the current product)
//   - limit: int - Number of products per page
//   - page: int - Page number for pagination
//
// Returns:
//   - []dtos.Product: Array of related products with complete details
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func GetRelatedProducts(db DBExecutor, categoryID, excludeProductID string, limit, page int) ([]dtos.Product, *dtos.PaginationMeta, error) {
	// Build query to get related products from the same category, excluding the specified product
	query, args := buildRelatedProductsQuery(categoryID, excludeProductID, limit, page)

	// Build count query for total items
	countQuery, countArgs := buildRelatedProductsCountQuery(categoryID, excludeProductID)

	// Execute main query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Execute count query
	var totalItems int64
	err = DB.QueryRow(countQuery, countArgs...).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Scan and enrich products
	var relatedProducts []dtos.Product
	for rows.Next() {
		product, err := scanRelatedProduct(DB, rows)
		if err != nil {
			return nil, nil, err
		}
		relatedProducts = append(relatedProducts, product)
	}

	// Calculate pagination metadata
	pagination := calculatePagination(page, limit, totalItems)

	return relatedProducts, &pagination, nil
}

func buildRelatedProductsQuery(categoryID, excludeProductID string, limit, page int) (string, []any) {
	var args []any

	query := `
        SELECT 
            p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
            p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, c.name as category_name, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
        FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
        WHERE p.category_id = ?
		AND p.product_type = 'single'
    `
	args = append(args, categoryID)

	if excludeProductID != "" {
		query += " AND p.product_id != ?"
		args = append(args, excludeProductID)
	}

	// Order by most recent or most relevant
	query += " ORDER BY p.created_at DESC"

	// Pagination
	if limit > 0 {
		offset := (page - 1) * limit
		query += limtOffset
		args = append(args, limit, offset)
	}

	return query, args
}

func buildRelatedProductsCountQuery(categoryID, excludeProductID string) (string, []any) {
	var args []any

	query := "SELECT COUNT(*) FROM products p WHERE p.category_id = ? AND p.product_type = 'single'"
	args = append(args, categoryID)

	if excludeProductID != "" {
		query += " AND p.product_id != ?"
		args = append(args, excludeProductID)
	}

	return query, args
}

func scanRelatedProduct(db DBExecutor, rows *sql.Rows) (dtos.Product, error) {
	data, err := scanProductRow(rows)
	if err != nil {
		return dtos.Product{}, err
	}

	product := buildProduct(data)

	if err := enrichProduct(db, &product); err != nil {
		return product, err
	}

	return product, nil
}

type productRow struct {
	productID, name, desc, sku, categoryID, searchVector, categoryName, tag sql.NullString
	price                                                                   sql.NullFloat64
	stockQuantity                                                           sql.NullInt64
	createdAt, updatedAt                                                    sql.NullTime
	weightLimit, weight, discount                                           sql.NullFloat64
	discountType, dimensions, manufacturer                                  sql.NullString
	isFeatured                                                              sql.NullBool
}
