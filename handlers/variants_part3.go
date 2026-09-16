// Package handlers provides HTTP request handlers for product variant management.
// This file contains handlers for managing product variants (like size, color, material)
// in the e-commerce platform. Variants allow products to have multiple options while
// maintaining shared base information. Essential for inventory management and customer choice.
package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ListProductVariants retrieves all variant options for a specific product.
// Returns variant details showing available choices for the product (e.g., all colors and sizes).
// Public endpoint for displaying product options on detail pages.
//
// @Summary      List product's variants
// @Description  Retrieve all variant options available for a specific product
// @Tags         Variants
// @Produce      json
// @Param        product_id  path      string                    true  "Product ID"
// @Success      200         {object}  map[string]any    "Product variant options"
// @Failure      400         {object}  dtos.ErrorResponse        "Failed to list variants"
// @Router       /api/products/{product_id}/variants [get]
func ListProductVariants(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract product ID from URL path parameters
	productID := c.Param("product_id")
	// Fetch all variant options for the specified product from database
	pv, err := models.ListProductVariants(models.DB, productID)
	if err != nil {
		// Database query failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to list variants for product with ID " + productID,
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
			Description: "Product variants retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   pv,
		Message:   "Product variants",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200  {object}  map[string]any   "Migration completed with rows affected count"
// @Failure      500  {object}  string                   "Migration failed"
// @Router       /api/admin/migrate-image-urls [post]
func MigrateImageURLs(c *gin.Context) {
	// Execute image URL migration service
	rows, err := MigrateImageURLsService()
	if err != nil {
		// Migration failed
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Return success response with affected rows count
	resp := map[string]any{
		"message":       "Image URLs updated successfully",
		"rows_affected": rows,
	}

	c.JSON(http.StatusOK, resp)
}

// MigrateImageURLsService handles the business logic for image URL migration.
// Delegates to the models layer for database operations.
func MigrateImageURLsService() (int64, error) {
	// Execute batch image URL update in database
	return models.UpdateImageURLs(models.DB)
}
