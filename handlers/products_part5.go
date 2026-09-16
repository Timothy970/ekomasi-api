package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	SortPriceHighToLow   = "price:high-to-low"
	SortPriceLowToHigh   = "price:low-to-high"
	SortDateOldToNew     = "date:old-to-new"
	SortDateNewToOld     = "date:new-to-old"
	SortFeatured         = "featured"
	SortBestSellers      = "best_sellers"
	SortAlphabeticallyAZ = "alphabetically:a-z"
	SortAlphabeticallyZA = "alphabetically:z-a"
)

// ParseSearchParams extracts and validates search query parameters from an HTTP request.
// It handles complex parameters like variants (color---red, size---L), price ranges, sorting, and pagination.
func ParseSearchParams(c *gin.Context, start time.Time, requestSummary string) (*dtos.SearchParams, bool) {
	// Extract all query parameters
	query := c.Request.URL.Query()

	// Parse variants (e.g., variant=color---red&variant=size---L)
	var variants []dtos.VariantFilter
	for _, variantParam := range query["variant"] {
		// Skip empty variant parameters
		if variantParam == "" {
			continue
		}
		// Split variant into type and value using "---" delimiter
		parts := strings.Split(variantParam, "---")
		if len(parts) == 2 {
			variants = append(variants, dtos.VariantFilter{
				Type:  parts[0],
				Value: parts[1],
			})
		}
	}

	// Build comprehensive search parameters structure
	searchParams := &dtos.SearchParams{
		Q:            query.Get("q"),                        // General search query
		CategoryName: query.Get("category_name"),            // Filter by category name
		ProductName:  query.Get("product_name"),             // Filter by product name
		Variants:     variants,                              // Variant filters (color, size, etc.)
		SortBy:       query.Get("sort_by"),                  // Sort order
		SKU:          query.Get("sku"),                      // Filter by SKU
		Tag:          query.Get("tag"),                      // Filter by tag
		MaxPrice:     getFloatQueryParam(query, "maxPrice"), // Maximum price filter
		MinPrice:     getFloatQueryParam(query, "minPrice"), // Minimum price filter
	}

	// Parse and add pagination parameters
	searchParams.Page, searchParams.Limit = parsePagination(query.Get("page"), query.Get("size"))

	// Validate sort parameter (returns error response if invalid)
	if !validateSortParam(searchParams.SortBy, c, start, requestSummary) {
		return nil, false
	}

	// Log search parameters for debugging
	log.Printf("search params: %+v", searchParams)
	return searchParams, true
}

// validateSortParam checks if the provided sort parameter is valid.
// It returns true if valid or empty, false if invalid (with error response sent).
func validateSortParam(sortBy string, c *gin.Context, start time.Time, requestSummary string) bool {
	// Empty sort parameter is valid (no sorting applied)
	if sortBy == "" {
		return true
	}

	// Define all valid sort options
	validSorts := map[string]bool{
		"price:high-to-low":  true, // Price descending
		"price:low-to-high":  true, // Price ascending
		"date:old-to-new":    true, // Oldest first
		"date:new-to-old":    true, // Newest first
		"featured":           true, // Featured products
		"best_sellers":       true, // Best selling products
		"alphabetically:a-z": true, // Name A to Z
		"alphabetically:z-a": true, // Name Z to A
	}

	// Check if the provided sort parameter is valid
	if !validSorts[sortBy] {
		// Return error for invalid sort parameter
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: fmt.Sprintf("Invalid sort parameter: %s", sortBy),
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("Invalid sort parameter: %s", sortBy),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return false
	}
	return true
}

// SearchProductsHandler searches for products based on various criteria.
// It returns a list of products and available filters.
//
// @Summary      Search products
// @Description  Search for products using keywords, categories, price range, and other filters
// @Tags         Products
// @Produce      json
// @Param        q            query     string  false  "Search query"
// @Param        category_name query     string  false  "Category name"
// @Param        product_name query     string  false  "Product name"
// @Param        variant      query     string  false  "Variant filter (e.g., color---red)"
// @Param        sort_by      query     string  false  "Sort order"
// @Param        maxPrice     query     number  false  "Maximum price"
// @Param        minPrice     query     number  false  "Minimum price"
// @Param        sku          query     string  false  "SKU"
// @Param        tag          query     string  false  "Tag"
// @Param        page         query     int     false  "Page number"
// @Param        size         query     int     false  "Page size"
// @Success      200          {object}  map[string]any
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      500          {object}  dtos.ErrorResponse
// @Router       /api/products/search [get]
func SearchProductsHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse and validate all search parameters from query string
	searchParams, ok := ParseSearchParams(c, start, requestSummary)
	if !ok {
		// Parsing failed, error response already sent by ParseSearchParams
		return
	}

	// Execute product search with all filters applied (false = include out of stock)
	products, pagination, err := models.SearchProducts(models.DB, *searchParams, false)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to search products",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to search products: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Respond with search results
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"products":   products,
			"pagination": pagination,
			"filters": map[string]any{
				"q":             searchParams.Q,
				"category_name": searchParams.CategoryName,
				"product_name":  searchParams.ProductName,
				"variants":      searchParams.Variants, // Now shows all variants
				"sort_by":       searchParams.SortBy,
				"maxPrice":      searchParams.MaxPrice,
				"minPrice":      searchParams.MinPrice,
				"sku":           searchParams.SKU,
				"tag":           searchParams.Tag,
			},
		},
		Message:   "Products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// getFloatQueryParam extracts and parses a float value from query parameters.
// Returns 0 if the parameter is missing or cannot be parsed.
func getFloatQueryParam(query url.Values, key string) float64 {
	// Get string value from query parameters
	valueStr := query.Get(key)
	if valueStr == "" {
		// Parameter not provided, return 0
		return 0
	}

	// Parse string to float64
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		// Invalid float format, return 0
		return 0
	}
	return value
}

// AddProductFeatures adds features to a product.
// This endpoint is restricted to administrators.
//
// @Summary      Add product features
// @Description  Add features, images, and specifications to a product
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id              path      string  true  "Product ID"
// @Param        image                   formData  file    false "Main feature image"
// @Param        images                  formData  file    false "Additional feature images"
// @Param        header                  formData  string  false "Feature header"
// @Param        description             formData  string  false "Feature description"
// @Param        image_position          formData  string  false "Image position"
// @Param        top_section             formData  string  false "Top section JSON"
// @Param        product_specifications  formData  string  false "Product specifications JSON"
// @Param        design_type             formData  string  false "Design type"
// @Success      201                     {object}  dtos.ProductFeature
// @Failure      400                     {object}  dtos.ErrorResponse
// @Failure      500                     {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/{product_id}/features [post]
func AddProductFeatures(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify admin permissions
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create"); !ok {
		return
	}

	productID := c.Param("product_id")
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	mainImageURL, imageURLs, err := parseFeatureImages(c, start)
	if err != nil {
		return
	}

	topSections, productSpecs, err := parseFeatureJSONFields(c, start)
	if err != nil {
		return
	}

	designType := c.Request.FormValue("design_type")
	imagePosition := c.Request.FormValue("image_position")
	header := c.Request.FormValue("header")
	description := c.Request.FormValue("description")

	//default image position to left if not provided
	if imagePosition == "" {
		imagePosition = "left"
	}

	// Build the request DTO
	req := dtos.ProductFeature{
		Image:                 &mainImageURL,
		Header:                header,
		Description:           description,
		ImagePosition:         imagePosition,
		Images:                &imageURLs,
		TopSection:            &topSections,
		ProductSpecifications: &productSpecs,
		DesignType:            &designType,
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	feature, err := models.AddProductFeature(models.DB, req, productID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product feature: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product feature added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   feature,
		Message:   "Product feature added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
