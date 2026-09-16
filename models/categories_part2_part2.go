package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"strings"
)

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
