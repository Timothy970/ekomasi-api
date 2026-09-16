// Package handlers provides HTTP request handlers for product bundle management.
// This file contains handlers for CRUD operations on product bundles, which are
// collections of multiple products sold together as a package at a special price.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RemoveProductsFromBundleHandler removes products from an existing bundle.
// This endpoint is restricted to admin users and allows removing specific products from a bundle.
//
// @Summary Remove products from a bundle
// @Description Removes one or more products from an existing product bundle
// @Tags Admin
// @Accept json
// @Produce json
// @Param bundle_id path string true "Bundle ID"
// @Param request body dtos.AddProductsToBundle true "Product IDs to remove from bundle"
// @Success 200 {object} map[string]any "Products removed successfully"
// @Failure 400 {object} map[string]any "Invalid request or validation failed"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 404 {object} map[string]any "Bundle not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/bundles/{bundle_id}/products [delete]
// @Security BearerAuth
func RemoveProductsFromBundleHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.delete")
	if !ok {
		return
	}

	// Extract bundle ID from URL path parameters
	bundleID := c.Param("bundle_id")

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.AddProductsToBundle](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	// Remove products from bundle in database
	if err := models.RemoveProductsFromBundle(models.DB, *req, bundleID); err != nil {
		// Return error if removing products fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to remove product(s) from bundle with ID " + bundleID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to remove product(s) from bundle",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached bundle data to ensure consistency
	utils.DeleteCacheByPrefix("products_bundles_page_")
	utils.DeleteCacheByPrefix("bundles_pagination_page_")

	// Return success response confirming products removed
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product(s) removed from bundle with ID " + bundleID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product(s) removed from bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
