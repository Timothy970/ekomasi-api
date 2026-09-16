// Package handlers provides HTTP request handlers for product variant management.
// This file contains handlers for managing product variants (like size, color, material)
// in the e-commerce platform. Variants allow products to have multiple options while
// maintaining shared base information. Essential for inventory management and customer choice.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
// @Success      200         {object}  map[string]any    "Variant updated successfully"
// @Failure      400         {object}  dtos.ErrorResponse      "Invalid request or update failed"
// @Failure      401         {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/products/variants/{variant_id} [patch]
func UpdateVariant(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can update variants)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with updated variant data
	req, ok := DecodeRequestBody[dtos.VariantRequest](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract variant ID from URL path parameters
	id := c.Param("variant_id")

	// Update variant information in database
	if err := models.UpdateVariantByID(models.DB, id, *req); err != nil {
		// Variant update failed (not found, constraint violation, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update variant with ID " + id,
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
			Description: variantWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Variants updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200         {object}  map[string]any    "Variant deleted successfully"
// @Failure      400         {object}  dtos.ErrorResponse      "Delete failed"
// @Failure      401         {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/products/variants/{variant_id} [delete]
func DeleteVariant(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can delete variants)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract variant ID from URL path parameters
	id := c.Param("variant_id")
	// Delete variant from database (may be soft delete)
	if err := models.DeleteVariantByID(models.DB, id); err != nil {
		// Variant deletion failed (not found, has dependencies, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete variant with ID " + id,
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
			Description: variantWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Variants deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200         {object}  map[string]any         "Product added to variant successfully"
// @Failure      400         {object}  dtos.ErrorResponse           "Invalid request or association failed"
// @Failure      401         {object}  dtos.ErrorResponse           "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/add-products/variants/{variant_id} [post]
func AddProductVariant(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can add product variants)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with product-variant association details
	req, ok := DecodeRequestBody[dtos.ProductVariantRequest](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (product_id, variant_value, etc.)
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract variant ID from URL path parameters
	id := c.Param("variant_id")

	// Create product-variant association in database
	if err := models.AddProductVariant(models.DB, id, *req); err != nil {
		// Association failed (duplicate, invalid IDs, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product to variant with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate product caches to reflect new variant associations
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product added to variant successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product added to variant successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200         {object}  map[string]any    "Product removed from variant successfully"
// @Failure      400         {object}  dtos.ErrorResponse      "Remove operation failed"
// @Failure      401         {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/remove-products/variants/{variant_id}/{product_id} [post]
func RemoveProductVariant(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can remove product variants)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Extract product and variant IDs from URL path parameters
	productID := c.Param("product_id")
	variantID := c.Param("variant_id")
	// Remove product-variant association from database
	if err := models.RemoveProductVariant(models.DB, productID, variantID); err != nil {
		// Remove operation failed (association not found or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to remove product with ID " + productID + " from variant with ID " + variantID,
				Code:        http.StatusBadRequest,
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
			Description: "Product with ID " + productID + " removed from variant with ID " + variantID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product removed from variant successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
