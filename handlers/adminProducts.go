// Package handlers provides HTTP request handlers for the Adenzo backend API.
// This file contains admin-specific product search and management handlers.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"log"
	"net/http"
	"strings"
	"time"
)

// AdminSearchProductsHandler provides advanced product search functionality for admin users.
// This endpoint supports complex filtering, sorting, and pagination of products with
// full access to all product data including inventory and financial information.
//
// @Summary Search products with admin privileges
// @Description Advanced product search with multiple filters, variants, price ranges, and sorting options. Admin version includes all product details.
// @Tags Admin
// @Produce json
// @Param q query string false "General search query (searches across product name, SKU, description)"
// @Param category_name query string false "Filter by category name"
// @Param product_name query string false "Filter by product name"
// @Param variant query string false "Filter by variant (format: type---value, can be specified multiple times)"
// @Param sort_by query string false "Sort order (price_high_to_low, price_low_to_high, date_old_to_new, date_new_to_old, featured, best_sellers, alphabetically_a_z, alphabetically_z_a)"
// @Param sku query string false "Filter by SKU"
// @Param tag query string false "Filter by product tag"
// @Param maxPrice query number false "Maximum price filter"
// @Param minPrice query number false "Minimum price filter"
// @Param start_date query string false "Filter products created after this date"
// @Param end_date query string false "Filter products created before this date"
// @Param page query int false "Page number for pagination (default: 1)"
// @Param size query int false "Number of items per page (default: 10)"
// @Success 200 {object} map[string]interface{} "Products retrieved successfully with pagination and filters"
// @Failure 400 {object} map[string]interface{} "Invalid sort parameter or other validation error"
// @Failure 500 {object} map[string]interface{} "Internal server error during search"
// @Router /api/admin/products/search [get]
// @Security BearerAuth
func AdminSearchProductsHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(r)

	// Extract all query parameters from the URL
	query := r.URL.Query()

	// Parse variant filters (supports multiple variant parameters)
	// Variants are specified as "type---value" format (e.g., "color---red", "size---large")
	var variants []dtos.VariantFilter
	variantParams := query["variant"] // Get all values for "variant" parameter

	// Process each variant parameter
	for _, variantParam := range variantParams {
		if variantParam != "" {
			// Split on "---" delimiter to extract type and value
			parts := strings.Split(variantParam, "---")
			if len(parts) == 2 {
				// Add valid variant filter to the list
				variants = append(variants, dtos.VariantFilter{
					Type:  parts[0],
					Value: parts[1],
				})
			}
		}
	}

	// Build comprehensive search parameters structure
	searchParams := dtos.SearchParams{
		Q:            query.Get("q"),                        // General search query across multiple fields
		CategoryName: query.Get("category_name"),            // Filter by specific category
		ProductName:  query.Get("product_name"),             // Filter by product name
		Variants:     variants,                              // Variant filters (supports multiple)
		SortBy:       query.Get("sort_by"),                  // Sorting preference
		SKU:          query.Get("sku"),                      // Filter by stock keeping unit
		Tag:          query.Get("tag"),                      // Filter by product tag
		MaxPrice:     getFloatQueryParam(query, "maxPrice"), // Maximum price boundary
		MinPrice:     getFloatQueryParam(query, "minPrice"), // Minimum price boundary
		StartDate:    query.Get("start_date"),               // Filter products created after this date
		EndDate:      query.Get("end_date"),                 // Filter products created before this date
	}

	// Log search parameters for debugging and analytics
	log.Printf("search params: %+v", searchParams)

	// Parse and apply pagination parameters
	page, limit := parsePagination(query.Get("page"), query.Get("size"))
	searchParams.Page = page
	searchParams.Limit = limit

	// Validate sort parameter against allowed values
	if searchParams.SortBy != "" {
		// Define map of valid sort options for quick lookup
		validSorts := map[string]bool{
			SortPriceHighToLow:   true, // Sort by price descending
			SortPriceLowToHigh:   true, // Sort by price ascending
			SortDateOldToNew:     true, // Sort by creation date ascending
			SortDateNewToOld:     true, // Sort by creation date descending
			SortFeatured:         true, // Sort by featured products first
			SortBestSellers:      true, // Sort by sales volume
			SortAlphabeticallyAZ: true, // Sort by name A-Z
			SortAlphabeticallyZA: true, // Sort by name Z-A
		}

		// Return error if sort parameter is invalid
		if !validSorts[searchParams.SortBy] {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Invalid sort parameter provided for product search",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid sort parameter",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
	}

	// Execute product search with admin privileges (second parameter = true)
	// Admin search includes all product data including inventory and restricted fields
	products, pagination, err := models.SearchProducts(searchParams, true)
	if err != nil {
		// Return error response if search operation fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to search products",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to search products: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Return success response with products, pagination, and applied filters
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"products":   products,   // Array of product objects
			"pagination": pagination, // Pagination metadata (total, pages, current page)
			"filters": map[string]interface{}{
				// Echo back applied filters for client confirmation
				"q":             searchParams.Q,
				"category_name": searchParams.CategoryName,
				"product_name":  searchParams.ProductName,
				"variants":      searchParams.Variants, // Shows all applied variant filters
				"sort_by":       searchParams.SortBy,
				"maxPrice":      searchParams.MaxPrice,
				"minPrice":      searchParams.MinPrice,
			},
		},
		Message:   "Products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
