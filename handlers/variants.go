// Package handlers provides HTTP request handlers for product variant management.
// This file contains handlers for managing product variants (like size, color, material)
// in the e-commerce platform. Variants allow products to have multiple options while
// maintaining shared base information. Essential for inventory management and customer choice.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
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
func CreateVariant(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can create variants)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with variant details
	req, ok := DecodeRequestBody[dtos.VariantRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (name, type, etc.)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Create variant record in database
	_, err := models.CreateVariant(models.DB, *req)
	if err != nil {
		// Variant creation failed (duplicate, constraint violation, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create variant",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate variants cache to ensure fresh data
	_ = utils.DeleteCache("all_variants")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Variant created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Variant created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func GetVariantProductsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract variant filter parameters from query string
	variantParams := r.URL.Query()["variants"]
	var variants []dtos.Variant
	// Parse each variant JSON string into structured objects
	for _, v := range variantParams {
		var variant dtos.Variant
		if err := json.Unmarshal([]byte(v), &variant); err != nil {
			// Invalid variant JSON format
			http.Error(w, "invalid variant JSON", http.StatusBadRequest)
			return
		}
		variants = append(variants, variant)
	}
	// Parse pagination parameters with defaults
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1 // Default to first page
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 10 // Default to 10 items per page
	}

	// Fetch products matching variant criteria with pagination
	variant, pagination, err := models.GetVariantsWithProductsPaginated(models.DB, variants, page, limit)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get products variants",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Products variants fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Products variants",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func GetVariant(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract variant ID from URL path parameters
	id := mux.Vars(r)["variant_id"]

	// Fetch specific variant details from database
	variant, err := models.GetVariant(models.DB, id)
	if err != nil {
		// Variant not found or database error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch variant with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: variantWithID + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   variant,
		Message:   "Variant fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func ListVariants(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
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
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to list variants",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Store results in Redis cache for faster subsequent requests
		_ = utils.SetCache("all_variants", &cachedVariants)
	} else {
		// Cache hit - use cached data
		variants = cachedVariants
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Variants fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   variants,
		Message:   "Variants fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateVariant modifies an existing product variant type.
// Admin-only operation for updating variant details like name, type, or available options.
// Essential for maintaining accurate product filtering and customer selection options.
//
// @Summary      Update variant
// @Description  Update product variant type information (admin only)
// @Tags         Variants
// @Accept       json
// @Produce      json
// @Param        variant_id  path      string                  true  "Variant ID"
// @Param        variant     body      dtos.VariantRequest     true  "Updated variant details"
// @Success      200         {object}  map[string]interface{}    "Variant updated successfully"
// @Failure      400         {object}  dtos.ErrorResponse      "Invalid request or update failed"
// @Failure      401         {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/products/variants/{variant_id} [patch]
func UpdateVariant(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can update variants)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with updated variant data
	req, ok := DecodeRequestBody[dtos.VariantRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract variant ID from URL path parameters
	id := mux.Vars(r)["variant_id"]

	// Update variant information in database
	if err := models.UpdateVariantByID(models.DB, id, *req); err != nil {
		// Variant update failed (not found, constraint violation, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update variant with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate variants cache to ensure fresh data
	_ = utils.DeleteCache("all_variants")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: variantWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Variants updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteVariant removes a product variant type from the system.
// Admin-only operation for deactivating variant categories.
// May perform soft delete to preserve historical product associations.
//
// @Summary      Delete variant
// @Description  Delete or deactivate a product variant type (admin only)
// @Tags         Variants
// @Produce      json
// @Param        variant_id  path      string                  true  "Variant ID"
// @Success      200         {object}  map[string]interface{}    "Variant deleted successfully"
// @Failure      400         {object}  dtos.ErrorResponse      "Delete failed"
// @Failure      401         {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/products/variants/{variant_id} [delete]
func DeleteVariant(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can delete variants)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract variant ID from URL path parameters
	id := mux.Vars(r)["variant_id"]
	// Delete variant from database (may be soft delete)
	if err := models.DeleteVariantByID(models.DB, id); err != nil {
		// Variant deletion failed (not found, has dependencies, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete variant with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate variants cache to ensure fresh data
	_ = utils.DeleteCache("all_variants")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: variantWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Variants deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// AddProductVariant associates a variant with a specific product.
// Admin-only operation for adding variant options to products (e.g., adding "Red" color to a shirt).
// Creates product-variant relationship with specific values.
//
// @Summary      Add variant to product
// @Description  Associate a variant option with a specific product (admin only)
// @Tags         Variants
// @Accept       json
// @Produce      json
// @Param        variant_id  path      string                       true  "Variant ID"
// @Param        product     body      dtos.ProductVariantRequest   true  "Product variant association details"
// @Success      200         {object}  map[string]interface{}         "Product added to variant successfully"
// @Failure      400         {object}  dtos.ErrorResponse           "Invalid request or association failed"
// @Failure      401         {object}  dtos.ErrorResponse           "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/add-products/variants/{variant_id} [post]
func AddProductVariant(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can add product variants)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with product-variant association details
	req, ok := DecodeRequestBody[dtos.ProductVariantRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (product_id, variant_value, etc.)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract variant ID from URL path parameters
	id := mux.Vars(r)["variant_id"]

	// Create product-variant association in database
	if err := models.AddProductVariant(models.DB, id, *req); err != nil {
		// Association failed (duplicate, invalid IDs, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product to variant with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate product caches to reflect new variant associations
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product added to variant successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product added to variant successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// RemoveProductVariant removes a variant association from a specific product.
// Admin-only operation for removing variant options from products.
// Essential for managing product catalog and variant availability.
//
// @Summary      Remove variant from product
// @Description  Remove variant option association from a specific product (admin only)
// @Tags         Variants
// @Produce      json
// @Param        variant_id  path      string                  true  "Variant ID"
// @Param        product_id  path      string                  true  "Product ID"
// @Success      200         {object}  map[string]interface{}    "Product removed from variant successfully"
// @Failure      400         {object}  dtos.ErrorResponse      "Remove operation failed"
// @Failure      401         {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/remove-products/variants/{variant_id}/{product_id} [post]
func RemoveProductVariant(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can remove product variants)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Extract product and variant IDs from URL path parameters
	productID := mux.Vars(r)["product_id"]
	variantID := mux.Vars(r)["variant_id"]
	// Remove product-variant association from database
	if err := models.RemoveProductVariant(models.DB, productID, variantID); err != nil {
		// Remove operation failed (association not found or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to remove product with ID " + productID + " from variant with ID " + variantID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product with ID " + productID + " removed from variant with ID " + variantID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product removed from variant successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ListProductVariants retrieves all variant options for a specific product.
// Returns variant details showing available choices for the product (e.g., all colors and sizes).
// Public endpoint for displaying product options on detail pages.
//
// @Summary      List product's variants
// @Description  Retrieve all variant options available for a specific product
// @Tags         Variants
// @Produce      json
// @Param        product_id  path      string                    true  "Product ID"
// @Success      200         {object}  map[string]interface{}    "Product variant options"
// @Failure      400         {object}  dtos.ErrorResponse        "Failed to list variants"
// @Router       /api/products/{product_id}/variants [get]
func ListProductVariants(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract product ID from URL path parameters
	productID := mux.Vars(r)["product_id"]
	// Fetch all variant options for the specified product from database
	pv, err := models.ListProductVariants(models.DB, productID)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to list variants for product with ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product variants retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   pv,
		Message:   "Product variants",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// MigrateImageURLs is a utility endpoint for batch updating product image URLs.
// Typically used for data migration or URL format changes across the product catalog.
// Consider adding admin authentication if exposed in production.
//
// @Summary      Migrate image URLs
// @Description  Batch update product image URLs (utility endpoint)
// @Tags         Utilities
// @Produce      json
// @Success      200  {object}  map[string]interface{}   "Migration completed with rows affected count"
// @Failure      500  {object}  string                   "Migration failed"
// @Router       /api/admin/migrate-image-urls [post]
func MigrateImageURLs(w http.ResponseWriter, r *http.Request) {
	// Execute image URL migration service
	rows, err := MigrateImageURLsService()
	if err != nil {
		// Migration failed
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response with affected rows count
	resp := map[string]interface{}{
		"message":       "Image URLs updated successfully",
		"rows_affected": rows,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// MigrateImageURLsService handles the business logic for image URL migration.
// Delegates to the models layer for database operations.
func MigrateImageURLsService() (int64, error) {
	// Execute batch image URL update in database
	return models.UpdateImageURLs(models.DB)
}
