package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// AutoCompleteHandler provides autocomplete suggestions for product searches.
// It caches results for performance.
//
// @Summary      Get autocomplete suggestions
// @Description  Retrieve autocomplete suggestions for products based on a query string
// @Tags         Products
// @Produce      json
// @Param        q     query     string  true   "Search query (min 2 chars)"
// @Param        size  query     int     false  "Number of suggestions (default 10)"
// @Success      200   {object}  dtos.AutoCompleteResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Router       /api/products/autocomplete [get]
func AutoCompleteHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Get query parameters
	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("size")

	// Parse limit with default value
	limit := 10
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Validate query length
	if len(query) >= 2 {
		// Perform autocomplete search
		suggestions, err := AutoCompleteSearchWithCache(query, limit)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to fetch autocomplete suggestions",
					Code:        http.StatusInternalServerError,
				},
				Message:   "Failed to fetch autocomplete suggestions: " + err.Error(),
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
				Description: "Autocomplete suggestions fetched successfully",
				Code:        http.StatusOK,
			},
			Payload: dtos.AutoCompleteResponse{
				Suggestions: suggestions,
				Total:       len(suggestions),
			},
			Message:   "Autocomplete suggestions",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
	}
}

// AutoCompleteSearchWithCache performs a search with caching.
// It checks the cache first, and if not found, queries the database and caches the result.
func AutoCompleteSearchWithCache(query string, limit int) ([]dtos.AutoCompleteResult, error) {
	// Create cache key
	cacheKey := fmt.Sprintf("autocomplete:%s:%d", query, limit)
	var cached []dtos.AutoCompleteResult
	_ = utils.GetCache(cacheKey, &cached)
	// Check cache
	if cached != nil {
		return cached, nil
	}
	// Perform search
	results, err := models.AutoCompleteSearch(query, limit)
	if err != nil {
		return nil, err
	}

	// Cache results for 1 minute
	utils.SetCache(cacheKey, results, 1*time.Minute)

	return results, nil
}
