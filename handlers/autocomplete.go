package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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
func AutoCompleteHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get query parameters
	query := c.Query("q")
	limitStr := c.Query("size")

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
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to fetch autocomplete suggestions",
					Code:        http.StatusInternalServerError,
				},
				Message:   "Failed to fetch autocomplete suggestions: " + err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return
		}

		utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
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
			Request:   c.Request,
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
