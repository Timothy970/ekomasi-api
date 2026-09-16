// Package models provides the category management functionality for the Ekomasi e-commerce platform.
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
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"strings"
)

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
	if input.Image != nil && *input.Image != "" {
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
func GetAdminCategories(db DBExecutor, tenantID int, page, limit int, categoryName, categoryType string) ([]dtos.AdminCategoryData, *dtos.PaginationMeta, error) {
	// Step 1: Get total count for pagination
	var total int
	args := []any{tenantID}
	countQuery := "SELECT COUNT(*) FROM categories WHERE tenant_id = ?"
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
		countQuery += " AND " + strings.Join(whereClauses, " AND ")
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
					       WHERE sc.parent_category_id = c.category_id AND sc.tenant_id = c.tenant_id
				       )
			       ELSE (
				       SELECT COUNT(*) 
				       FROM products p 
				       WHERE p.category_id = c.category_id AND p.tenant_id = c.tenant_id
			       )
		       END AS items,
		       (SELECT COUNT(*) FROM categories sc WHERE sc.parent_category_id = c.category_id AND sc.tenant_id = c.tenant_id) AS subcategories,
		       c.description,
		       CASE WHEN c.parent_category_id IS NOT NULL THEN (SELECT name FROM categories pc WHERE pc.category_id = c.parent_category_id AND pc.tenant_id = c.tenant_id) ELSE NULL END AS parent_name
	       FROM categories c`

	mainWhereClauses := []string{}
	queryArgs := []any{tenantID}

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

	mainQuery += " WHERE c.tenant_id = ?"
	if len(mainWhereClauses) > 0 {
		mainQuery += " AND " + strings.Join(mainWhereClauses, " AND ")
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
