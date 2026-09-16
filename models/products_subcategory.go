package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"fmt"
	"strings"
)

func fetchSubcategoryProducts(db DBExecutor, whereClause string, args []any, params dtos.SearchParams, page, size int) ([]dtos.CategoryProduct, error) {
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

	rows, err := db.Query(dataQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]dtos.CategoryProduct, 0)
	for rows.Next() {
		product, err := scanCategoryProduct(db, rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

// scanCategoryProduct scans and enriches a single category product row
func scanCategoryProduct(db DBExecutor, rows *sql.Rows) (dtos.CategoryProduct, error) {
	var pr dtos.CategoryProduct
	var subcategoryID string
	var parentCategoryNull sql.NullString
	var detailsData []byte

	if err := rows.Scan(
		&pr.ID, &pr.Name, &pr.Description, &pr.SKU, &pr.Price,
		&subcategoryID, &parentCategoryNull, &pr.StockQuantity,
		&pr.SearchVector, &pr.CreatedAt, &pr.LastUpdated, &pr.Tag, &detailsData, &pr.Discount, &pr.DiscountType, &pr.Weight, &pr.Dimensions, &pr.Manufacturer, &pr.WeightLimit,
	); err != nil {
		return pr, err
	}

	if parentCategoryNull.Valid {
		pr.CategoryID = parentCategoryNull.String
		pr.SubcategoryID = subcategoryID
	} else {
		pr.CategoryID = subcategoryID
		pr.SubcategoryID = ""
	}

	if len(detailsData) > 0 {
		if err := json.Unmarshal(detailsData, &pr.Details); err != nil {
			return pr, err
		}
	} else {
		pr.Details = []string{}
	}

	return pr, enrichCategoryProduct(db, &pr)
}

// enrichCategoryProduct fetches and attaches all related data to a category product
func enrichCategoryProduct(db DBExecutor, pr *dtos.CategoryProduct) error {
	var err error

	if pr.Images, err = fetchProductImages(db, pr.ID); err != nil {
		return err
	}

	warranty, err := FetchProductWarranties(db, pr.ID)
	if err != nil {
		return err
	}
	pr.Warranty = &warranty

	if pr.Features, err = fetchProductFeatures(db, pr.ID); err != nil {
		return err
	}

	if pr.ProductVariants, err = getProductVariants(db, pr.ID); err != nil {
		return err
	}

	// tax, err := fetchProductTax(db, pr.ID)
	// if err != nil {
	// 	return err
	// }
	// pr.Tax = &tax
	pr.VariantSelection, err = GetVariantSelection(db, pr.ID)
	if err != nil {
		return err
	}

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
//   - []any: Query parameters for prepared statement
func buildProductFilters(params dtos.SearchParams) (string, []any) {
	var (
		conditions []string // Array of SQL condition strings
		args       []any    // Array of query parameters
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
func SearchProducts(db DBExecutor, params dtos.SearchParams, isAdmin bool) ([]dtos.Product, *dtos.PaginationMeta, error) {
	// Build the main data query with filters and sorting
	query, args := buildSearchQuery(params)

	// Build count query for pagination (same filters, no sorting/pagination)
	countQuery, countArgs := buildCountQuerySearch(params)

	// Get total matching items for pagination metadata
	var totalItems int64
	if err := db.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Execute main query to fetch products
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan rows and enrich products with associated data
	var products []dtos.Product
	for rows.Next() {
		// scanProduct handles null-safe scanning and fetches related data
		product, err := scanProduct(db, rows, isAdmin)
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
//   - []any: Query parameters for prepared statement
