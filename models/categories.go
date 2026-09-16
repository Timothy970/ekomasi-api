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
	"errors"
	"fmt"
	"log"

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
func GetAllCategories(db DBExecutor, tenantID int) ([]dtos.CategoryData, error) {
	// Step 1: Get top-level categories (parent categories with no parent_category_id)
	rows, err := db.Query("SELECT category_id, name, parent_category_id, image, description FROM categories WHERE parent_category_id IS NULL AND tenant_id = ?", tenantID)
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
		subcategories, err := getSubcategories(db, cat.ID, tenantID)
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
func getSubcategories(db DBExecutor, parentID string, tenantID int) ([]dtos.CategoryData, error) {
	// Query subcategories with this parent_category_id
	rows, err := db.Query("SELECT category_id, name, parent_category_id, image,description FROM categories WHERE parent_category_id = ? AND tenant_id = ?", parentID, tenantID)
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
func AddNewCategory(db DBExecutor, input dtos.CreateCategory, tenantID int) (*dtos.Category, error) {

	// Check if the category name already exists (must be unique for this tenant)
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM categories WHERE name = ? AND tenant_id = ?
		)
	`, input.Name, tenantID).Scan(&exists)

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
		INSERT INTO categories (category_id, name, description, image, tenant_id)
		VALUES (?, ?, ?, ?, ?)`,
			categoryID, input.Name, input.Description, input.Image, tenantID,
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
		INSERT INTO categories (category_id, name, parent_category_id, description, image, tenant_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
			categoryID, input.Name, input.ParentID, input.Description, input.Image, tenantID,
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
