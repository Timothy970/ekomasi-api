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
	"ekomasi_backend/dtos"
	"fmt"
)

// GetCategoriesWithSubCategories retrieves all parent categories with their subcategories.
func GetCategoriesWithSubCategories(db DBExecutor, tenantID int) ([]dtos.CategoryWithSubCategories, error) {
	// Step 1: Fetch all parent categories (no parent_category_id)
	parentQuery := `
		SELECT category_id, name 
		FROM categories
		WHERE parent_category_id IS NULL AND tenant_id = ?
	`

	rows, err := db.Query(parentQuery, tenantID)
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
			WHERE parent_category_id = ? AND tenant_id = ?
		`

		subRows, err := db.Query(subQuery, cat.CategoryID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch subcategories for %s: %w", cat.CategoryID, err)
		}

		var subcategories []dtos.SubCategory
		// Collect subcategories for this parent
		for subRows.Next() {
			var sub dtos.SubCategory
			if err := subRows.Scan(&sub.CategoryID, &sub.Category); err != nil {
				subRows.Close()
				return nil, fmt.Errorf("failed to scan subcategory: %w", err)
			}
			subcategories = append(subcategories, sub)
		}
		subRows.Close()

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
