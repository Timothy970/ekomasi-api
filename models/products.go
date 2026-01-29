// Package models provides data access functions for the Adenzo backend application.
//
// This file contains comprehensive product management functionality including:
//   - Product CRUD operations (create, read, update, delete)
//   - Product listing with filtering, pagination, and search
//   - Product variants management (size, color, etc.)
//   - Product images and specifications
//   - Related products and category hierarchies
//   - Product performance analytics and reporting
//   - Wishlist integration
//   - Stock management and low stock alerts
package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

// Common error messages and query fragments for reuse
var nobundle = "bundle not found"
var limtOffset = " LIMIT ? OFFSET ?"                                           // SQL pagination clause
var lowerCname = " AND LOWER(c.name) LIKE ?"                                   // Case-insensitive category name filter
var lowerPname = " AND LOWER(p.name) LIKE ?"                                   // Case-insensitive product name filter
var lowerVariant = "(LOWER(v.variant_type) = ?)"                               // Case-insensitive variant type match
var lowerVariantTypeName = "(LOWER(v.variant_type) = ? AND LOWER(v.name) = ?)" // Variant type and name match

// GetAllProducts retrieves paginated products with optional filtering by category and name.
//
// This function supports hierarchical category filtering, case-insensitive name search,
// and enriches products with images, warranties, features, variants, and tax information.
//
// Parameters:
//   - categoryFilter: string - Optional case-insensitive category name filter (partial match)
//   - productFilter: string - Optional case-insensitive product name filter (partial match)
//   - categoryID: string - Optional category ID to filter by (includes parent/child relationships)
//   - page: int - Page number for pagination (1-indexed)
//   - limit: int - Number of products per page
//
// Returns:
//   - []dtos.Product: Array of products with complete details:
//   - Basic info: ID, Name, Description, SKU, Price, CategoryID, CategoryName, Tag
//   - Stock: StockQuantity
//   - Specifications: Weight, Dimensions, Manufacturer, WeightLimit
//   - Deal info: Discount, DiscountType (from active deals)
//   - Images: Product image gallery
//   - Warranty: Warranty details
//   - Features: Product features list
//   - ProductVariants: Available variants
//   - Tax: Tax information
//   - Timestamps: CreatedAt, LastUpdated
//   - *dtos.PaginationMeta: Pagination metadata (Page, Size, TotalItems, TotalPages, HasPrev, HasNext)
//   - error: Category not found, database error, or nil on success
func GetAllProducts(categoryFilter, productFilter, categoryID string, page, limit int) ([]dtos.Product, *dtos.PaginationMeta, error) {
	// Validate category exists if filtering by category ID
	if categoryID != "" {
		if err := CategoryExists(categoryID); err != nil {
			return nil, nil, err
		}
	}

	// Build queries with filters and pagination
	query, args := buildProductQuery(categoryFilter, productFilter, categoryID, page, limit)
	countQuery, countArgs := buildCountQuery(categoryFilter, productFilter, categoryID)

	// Execute main product query
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Get total count for pagination
	var totalItems int64
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Scan product rows and enrich with associated data
	var products []dtos.Product
	for rows.Next() {
		product, err := scanAndEnrichProduct(rows)
		if err != nil {
			return nil, nil, err
		}
		products = append(products, product)
	}

	// Calculate pagination metadata
	pagination := calculatePagination(page, limit, totalItems)
	return products, &pagination, nil
}

// buildCountQuery constructs a COUNT query for products with optional filters.
//
// This function builds a query that counts products with category hierarchy support.
// When a categoryID is provided, it uses recursive CTEs to count products in the
// specified category and all its ancestors and descendants.
//
// Parameters:
//   - categoryFilter: string - Optional category name filter (case-insensitive partial match)
//   - productFilter: string - Optional product name filter (case-insensitive partial match)
//   - categoryID: string - Optional category ID (includes parent and child categories via recursive CTE)
//
// Returns:
//   - string: SQL COUNT query
//   - []interface{}: Query parameters for prepared statement
func buildCountQuery(categoryFilter, productFilter, categoryID string) (string, []interface{}) {
	// Base query for simple filtering (no category hierarchy)
	query := `
        SELECT COUNT(DISTINCT p.product_id)
        FROM products p
        JOIN categories c ON p.category_id = c.category_id
        WHERE p.product_type = 'single'`
	var args []interface{}

	// Add category name filter
	if categoryFilter != "" {
		query += lowerCname
		args = append(args, "%"+strings.ToLower(categoryFilter)+"%")
	}

	// Add product name filter
	if productFilter != "" {
		query += lowerPname
		args = append(args, "%"+strings.ToLower(productFilter)+"%")
	}

	// Build recursive CTE for category hierarchy when filtering by category ID
	if categoryID != "" {
		query = `
        WITH RECURSIVE ancestors AS (
            SELECT category_id, parent_category_id
            FROM categories
            WHERE category_id = ?
            UNION ALL
            SELECT c.category_id, c.parent_category_id
            FROM categories c
            INNER JOIN ancestors a ON c.category_id = a.parent_category_id
        ),
        descendants AS (
            SELECT category_id, parent_category_id
            FROM categories
            WHERE category_id = ?
            UNION ALL
            SELECT c.category_id, c.parent_category_id
            FROM categories c
            INNER JOIN descendants d ON c.parent_category_id = d.category_id
        )
        SELECT COUNT(DISTINCT p.product_id)
        FROM products p
        JOIN categories c ON p.category_id = c.category_id
        WHERE p.product_type = 'single'
          AND c.category_id IN (
              SELECT category_id FROM ancestors
              UNION
              SELECT category_id FROM descendants
          )`
		args = append(args, categoryID, categoryID) // categoryID used twice for both CTEs
	}

	return query, args
}

// buildProductQuery constructs a SELECT query for products with filters and pagination.
//
// This function builds a query that fetches products with their category, deal, and
// specification information using LEFT JOINs for optional data.
//
// Parameters:
//   - categoryFilter: string - Optional category name filter (case-insensitive partial match)
//   - productFilter: string - Optional product name filter (case-insensitive partial match)
//   - categoryID: string - Optional category ID (includes parent/child via hierarchy)
//   - page: int - Page number for pagination (1-indexed)
//   - limit: int - Number of products per page (0 = no limit)
//
// Returns:
//   - string: SQL SELECT query with JOINs and filters
//   - []interface{}: Query parameters for prepared statement
func buildProductQuery(categoryFilter, productFilter, categoryID string, page, limit int) (string, []interface{}) {
	// Base query with LEFT JOINs for optional data
	query := `
        SELECT 
            c.category_id, c.name,
            p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
            p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
        FROM categories c
        JOIN products p ON c.category_id = p.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
        WHERE p.product_type = 'single'`
	var args []interface{}

	// Add category name filter (case-insensitive partial match)
	if categoryFilter != "" {
		query += lowerCname
		args = append(args, "%"+strings.ToLower(categoryFilter)+"%")
	}

	// Add product name filter (case-insensitive partial match)
	if productFilter != "" {
		query += lowerPname
		args = append(args, "%"+strings.ToLower(productFilter)+"%")
	}

	// Add category hierarchy filter (current category, parent, or grandparent)
	if categoryID != "" {
		query += " AND (c.category_id = ? OR c.parent_category_id = ? OR c.category_id IN (SELECT parent_category_id FROM categories WHERE category_id = ? AND parent_category_id IS NOT NULL))"
		args = append(args, categoryID, categoryID, categoryID) // categoryID used three times for hierarchy check
	}

	// Sort by category hierarchy: root categories first, then by parent, then by category ID
	query += " ORDER BY c.parent_category_id IS NULL DESC, c.parent_category_id, c.category_id"

	// Add pagination if limit is specified
	if limit > 0 {
		offset := (page - 1) * limit
		query += limtOffset
		args = append(args, limit, offset)
	}

	return query, args
}

// calculatePagination computes pagination metadata from page, limit, and total items.
//
// This function ensures valid pagination values (minimum 1 for page, minimum 10 for limit)
// and calculates navigation flags (HasPrev, HasNext) and total pages.
//
// Parameters:
//   - page: int - Current page number (auto-corrected to minimum 1 if ≤ 0)
//   - limit: int - Items per page (auto-corrected to minimum 10 if ≤ 0)
//   - totalItems: int64 - Total number of items across all pages
//
// Returns:
//   - dtos.PaginationMeta: Pagination metadata containing:
//   - Page: Validated page number
//   - Size: Validated page size
//   - TotalItems: Total items count
//   - TotalPages: Calculated total pages (ceiling division)
//   - HasPrev: true if page > 1
//   - HasNext: true if more pages exist
//
// scanAndEnrichProduct scans a product row and enriches it with related data.
//
// This helper function reduces cognitive complexity by extracting the scanning
// and enrichment logic from GetAllProducts.
//
// Parameters:
//   - rows: *sql.Rows - Current row from query result
//
// Returns:
//   - dtos.Product: Product with complete details
//   - error: Scan error, database error, or nil on success
func scanAndEnrichProduct(rows *sql.Rows) (dtos.Product, error) {
	var product dtos.Product
	var tag sql.NullString // Handle nullable tag field

	// Scan basic product data
	err := rows.Scan(
		&product.CategoryID, &product.CategoryName, &product.ID, &product.Name, &product.Description, &product.SKU, &product.Price, &product.CategoryID, &product.StockQuantity, &product.SearchVector, &product.CreatedAt, &product.LastUpdated, &tag, &product.Discount, &product.DiscountType, &product.Weight, &product.Dimensions, &product.Manufacturer, &product.WeightLimit,
	)
	if err != nil {
		return dtos.Product{}, err
	}

	// Handle nullable tag
	if tag.Valid {
		product.Tag = &tag.String
	}

	// Enrich product with associated data
	product.Images, err = fetchProductImages(product.ID)
	if err != nil {
		return dtos.Product{}, err
	}

	warranty, err := FetchProductWarranties(product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Warranty = &warranty

	features, err := fetchProductFeatures(product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Features = features

	variants, err := getProductVariants(product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.ProductVariants = variants

	tax, err := fetchProductTax(product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Tax = &tax

	return product, nil
}

func calculatePagination(page, limit int, totalItems int64) dtos.PaginationMeta {
	// Ensure minimum valid page size
	if limit <= 0 {
		limit = 10
	}

	// Ensure minimum valid page number
	if page <= 0 {
		page = 1
	}

	// Calculate total pages using ceiling division
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))

	return dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: int(totalItems),
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
}

// getProductVariants fetches all variants for a given product.
//
// This function retrieves product variants (e.g., sizes, colors) with their pricing,
// stock levels, and visual attributes (hex color codes).
//
// Parameters:
//   - productID: string - The product_id to fetch variants for
//
// Returns:
//   - []dtos.ProductVariants: Array of variants containing:
//   - VariantID: Unique variant identifier
//   - VariantType: Type of variant (e.g., "size", "color")
//   - Name: Variant name (e.g., "Large", "Red")
//   - HexCode: Color hex code (for color variants, nullable)
//   - AdditionalPrice: Price adjustment for this variant
//   - StockQuantity: Stock level for this specific variant
//   - error: Database error or nil on success
func getProductVariants(productID string) ([]dtos.ProductVariants, error) {
	// Join product_variants with variants table to get complete variant details
	query := `
		SELECT 
			v.variant_id,
			v.variant_type,
			v.name,
			v.hex_code,
			pv.additional_price,
			pv.stock_quantity
		FROM product_variants pv
		INNER JOIN variants v ON pv.variant_id = v.variant_id
		WHERE pv.product_id = ?
	`

	rows, err := DB.Query(query, productID)
	if err != nil {
		return nil, fmt.Errorf("querying product variants: %w", err)
	}
	defer rows.Close()

	// Scan variant rows
	var variants []dtos.ProductVariants
	for rows.Next() {
		var pv dtos.ProductVariants
		if err := rows.Scan(
			&pv.VariantID,
			&pv.VariantType,
			&pv.Name,
			&pv.HexCode,
			&pv.AdditionalPrice,
			&pv.StockQuantity,
		); err != nil {
			return nil, fmt.Errorf("scanning product variant: %w", err)
		}
		variants = append(variants, pv)
	}

	// Check for row iteration errors
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating product variants: %w", err)
	}

	return variants, nil
}

// GetProductByID retrieves complete details for a single product by its ID.
//
// This function fetches a product with all associated data including images, warranties,
// features, variants, tax information, and JSON details.
//
// Parameters:
//   - productID: string - The unique product_id to retrieve
//
// Returns:
//   - *dtos.Product: Complete product details including:
//   - Basic info: ID, Name, Description, SKU, Price, CategoryID, CategoryName, Tag
//   - Stock: StockQuantity
//   - Specifications: Weight, Dimensions, Manufacturer, WeightLimit
//   - Deal info: Discount, DiscountType (from active deals)
//   - Details: Array of detail strings (unmarshaled from JSON)
//   - Images: Product image gallery
//   - Warranty: Warranty details
//   - Features: Product features list
//   - ProductVariants: Available variants
//   - Tax: Tax information
//   - Timestamps: CreatedAt, LastUpdated
//   - error: "product not found" if ID doesn't exist, database error, or nil on success
func GetProductByID(productID string) (*dtos.Product, error) {
	// Validate product exists
	err := IsProductThere(productID)
	if err != nil {
		return nil, err
	}

	// Query product with LEFT JOINs for optional data
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, c.name, p.tag, p.details, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE p.product_id = ?
	`

	var (
		p           dtos.Product
		detailsData []byte // JSON blob for product details
	)

	// Scan basic product data
	err = DB.QueryRow(query, productID).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&p.CategoryName, &p.Tag, &detailsData, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON details if present
	if len(detailsData) > 0 {
		err := json.Unmarshal(detailsData, &p.Details)
		if err != nil {
			return nil, err
		}
	} else {
		p.Details = []string{} // Empty array for no details
	}

	// Fetch associated product images
	images, err := fetchProductImages(p.ID)
	if err != nil {
		return nil, err
	}
	p.Images = images

	// Fetch product warranties
	warranties, err := FetchProductWarranties(p.ID)
	if err != nil {
		return nil, err
	}
	p.Warranty = &warranties
	// Fetch product features
	features, err := fetchProductFeatures(p.ID)
	if err != nil {
		return nil, err
	}
	p.Features = features

	// Fetch product variants (sizes, colors, etc.)
	variants, err := getProductVariants(p.ID)
	if err != nil {
		return nil, err
	}
	p.ProductVariants = variants

	// Fetch tax information
	tax, err := fetchProductTax(p.ID)
	if err != nil {
		return nil, err
	}
	p.Tax = &tax

	return &p, nil
}

// IsSkuThere validates that a SKU does not already exist in the database.
//
// This function is used before product creation to ensure SKU uniqueness.
//
// Parameters:
//   - sku: string - The SKU (Stock Keeping Unit) to check
//
// Returns:
//   - error: "duplicate SKU found: <sku>" if SKU exists, database error, or nil if unique
func IsSkuThere(sku string) error {
	// Check if SKU exists in products table
	skuExists, err := RecordExists("products", "sku = ?", sku)
	if err != nil {
		return err
	}
	if skuExists {
		return fmt.Errorf("duplicate SKU found: %s", sku)
	}
	return nil
}

// IsCategoryParent validates that a category is not a parent category.
//
// This function prevents adding products to parent categories, enforcing
// that products must be added to leaf (subcategory) nodes only.
//
// Parameters:
//   - categoryID: string - The category_id to validate
//
// Returns:
//   - error: "cannot add product to a parent category..." if category is a parent,
//     database error, or nil if category is a valid subcategory
func IsCategoryParent(categoryID string) error {
	var parentID *string

	// Check if category has a parent (null parent_category_id = root/parent category)
	err := DB.QueryRow("SELECT parent_category_id FROM categories WHERE category_id = ?", categoryID).Scan(&parentID)
	if err != nil {
		return err
	}

	// Prevent adding product to parent category (parentID is null)
	if parentID == nil {
		return fmt.Errorf("cannot add product to a parent category with ID %s, choose a subcategory instead", categoryID)
	}
	return nil

}

// AddNewProduct creates a new product with validation and default values.
//
// This function validates SKU uniqueness, category existence, and ensures products
// are only added to subcategories (not parent categories). It handles optional fields
// and marshals product details to JSON.
//
// Parameters:
//   - input: dtos.CreateProduct - Product details containing:
//   - Name, Description, SKU, Price (required)
//   - CategoryID: Must be a valid subcategory (not parent)
//   - StockQuantity, SearchVector, Tag
//   - LowStockAlert: Low stock warning threshold
//   - SellWhenOOS: Pointer to bool (allow selling when out of stock)
//   - ShowStock: Pointer to bool (show stock quantity to customers)
//   - BuyingPrice: Cost price
//   - Details: Array of strings (marshaled to JSON)
//   - userID: string - ID of the user creating the product (for audit trail)
//
// Returns:
//   - *dtos.CreateProduct: Created product with generated ID
//   - error: "duplicate SKU", category validation error, database error, or nil on success
func AddNewProduct(input dtos.CreateProduct, userID string) (*dtos.CreateProduct, error) {
	// Validate SKU uniqueness
	skuExists, err := RecordExists("products", "sku = ?", input.SKU)
	if err != nil {
		return nil, err
	}
	if skuExists {
		return nil, fmt.Errorf("duplicate SKU")
	}

	// Validate category exists
	err = CategoryExists(input.CategoryID)
	if err != nil {
		return nil, err
	}

	// Ensure product is added to subcategory, not parent category
	err = IsCategoryParent(input.CategoryID)
	if err != nil {
		return nil, err
	}

	// Generate unique product ID
	productID, _ := shortid.Generate()

	// Set default values for optional boolean fields
	sellWhenOOs := false
	showStock := false
	if input.SellWhenOOS != nil {
		sellWhenOOs = *input.SellWhenOOS
	}
	if input.ShowStock != nil {
		showStock = *input.ShowStock
	}

	// Marshal product details to JSON
	var detailsJSON []byte
	if input.Details != nil {
		detailsJSON, err = json.Marshal(input.Details)
		if err != nil {
			return nil, err
		}
	} else {
		detailsJSON = nil
	}

	// Insert product record
	_, err = DB.Exec(`
		INSERT INTO products (product_id, name, description, sku, price, category_id, stock_quantity, search_vector, tag, low_stock_quantity_warning, sell_when_out_of_stock, show_stock_quantity, created_by_id, buying_price, details)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		productID, input.Name, input.Description, input.SKU, input.Price, input.CategoryID, input.StockQuantity, input.SearchVector, input.Tag, input.LowStockAlert, sellWhenOOs, showStock, userID, input.BuyingPrice, detailsJSON,
	)
	if err != nil {
		return nil, err
	}

	// Return created product summary
	return &dtos.CreateProduct{
		ID:            productID,
		Name:          input.Name,
		Description:   input.Description,
		SKU:           input.SKU,
		Price:         input.Price,
		CategoryID:    input.CategoryID,
		StockQuantity: input.StockQuantity,
		SearchVector:  input.SearchVector,
	}, nil
}

// UpdateProductByID updates an existing product with SKU uniqueness validation.
//
// This function updates product details while ensuring the SKU remains unique
// (excluding the current product). It auto-updates the last_updated_at timestamp.
//
// Parameters:
//   - productID: string - The product_id to update
//   - input: dtos.CreateProduct - Updated product details:
//   - Name, Description, SKU, Price
//   - StockQuantity, SearchVector, Tag
//   - LowStockAlert: Low stock warning threshold
//   - SellWhenOOS: Allow selling when out of stock
//   - ShowStock: Show stock quantity to customers
//
// Returns:
//   - *dtos.CreateProduct: Updated product summary
//   - error: "product not found", "duplicate SKU", database error, or nil on success
func UpdateProductByID(productID string, input dtos.CreateProduct) (*dtos.CreateProduct, error) {
	// Validate product exists
	errr := IsProductThere(productID)
	if errr != nil {
		return nil, errr
	}

	// Check if SKU exists in another product (exclude current product from check)
	var exists bool
	err := DB.QueryRow(`
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
	_, err = DB.Exec(`
		UPDATE products
		SET name = ?, description = ?, sku = ?, price = ?, stock_quantity = ?, search_vector = ?, last_updated_at = CURRENT_TIMESTAMP, tag = ?, low_stock_quantity_warning = ?, sell_when_out_of_stock = ?, show_stock_quantity = ?
		WHERE product_id = ?`,
		input.Name, input.Description, input.SKU, input.Price, input.StockQuantity, input.SearchVector, input.Tag, input.LowStockAlert, input.SellWhenOOS, input.ShowStock,
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
		Price:         input.Price,
		CategoryID:    input.CategoryID,
		StockQuantity: input.StockQuantity,
		SearchVector:  input.SearchVector,
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
func DeleteProductByID(productID string) error {
	// Validate product exists
	err := IsProductThere(productID)
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
	err = DB.QueryRow(query, productID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check product usage in orders: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete product; it's used in active orders")
	}

	// Check if product is part of any bundles
	var bundleCount int
	err = DB.QueryRow(`SELECT COUNT(*) FROM bundle_products WHERE product_id = ?`, productID).Scan(&bundleCount)
	if err != nil {
		return fmt.Errorf("failed to check product usage in bundles: %w", err)
	}
	if bundleCount > 0 {
		return fmt.Errorf("cannot delete product; it's part of a bundle")
	}

	// Delete product (CASCADE should handle related records)
	_, err = DB.Exec("DELETE FROM products WHERE product_id = ?", productID)
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
func InsertProductImage(productID, imageURL, fileType string, isPrimary bool) error {
	// Generate unique image ID
	imageID, _ := shortid.Generate()

	// Insert image record
	query := `INSERT INTO product_images (image_id, product_id, url, is_primary, type) VALUES (?, ?, ?, ?, ?)`
	_, err := DB.Exec(query, imageID, productID, imageURL, isPrimary, fileType)
	return err
}

// DeleteProductImage removes an image from a product's gallery.
//
// Parameters:
//   - imageID: string - The image_id to delete
//
// Returns:
//   - error: Database error or nil on success
func DeleteProductImage(imageID string) error {
	_, err := DB.Exec("DELETE FROM product_images WHERE image_id = ?", imageID)
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
func GetProductImages(productID string) ([]dtos.Image, error) {
	query := `SELECT image_id, product_id, url, is_primary, type FROM product_images WHERE product_id = ?`
	rows, err := DB.Query(query, productID)
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
func GetRelatedProducts(categoryID, excludeProductID string, limit, page int) ([]dtos.Product, *dtos.PaginationMeta, error) {
	// Build query to get related products from the same category, excluding the specified product
	query, args := buildRelatedProductsQuery(categoryID, excludeProductID, limit, page)

	// Build count query for total items
	countQuery, countArgs := buildRelatedProductsCountQuery(categoryID, excludeProductID)

	// Execute main query
	rows, err := DB.Query(query, args...)
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
		product, err := scanRelatedProduct(rows)
		if err != nil {
			return nil, nil, err
		}
		relatedProducts = append(relatedProducts, product)
	}

	// Calculate pagination metadata
	pagination := calculatePagination(page, limit, totalItems)

	return relatedProducts, &pagination, nil
}

func buildRelatedProductsQuery(categoryID, excludeProductID string, limit, page int) (string, []interface{}) {
	var args []interface{}

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

func buildRelatedProductsCountQuery(categoryID, excludeProductID string) (string, []interface{}) {
	var args []interface{}

	query := "SELECT COUNT(*) FROM products p WHERE p.category_id = ? AND p.product_type = 'single'"
	args = append(args, categoryID)

	if excludeProductID != "" {
		query += " AND p.product_id != ?"
		args = append(args, excludeProductID)
	}

	return query, args
}

func scanRelatedProduct(rows *sql.Rows) (dtos.Product, error) {
	data, err := scanProductRow(rows)
	if err != nil {
		return dtos.Product{}, err
	}

	product := buildProduct(data)

	if err := enrichProduct(&product); err != nil {
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
}

func enrichProduct(product *dtos.Product) error {
	images, err := fetchProductImages(product.ID)
	if err != nil {
		return err
	}
	product.Images = images

	warranties, err := FetchProductWarranties(product.ID)
	if err != nil {
		return err
	}
	product.Warranty = &warranties

	tax, err := fetchProductTax(product.ID)
	if err != nil {
		return err
	}
	product.Tax = &tax

	features, err := fetchProductFeatures(product.ID)
	if err != nil {
		return err
	}
	product.Features = features

	variants, err := getProductVariants(product.ID)
	if err != nil {
		return err
	}
	product.ProductVariants = variants

	return nil
}
func ptr[T any](v T) *T {
	return &v
}

// products reports
func GetProductPerformance(start, end time.Time, categoryID string) ([]map[string]interface{}, error) {
	//check if category exists
	err := isCategoryThere(categoryID)
	if err != nil {
		return nil, err
	}
	rows, err := DB.Query(`
		SELECT 
			p.product_id,
			p.name,
			c.name AS category,
			COALESCE(SUM(oi.quantity), 0) AS sales_volume,
			COALESCE(SUM(r.quantity), 0) AS total_returns,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.buying_price)) / NULLIF(SUM(oi.quantity * oi.unit_price), 0), 0) AS profit_margin,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.buying_price)), 0) AS net_profit
		FROM products p
		JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN order_items oi ON p.product_id = oi.product_id
		LEFT JOIN return_products r ON p.product_id = r.product_id
		JOIN orders o ON oi.order_id = o.order_id
		WHERE o.created_at BETWEEN ? AND ?
		  AND c.category_id = ?
		GROUP BY p.product_id, p.name, c.name
		ORDER BY net_profit DESC
	`, start, end, categoryID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summary []map[string]interface{}
	for rows.Next() {
		var productID, name, category string
		var salesVolume, totalReturns int
		var profitMargin, netProfit float64

		if err := rows.Scan(&productID, &name, &category, &salesVolume, &totalReturns, &profitMargin, &netProfit); err != nil {
			return nil, err
		}

		summary = append(summary, map[string]interface{}{
			"product_id":    productID,
			"name":          name,
			"category":      category,
			"sales_volume":  salesVolume,
			"total_returns": totalReturns,
			"profit_margin": profitMargin,
			"net_profit":    netProfit,
		})
	}
	return summary, nil
}

func GetSingleProductPerformance(productID string, start, end time.Time) (map[string]interface{}, error) {

	var name, category string
	var salesVolume, totalReturns int
	var returnRate, profitMargin, netProfit float64

	err := DB.QueryRow(`
		SELECT 
			p.product_id,
			p.name,
			c.name AS category,
			COALESCE(SUM(oi.quantity), 0) AS sales_volume,
			COALESCE(SUM(r.quantity), 0) AS total_returns,
			(COALESCE(SUM(r.quantity), 0) / NULLIF(SUM(oi.quantity), 0)) * 100 AS return_rate,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.buying_price)) / NULLIF(SUM(oi.quantity * oi.unit_price), 0), 0) AS profit_margin,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.buying_price)), 0) AS net_profit
		FROM products p
		JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN order_items oi ON p.product_id = oi.product_id
		LEFT JOIN return_products r ON p.product_id = r.product_id
		JOIN orders o ON oi.order_id = o.order_id
		WHERE o.created_at BETWEEN ? AND ?
		  AND p.product_id = ?
		GROUP BY p.product_id, p.name, c.name
	`, start, end, productID).Scan(&productID, &name, &category, &salesVolume, &totalReturns, &returnRate, &profitMargin, &netProfit)

	if err != nil {
		return nil, err
	}

	response := map[string]interface{}{
		"product_id":    productID,
		"name":          name,
		"category":      category,
		"sales_volume":  salesVolume,
		"total_returns": totalReturns,
		"return_rate":   returnRate,
		"profit_margin": profitMargin,
		"net_profit":    netProfit,
		"period": map[string]time.Time{
			"start": start,
			"end":   end,
		},
	}
	return response, nil
}

// FetchSubcategoryProducts retrieves products under a given subcategory with pagination
func FetchSubcategoryProducts(subcategoryID string, page, size int) (*dtos.SubcategoryProducts, *dtos.PaginationMeta, error) {
	valid, err := IsValidSubcategory(subcategoryID)
	if err != nil {
		return nil, nil, err
	}
	if !valid {
		return nil, nil, fmt.Errorf("cannot use a main category ID, must be a subcategory")
	}

	// Fetch subcategory details
	sub, err := fetchSubcategoryDetails(subcategoryID)
	if err != nil {
		return nil, nil, err
	}

	// Count total products for pagination
	totalItems, err := countSubcategoryProducts(subcategoryID)
	if err != nil {
		return nil, nil, err
	}

	// Fetch products for the subcategory with pagination
	products, err := fetchSubcategoryProductList(subcategoryID, page, size)
	if err != nil {
		return nil, nil, err
	}
	sub.Products = products

	// Build pagination metadata
	meta := buildSubcategoryPaginationMeta(page, size, totalItems)

	return sub, meta, nil
}

// fetchSubcategoryDetails retrieves subcategory information with parent category details
func fetchSubcategoryDetails(subcategoryID string) (*dtos.SubcategoryProducts, error) {
	var sub dtos.SubcategoryProducts
	err := DB.QueryRow(`
		SELECT 
			c.category_id, c.name, c.image, c.parent_category_id,
			p.name as parent_name, p.image as parent_image
		FROM categories c
		LEFT JOIN categories p ON c.parent_category_id = p.category_id
		WHERE c.category_id = ? AND c.parent_category_id IS NOT NULL`, subcategoryID).
		Scan(&sub.ID, &sub.Name, &sub.ImageURL, &sub.ParentID, &sub.ParentCategoryName, &sub.ParentCategoryImageURL)

	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// countSubcategoryProducts counts total products in a subcategory
func countSubcategoryProducts(subcategoryID string) (int, error) {
	var totalItems int
	err := DB.QueryRow(`SELECT COUNT(*) FROM products WHERE category_id = ? AND product_type = 'single'`, subcategoryID).Scan(&totalItems)
	return totalItems, err
}

// fetchSubcategoryProductList retrieves paginated products for a subcategory
func fetchSubcategoryProductList(subcategoryID string, page, size int) ([]dtos.Product, error) {
	offset := (page - 1) * size

	rows, err := DB.Query(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id, 
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, p.tag, p.details, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM products p
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE category_id = ? AND product_type = 'single'
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, subcategoryID, size, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.Product
	for rows.Next() {
		product, err := scanSubcategoryProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

// scanSubcategoryProduct scans a product row and enriches it with related data
func scanSubcategoryProduct(rows *sql.Rows) (dtos.Product, error) {
	var (
		p           dtos.Product
		detailsData []byte
	)

	if err := rows.Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price,
		&p.CategoryID, &p.StockQuantity, &p.SearchVector,
		&p.CreatedAt, &p.LastUpdated, &p.Tag, &detailsData, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
	); err != nil {
		return dtos.Product{}, err
	}

	// Unmarshal details
	if len(detailsData) > 0 {
		if err := json.Unmarshal(detailsData, &p.Details); err != nil {
			return dtos.Product{}, err
		}
	} else {
		p.Details = []string{}
	}

	// Enrich with related data
	if err := enrichSubcategoryProduct(&p); err != nil {
		return dtos.Product{}, err
	}

	return p, nil
}

// enrichSubcategoryProduct fetches and attaches related data to a product
func enrichSubcategoryProduct(p *dtos.Product) error {
	images, err := fetchProductImages(p.ID)
	if err != nil {
		return err
	}
	p.Images = images

	warranties, err := FetchProductWarranties(p.ID)
	if err != nil {
		return err
	}
	p.Warranty = &warranties

	features, err := fetchProductFeatures(p.ID)
	if err != nil {
		return err
	}
	p.Features = features

	variants, err := getProductVariants(p.ID)
	if err != nil {
		return err
	}
	p.ProductVariants = variants

	tax, err := fetchProductTax(p.ID)
	if err != nil {
		return err
	}
	p.Tax = &tax

	return nil
}

// buildSubcategoryPaginationMeta creates pagination metadata for subcategory products
func buildSubcategoryPaginationMeta(page, size, totalItems int) *dtos.PaginationMeta {
	totalPages := int(math.Ceil(float64(totalItems) / float64(size)))
	return &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
}

// IsValidSubcategory checks whether a category is a valid subcategory (not a main category).
func IsValidSubcategory(categoryID string) (bool, error) {
	var parentID sql.NullString
	err := DB.QueryRow(`SELECT parent_category_id FROM categories WHERE category_id = ?`, categoryID).Scan(&parentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("category not found")
		}
		return false, err
	}

	// If parent_category_id is NULL → it's a main category → invalid
	if !parentID.Valid {
		return false, nil
	}
	return true, nil
}
func IsValidCategory(categoryID string) (bool, error) {
	var parentID sql.NullString
	err := DB.QueryRow(`
		SELECT parent_category_id 
		FROM categories 
		WHERE category_id = ?`, categoryID).
		Scan(&parentID)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("category not found")
		}
		return false, err
	}

	// Only valid if it's a parent (i.e., has no parent itself)
	if !parentID.Valid {
		return true, nil // main category
	}
	return false, nil // subcategory, not valid
}

// GetCategoriesWithSubcategoriesAndProducts retrieves hierarchical category structure with products.
//
// This function returns main categories (optionally filtered), their subcategories, and
// products within each subcategory. It builds a complete category tree with paginated products.
//
// Parameters:
//   - searchParams: dtos.SearchParams - Search/filter parameters containing:
//   - Page, Limit: Pagination for products
//   - Q: Product name search query (case-insensitive)
//   - CategoryName: Category name filter (case-insensitive)
//   - Additional filters: Price range, variants, etc.
//   - filterCategoryID: string - Optional main category ID to filter (must be parent category, not subcategory)
//
// Returns:
//   - []dtos.CategoryResponse: Array of categories containing:
//   - ID, Name, ParentID, ImageURL, Description
//   - Subcategories: Array of subcategories
//   - Products: Paginated products under subcategories
//   - *dtos.PaginationMeta: Pagination metadata for products
//   - error: "cannot use a subcategory ID, must be a main category", database error, or nil on success
func GetCategoriesWithSubcategoriesAndProducts(
	searchParams dtos.SearchParams,
	filterCategoryID string,
) ([]dtos.CategoryResponse, *dtos.PaginationMeta, error) {

	// Validate category filter (must be main category, not subcategory)
	if filterCategoryID != "" {
		valid, err := IsValidCategory(filterCategoryID)
		if err != nil {
			return nil, nil, err
		}
		if !valid {
			return nil, nil, fmt.Errorf("cannot use a subcategory ID, must be a main category")
		}
	}

	// Fetch main categories with optional filter
	categories, err := getMainCategories(filterCategoryID, searchParams)
	if err != nil {
		return nil, nil, err
	}

	var paginationMeta *dtos.PaginationMeta

	// For each category, attach subcategories and products
	for i := range categories {
		// Get subcategories for this main category
		subs, subIDs, err := getSubcategoriesWithParentID(categories[i].ID)
		if err != nil {
			return nil, nil, err
		}
		categories[i].Subcategories = subs

		// Get products for all subcategories
		products, meta, err := getProductsForSubcategories(subIDs, searchParams.Page, searchParams.Limit, searchParams)
		if err != nil {
			return nil, nil, err
		}
		categories[i].Products = products
		paginationMeta = meta
	}

	return categories, paginationMeta, nil
}

// getMainCategories fetches parent categories that have products in their subcategories.
//
// This internal function retrieves top-level categories with optional filters.
// Only returns categories that have at least one product in their subcategories.
//
// Parameters:
//   - filterCategoryID: string - Optional category ID filter
//   - params: dtos.SearchParams - Search parameters (Q, CategoryName)
//
// Returns:
//   - []dtos.CategoryResponse: Main categories
//   - error: Database error or nil on success
func getMainCategories(filterCategoryID string, params dtos.SearchParams) ([]dtos.CategoryResponse, error) {
	var (
		// Query main categories that have products in subcategories
		query = `
            SELECT c.category_id, c.name, c.parent_category_id, c.image, c.description
            FROM categories c
            WHERE c.parent_category_id IS NULL
            AND EXISTS (
                SELECT 1 FROM categories sub
                LEFT JOIN products p ON sub.category_id = p.category_id
                WHERE sub.parent_category_id = c.category_id
                AND p.product_id IS NOT NULL
				AND p.product_type = 'single'
            )`
		args []interface{}
	)

	// Add category ID filter
	if filterCategoryID != "" {
		query += " AND c.category_id = ?"
		args = append(args, filterCategoryID)
	}

	// Add product name filter
	if params.Q != "" {
		query += lowerPname
		args = append(args, "%"+strings.ToLower(params.Q)+"%")
	}

	// Add category name filter
	if params.CategoryName != "" {
		query += lowerCname
		args = append(args, "%"+strings.ToLower(params.CategoryName)+"%")
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan category rows
	var categories []dtos.CategoryResponse
	for rows.Next() {
		var cat dtos.CategoryResponse
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.ParentID, &cat.ImageURL, &cat.Description); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}

	return categories, nil
}

// getSubcategoriesWithParentID fetches subcategories for a parent category.
//
// This internal function retrieves child categories that have products.
//
// Parameters:
//   - parentID: string - The parent category_id
//
// Returns:
//   - []dtos.SubcategoryResponse: Subcategories
//   - []string: Array of subcategory IDs (for product queries)
//   - error: Database error or nil on success
func getSubcategoriesWithParentID(parentID string) ([]dtos.SubcategoryResponse, []string, error) {
	// Query subcategories that have products
	rows, err := DB.Query(`
        SELECT c.category_id, c.name, c.parent_category_id, c.image, c.description
        FROM categories c
        LEFT JOIN products p ON c.category_id = p.category_id
        WHERE c.parent_category_id = ?
        AND p.product_id IS NOT NULL
		AND p.product_type = 'single'
        GROUP BY c.category_id, c.name, c.parent_category_id, c.image, c.description`, parentID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var (
		subs []dtos.SubcategoryResponse
		ids  []string
	)

	for rows.Next() {
		var sub dtos.SubcategoryResponse
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.ParentID, &sub.ImageURL, &sub.Description); err != nil {
			return nil, nil, err
		}
		subs = append(subs, sub)
		ids = append(ids, sub.ID)
	}

	return subs, ids, nil
}

//
// --- PRODUCTS FOR SUBCATEGORIES ---
//

// getProductsForSubcategories fetches paginated products from multiple subcategories with filtering.
//
// This function retrieves products belonging to any of the specified subcategory IDs,
// applies optional filters (price, variants, etc.), and returns paginated results with metadata.
//
// Parameters:
//   - subIDs: []string - Array of subcategory IDs to fetch products from
//   - page: int - Page number for pagination (1-indexed)
//   - size: int - Number of products per page
//   - params: dtos.SearchParams - Search/filter parameters containing:
//   - ProductName: Product name filter
//   - SKU: SKU filter
//   - Tag: Tag filter
//   - MinPrice, MaxPrice: Price range filter
//   - Variants: Array of variant filters (type and value)
//   - SortBy: Sort order specification
//
// Returns:
//   - []dtos.CategoryProduct: Array of products with:
//   - Basic info: ID, Name, Description, SKU, Price
//   - CategoryID: Parent category ID
//   - SubcategoryID: Direct category ID
//   - Stock, timestamps, tax, variants, images, etc.
//   - *dtos.PaginationMeta: Pagination metadata (Page, Size, TotalItems, TotalPages, HasPrev, HasNext)
//   - error: Database error or nil on success
func getProductsForSubcategories(subIDs []string, page, size int, params dtos.SearchParams) ([]dtos.CategoryProduct, *dtos.PaginationMeta, error) {
	// Early return if no subcategories specified
	if len(subIDs) == 0 {
		return nil, nil, nil
	}

	// Build query components
	whereClause, args := buildSubcategoryWhereClause(subIDs, params)

	// Get total count for pagination
	totalItems, err := countSubcategoryProduct(whereClause, args)
	if err != nil {
		return nil, nil, err
	}

	// Fetch products with pagination
	products, err := fetchSubcategoryProducts(whereClause, args, params, page, size)
	if err != nil {
		return nil, nil, err
	}

	// Build pagination metadata
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: int(math.Ceil(float64(totalItems) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < totalItems,
	}

	return products, meta, nil
}

// buildSubcategoryWhereClause constructs the WHERE clause for subcategory product queries
func buildSubcategoryWhereClause(subIDs []string, params dtos.SearchParams) (string, []interface{}) {
	placeholders := strings.Repeat(",?", len(subIDs)-1)
	baseWhere := fmt.Sprintf(`WHERE p.category_id IN (?%s)
		AND p.product_type = 'single'`, placeholders)

	args := make([]interface{}, len(subIDs))
	for i, id := range subIDs {
		args[i] = id
	}

	filterQuery, filterArgs := buildProductFilters(params)
	args = append(args, filterArgs...)
	return baseWhere + filterQuery, args
}

// countSubcategoryProducts returns the total count of products matching the criteria
func countSubcategoryProduct(whereClause string, args []interface{}) (int, error) {
	joinClause := `FROM products p
		JOIN categories c ON p.category_id = c.category_id`
	countQuery := "SELECT COUNT(*) " + joinClause + " " + whereClause

	var totalItems int
	err := DB.QueryRow(countQuery, args...).Scan(&totalItems)
	return totalItems, err
}

// fetchSubcategoryProducts retrieves paginated products with all related data
func fetchSubcategoryProducts(whereClause string, args []interface{}, params dtos.SearchParams, page, size int) ([]dtos.CategoryProduct, error) {
	offset := (page - 1) * size
	sortClause := getSortClause(params.SortBy)

	dataQuery := fmt.Sprintf(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price,
			p.category_id, c.parent_category_id, p.stock_quantity,
			p.search_vector, p.created_at, p.last_updated_at, p.tag, p.details, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?`, whereClause, sortClause)

	args = append(args, size, offset)

	rows, err := DB.Query(dataQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.CategoryProduct
	for rows.Next() {
		product, err := scanCategoryProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

// scanCategoryProduct scans and enriches a single category product row
func scanCategoryProduct(rows *sql.Rows) (dtos.CategoryProduct, error) {
	var pr dtos.CategoryProduct
	var subcategoryID, parentCategoryID string
	var detailsData []byte

	if err := rows.Scan(
		&pr.ID, &pr.Name, &pr.Description, &pr.SKU, &pr.Price,
		&subcategoryID, &parentCategoryID, &pr.StockQuantity,
		&pr.SearchVector, &pr.CreatedAt, &pr.LastUpdated, &pr.Tag, &detailsData, &pr.Discount, &pr.DiscountType, &pr.Weight, &pr.Dimensions, &pr.Manufacturer, &pr.WeightLimit,
	); err != nil {
		return pr, err
	}

	pr.CategoryID = parentCategoryID
	pr.SubcategoryID = subcategoryID

	if len(detailsData) > 0 {
		if err := json.Unmarshal(detailsData, &pr.Details); err != nil {
			return pr, err
		}
	} else {
		pr.Details = []string{}
	}

	return pr, enrichCategoryProduct(&pr)
}

// enrichCategoryProduct fetches and attaches all related data to a category product
func enrichCategoryProduct(pr *dtos.CategoryProduct) error {
	var err error

	if pr.Images, err = fetchProductImages(pr.ID); err != nil {
		return err
	}

	warranty, err := FetchProductWarranties(pr.ID)
	if err != nil {
		return err
	}
	pr.Warranty = &warranty

	if pr.Features, err = fetchProductFeatures(pr.ID); err != nil {
		return err
	}

	if pr.ProductVariants, err = getProductVariants(pr.ID); err != nil {
		return err
	}

	tax, err := fetchProductTax(pr.ID)
	if err != nil {
		return err
	}
	pr.Tax = &tax

	return nil
}

// buildProductFilters constructs a WHERE clause fragment and args for product filtering.
//
// This function builds SQL conditions based on search parameters including name, SKU, tag,
// price range, and variant filters. It supports complex variant filtering with type-only
// or type+value combinations.
//
// Parameters:
//   - params: dtos.SearchParams - Search parameters containing:
//   - ProductName: Case-insensitive product name filter (partial match)
//   - SKU: Exact SKU match (case-insensitive)
//   - Tag: Exact tag match (case-insensitive)
//   - MinPrice, MaxPrice: Price range filter (inclusive)
//   - Variants: Array of variant filters with Type and Value fields
//   - Value = "all": Matches any variant of the specified type
//   - Value = specific: Matches exact type and value combination
//
// Returns:
//   - string: SQL WHERE clause fragment (starts with " AND " if conditions exist, empty otherwise)
//   - []interface{}: Query parameters for prepared statement
func buildProductFilters(params dtos.SearchParams) (string, []interface{}) {
	var (
		conditions []string      // Array of SQL condition strings
		args       []interface{} // Array of query parameters
	)

	// Filter by product name (case-insensitive partial match)
	if params.ProductName != "" {
		conditions = append(conditions, "LOWER(p.name) LIKE ?")
		args = append(args, "%"+strings.ToLower(params.ProductName)+"%")
	}

	// Filter by SKU (case-insensitive exact match)
	if params.SKU != "" {
		conditions = append(conditions, "LOWER(p.sku) = ?")
		args = append(args, strings.ToLower(params.SKU))
	}

	// Filter by tag (case-insensitive exact match)
	if params.Tag != "" {
		conditions = append(conditions, "LOWER(p.tag) = ?")
		args = append(args, strings.ToLower(params.Tag))
	}

	// Filter by price range (inclusive, only if both min and max are valid)
	if params.MinPrice >= 0 && params.MaxPrice > 0 {
		conditions = append(conditions, "p.price BETWEEN ? AND ?")
		args = append(args, params.MinPrice, params.MaxPrice)
	}

	// --- Variants filtering (supports multiple variant filters with OR logic) ---
	if len(params.Variants) > 0 {
		var variantConds []string // Individual variant conditions

		// Build condition for each variant filter
		for _, v := range params.Variants {
			if strings.ToLower(v.Value) == "all" {
				// Match any variant of the specified type (e.g., any size)
				// lowerVariant should be: "(LOWER(v.variant_type) = ?)"
				variantConds = append(variantConds, lowerVariant)
				args = append(args, strings.ToLower(v.Type))
			} else {
				// Match specific variant type and value (e.g., size=large)
				// lowerVariantTypeName should be: "(LOWER(v.variant_type) = ? AND LOWER(v.name) = ?)"
				variantConds = append(variantConds, lowerVariantTypeName)
				args = append(args, strings.ToLower(v.Type), strings.ToLower(v.Value))
			}
		}

		// Build subquery to find products matching any of the variant conditions
		// IMPORTANT: No leading "AND" here - this is the complete IN(...) condition
		variantQuery := fmt.Sprintf(
			`p.product_id IN (
				SELECT pv.product_id
				FROM product_variants pv
				JOIN variants v ON pv.variant_id = v.variant_id
				WHERE %s
			)`,
			strings.Join(variantConds, " OR "), // OR logic: matches if any variant condition is true
		)

		// Append the complete variant condition (no leading AND)
		conditions = append(conditions, variantQuery)
	}

	// Return empty string if no filters applied
	if len(conditions) == 0 {
		return "", args
	}

	// Join all conditions with AND and prepend " AND " for SQL WHERE clause
	return " AND " + strings.Join(conditions, " AND "), args
}

// SearchProducts performs a comprehensive product search with filtering, sorting, and pagination.
//
// This function is the main entry point for product search functionality. It supports:
//   - Full-text search across product names and category names
//   - Multiple filter types (category, price, variants, dates, tags)
//   - Flexible sorting options (price, date, popularity, alphabetical)
//   - Admin-specific enrichment (creator info, deal status, inventory)
//
// Parameters:
//   - params: dtos.SearchParams - Search parameters containing:
//   - Q: Search query (searches both product and category names)
//   - CategoryName: Category name filter
//   - ProductName: Product name filter
//   - SKU: Exact SKU filter
//   - Tag: Tag filter
//   - MinPrice, MaxPrice: Price range
//   - StartDate, EndDate: Creation date range (format: YYYY-MM-DD)
//   - Variants: Variant filters (type and value)
//   - SortBy: Sort order (see getSortClause constants)
//   - Page, Limit: Pagination parameters
//   - isAdmin: bool - Whether to include admin-specific fields
//
// Returns:
//   - []dtos.Product: Array of products with complete details
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func SearchProducts(params dtos.SearchParams, isAdmin bool) ([]dtos.Product, *dtos.PaginationMeta, error) {
	// Build the main data query with filters and sorting
	query, args := buildSearchQuery(params)

	// Build count query for pagination (same filters, no sorting/pagination)
	countQuery, countArgs := buildCountQuerySearch(params)

	// Get total matching items for pagination metadata
	var totalItems int64
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Execute main query to fetch products
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan rows and enrich products with associated data
	var products []dtos.Product
	for rows.Next() {
		// scanProduct handles null-safe scanning and fetches related data
		product, err := scanProduct(rows, isAdmin)
		if err != nil {
			return nil, nil, err
		}
		products = append(products, product)
	}

	// Check for row iteration errors
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Calculate pagination metadata
	pagination := calculatePagination(params.Page, params.Limit, totalItems)

	return products, &pagination, nil
}

// buildSearchQuery constructs a SELECT query for product search with filters and sorting.
//
// This function builds a comprehensive query with:
//   - LEFT JOINs for optional data (categories, deals, specifications)
//   - DISTINCT to handle multiple variant matches
//   - Full-text search across product and category names
//   - Multiple filter types (name, SKU, tag, price, date range, variants)
//   - Flexible sorting options
//   - Pagination support
//
// Parameters:
//   - params: dtos.SearchParams - Search parameters (see SearchProducts for details)
//
// Returns:
//   - string: SQL SELECT query
//   - []interface{}: Query parameters for prepared statement
func buildSearchQuery(params dtos.SearchParams) (string, []interface{}) {
	// Base query with LEFT JOINs for optional data (deals, specifications)
	query := `
		SELECT DISTINCT
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at,
			c.name as category_name, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE p.product_type = 'single'
	`
	var args []interface{}

	// Apply all filters
	query, args = applyBasicSearchFilters(query, args, params)
	query, args = applyVariantFilters(query, args, params)

	// Apply sorting based on params.SortBy
	query += " ORDER BY " + getSortClause(params.SortBy)

	// Apply pagination (limit and offset)
	if params.Limit > 0 {
		offset := (params.Page - 1) * params.Limit
		query += limtOffset // " LIMIT ? OFFSET ?"
		args = append(args, params.Limit, offset)
	}

	return query, args
}

// applyBasicSearchFilters applies name, SKU, tag, price, and date filters to the query
func applyBasicSearchFilters(query string, args []interface{}, params dtos.SearchParams) (string, []interface{}) {
	// Apply general search query (Q parameter) - searches both product name and category name
	if params.Q != "" {
		query += " AND (LOWER(p.name) LIKE ? OR LOWER(c.name) LIKE ?)"
		searchTerm := "%" + strings.ToLower(params.Q) + "%"
		args = append(args, searchTerm, searchTerm)
	}

	// Apply category name filter (case-insensitive partial match)
	if params.CategoryName != "" {
		query += lowerCname
		args = append(args, "%"+strings.ToLower(params.CategoryName)+"%")
	}

	// Apply product name filter (case-insensitive partial match)
	if params.ProductName != "" {
		query += lowerPname
		args = append(args, "%"+strings.ToLower(params.ProductName)+"%")
	}

	// Apply SKU filter (case-insensitive exact match)
	if params.SKU != "" {
		query += " AND LOWER(p.sku) = ?"
		args = append(args, strings.ToLower(params.SKU))
	}

	// Apply tag filter (case-insensitive exact match)
	if params.Tag != "" {
		query += " AND LOWER(p.tag) = ?"
		args = append(args, strings.ToLower(params.Tag))
	}

	// Apply price range filter (inclusive)
	if params.MinPrice >= 0 && params.MaxPrice > 0 {
		query += " AND p.price BETWEEN ? AND ?"
		args = append(args, params.MinPrice, params.MaxPrice)
	}

	// Apply creation date range filter
	if params.StartDate != "" && params.EndDate != "" {
		query += " AND DATE(p.created_at) BETWEEN ? AND ?"
		args = append(args, params.StartDate, params.EndDate)
	}

	return query, args
}

// applyVariantFilters applies variant-based filters to the query
func applyVariantFilters(query string, args []interface{}, params dtos.SearchParams) (string, []interface{}) {
	if len(params.Variants) == 0 {
		return query, args
	}

	// Subquery to find products matching any of the variant conditions
	variantSubquery := `
		AND p.product_id IN (
			SELECT pv.product_id 
			FROM product_variants pv
			JOIN variants v ON pv.variant_id = v.variant_id
			WHERE `

	variantConditions := []string{}
	for _, variant := range params.Variants {
		if strings.ToLower(variant.Value) == "all" {
			// Match any variant of the specified type (e.g., any size)
			variantConditions = append(variantConditions, lowerVariant)
			args = append(args, strings.ToLower(variant.Type))
		} else {
			// Match specific variant type and value (e.g., size=large)
			variantConditions = append(variantConditions, lowerVariantTypeName)
			args = append(args, strings.ToLower(variant.Type), strings.ToLower(variant.Value))
		}
	}

	// Join variant conditions with OR (product matches if ANY condition is true)
	variantSubquery += strings.Join(variantConditions, " OR ")
	variantSubquery += ")"
	query += variantSubquery

	return query, args
}

// buildCountQuerySearch constructs a COUNT query for search results.
//
// This function builds a query identical to buildSearchQuery but returns only the count.
// It applies the same filters to ensure accurate pagination metadata.
//
// Parameters:
//   - params: dtos.SearchParams - Search parameters (see SearchProducts for details)
//
// Returns:
//   - string: SQL COUNT query
//   - []interface{}: Query parameters for prepared statement
func buildCountQuerySearch(params dtos.SearchParams) (string, []interface{}) {
	// Base count query (no sorting or pagination needed)
	query := `
		SELECT COUNT(DISTINCT p.product_id)
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE p.product_type = 'single'
	`
	var args []interface{}

	// Apply all filters using the same helper functions as buildSearchQuery
	query, args = applyBasicSearchFilters(query, args, params)
	query, args = applyVariantFilters(query, args, params)

	return query, args
}

// Sort order constants for product queries
const (
	SortPriceHighToLow   = "price:high-to-low"  // Most expensive first
	SortPriceLowToHigh   = "price:low-to-high"  // Cheapest first
	SortDateOldToNew     = "date:old-to-new"    // Oldest products first
	SortDateNewToOld     = "date:new-to-old"    // Newest products first (default)
	SortFeatured         = "featured"           // Featured products first, then by date
	SortBestSellers      = "best_sellers"       // Most sold products first
	SortAlphabeticallyAZ = "alphabetically:a-z" // A to Z
	SortAlphabeticallyZA = "alphabetically:z-a" // Z to A
)

// getSortClause returns an SQL ORDER BY clause based on the sort type.
//
// This function maps sort type strings to SQL ORDER BY clauses. It supports
// simple sorting (price, date, alphabetical) and complex sorting (featured, best sellers).
//
// Parameters:
//   - sortBy: string - Sort type (use constants above)
//
// Returns:
//   - string: SQL ORDER BY clause (without "ORDER BY" keyword)
//
// Default: Sorts by creation date descending (newest first)
func getSortClause(sortBy string) string {
	switch sortBy {
	case SortPriceHighToLow:
		return "p.price DESC"
	case SortPriceLowToHigh:
		return "p.price ASC"
	case SortDateOldToNew:
		return "p.created_at ASC"
	case SortDateNewToOld:
		return "p.created_at DESC"
	case SortFeatured:
		// Featured products first (CASE = 0), then non-featured (CASE = 1), then by date
		return `
			CASE WHEN p.product_id IN (SELECT product_id FROM featured_products) THEN 0 ELSE 1 END,
			p.created_at DESC
		`
	case SortBestSellers:
		// Sort by total quantity sold (descending), then by date
		return `
			(SELECT COALESCE(SUM(oi.quantity), 0) 
			 FROM order_items oi 
			 WHERE oi.product_id = p.product_id) DESC,
			p.created_at DESC
		`
	case SortAlphabeticallyAZ:
		return "p.name ASC"
	case SortAlphabeticallyZA:
		return "p.name DESC"
	default:
		return "p.created_at DESC" // Default: newest first
	}
}

// scanProduct scans a product row and enriches it with associated data.
//
// This function handles null-safe scanning of product fields and fetches related data
// (images, warranties, features, variants, tax). For admin users, it also fetches
// creator information, deal status, and inventory data.
//
// Parameters:
//   - rows: *sql.Rows - Current row from query result
//   - isAdmin: bool - Whether to include admin-specific fields
//
// Returns:
//   - dtos.Product: Product with complete details
//   - error: Scan error, database error, or nil on success
func scanProduct(rows *sql.Rows, isAdmin bool) (dtos.Product, error) {
	// Scan raw database values
	data, err := scanProductRow(rows)
	if err != nil {
		return dtos.Product{}, err
	}

	// Build base product from scanned values
	product := buildProduct(data)

	// Fetch and attach related entities (images, tax, variants, etc.)
	if err := enrichProduct(&product); err != nil {
		return product, err
	}

	// Optionally enrich admin-only fields
	if isAdmin {
		if err := enrichProductAdmin(&product); err != nil {
			return product, err
		}
	}

	return product, nil
}

// productRow represents a single scanned DB row with null-safe fields
// type productRow struct {
// 	productID, name, desc, sku, categoryID, searchVector, categoryName, tag sql.NullString
// 	price                                                                   sql.NullFloat64
// 	stockQuantity                                                           sql.NullInt64
// 	createdAt, updatedAt                                                    sql.NullTime
// 	discount, weight, weightLimit                                           sql.NullFloat64
// 	discountType, dimensions, manufacturer                                  sql.NullString
// }

// scanProductRow scans the SQL row into a null-safe struct
func scanProductRow(rows *sql.Rows) (*productRow, error) {
	var r productRow

	if err := rows.Scan(
		&r.productID, &r.name, &r.desc, &r.sku, &r.price, &r.categoryID,
		&r.stockQuantity, &r.searchVector, &r.createdAt, &r.updatedAt,
		&r.categoryName, &r.tag, &r.discount, &r.discountType,
		&r.weight, &r.dimensions, &r.manufacturer, &r.weightLimit,
	); err != nil {
		return nil, err
	}

	return &r, nil
}

// buildProduct converts a scanned DB row into a Product domain object
func buildProduct(r *productRow) dtos.Product {
	product := dtos.Product{
		ID:            r.productID.String,
		Name:          r.name.String,
		Description:   r.desc.String,
		SKU:           r.sku.String,
		Price:         r.price.Float64,
		CategoryID:    r.categoryID.String,
		CategoryName:  r.categoryName.String,
		StockQuantity: intOrZero(r.stockQuantity),
		SearchVector:  r.searchVector.String,
	}

	// Optional string fields
	if r.tag.Valid {
		product.Tag = ptr(strings.TrimSpace(r.tag.String))
	}
	if r.discountType.Valid {
		product.DiscountType = ptr(r.discountType.String)
	}
	if r.dimensions.Valid {
		product.Dimensions = ptr(r.dimensions.String)
	}
	if r.manufacturer.Valid {
		product.Manufacturer = ptr(r.manufacturer.String)
	}

	// Optional numeric fields
	if r.discount.Valid {
		product.Discount = ptr(r.discount.Float64)
	}
	if r.weight.Valid {
		product.Weight = ptr(r.weight.Float64)
	}
	if r.weightLimit.Valid {
		product.WeightLimit = ptr(r.weightLimit.Float64)
	}

	// Optional timestamps
	if r.createdAt.Valid {
		product.CreatedAt = r.createdAt.Time
	}
	if r.updatedAt.Valid {
		product.LastUpdated = r.updatedAt.Time
	}

	return product
}

// intOrZero returns 0 if the sql.NullInt64 is invalid
func intOrZero(v sql.NullInt64) int {
	if v.Valid {
		return int(v.Int64)
	}
	return 0
}

// enrichProductAdmin adds admin-specific fields to a product.
//
// This function fetches additional data only needed by administrators:
//   - CreatedBy: Full name of the user who created the product
//   - IsInTodaysDeals: Whether product is in today's deals
//   - MaxStockQuantity: Total inventory across all locations
//
// Parameters:
//   - product: *dtos.Product - Product to enrich (modified in place)
//
// Returns:
//   - error: Database error or nil on success
func enrichProductAdmin(product *dtos.Product) error {
	var err error

	// Get full name of user who created the product
	product.CreatedBy, err = getProductCreator(product.ID)
	if err != nil {
		return err
	}

	// Check if product is in today's deals
	product.IsInTodaysDeals, err = isProductInTodaysDeal(product.ID)
	if err != nil {
		return err
	}

	// Get maximum stock quantity from inventory table
	product.MaxStockQuantity, err = getMaxQuantity(product.ID)
	// Ensure max quantity is at least current stock quantity
	if product.MaxStockQuantity < product.StockQuantity {
		product.MaxStockQuantity = product.StockQuantity
	}
	return err
}

// getProductCreator fetches the full name of the user who created a product.
//
// Parameters:
//   - productID: string - The product_id to look up
//
// Returns:
//   - string: Full name (first + last) of creator, empty string if not found or NULL
//   - error: Database error or nil on success
func getProductCreator(productID string) (string, error) {
	var createdByID sql.NullString
	query := `SELECT created_by_id FROM products WHERE product_id = ?`

	err := DB.QueryRow(query, productID).Scan(&createdByID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Product not found → no creator
		}
		return "", fmt.Errorf("failed to get product creator ID: %v", err)
	}

	// Handle NULL or empty created_by_id
	if !createdByID.Valid || createdByID.String == "" {
		return "", nil
	}

	// Fetch user's name from users table
	var firstName, lastName sql.NullString
	userQuery := `SELECT first_name, last_name FROM users WHERE user_id = ?`
	err = DB.QueryRow(userQuery, createdByID.String).Scan(&firstName, &lastName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // User not found → return empty
		}
		return "", fmt.Errorf("failed to get user details: %v", err)
	}

	// Handle possible NULL first/last names
	fn := ""
	ln := ""
	if firstName.Valid {
		fn = firstName.String
	}
	if lastName.Valid {
		ln = lastName.String
	}

	// Combine names and trim whitespace
	fullName := strings.TrimSpace(fn + " " + ln)
	return fullName, nil
}

// get user full name by user id
func getUserNames(userID string) (string, error) {
	// Fetch user's name from users table
	var firstName, lastName sql.NullString
	userQuery := `SELECT first_name, last_name FROM users WHERE user_id = ?`
	err := DB.QueryRow(userQuery, userID).Scan(&firstName, &lastName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // User not found → return empty
		}
		return "", fmt.Errorf("failed to get user details: %v", err)
	}

	// Handle possible NULL first/last names
	fn := ""
	ln := ""
	if firstName.Valid {
		fn = firstName.String
	}
	if lastName.Valid {
		ln = lastName.String
	}

	// Combine names and trim whitespace
	fullName := strings.TrimSpace(fn + " " + ln)
	return fullName, nil
}

// isProductInTodaysDeal checks if a product is part of today's deals.
//
// This function looks for a deal with "today" in its name (case-insensitive regex)
// and checks if the product is linked to that deal.
//
// Parameters:
//   - productID: string - The product_id to check
//
// Returns:
//   - bool: true if product is in today's deals, false otherwise
//   - error: Database error or nil on success
func isProductInTodaysDeal(productID string) (bool, error) {
	var dealID string

	// Find deal_id for a deal whose name matches 'today' (case-insensitive)
	dealQuery := `SELECT deal_id FROM deals WHERE name REGEXP '(?i)today' LIMIT 1`
	err := DB.QueryRow(dealQuery).Scan(&dealID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // No "today" deal exists
		}
		return false, fmt.Errorf("failed to get today's deal: %v", err)
	}

	// Check if this product is linked to that deal
	var exists bool
	checkQuery := `SELECT EXISTS(
		SELECT 1 FROM deal_products WHERE product_id = ? AND deal_id = ?
	)`
	err = DB.QueryRow(checkQuery, productID, dealID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check product-deal link: %v", err)
	}

	return exists, nil
}

// getMaxQuantity retrieves the total inventory quantity for a product across all locations.
//
// Parameters:
//   - productID: string - The product_id to sum inventory for
//
// Returns:
//   - int: Total quantity across all inventory records, 0 if none found
//   - error: Database error or nil on success
func getMaxQuantity(productID string) (int, error) {
	var totalQuantity sql.NullInt64

	// Sum all inventory quantities for this product
	query := `
		SELECT SUM(quantity)
		FROM inventory
		WHERE product_id = ?
	`
	err := DB.QueryRow(query, productID).Scan(&totalQuantity)
	if err != nil {
		return 0, fmt.Errorf("failed to get quantity for product %s: %v", productID, err)
	}

	// Handle NULL result (no inventory records found)
	if !totalQuantity.Valid {
		return 0, nil
	}

	return int(totalQuantity.Int64), nil
}

// InsertProductSpecs creates a new product specification record.
//
// Parameters:
//   - req: dtos.ProductSpecs - Specification data containing:
//   - ProductID: Product to attach specs to
//   - Weight: Product weight
//   - WeightLimit: Maximum weight for shipping
//   - Dimensions: Product dimensions (e.g., "10x5x3")
//   - Manufacturer: Manufacturer name
//
// Returns:
//   - error: "product not found", database error, or nil on success
func InsertProductSpecs(req dtos.ProductSpecs) error {
	// Validate product exists
	err := IsProductThere(req.ProductID)
	if err != nil {
		return err
	}

	insertQuery := `INSERT INTO product_specifications (specifications_id, product_id, weight, weight_limit, dimensions, manufacturer) VALUES (?, ?, ?, ?,?,?)`

	// Generate unique specifications ID
	specificationsID, _ := shortid.Generate()

	// Insert specification record
	if _, err := DB.Exec(insertQuery, specificationsID, req.ProductID, req.Weight, req.WeightLimit, req.Dimensions, req.Manufacturer); err != nil {
		return err
	}

	return nil
}

// GetExpensiveAndCheapProducts fetches the most and least expensive products.
//
// Returns:
//   - *dtos.ExpensiveCheapProduct: Struct containing:
//   - CheapestProduct: Product with lowest price
//   - ExpensiveProduct: Product with highest price
//   - error: Database error or nil on success
func GetExpensiveAndCheapProducts() (*dtos.ExpensiveCheapProduct, error) {
	// Fetch cheapest product
	cheapestProduct, err := getProductByPriceType("cheapest")
	if err != nil {
		return nil, err
	}

	// Fetch most expensive product
	expensiveProduct, err := getProductByPriceType("expensive")
	if err != nil {
		return nil, err
	}

	// Combine into response struct
	combinedProducts := &dtos.ExpensiveCheapProduct{
		CheapestProduct:  *cheapestProduct,
		ExpensiveProduct: *expensiveProduct,
	}
	return combinedProducts, nil
}

// getProductByPriceType fetches a single product with either the lowest or highest price.
//
// This internal function is used by GetExpensiveAndCheapProducts to fetch extreme price points.
//
// Parameters:
//   - priceType: string - Either "cheapest" or "expensive"
//
// Returns:
//   - *dtos.Product: Product with complete details, nil if no products found
//   - error: Invalid priceType, database error, or nil on success
func getProductByPriceType(priceType string) (*dtos.Product, error) {
	// Determine sort order based on price type
	var orderClause string
	switch strings.ToLower(priceType) {
	case "cheapest":
		orderClause = "ASC" // Ascending = lowest price first
	case "expensive":
		orderClause = "DESC" // Descending = highest price first
	default:
		return nil, fmt.Errorf("invalid price type: %s", priceType)
	}

	// Query to fetch product with extreme price
	query := fmt.Sprintf(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, 
			p.category_id, p.stock_quantity, p.search_vector, 
			p.created_at, p.last_updated_at, c.name AS category_name, p.tag
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE p.product_type = 'single'
		ORDER BY p.price %s
		LIMIT 1
	`, orderClause)

	var p dtos.Product
	err := DB.QueryRow(query).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&p.CategoryName, &p.Tag,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No products found
		}
		return nil, err
	}

	// Fetch product images
	if p.Images, err = fetchProductImages(p.ID); err != nil {
		return nil, err
	}

	// Fetch product warranties
	warranty, err := FetchProductWarranties(p.ID)
	if err != nil {
		return nil, err
	}
	p.Warranty = &warranty

	// Fetch product features
	if p.Features, err = fetchProductFeatures(p.ID); err != nil {
		return nil, err
	}

	// Fetch product variants
	if p.ProductVariants, err = getProductVariants(p.ID); err != nil {
		return nil, err
	}

	// Fetch tax information
	tax, err := fetchProductTax(p.ID)
	if err != nil {
		return nil, err
	}
	p.Tax = &tax

	return &p, nil
}

// IsProductInUserWishlist checks if a product is in a user's wishlist.
//
// Parameters:
//   - userID: string - The user_id to check
//   - productID: string - The product_id to check
//
// Returns:
//   - bool: true if product is in user's wishlist, false otherwise
//   - error: Database error or nil on success
func IsProductInUserWishlist(userID, productID string) (bool, error) {
	var exists bool

	// Check if product exists in user's wishlist via wishlist_items join
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM wishlists w
			JOIN wishlist_items witems ON w.wishlist_id = witems.wishlist_id
			WHERE w.user_id = ? AND witems.product_id = ?
		)
	`

	err := DB.QueryRow(query, userID, productID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check wishlist: %w", err)
	}

	return exists, nil
}

// RemoveHeldProductSpecs deletes a product specification record.
//
// Parameters:
//   - specID: string - The specifications_id to delete
//
// Returns:
//   - error: Database error or nil on success
func RemoveHeldProductSpecs(specID string) error {
	_, err := DB.Exec(`DELETE FROM product_specifications WHERE specifications_id = ?`, specID)
	return err
}

// HoldProductSpecs retrieves all specification IDs for a product.
//
// This function is used to get specification IDs before deletion operations
// to ensure proper cleanup of related records.
//
// Parameters:
//   - productID: string - The product_id to fetch specification IDs for
//
// Returns:
//   - []string: Array of specifications_id values
//   - error: Database error or nil on success
func HoldProductSpecs(productID string) ([]string, error) {
	// Query all specification IDs for this product
	rows, err := DB.Query(`SELECT specifications_id FROM product_specifications WHERE product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect specification IDs
	var specIDs []string
	for rows.Next() {
		var specID string
		if err := rows.Scan(&specID); err != nil {
			return nil, err
		}
		specIDs = append(specIDs, specID)
	}
	return specIDs, nil
}

// RemoveAllProductVariants deletes all variant associations for a product.
//
// This function removes records from product_variants table (the join table),
// but does NOT delete the variant definitions from the variants table.
//
// Parameters:
//   - productID: string - The product_id to remove variants for
//
// Returns:
//   - error: Database error or nil on success
func RemoveAllProductVariants(productID string) error {
	_, err := DB.Exec(`DELETE FROM product_variants WHERE product_id = ?`, productID)
	return err
}
