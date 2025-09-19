package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

func AutoCompleteSearch(query string, limit int) ([]dtos.AutoCompleteResult, error) {
	searchTerm := "%" + strings.ToLower(query) + "%"

	// Search across products, categories
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []dtos.AutoCompleteResult

	// Search products
	wg.Add(1)
	go func() {
		defer wg.Done()
		productResults, err := searchProductsAutoCompleteEnhanced(searchTerm, limit)
		if err == nil {
			mu.Lock()
			results = append(results, productResults...)
			mu.Unlock()
		}
	}()

	// Search categories
	wg.Add(1)
	go func() {
		defer wg.Done()
		categoryResults, err := searchCategoriesAutoComplete(searchTerm, limit)
		if err == nil {
			mu.Lock()
			results = append(results, categoryResults...)
			mu.Unlock()
		}
	}()

	// // Search variants
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	variantResults, err := searchVariantsAutoComplete(searchTerm, limit)
	// 	if err == nil {
	// 		mu.Lock()
	// 		results = append(results, variantResults...)
	// 		mu.Unlock()
	// 	}
	// }()

	wg.Wait()

	// Sort results by relevance (you can implement more sophisticated ranking)
	sort.Slice(results, func(i, j int) bool {
		return strings.Contains(strings.ToLower(results[i].Name), strings.ToLower(query)) &&
			!strings.Contains(strings.ToLower(results[j].Name), strings.ToLower(query))
	})

	// Limit final results
	if len(results) > limit {
		results = results[:limit]
	}
	results = append(results, dtos.AutoCompleteResult{
		ID:          "search",
		Type:        "search",
		Name:        query,
		DisplayName: fmt.Sprintf("Search for \"%s\"", query),
		Link:        fmt.Sprintf("/search?q=%s", url.QueryEscape(query)),
	})

	return results, nil
}

func searchProductsAutoCompleteEnhanced(searchTerm string, limit int) ([]dtos.AutoCompleteResult, error) {
	query := `
		SELECT 
		    p.product_id, 
		    p.name, 
		    pi.url,
		    COALESCE(SUM(oi.quantity), 0) as popularity
		FROM products p
		LEFT JOIN product_images pi 
		    ON pi.product_id = p.product_id
		LEFT JOIN order_items oi
		    ON oi.product_id = p.product_id
		WHERE LOWER(p.name) LIKE ? 
		   OR LOWER(p.description) LIKE ? 
		   OR LOWER(p.sku) LIKE ?
		GROUP BY p.product_id, p.name, pi.url
		ORDER BY 
			CASE 
				WHEN LOWER(p.name) LIKE ? THEN 1
				WHEN LOWER(p.description) LIKE ? THEN 2
				WHEN LOWER(p.sku) LIKE ? THEN 3
				ELSE 4
			END,
			popularity DESC,
			p.name
		LIMIT ?
	`

	rows, err := DB.Query(query,
		searchTerm, searchTerm, searchTerm,
		searchTerm, searchTerm, searchTerm,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dtos.AutoCompleteResult
	for rows.Next() {
		var productID, name, imageURL sql.NullString
		var popularity int
		if err := rows.Scan(&productID, &name, &imageURL, &popularity); err != nil {
			continue
		}

		if productID.Valid && name.Valid {
			results = append(results, dtos.AutoCompleteResult{
				Type:        "product",
				ID:          productID.String,
				Name:        name.String,
				DisplayName: name.String,
				ImageURL:    imageURL.String,
				Link:        fmt.Sprintf("/product/%s", productID.String),
			})
		}
	}

	return results, nil
}

func searchCategoriesAutoComplete(searchTerm string, limit int) ([]dtos.AutoCompleteResult, error) {
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

	rows, err := DB.Query(query, searchTerm, searchTerm, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dtos.AutoCompleteResult
	for rows.Next() {
		var categoryID, name, parentCategoryID sql.NullString
		if err := rows.Scan(&categoryID, &name, &parentCategoryID); err != nil {
			continue
		}
		categoryType := "category"
		if parentCategoryID.Valid {
			categoryType = "subcategory"
		}
		if categoryID.Valid && name.Valid {
			results = append(results, dtos.AutoCompleteResult{
				Type:        categoryType,
				ID:          categoryID.String,
				Name:        name.String,
				DisplayName: "Category: " + name.String,
				Link:        fmt.Sprintf("/%s/%s", categoryType, categoryID.String),
			})
		}
	}

	return results, nil
}

// func searchVariantsAutoComplete(searchTerm string, limit int) ([]dtos.AutoCompleteResult, error) {
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

// 	rows, err := DB.Query(query, searchTerm, searchTerm, searchTerm, searchTerm, limit)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var results []dtos.AutoCompleteResult
// 	for rows.Next() {
// 		var variantID, variantType, name, imageURL sql.NullString
// 		if err := rows.Scan(&variantID, &variantType, &name, &imageURL); err != nil {
// 			continue
// 		}

// 		if variantID.Valid && variantType.Valid && name.Valid {
// 			displayName := fmt.Sprintf("%s: %s", variantType.String, name.String)
// 			results = append(results, dtos.AutoCompleteResult{
// 				Type:        "variant",
// 				ID:          variantID.String,
// 				Name:        name.String,
// 				DisplayName: displayName,
// 				ImageURL:    imageURL.String,
// 			})
// 		}
// 	}

// 	return results, nil
// }
