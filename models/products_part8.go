package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"strings"
)

func buildSearchQuery(params dtos.SearchParams) (string, []any) {
	// Base query with LEFT JOINs for optional data (deals, specifications)
	query := `
		SELECT DISTINCT
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at,
			c.name as category_name, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit,
			CASE 
			WHEN fp.product_id IS NOT NULL THEN TRUE
			ELSE FALSE
		END AS is_featured
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		LEFT JOIN featured_products fp ON p.product_id = fp.product_id
		WHERE p.product_type = 'single'
	`
	var args []any

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
func applyBasicSearchFilters(query string, args []any, params dtos.SearchParams) (string, []any) {
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
func applyVariantFilters(query string, args []any, params dtos.SearchParams) (string, []any) {
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
//   - []any: Query parameters for prepared statement
func buildCountQuerySearch(params dtos.SearchParams) (string, []any) {
	// Base count query (no sorting or pagination needed)
	query := `
		SELECT COUNT(DISTINCT p.product_id)
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE p.product_type = 'single'
	`
	var args []any

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
func scanProduct(db DBExecutor, rows *sql.Rows, isAdmin bool) (dtos.Product, error) {
	// Scan raw database values
	data, err := scanProductRow(rows)
	if err != nil {
		return dtos.Product{}, err
	}

	// Build base product from scanned values
	product := buildProduct(data)

	// Fetch and attach related entities (images, tax, variants, etc.)
	if err := enrichProduct(db, &product); err != nil {
		return product, err
	}

	// Optionally enrich admin-only fields
	if isAdmin {
		if err := enrichProductAdmin(db, &product); err != nil {
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
// 	isFeatured                                                              sql.NullBool
// }

// scanProductRow scans the SQL row into a null-safe struct
func scanProductRow(rows *sql.Rows) (*productRow, error) {
	var r productRow

	if err := rows.Scan(
		&r.productID, &r.name, &r.desc, &r.sku, &r.price, &r.categoryID,
		&r.stockQuantity, &r.searchVector, &r.createdAt, &r.updatedAt,
		&r.categoryName, &r.tag, &r.discount, &r.discountType,
		&r.weight, &r.dimensions, &r.manufacturer, &r.weightLimit, &r.isFeatured,
	); err != nil {
		return nil, err
	}

	return &r, nil
}

// buildProduct converts a scanned DB row into a Product domain object
