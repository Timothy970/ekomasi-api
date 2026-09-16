package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

func AutoCompleteSearch(query string, limit int) ([]dtos.AutoCompleteResult, error) {
	// Convert query to lowercase with SQL LIKE wildcards for pattern matching
	searchTerm := "%" + strings.ToLower(query) + "%"

	// Initialize concurrent search infrastructure
	var wg sync.WaitGroup                 // Coordinates goroutine completion
	var mu sync.Mutex                     // Protects results array from race conditions
	var results []dtos.AutoCompleteResult // Aggregated search results

	// Launch concurrent product search goroutine
	wg.Add(1)
	go func() {
		defer wg.Done() // Signal completion when goroutine finishes
		// Search products by name, description, SKU with popularity ranking
		productResults, err := searchProductsAutoCompleteEnhanced(searchTerm, limit)
		if err == nil {
			// Thread-safe append to shared results array
			mu.Lock()
			results = append(results, productResults...)
			mu.Unlock()
		}
		// Errors are silently ignored to prevent one failed search from blocking autocomplete
	}()

	// Launch concurrent category search goroutine
	wg.Add(1)
	go func() {
		defer wg.Done() // Signal completion when goroutine finishes
		// Search categories and subcategories by name and description
		categoryResults, err := searchCategoriesAutoComplete(searchTerm, limit)
		if err == nil {
			// Thread-safe append to shared results array
			mu.Lock()
			results = append(results, categoryResults...)
			mu.Unlock()
		}
		// Errors are silently ignored to prevent one failed search from blocking autocomplete
	}()

	// Variant search is currently disabled (uncomment to enable)
	// This would search product variants by variant_type and name
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

	// Wait for all concurrent searches to complete
	wg.Wait()

	// Sort results by relevance: prioritize items whose name contains the query
	// Items with query in name appear before items with query only in description
	sort.Slice(results, func(i, j int) bool {
		return strings.Contains(strings.ToLower(results[i].Name), strings.ToLower(query)) &&
			!strings.Contains(strings.ToLower(results[j].Name), strings.ToLower(query))
	})

	// Apply final limit to total aggregated results
	if len(results) > limit {
		results = results[:limit]
	}
	// Append fallback search option for full search page
	// This allows users to see all results beyond the autocomplete limit
	results = append(results, dtos.AutoCompleteResult{
		ID:          "search",
		Type:        "search",
		Name:        query,
		DisplayName: fmt.Sprintf("Search for \"%s\"", query),
		Link:        fmt.Sprintf("/search?q=%s", url.QueryEscape(query)), // URL-encoded query
	})

	return results, nil
}

// searchProductsAutoCompleteEnhanced searches products with popularity-based ranking.
//
// This function searches products by name, description, and SKU, then ranks results by:
//  1. Relevance (name matches > description matches > SKU matches)
//  2. Popularity (total quantity sold across all orders)
//  3. Alphabetical order
//
// It includes product images via LEFT JOIN and calculates popularity from order history.
//
// Parameters:
//   - searchTerm: The search pattern with SQL wildcards (e.g., "%laptop%")
//   - limit: Maximum number of product results to return
//
// Returns:
//   - []dtos.AutoCompleteResult: Array of product results with ID, name, image URL, and link
//   - error: Database error if query fails, nil on success
//
// Query Performance:
//   - Uses LEFT JOINs to include products without images or order history
//   - Groups by product to calculate total popularity
//   - COALESCE ensures products with no orders show 0 popularity instead of NULL
func searchProductsAutoCompleteEnhanced(searchTerm string, limit int) ([]dtos.AutoCompleteResult, error) {
	// SQL query with relevance ranking and popularity calculation
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

	// Execute query with search term repeated for WHERE and ORDER BY clauses
	// WHERE: 3 parameters (name, description, sku)
	// ORDER BY CASE: 3 parameters (name, description, sku)
	// Total: 6 searchTerm parameters + 1 limit parameter
	rows, err := DB.Query(query,
		searchTerm, searchTerm, searchTerm, // WHERE clause
		searchTerm, searchTerm, searchTerm, // ORDER BY CASE clause
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dtos.AutoCompleteResult
	// Iterate through query results and build autocomplete suggestions
	for rows.Next() {
		// Handle nullable fields (imageURL might be NULL if product has no images)
		var productID, name, imageURL sql.NullString
		var popularity int

		// Scan row into variables
		if err := rows.Scan(&productID, &name, &imageURL, &popularity); err != nil {
			continue // Skip malformed rows
		}

		// Only include results with valid product ID and name
		if productID.Valid && name.Valid {
			results = append(results, dtos.AutoCompleteResult{
				Type:        "product",
				ID:          productID.String,
				Name:        name.String,
				DisplayName: name.String,
				ImageURL:    imageURL.String, // Empty string if NULL
				Link:        fmt.Sprintf("/product/%s", productID.String),
			})
		}
	}

	return results, nil
}

// searchCategoriesAutoComplete searches categories and subcategories by name and description.
//
// This function differentiates between top-level categories (parent_category_id is NULL)
// and subcategories (parent_category_id has a value), returning appropriate type and links.
//
// Parameters:
//   - searchTerm: The search pattern with SQL wildcards (e.g., "%electronics%")
//   - limit: Maximum number of category results to return
//
// Returns:
//   - []dtos.AutoCompleteResult: Array of category/subcategory results with ID, name, type, and link
//   - error: Database error if query fails, nil on success
//
// Result Types:
//   - "category": Top-level categories (parent_category_id is NULL), link: /category/{id}
//   - "subcategory": Child categories (parent_category_id has value), link: /subcategory/{id}
//
// Ranking Strategy:
//   - Name matches ranked higher than description matches
//   - Within same rank, sorted alphabetically by name
