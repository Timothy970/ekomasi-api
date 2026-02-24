// Package models provides the category management functionality for the Adenzo e-commerce platform.
//
// This package handles core category operations including:
//   - Hierarchical category structure (parent categories and subcategories)
//   - Category CRUD operations with validation
//   - Product association with categories
//   - Category existence validation
//   - Admin category listing with pagination and search
//   - Category tree retrieval with subcategories
//
// Category Features:
//   - Two-level hierarchy: parent categories and subcategories
//   - Name uniqueness validation
//   - Optional parent category assignment
//   - Product preview (up to 6 products per subcategory)
//   - Dynamic query building for flexible updates
//   - Pagination support for admin views
//   - Search by category name
//
// Database Schema:
//   - categories table: Stores category metadata (category_id, name, parent_category_id, image, description)
//   - products table: Referenced for category-product associations
//   - product_images table: Referenced for product image URLs
//
// Category Hierarchy:
//   - Parent categories: parent_category_id IS NULL
//   - Subcategories: parent_category_id references parent category
//   - Products: Associated with subcategories via category_id
package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/teris-io/shortid"
)

// GetAllCategories retrieves the complete category hierarchy with products.
//
// This function builds a three-level structure:
//  1. Parent categories (top-level categories with no parent)
//  2. Subcategories (categories with parent_category_id)
//  3. Products (up to 6 products per subcategory with images)
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.CategoryData: Array of parent categories with nested subcategories and products
//   - error: Database error if queries fail
//
// Structure:
//   - Each parent category contains an array of subcategories
//   - Each subcategory contains an array of up to 6 products
//   - Products include ID, name, price, and image URL
func GetAllCategories(db DBExecutor) ([]dtos.CategoryData, error) {
	// Step 1: Get top-level categories (parent categories with no parent_category_id)
	rows, err := db.Query("SELECT category_id, name, parent_category_id, image, description FROM categories WHERE parent_category_id IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []dtos.CategoryData
	log.Printf("fetching sub categories888888")
	// Iterate through parent categories
	for rows.Next() {
		var cat dtos.CategoryData
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.ParentCategoryID, &cat.Image, &cat.Description); err != nil {
			return nil, err
		}

		// Step 2: Get subcategories for this parent category
		subcategories, err := getSubcategories(db, cat.ID)
		if err != nil {
			return nil, err
		}
		cat.Subcategories = subcategories

		categories = append(categories, cat)
	}

	return categories, nil
}

// getSubcategories retrieves all subcategories for a parent category with their products.
//
// This helper function is called by GetAllCategories to build the category hierarchy.
// It fetches subcategories and populates each with up to 6 products.
//
// Parameters:
//   - parentID: The category_id of the parent category
//
// Returns:
//   - []dtos.CategoryData: Array of subcategories with nested products
//   - error: Database error if queries fail
//
// Product Limit:
//   - Each subcategory includes up to 6 products for preview purposes
//   - Products include basic info (ID, name, price, image URL)
func getSubcategories(db DBExecutor, parentID string) ([]dtos.CategoryData, error) {
	// Query subcategories with this parent_category_id
	rows, err := db.Query("SELECT category_id, name, parent_category_id, image,description FROM categories WHERE parent_category_id = ?", parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []dtos.CategoryData
	// Iterate through subcategories
	for rows.Next() {
		var sub dtos.CategoryData
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.ParentCategoryID, &sub.Image, &sub.Description); err != nil {
			return nil, err
		}

		// Step 3: Get products for this subcategory (limit 6 for preview)
		products, err := getProductsByCategory(db, sub.ID)
		if err != nil {
			return nil, err
		}
		sub.Products = products

		subs = append(subs, sub)
	}

	return subs, nil
}

// getProductsByCategory retrieves up to 6 products for a specific category.
//
// This helper function fetches product previews for category display.
// It includes product image via LEFT JOIN with product_images table.
//
// Parameters:
//   - categoryID: The category_id to fetch products for
//
// Returns:
//   - []dtos.ProductData: Array of up to 6 products with basic info and image URL
//   - error: Database error if query fails
//
// Product Fields:
//   - ID: product_id
//   - Name: Product name
//   - Price: Product price
//   - URL: Product image URL (from first product_images record)
//
// LIMIT 6 restricts results for preview purposes in category listings
func getProductsByCategory(db DBExecutor, categoryID string) ([]dtos.ProductData, error) {
	// Query products with image via LEFT JOIN
	rows, err := db.Query(`
	SELECT p.product_id, p.name, p.price, pg.url
	FROM products p
	LEFT JOIN product_images pg ON p.product_id = pg.product_id
	WHERE p.category_id = ?
	LIMIT 6
`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.ProductData
	// Collect product data
	for rows.Next() {
		var p dtos.ProductData
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}

// isCategoryThere validates that a category exists in the database by category_id.
//
// This function uses the RecordExists helper to check for category existence.
//
// Parameters:
//   - value: The category_id to validate
//
// Returns:
//   - error: nil if category exists, "category not found" error if not found, or database error
func isCategoryThere(db DBExecutor, value string) error {
	// Check if category record exists in categories table
	exists, err := RecordExists(db, "categories", "category_id = ?", value)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("category not found")
	}
	return nil
}

// AddNewCategory creates a new category with name uniqueness validation.
//
// This function handles both parent categories and subcategories:
//   - Parent category: ParentID is nil (inserted without parent_category_id)
//   - Subcategory: ParentID provided (validates parent exists before inserting)
//
// Parameters:
//   - input: CreateCategory DTO containing Name, ParentID (optional), Description, Image
//
// Returns:
//   - *dtos.Category: Pointer to created category object with generated ID
//   - error: Validation error (duplicate name, parent not found) or database error
//
// Validation:
//   - Category name must be unique across all categories
//   - If ParentID provided, parent category must exist
func AddNewCategory(db DBExecutor, input dtos.CreateCategory) (*dtos.Category, error) {

	// Check if the category name already exists (must be unique)
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM categories WHERE name = ?
		)
	`, input.Name).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf("failed to check if category exists: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("category with name '%s' already exists", input.Name)
	}

	// Generate unique category ID
	categoryID, _ := shortid.Generate()

	// If ParentID is nil, create parent category
	if input.ParentID == nil {
		// Insert parent category without parent_category_id
		_, err := db.Exec(`
		INSERT INTO categories (category_id, name, description, image)
		VALUES (?, ?, ?, ?)`,
			categoryID, input.Name, input.Description, input.Image,
		)

		if err != nil {
			return nil, err
		}

	} else {
		// Validate parent category exists before creating subcategory
		err := isCategoryThere(db, *input.ParentID)
		if err != nil {
			return nil, err
		}
		// Insert subcategory with parent_category_id
		_, err = db.Exec(`
		INSERT INTO categories (category_id, name, parent_category_id, description, image)
		VALUES (?, ?, ?, ?, ?)`,
			categoryID, input.Name, input.ParentID, input.Description, input.Image,
		)
		if err != nil {
			return nil, err
		}
	}
	// Return created category object
	return &dtos.Category{
		ID:               categoryID,
		Name:             input.Name,
		ParentCategoryID: input.ParentID,
		Description:      input.Description,
		Image:            input.Image,
	}, err
}

// UpdateCategory updates an existing category with validation.
//
// This function builds a dynamic UPDATE query to modify only the fields provided in the input.
// It validates category existence and name uniqueness before updating.
//
// Parameters:
//   - id: The category_id to update
//   - input: CreateCategory DTO with fields to update (empty fields are skipped)
//
// Returns:
//   - *dtos.Category: Pointer to updated category object
//   - error: Validation error (not found, duplicate name, no fields) or database error
//
// Validation:
//   - Category must exist
//   - New name (if provided) must be unique (excluding current category)
//   - At least one field must be provided for update
//
// Dynamic Update:
//   - Only non-empty fields in input are included in UPDATE query
//   - Fields: Name, ParentID, Description, Image
func UpdateCategory(db DBExecutor, id string, input dtos.UpdateCategoryPayload) (*dtos.Category, error) {
	// Step 1: Validate category exists
	if err := CategoryExists(db, id); err != nil {
		return nil, err
	}

	// Step 2: Check uniqueness of name if being updated
	if input.Name != "" {
		var nameExists bool
		err := db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM categories WHERE name = ? AND category_id != ?
			)
		`, input.Name, id).Scan(&nameExists)
		if err != nil {
			return nil, fmt.Errorf("failed to check name uniqueness: %w", err)
		}
		if nameExists {
			return nil, fmt.Errorf("a category with the name '%s' already exists", input.Name)
		}
	}

	// Step 3: Build dynamic UPDATE query with only non-empty fields
	setClauses := []string{}
	args := []interface{}{}

	// Add Name field if provided
	if input.Name != "" {
		setClauses = append(setClauses, "name = ?")
		args = append(args, input.Name)
	}
	// Add ParentID field if provided
	if input.ParentID != nil {
		setClauses = append(setClauses, "parent_category_id = ?")
		args = append(args, *input.ParentID)
	}
	// Add Description field if provided
	if input.Description != "" {
		setClauses = append(setClauses, "description = ?")
		args = append(args, input.Description)
	}
	// Add Image field if provided
	if input.Image != nil {
		setClauses = append(setClauses, "image = ?")
		args = append(args, *input.Image)
	}

	// Validate at least one field is being updated
	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add category_id to args for WHERE clause
	args = append(args, id)

	// Build and execute dynamic UPDATE query
	query := fmt.Sprintf(`UPDATE categories SET %s WHERE category_id = ?`, strings.Join(setClauses, ", "))

	_, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	// Step 4: Return updated category DTO
	return &dtos.Category{
		ID:               id,
		Name:             input.Name,
		ParentCategoryID: input.ParentID,
		Description:      input.Description,
	}, nil
}

// DeleteCategory removes a category from the database.
//
// This function validates category existence before deletion.
//
// Parameters:
//   - id: The category_id to delete
//
// Returns:
//   - error: "category not found" if category doesn't exist, or database error if deletion fails
//
// Important:
//   - The commented-out code would prevent deletion of categories with subcategories
//   - Currently allows cascade deletion or orphaning of subcategories (depends on DB constraints)
//   - Consider uncommenting child check to prevent accidental data loss
func DeleteCategory(db DBExecutor, id string) error {
	// Validate category exists
	err := CategoryExists(db, id)
	if err != nil {
		return err
	}

	// Optional: Check for child categories (currently commented out)
	// Uncommenting this prevents deletion of categories with subcategories
	// var childCount int
	// err = DB.QueryRow("SELECT COUNT(*) FROM categories WHERE parent_category_id = ?", id).Scan(&childCount)
	// if err != nil {
	// 	return fmt.Errorf("failed to check child categories: %w", err)
	// }
	// if childCount > 0 {
	// 	return fmt.Errorf("cannot delete category with existing child categories")
	// }

	// Delete the category record
	_, err = db.Exec("DELETE FROM categories WHERE category_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}

// GetCategoryByID retrieves a single category by its ID.
//
// This function fetches basic category information without subcategories or products.
//
// Parameters:
//   - id: The category_id to retrieve
//
// Returns:
//   - *dtos.Category: Pointer to category object, nil if not found
//   - error: Database error if query fails (sql.ErrNoRows returns nil, not error)
func GetCategoryByID(db DBExecutor, id string) (*dtos.Category, error) {
	// Query category by ID
	row := db.QueryRow(`
		SELECT category_id, name, parent_category_id, description
		FROM categories
		WHERE category_id = ?`, id)

	var cat dtos.Category
	err := row.Scan(&cat.ID, &cat.Name, &cat.ParentCategoryID, &cat.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Category not found - not an error
		}
		return nil, err
	}
	return &cat, nil
}

// RecordExists checks if a record exists in any table with custom WHERE clause.
//
// This is a generic helper function used by other validation functions.
// It builds a dynamic EXISTS query for any table and condition.
//
// Parameters:
//   - table: The table name to check
//   - clause: The WHERE clause (e.g., "category_id = ?", "name = ? AND status = ?")
//   - args: Variable arguments for the WHERE clause placeholders
//
// Returns:
//   - bool: true if record exists, false if not found
//   - error: Database error if query fails
func RecordExists(db DBExecutor, table, clause string, args ...interface{}) (bool, error) {
	// Build dynamic EXISTS query
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s)", table, clause)

	var exists bool
	// Execute query with provided arguments
	err := db.QueryRow(query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence in %s: %w", table, err)
	}

	return exists, nil
}

// CategoryExists validates that a category exists by category_id.
//
// This function is a convenience wrapper around existence checking that returns
// an error (rather than bool) for easier use in validation flows.
//
// Parameters:
//   - id: The category_id to validate
//
// Returns:
//   - error: nil if category exists, "category not found" error if not found, or database error
//
// Usage:
//   - Use this in validation chains where you want to return early on error
//   - Use isCategoryThere or RecordExists if you need boolean result
func CategoryExists(db DBExecutor, id string) error {
	var exists bool
	// Check category existence
	err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM categories WHERE category_id = ?)`,
		id,
	).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check category existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("category not found")
	}

	return nil
}

// GetAdminCategories retrieves paginated categories with statistics for admin dashboard.
//
// This function provides category listing with product counts, subcategory counts, and pagination.
// It supports optional search by category name and orders results by most recently updated.
//
// Parameters:
//   - page: Page number (1-based)
//   - limit: Number of categories per page
//   - categoryName: Optional search filter (empty string for no filter)
//
// Returns:
//   - []dtos.AdminCategoryData: Array of categories with statistics
//   - *dtos.PaginationMeta: Pagination metadata (page, size, total, has_prev, has_next)
//   - error: Database error if queries fail
//
// Category Statistics:
//   - Type: "Parent" or "Subcategory" based on parent_category_id
//   - Items: Product count (for parent: products in all subcategories; for subcategory: direct products)
//   - Subcategories: Count of child categories (0 for subcategories)
//
// Search:
//   - Case-insensitive LIKE search on category name
//   - Empty categoryName returns all categories
//
// Ordering:
//   - Results ordered by updated_at DESC (most recently updated first)
func GetAdminCategories(db DBExecutor, page, limit int, categoryName, categoryType string) ([]dtos.AdminCategoryData, *dtos.PaginationMeta, error) {
	// Step 1: Get total count for pagination
	var total int
	args := []any{}
	countQuery := "SELECT COUNT(*) FROM categories"
	whereClauses := []string{}

	// Add search filter if category name provided
	if categoryName != "" {
		whereClauses = append(whereClauses, "name LIKE ?")
		args = append(args, "%"+categoryName+"%")
	}

	if categoryType != "" {
		if strings.ToLower(categoryType) == "parent" {
			whereClauses = append(whereClauses, "parent_category_id IS NULL")
		} else if strings.ToLower(categoryType) == "subcategory" {
			whereClauses = append(whereClauses, "parent_category_id IS NOT NULL")
		}
	}

	if len(whereClauses) > 0 {
		countQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	// Calculate pagination offset
	offset := (page - 1) * limit

	// Step 2: Build main query with category statistics
	mainQuery := `
	       SELECT 
		       c.category_id,
		       c.name,
		       c.parent_category_id,
		       c.image,
		       IF(c.parent_category_id IS NULL, 'Parent', 'Subcategory') AS type,
		       CASE 
			       WHEN c.parent_category_id IS NULL 
				       THEN (
					       SELECT COUNT(*) 
					       FROM products p 
					       JOIN categories sc ON sc.category_id = p.category_id
					       WHERE sc.parent_category_id = c.category_id
				       )
			       ELSE (
				       SELECT COUNT(*) 
				       FROM products p 
				       WHERE p.category_id = c.category_id
			       )
		       END AS items,
		       (SELECT COUNT(*) FROM categories sc WHERE sc.parent_category_id = c.category_id) AS subcategories,
		       c.description,
		       CASE WHEN c.parent_category_id IS NOT NULL THEN (SELECT name FROM categories pc WHERE pc.category_id = c.parent_category_id) ELSE NULL END AS parent_name
	       FROM categories c`

	mainWhereClauses := []string{}
	queryArgs := []any{}

	if categoryName != "" {
		mainWhereClauses = append(mainWhereClauses, "c.name LIKE ?")
		queryArgs = append(queryArgs, "%"+categoryName+"%")
	}

	if categoryType != "" {
		if strings.ToLower(categoryType) == "parent" {
			mainWhereClauses = append(mainWhereClauses, "c.parent_category_id IS NULL")
		} else if strings.ToLower(categoryType) == "subcategory" {
			mainWhereClauses = append(mainWhereClauses, "c.parent_category_id IS NOT NULL")
		}
	}

	if len(mainWhereClauses) > 0 {
		mainQuery += " WHERE " + strings.Join(mainWhereClauses, " AND ")
	}

	mainQuery += " ORDER BY c.updated_at DESC LIMIT ? OFFSET ?"
	queryArgs = append(queryArgs, limit, offset)

	rows, err := db.Query(mainQuery, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var categories []dtos.AdminCategoryData
	for rows.Next() {
		var cat dtos.AdminCategoryData
		var parentName sql.NullString
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.ParentID, &cat.Image, &cat.Type, &cat.Items, &cat.Subcategories, &cat.Description, &parentName); err != nil {
			return nil, nil, err
		}
		if parentName.Valid {
			cat.ParentName = &parentName.String
		}
		categories = append(categories, cat)
	}

	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: (total + limit - 1) / limit,
		HasPrev:    page > 1,
		HasNext:    offset+limit < total,
	}
	return categories, pagination, nil
}

// GetCategoriesWithSubCategories retrieves all parent categories with their subcategories.
//
// This function builds a simple two-level category tree without product details.
// It's optimized for dropdown menus and category navigation where product info isn't needed.
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.CategoryWithSubCategories: Array of parent categories with nested subcategory arrays
//   - error: Database error if queries fail
//
// Structure:
//   - Each parent category (parent_category_id IS NULL) contains:
//   - CategoryID: Parent category ID
//   - Category: Parent category name
//   - SubCategory: Array of subcategories with ID and name
func GetCategoriesWithSubCategories(db DBExecutor) ([]dtos.CategoryWithSubCategories, error) {
	// Step 1: Fetch all parent categories (no parent_category_id)
	parentQuery := `
		SELECT category_id, name 
		FROM categories
		WHERE parent_category_id IS NULL		
	`

	rows, err := db.Query(parentQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch parent categories: %w", err)
	}
	defer rows.Close()

	var categories []dtos.CategoryWithSubCategories

	// Iterate through parent categories
	for rows.Next() {
		var cat dtos.CategoryWithSubCategories
		if err := rows.Scan(&cat.CategoryID, &cat.Category); err != nil {
			return nil, fmt.Errorf("failed to scan parent category: %w", err)
		}

		// Step 2: Fetch subcategories for this parent
		subQuery := `
			SELECT category_id, name 
			FROM categories
			WHERE parent_category_id = ?
		`

		subRows, err := db.Query(subQuery, cat.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch subcategories for %s: %w", cat.CategoryID, err)
		}

		defer subRows.Close()

		var subcategories []dtos.SubCategory
		// Collect subcategories for this parent
		for subRows.Next() {
			var sub dtos.SubCategory
			if err := subRows.Scan(&sub.CategoryID, &sub.Category); err != nil {
				return nil, fmt.Errorf("failed to scan subcategory: %w", err)
			}
			subcategories = append(subcategories, sub)
		}

		// Assign subcategories to parent
		cat.SubCategory = subcategories
		categories = append(categories, cat)
	}

	// Check for any row iteration errors
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}
