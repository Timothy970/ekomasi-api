package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"

	"github.com/teris-io/shortid"
)

func GetAllCategories() ([]dtos.CategoryData, error) {
	// 1. Get top-level categories
	rows, err := DB.Query("SELECT category_id, name, parent_category_id, description FROM categories WHERE parent_category_id IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []dtos.CategoryData
	for rows.Next() {
		var cat dtos.CategoryData
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.ParentCategoryID, &cat.Description); err != nil {
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
	rows, err := DB.Query("SELECT category_id, name, parent_category_id, description FROM categories WHERE parent_category_id = ?", parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []dtos.CategoryData
	for rows.Next() {
		var sub dtos.CategoryData
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.ParentCategoryID, &sub.Description); err != nil {
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
		INSERT INTO categories (category_id, name, description)
		VALUES (?, ?, ?)`,
			categoryID, input.Name, categoryID,
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
		INSERT INTO categories (category_id, name, parent_category_id, description)
		VALUES (?, ?, ?, ?)`,
			categoryID, input.Name, input.ParentID, input.Description,
		)
	}
	return &dtos.Category{
		ID:               categoryID,
		Name:             input.Name,
		ParentCategoryID: &categoryID,
		Description:      input.Description,
	}, err
}
func UpdateCategory(id string, input dtos.CreateCategory) (*dtos.Category, error) {
	// 1. Check if the category with the given ID exists
	err := CategoryExists(id)
	if err != nil {
		return nil, err
	}

	// 2. Check if another category with the same name exists
	var nameExists bool
	err = DB.QueryRow(`
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
	_, err = DB.Exec(`
		UPDATE categories
		SET name = ?, parent_category_id = ?, description = ?
		WHERE category_id = ?`,
		input.Name, id, input.Description, id,
	)

	return &dtos.Category{
		ID:               id,
		Name:             input.Name,
		ParentCategoryID: &id,
		Description:      input.Description,
	}, err
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
