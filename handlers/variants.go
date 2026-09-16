// Package handlers provides HTTP request handlers for product variant management.
// This file contains handlers for managing product variants (like size, color, material)
// in the e-commerce platform. Variants allow products to have multiple options while
// maintaining shared base information. Essential for inventory management and customer choice.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// variantWithID is a constant prefix for variant-related log messages
var variantWithID = "Variant with ID "

// CreateVariant creates a new product variant type.
// Admin-only operation for defining variant categories (e.g., Size, Color, Material).
// Variants can then be associated with products to provide customer options.
//
// @Summary      Create product variant
// @Description  Create a new product variant type like size, color, or material (admin only)
// @Tags         Variants
// @Accept       json
// @Produce      json
// @Param        variant  body      dtos.VariantRequest     true  "Variant details"
// @Success      200      {object}  map[string]interface{}    "Variant created successfully"
// @Failure      400      {object}  dtos.ErrorResponse      "Invalid request or creation failed"
// @Failure      401      {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/products/variants [post]
func CreateVariant(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can create variants)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with variant details
	req, ok := DecodeRequestBody[dtos.VariantRequest](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (name, type, etc.)
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Create variant record in database
	_, err := models.CreateVariant(models.DB, *req)
	if err != nil {
		// Variant creation failed (duplicate, constraint violation, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create variant",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate variants cache to ensure fresh data
	_ = utils.DeleteCache("all_variants")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Variant created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Variant created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetVariantProductsHandler retrieves products filtered by specific variant combinations.
// Allows customers to find products matching their desired variant selections (e.g., Red + Large).
// Returns paginated product list with matching variant criteria.
//
// @Summary      Get products by variant criteria
// @Description  Retrieve products matching specified variant combinations with pagination
// @Tags         Variants
// @Produce      json
// @Param        variants  query     array                    false  "Array of variant JSON objects"
// @Param        page      query     int                      false  "Page number (default: 1)"
// @Param        limit     query     int                      false  "Items per page (default: 10)"
// @Success      200       {object}  map[string]interface{}   "Products matching variants with pagination"
// @Failure      400       {object}  dtos.ErrorResponse       "Invalid variant JSON or query failed"
// @Router       /api/products/variants-products [get]
func GetVariantProductsHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract variant filter parameters from query string
	variantParams := c.Request.URL.Query()["variants"]
	var variants []dtos.Variant
	// Parse each variant JSON string into structured objects
	for _, v := range variantParams {
		var variant dtos.Variant
		if err := json.Unmarshal([]byte(v), &variant); err != nil {
			// Invalid variant JSON format
			c.String(http.StatusBadRequest, "invalid variant JSON")
			return
		}
		variants = append(variants, variant)
	}
	// Parse pagination parameters with defaults
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1 // Default to first page
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = 10 // Default to 10 items per page
	}

	// Fetch products matching variant criteria with pagination
	variant, pagination, err := models.GetVariantsWithProductsPaginated(models.DB, variants, page, limit)
	if err != nil {
		// Database query failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get products variants",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Construct response with products and pagination metadata
	response := struct {
		Variant    []*dtos.VariantWithProducts `json:"variant"`
		Pagination *dtos.PaginationMeta        `json:"pagination"`
	}{
		Variant:    variant,
		Pagination: pagination,
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Products variants fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Products variants",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetVariant retrieves detailed information for a specific variant type.
// Returns variant details including name, type, and available options.
// Public endpoint for displaying variant information on product pages.
//
// @Summary      Get variant by ID
// @Description  Retrieve detailed information for a specific product variant type
// @Tags         Variants
// @Produce      json
// @Param        variant_id  path      string               true  "Variant ID"
// @Success      200         {object}  dtos.Variant         "Variant details"
// @Failure      404         {object}  dtos.ErrorResponse   "Variant not found"
// @Router       /api/products/variants/{variant_id} [get]
func GetVariant(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract variant ID from URL path parameters
	id := c.Param("variant_id")

	// Fetch specific variant details from database
	variant, err := models.GetVariant(models.DB, id)
	if err != nil {
		// Variant not found or database error
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch variant with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: variantWithID + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   variant,
		Message:   "Variant fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListVariants retrieves all available product variant types.
// Returns grouped variants (e.g., all sizes, all colors) with Redis caching.
// Public endpoint for displaying filter options on product listing pages.
//
// @Summary      List all variant types
// @Description  Retrieve all product variant types grouped by category with caching
// @Tags         Variants
// @Produce      json
// @Success      200  {object}  []dtos.GroupedVariants   "Grouped variants list"
// @Failure      400  {object}  dtos.ErrorResponse       "Failed to list variants"
// @Router       /api/products/variants [get]
func ListVariants(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	var variants []dtos.GroupedVariants
	var cachedVariants []dtos.GroupedVariants
	// Attempt to retrieve variants from Redis cache
	_ = utils.GetCache("all_variants", &cachedVariants)
	// If cache miss, fetch from database
	if cachedVariants == nil {
		var err error
		// Fetch all variants grouped by type from database
		variants, err = models.ListVariants(models.DB)
		if err != nil {
			// Database query failed
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to list variants",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		// Store results in Redis cache for faster subsequent requests
		_ = utils.SetCache("all_variants", &cachedVariants)
	} else {
		// Cache hit - use cached data
		variants = cachedVariants
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Variants fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   variants,
		Message:   "Variants fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
