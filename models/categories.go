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

func GetAllCategories() ([]dtos.CategoryData, error) {
	// 1. Get top-level categories
	rows, err := DB.Query("SELECT category_id, name, parent_category_id, image, description FROM categories WHERE parent_category_id IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []dtos.CategoryData
	log.Printf("fetching sub categories888888")
	for rows.Next() {
		var cat dtos.CategoryData
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.ParentCategoryID, &cat.Image, &cat.Description); err != nil {
			return nil, err
		}

		// 2. Get subcategories
		subcategories, err := getSubcategories(cat.ID)
		if err != nil {
			return nil, err
		}
		cat.Subcategories = subcategories

		categories = append(categories, cat)
	}

	return categories, nil
}

func getSubcategories(parentID string) ([]dtos.CategoryData, error) {
	rows, err := DB.Query("SELECT category_id, name, parent_category_id, image,description FROM categories WHERE parent_category_id = ?", parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []dtos.CategoryData
	for rows.Next() {
		var sub dtos.CategoryData
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.ParentCategoryID, &sub.Image, &sub.Description); err != nil {
			return nil, err
		}

		// 3. Get products for this subcategory (limit 6)
		products, err := getProductsByCategory(sub.ID)
		if err != nil {
			return nil, err
		}
		sub.Products = products

		subs = append(subs, sub)
	}

	return subs, nil
}

func getProductsByCategory(categoryID string) ([]dtos.ProductData, error) {
	rows, err := DB.Query(`
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
	for rows.Next() {
		var p dtos.ProductData
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}
func isCategoryThere(value string) error {
	exists, err := RecordExists("categories", "category_id = ?", value)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("category not found")
	}
	return nil
}

// Helper model to insert new category to the DB
func AddNewCategory(input dtos.CreateCategory) (*dtos.Category, error) {

	// Check if the category name already exists
	var exists bool
	err := DB.QueryRow(`
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
	categoryID, _ := shortid.Generate()
	// If ParentID is nil, we insert first then set it to self
	if input.ParentID == nil {
		// Insert without parent_id
		//Safe to insert to DB
		_, err := DB.Exec(`
		INSERT INTO categories (category_id, name, description, image)
		VALUES (?, ?, ?, ?)`,
			categoryID, input.Name, categoryID, input.Image,
		)
		if err != nil {
			return nil, err
		}

	} else {
		//Safe to insert to DB
		err := isCategoryThere(*input.ParentID)
		if err != nil {
			return nil, err
		}
		_, err = DB.Exec(`
		INSERT INTO categories (category_id, name, parent_category_id, description, image)
		VALUES (?, ?, ?, ?, ?)`,
			categoryID, input.Name, input.ParentID, input.Description, input.Image,
		)
		if err != nil {
			return nil, err
		}
	}
	return &dtos.Category{
		ID:               categoryID,
		Name:             input.Name,
		ParentCategoryID: input.ParentID,
		Description:      input.Description,
		Image:            input.Image,
	}, err
}
func UpdateCategory(id string, input dtos.CreateCategory) (*dtos.Category, error) {
	// 1. Ensure the category exists
	if err := CategoryExists(id); err != nil {
		return nil, err
	}

	// 2. Check uniqueness of name (if provided)
	if input.Name != "" {
		var nameExists bool
		err := DB.QueryRow(`
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

	// 3. Build update query dynamically
	setClauses := []string{}
	args := []interface{}{}

	if input.Name != "" {
		setClauses = append(setClauses, "name = ?")
		args = append(args, input.Name)
	}
	if input.ParentID != nil {
		setClauses = append(setClauses, "parent_category_id = ?")
		args = append(args, *input.ParentID)
	}
	if input.Description != "" {
		setClauses = append(setClauses, "description = ?")
		args = append(args, input.Description)
	}
	if input.Image != "" { // assuming image is a string, not *string
		setClauses = append(setClauses, "image = ?")
		args = append(args, input.Image)
	}

	// No fields to update
	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add category_id to args
	args = append(args, id)

	query := fmt.Sprintf(`UPDATE categories SET %s WHERE category_id = ?`, strings.Join(setClauses, ", "))

	_, err := DB.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	// 4. Return updated category DTO
	return &dtos.Category{
		ID:               id,
		Name:             input.Name,
		ParentCategoryID: input.ParentID,
		Description:      input.Description,
		Image:            input.Image,
	}, nil
}

func DeleteCategory(id string) error {
	err := CategoryExists(id)
	if err != nil {
		return err
	}

	// Check for child categories
	// var childCount int
	// err = DB.QueryRow("SELECT COUNT(*) FROM categories WHERE parent_category_id = ?", id).Scan(&childCount)
	// if err != nil {
	// 	return fmt.Errorf("failed to check child categories: %w", err)
	// }
	// if childCount > 0 {
	// 	return fmt.Errorf("cannot delete category with existing child categories")
	// }

	// Delete the category itself
	_, err = DB.Exec("DELETE FROM categories WHERE category_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}
func GetCategoryByID(id string) (*dtos.Category, error) {
	row := DB.QueryRow(`
		SELECT category_id, name, parent_category_id, description
		FROM categories
		WHERE category_id = ?`, id)

	var cat dtos.Category
	err := row.Scan(&cat.ID, &cat.Name, &cat.ParentCategoryID, &cat.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // not found
		}
		return nil, err
	}
	return &cat, nil
}
func RecordExists(table, clause string, args ...interface{}) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s)", table, clause)

	var exists bool
	err := DB.QueryRow(query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence in %s: %w", table, err)
	}

	return exists, nil
}
func CategoryExists(id string) error {
	var exists bool
	err := DB.QueryRow(
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

// Retrieves admin categories with pagination
func GetAdminCategories(page, limit int) ([]dtos.AdminCategoryData, *dtos.PaginationMeta, error) {
	//get total count
	var total int
	err := DB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&total)
	if err != nil {
		return nil, nil, err
	}
	offset := (page - 1) * limit
	// Parent Categories have the parent is null
	// Subcategories will have the subcategories count as  _
	// The items count for for parent categories will be the total products in all its subcategories
	rows, err := DB.Query(`
    SELECT 
        c.category_id,
        c.name,
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
        c.description
    FROM categories c
    ORDER BY c.name
    LIMIT ? OFFSET ?`, limit, offset)

	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var categories []dtos.AdminCategoryData
	for rows.Next() {
		var cat dtos.AdminCategoryData
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Items, &cat.Subcategories, &cat.Description); err != nil {
			return nil, nil, err
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

// GetCategoriesWithSubCategories returns all parent categories and their subcategories
func GetCategoriesWithSubCategories() ([]dtos.CategoryWithSubCategories, error) {
	// Fetch all parent categories (no parent_category_id)
	parentQuery := `
		SELECT category_id, name 
		FROM categories
		WHERE parent_category_id IS NULL
	`

	rows, err := DB.Query(parentQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch parent categories: %w", err)
	}
	defer rows.Close()

	var categories []dtos.CategoryWithSubCategories

	for rows.Next() {
		var cat dtos.CategoryWithSubCategories
		if err := rows.Scan(&cat.CategoryID, &cat.Category); err != nil {
			return nil, fmt.Errorf("failed to scan parent category: %w", err)
		}

		// Fetch subcategories for this parent
		subQuery := `
			SELECT category_id, name 
			FROM categories
			WHERE parent_category_id = ?
		`

		subRows, err := DB.Query(subQuery, cat.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch subcategories for %s: %w", cat.CategoryID, err)
		}

		defer subRows.Close()

		var subcategories []dtos.SubCategory
		for subRows.Next() {
			var sub dtos.SubCategory
			if err := subRows.Scan(&sub.CategoryID, &sub.Category); err != nil {
				return nil, fmt.Errorf("failed to scan subcategory: %w", err)
			}
			subcategories = append(subcategories, sub)
		}

		cat.SubCategory = subcategories
		categories = append(categories, cat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}
