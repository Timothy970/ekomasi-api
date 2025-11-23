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

// const (
// 	SortPriceHighToLow   = "price:high-to-low"
// 	SortPriceLowToHigh   = "price:low-to-high"
// 	SortDateOldToNew     = "date:old-to-new"
// 	SortDateNewToOld     = "date:new-to-old"
// 	SortFeatured         = "featured"
// 	SortBestSellers      = "best_sellers"
// 	SortAlphabeticallyAZ = "alphabetically:a-z"
// 	SortAlphabeticallyZA = "alphabetically:z-a"
// )

func AdminSearchProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Parse query parameters
	query := r.URL.Query()

	// Parse variants (multiple variant parameters)
	var variants []dtos.VariantFilter
	variantParams := query["variant"] // This gets all values for "variant" parameter

	for _, variantParam := range variantParams {
		if variantParam != "" {
			parts := strings.Split(variantParam, "---")
			if len(parts) == 2 {
				variants = append(variants, dtos.VariantFilter{
					Type:  parts[0],
					Value: parts[1],
				})
			}
		}
	}

	searchParams := dtos.SearchParams{
		Q:            query.Get("q"), // New search query parameter
		CategoryName: query.Get("category_name"),
		ProductName:  query.Get("product_name"),
		Variants:     variants, // Now supports multiple variants
		SortBy:       query.Get("sort_by"),
		SKU:          query.Get("sku"),
		Tag:          query.Get("tag"),
		MaxPrice:     getFloatQueryParam(query, "maxPrice"),
		MinPrice:     getFloatQueryParam(query, "minPrice"),
		StartDate:    query.Get("start_date"),
		EndDate:      query.Get("end_date"),
	}
	log.Printf("search params: %+v", searchParams)
	// Parse pagination
	page, limit := parsePagination(query.Get("page"), query.Get("size"))
	searchParams.Page = page
	searchParams.Limit = limit

	// Validate sort parameter
	if searchParams.SortBy != "" {
		validSorts := map[string]bool{
			SortPriceHighToLow:   true,
			SortPriceLowToHigh:   true,
			SortDateOldToNew:     true,
			SortDateNewToOld:     true,
			SortFeatured:         true,
			SortBestSellers:      true,
			SortAlphabeticallyAZ: true,
			SortAlphabeticallyZA: true,
		}
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

	// Perform search
	products, pagination, err := models.SearchProducts(searchParams, true)
	if err != nil {
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"products":   products,
			"pagination": pagination,
			"filters": map[string]interface{}{
				"q":             searchParams.Q,
				"category_name": searchParams.CategoryName,
				"product_name":  searchParams.ProductName,
				"variants":      searchParams.Variants, // Now shows all variants
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
