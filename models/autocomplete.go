// Package models provides the autocomplete search functionality for the Ekomasi e-commerce platform.
//
// This package implements a fast, concurrent autocomplete search system that searches across:
//   - Products: Searches by name, description, SKU with popularity-based ranking
//   - Categories: Searches by name and description, differentiates categories from subcategories
//   - Variants: Commented out variant search functionality (available for future use)
//
// The autocomplete system features:
//   - Concurrent search execution using goroutines for optimal performance
//   - Relevance-based sorting (exact matches prioritized)
//   - Popularity ranking for products based on order history
//   - Configurable result limits per search type
//   - URL-encoded search fallback option
//   - Thread-safe result aggregation using sync.Mutex
//
// Search Strategy:
//   - Case-insensitive matching using LOWER() and LIKE patterns
//   - Exact name matches ranked higher than description matches
//   - Products sorted by relevance first, then popularity, then alphabetically
//   - Categories sorted by relevance, then alphabetically
//   - Results include direct links to product/category pages
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

// AutoCompleteSearch performs a concurrent search across products and categories for autocomplete suggestions.
//
// This function launches multiple goroutines to search different data types simultaneously,
// aggregates the results, sorts them by relevance, and appends a fallback search option.
// It uses sync.WaitGroup for goroutine coordination and sync.Mutex for thread-safe result aggregation.
//
// Parameters:
//   - query: The search term entered by the user (case-insensitive)
//   - limit: Maximum number of results to return per search type (before aggregation)
//
// Returns:
//   - []dtos.AutoCompleteResult: Array of autocomplete suggestions with type, ID, name, display name, image URL, and link
//   - error: Always returns nil (individual search errors are silently ignored)
//
// Result Structure:
//   - Products: Include product_id, name, image URL, popularity-based ranking, and link to /product/{id}
//   - Categories: Include category_id, name, type (category/subcategory), and link to /{type}/{id}
//   - Search Fallback: Appended at the end with link to /search?q={query}
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
