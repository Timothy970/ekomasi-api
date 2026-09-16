package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
)

func searchCategoriesAutoComplete(searchTerm string, limit int) ([]dtos.AutoCompleteResult, error) {
	// SQL query with relevance ranking by match location (name vs description)
	query := `
		SELECT category_id, name, parent_category_id
		FROM categories
		WHERE LOWER(name) LIKE ? OR LOWER(description) LIKE ?
		ORDER BY
			CASE
				WHEN LOWER(name) LIKE ? THEN 1
				WHEN LOWER(description) LIKE ? THEN 2
				ELSE 3
			END,
			name
		LIMIT ?
	`

	// Execute query with search term repeated for WHERE and ORDER BY clauses
	// WHERE: 2 parameters (name, description)
	// ORDER BY CASE: 2 parameters (name, description)
	// Total: 4 searchTerm parameters + 1 limit parameter
	rows, err := DB.Query(query, searchTerm, searchTerm, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dtos.AutoCompleteResult
	// Iterate through query results and build autocomplete suggestions
	for rows.Next() {
		// Handle nullable parent_category_id (NULL for top-level categories)
		var categoryID, name, parentCategoryID sql.NullString

		// Scan row into variables
		if err := rows.Scan(&categoryID, &name, &parentCategoryID); err != nil {
			continue // Skip malformed rows
		}

		// Determine category type based on parent_category_id
		categoryType := "category" // Default to top-level category
		if parentCategoryID.Valid {
			categoryType = "subcategory" // Has parent, so it's a subcategory
		}

		// Only include results with valid category ID and name
		if categoryID.Valid && name.Valid {
			results = append(results, dtos.AutoCompleteResult{
				Type:        categoryType,
				ID:          categoryID.String,
				Name:        name.String,
				DisplayName: name.String,
				Link:        fmt.Sprintf("/%s/%s", categoryType, categoryID.String),
			})
		}
	}

	return results, nil
}

// searchVariantsAutoComplete searches product variants by type and name.
//
// This function is currently DISABLED but available for future use if variant-level
// autocomplete is needed. It searches variants by variant_type (e.g., "Color", "Size")
// and variant name (e.g., "Red", "Large"), including associated product images.
//
// Parameters:
//   - searchTerm: The search pattern with SQL wildcards (e.g., "%red%")
//   - limit: Maximum number of variant results to return
//
// Returns:
//   - []dtos.AutoCompleteResult: Array of variant results with ID, name, display name, and image URL
//   - error: Database error if query fails, nil on success
//
// Display Format:
//   - DisplayName combines type and name: "Color: Red", "Size: Large"
//   - Image retrieved from first associated product via product_variants join
//
// Ranking Strategy:
//   - variant_type matches ranked higher than name matches
//   - Within same rank, sorted by variant_type then name
//
// To enable: Uncomment this function and the corresponding goroutine in AutoCompleteSearch
// func searchVariantsAutoComplete(searchTerm string, limit int) ([]dtos.AutoCompleteResult, error) {
// 	// SQL query with subquery to fetch associated product image
// 	query := `
// 		SELECT v.variant_id, v.variant_type, v.name,
// 		       (SELECT pi.image_url
// 		        FROM product_images pi
// 		        JOIN product_variants pv ON pi.product_id = pv.product_id
// 		        WHERE pv.variant_id = v.variant_id
// 		        LIMIT 1) as image_url
// 		FROM variants v
// 		WHERE LOWER(v.variant_type) LIKE ? OR LOWER(v.name) LIKE ?
// 		ORDER BY
// 			CASE
// 				WHEN LOWER(v.variant_type) LIKE ? THEN 1
// 				WHEN LOWER(v.name) LIKE ? THEN 2
// 				ELSE 3
// 			END,
// 			v.variant_type, v.name
// 		LIMIT ?
// 	`
//
// 	// Execute query with search term repeated for WHERE and ORDER BY clauses
// 	rows, err := DB.Query(query, searchTerm, searchTerm, searchTerm, searchTerm, limit)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
//
// 	var results []dtos.AutoCompleteResult
// 	// Iterate through query results and build autocomplete suggestions
// 	for rows.Next() {
// 		// Handle nullable image_url (might be NULL if no product image found)
// 		var variantID, variantType, name, imageURL sql.NullString
//
// 		// Scan row into variables
// 		if err := rows.Scan(&variantID, &variantType, &name, &imageURL); err != nil {
// 			continue // Skip malformed rows
// 		}
//
// 		// Only include results with valid variant ID, type, and name
// 		if variantID.Valid && variantType.Valid && name.Valid {
// 			// Create combined display name showing variant type and name
// 			displayName := fmt.Sprintf("%s: %s", variantType.String, name.String)
// 			results = append(results, dtos.AutoCompleteResult{
// 				Type:        "variant",
// 				ID:          variantID.String,
// 				Name:        name.String,
// 				DisplayName: displayName, // e.g., "Color: Red"
// 				ImageURL:    imageURL.String, // Empty string if NULL
// 			})
// 		}
// 	}
//
// 	return results, nil
// }
