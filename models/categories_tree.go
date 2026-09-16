package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"strings"
)

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
	args := []any{}

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
func RecordExists(db DBExecutor, table, clause string, args ...any) (bool, error) {
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
