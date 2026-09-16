package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"math"
	"strings"
)

func IsValidSubcategory(db DBExecutor, categoryID string) (bool, error) {
	var parentID sql.NullString
	err := db.QueryRow(`SELECT parent_category_id FROM categories WHERE category_id = ?`, categoryID).Scan(&parentID)
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
func IsValidCategory(db DBExecutor, categoryID string, tenantID int) (bool, error) {
	var parentID sql.NullString
	err := db.QueryRow(`
		SELECT parent_category_id 
		FROM categories 
		WHERE category_id = ? AND (tenant_id = ? OR tenant_id IS NULL OR tenant_id = 0)`, categoryID, tenantID).
		Scan(&parentID)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("category not found")
		}
		return false, err
	}

	// Only valid if it's a parent (i.e., has no parent itself or parent_category_id is empty string)
	if !parentID.Valid || parentID.String == "" {
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
//   - tenantID: int - Authenticated/resolved tenant ID
//
// Returns:
//   - []dtos.CategoryResponse: Array of categories containing:
//   - ID, Name, ParentID, ImageURL, Description
//   - Subcategories: Array of subcategories
//   - Products: Paginated products under subcategories
//   - *dtos.PaginationMeta: Pagination metadata for products
//   - error: "cannot use a subcategory ID, must be a main category", database error, or nil on success
func GetCategoriesWithSubcategoriesAndProducts(
	db DBExecutor,
	searchParams dtos.SearchParams,
	filterCategoryID string,
	tenantID int,
) ([]dtos.CategoryResponse, *dtos.PaginationMeta, error) {

	// Validate category filter (must be main category, not subcategory)
	if filterCategoryID != "" {
		valid, err := IsValidCategory(db, filterCategoryID, tenantID)
		if err != nil {
			return nil, nil, err
		}
		if !valid {
			return nil, nil, fmt.Errorf("cannot use a subcategory ID, must be a main category")
		}
	}

	// Fetch main categories with optional filter
	categories, err := getMainCategories(db, filterCategoryID, searchParams, tenantID)
	if err != nil {
		return nil, nil, err
	}

	var paginationMeta *dtos.PaginationMeta

	// For each category, attach subcategories and products
	for i := range categories {
		// Get subcategories for this main category
		subs, subIDs, err := getSubcategoriesWithParentID(db, categories[i].ID, tenantID)
		if err != nil {
			return nil, nil, err
		}
		categories[i].Subcategories = subs

		// Get products for main category and all subcategories
		allIDs := append([]string{categories[i].ID}, subIDs...)
		products, meta, err := getProductsForSubcategories(db, allIDs, searchParams.Page, searchParams.Limit, searchParams, tenantID)
		if err != nil {
			return nil, nil, err
		}
		if products == nil {
			products = make([]dtos.CategoryProduct, 0)
		}
		categories[i].Products = products
		paginationMeta = meta
	}

	return categories, paginationMeta, nil
}

// getMainCategories fetches parent categories that have products in their subcategories.
//
// This internal function retrieves top-level categories with optional filters.
// Only returns categories that have at least one product in their subcategories or directly.
//
// Parameters:
//   - filterCategoryID: string - Optional category ID filter
//   - params: dtos.SearchParams - Search parameters (Q, CategoryName)
//   - tenantID: int - Tenant identifier
//
// Returns:
//   - []dtos.CategoryResponse: Main categories
//   - error: Database error or nil on success
func getMainCategories(db DBExecutor, filterCategoryID string, params dtos.SearchParams, tenantID int) ([]dtos.CategoryResponse, error) {
	var (
		query = `
            SELECT c.category_id, c.name, c.parent_category_id, c.image, c.description
            FROM categories c
            WHERE (c.parent_category_id IS NULL OR c.parent_category_id = '')
            AND (c.tenant_id = ? OR c.tenant_id IS NULL OR c.tenant_id = 0)`
		args = []any{tenantID}
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

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan category rows
	categories := make([]dtos.CategoryResponse, 0)
	for rows.Next() {
		var cat dtos.CategoryResponse
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.ParentID, &cat.ImageURL, &cat.Description); err != nil {
			return nil, err
		}
		if cat.Subcategories == nil {
			cat.Subcategories = make([]dtos.SubcategoryResponse, 0)
		}
		if cat.Products == nil {
			cat.Products = make([]dtos.CategoryProduct, 0)
		}
		categories = append(categories, cat)
	}

	return categories, nil
}

// getSubcategoriesWithParentID fetches subcategories for a parent category.
//
// This internal function retrieves child categories.
//
// Parameters:
//   - parentID: string - The parent category_id
//   - tenantID: int - Tenant identifier
//
// Returns:
//   - []dtos.SubcategoryResponse: Subcategories
//   - []string: Array of subcategory IDs (for product queries)
//   - error: Database error or nil on success
func getSubcategoriesWithParentID(db DBExecutor, parentID string, tenantID int) ([]dtos.SubcategoryResponse, []string, error) {
	rows, err := db.Query(`
        SELECT c.category_id, c.name, c.parent_category_id, c.image, c.description
        FROM categories c
        WHERE c.parent_category_id = ?
        AND (c.tenant_id = ? OR c.tenant_id IS NULL OR c.tenant_id = 0)`, parentID, tenantID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	subs := make([]dtos.SubcategoryResponse, 0)
	ids := make([]string, 0)

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
//   - tenantID: int - Tenant identifier
//
// Returns:
//   - []dtos.CategoryProduct: Array of products with:
//   - Basic info: ID, Name, Description, SKU, Price
//   - CategoryID: Parent category ID
//   - SubcategoryID: Direct category ID
//   - Stock, timestamps, tax, variants, images, etc.
//   - *dtos.PaginationMeta: Pagination metadata (Page, Size, TotalItems, TotalPages, HasPrev, HasNext)
//   - error: Database error or nil on success
func getProductsForSubcategories(db DBExecutor, subIDs []string, page, size int, params dtos.SearchParams, tenantID int) ([]dtos.CategoryProduct, *dtos.PaginationMeta, error) {
	// Early return if no subcategories specified
	if len(subIDs) == 0 {
		return make([]dtos.CategoryProduct, 0), &dtos.PaginationMeta{
			Page:       page,
			Size:       size,
			TotalItems: 0,
			TotalPages: 0,
			HasPrev:    false,
			HasNext:    false,
		}, nil
	}

	// Build query components
	whereClause, args := buildSubcategoryWhereClause(subIDs, params, tenantID)

	// Get total count for pagination
	totalItems, err := countSubcategoryProduct(db, whereClause, args)
	if err != nil {
		return nil, nil, err
	}

	// Fetch products with pagination
	products, err := fetchSubcategoryProducts(db, whereClause, args, params, page, size)
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
func buildSubcategoryWhereClause(subIDs []string, params dtos.SearchParams, tenantID int) (string, []any) {
	placeholders := strings.Repeat(",?", len(subIDs)-1)
	baseWhere := fmt.Sprintf(`WHERE p.category_id IN (?%s)
		AND p.product_type = 'single'
		AND (p.tenant_id = ? OR p.tenant_id IS NULL OR p.tenant_id = 0)`, placeholders)

	args := make([]any, len(subIDs)+1)
	for i, id := range subIDs {
		args[i] = id
	}
	args[len(subIDs)] = tenantID

	filterQuery, filterArgs := buildProductFilters(params)
	args = append(args, filterArgs...)
	return baseWhere + filterQuery, args
}

// countSubcategoryProducts returns the total count of products matching the criteria
func countSubcategoryProduct(db DBExecutor, whereClause string, args []any) (int, error) {
	joinClause := `FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id`
	countQuery := "SELECT COUNT(*) " + joinClause + " " + whereClause

	var totalItems int
	err := db.QueryRow(countQuery, args...).Scan(&totalItems)
	return totalItems, err
}

// fetchSubcategoryProducts retrieves paginated products with all related data
