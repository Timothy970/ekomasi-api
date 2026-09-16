package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"strings"
)

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
func GetAllProducts(db DBExecutor, tenantID int, categoryFilter, productFilter, categoryID string, page, limit int) ([]dtos.Product, *dtos.PaginationMeta, error) {
	// Validate category exists if filtering by category ID
	if categoryID != "" {
		if err := CategoryExists(db, categoryID); err != nil {
			return nil, nil, err
		}
	}

	// Build queries with filters and pagination
	query, args := buildProductQuery(tenantID, categoryFilter, productFilter, categoryID, page, limit)
	countQuery, countArgs := buildCountQuery(tenantID, categoryFilter, productFilter, categoryID)

	// Execute main product query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Get total count for pagination
	var totalItems int64
	if err := db.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Scan product rows and enrich with associated data
	var products []dtos.Product
	for rows.Next() {
		product, err := scanAndEnrichProduct(db, rows)
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
//   - tenantID: int - Tenant identifier
//   - categoryFilter: string - Optional category name filter (case-insensitive partial match)
//   - productFilter: string - Optional product name filter (case-insensitive partial match)
//   - categoryID: string - Optional category ID (includes parent and child categories via recursive CTE)
//
// Returns:
//   - string: SQL COUNT query
//   - []any: Query parameters for prepared statement
func buildCountQuery(tenantID int, categoryFilter, productFilter, categoryID string) (string, []any) {
	// Base query for simple filtering (no category hierarchy)
	query := `
        SELECT COUNT(DISTINCT p.product_id)
        FROM products p
        JOIN categories c ON p.category_id = c.category_id
        WHERE p.product_type = 'single' AND p.tenant_id = ?`
	args := []any{tenantID}

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
            WHERE category_id = ? AND tenant_id = ?
            UNION ALL
            SELECT c.category_id, c.parent_category_id
            FROM categories c
            INNER JOIN ancestors a ON c.category_id = a.parent_category_id
        ),
        descendants AS (
            SELECT category_id, parent_category_id
            FROM categories
            WHERE category_id = ? AND tenant_id = ?
            UNION ALL
            SELECT c.category_id, c.parent_category_id
            FROM categories c
            INNER JOIN descendants d ON c.parent_category_id = d.category_id
        )
        SELECT COUNT(DISTINCT p.product_id)
        FROM products p
        JOIN categories c ON p.category_id = c.category_id
        WHERE p.product_type = 'single' AND p.tenant_id = ?
          AND c.category_id IN (
              SELECT category_id FROM ancestors
              UNION
              SELECT category_id FROM descendants
          )`
		args = []any{categoryID, tenantID, categoryID, tenantID, tenantID}
	}

	return query, args
}

// buildProductQuery constructs a SELECT query for products with filters and pagination.
//
// This function builds a query that fetches products with their category, deal, and
// specification information using LEFT JOINs for optional data.
//
// Parameters:
//   - tenantID: int - Tenant identifier
//   - categoryFilter: string - Optional category name filter (case-insensitive partial match)
//   - productFilter: string - Optional product name filter (case-insensitive partial match)
//   - categoryID: string - Optional category ID (includes parent/child via hierarchy)
//   - page: int - Page number for pagination (1-indexed)
//   - limit: int - Number of products per page (0 = no limit)
//
// Returns:
//   - string: SQL SELECT query with JOINs and filters
//   - []any: Query parameters for prepared statement
func buildProductQuery(tenantID int, categoryFilter, productFilter, categoryID string, page, limit int) (string, []any) {
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
        WHERE p.product_type = 'single' AND p.tenant_id = ?`
	args := []any{tenantID}

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
func scanAndEnrichProduct(db DBExecutor, rows *sql.Rows) (dtos.Product, error) {
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
	product.Images, err = fetchProductImages(db, product.ID)
	if err != nil {
		return dtos.Product{}, err
	}

	warranty, err := FetchProductWarranties(db, product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Warranty = &warranty

	features, err := fetchProductFeatures(db, product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Features = features

	variants, err := getProductVariants(db, product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.ProductVariants = variants
	product.VariantSelection, err = GetVariantSelection(db, product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	// tax, err := fetchProductTax(db, product.ID)
	// if err != nil {
	// 	return dtos.Product{}, err
	// }
	// product.Tax = &tax

	return product, nil
}
