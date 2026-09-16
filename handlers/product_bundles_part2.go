// Package handlers provides HTTP request handlers for product bundle management.
// This file contains handlers for CRUD operations on product bundles, which are
// collections of multiple products sold together as a package at a special price.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// UpdateBundleHandler updates an existing product bundle.
// This endpoint is restricted to admin users and allows updating bundle details,
// including optional image replacement.
//
// @Summary Update an existing product bundle
// @Description Updates an existing product bundle's details, products, pricing, and optionally the image
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Param bundle_id path string true "Bundle ID"
// @Param bundle_name formData string false "Bundle name"
// @Param bundle_description formData string false "Bundle description"
// @Param bundle_price formData number false "Bundle price"
// @Param image formData file false "Bundle image (optional)"
// @Param products formData string false "JSON array of bundle products"
// @Param keep_selling formData boolean false "Continue selling when out of stock"
// @Param compare_at_price formData number false "Original price before discount"
// @Param stock_quantity formData int false "Stock quantity"
// @Success 200 {object} map[string]interface{} "Bundle updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Bundle not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/bundles/{bundle_id} [patch]
// @Security BearerAuth
func UpdateBundleHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.update")
	if !ok {
		return
	}

	// Extract bundle ID from URL path parameters
	bundleID := c.Param("bundle_id")

	// Parse multipart form (max 20MB for optional image upload)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		// Return error if form parsing fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse form data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to parse form: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Handle optional image upload (if new image provided, upload it)
	var imageURL string
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		// Upload new image to Google Cloud Storage
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			// Return error if image upload fails
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to upload image to storage :" + err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   "Failed to upload image",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return
		}
		imageURL = url
	} else {
		// If no file uploaded, use existing image URL from form value
		imageURL = c.Request.FormValue("image")
	}

	// Construct bundle object from form data using helper functions
	req := &dtos.Bundle{
		Name:           c.Request.FormValue("bundle_name"),
		Description:    c.Request.FormValue("bundle_description"),
		Price:          parseBundlePrice(c.Request.FormValue("bundle_price")),
		Image:          imageURL,
		Products:       parseBundleProducts(c.Request.FormValue("products")),
		CompareAtPrice: parseCompareAtPrice(c.Request.FormValue("compare_at_price")),
		StockQuantity:  parseStockQuantity(c.Request.FormValue("stock_quantity")),
	}

	// Validate the updated bundle data
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	// Update bundle in database
	err := models.UpdateBundle(models.DB, *req, bundleID)
	if err != nil {
		// Return error if bundle update fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product bundle with ID " + bundleID,
				Code:        http.StatusInternalServerError,
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
			Description: "Product bundle with ID " + bundleID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product bundle updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteBundleHandler permanently removes a product bundle from the system.
// This endpoint is restricted to admin users and deletes the bundle and its associations.
//
// @Summary Delete a product bundle
// @Description Permanently deletes a product bundle from the system
// @Tags Admin
// @Produce json
// @Param bundle_id path string true "Bundle ID"
// @Success 200 {object} map[string]interface{} "Bundle deleted successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Bundle not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/bundles/{bundle_id} [delete]
// @Security BearerAuth
func DeleteBundleHandler(c *gin.Context) {
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

	// Delete bundle from database
	if err := models.DeleteBundle(models.DB, bundleID); err != nil {
		// Return error if bundle deletion fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete product bundle with ID " + bundleID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response confirming deletion
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product bundle with ID " + bundleID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product bundle deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AddProductsToBundleHandler adds one or more products to an existing bundle.
// This endpoint is restricted to admin users and allows expanding bundle contents.
//
// @Summary Add products to a bundle
// @Description Adds one or more products to an existing product bundle
// @Tags Admin
// @Accept json
// @Produce json
// @Param bundle_id path string true "Bundle ID"
// @Param request body []dtos.BundleProducts true "Array of products to add with IDs and quantities"
// @Success 200 {object} map[string]interface{} "Products added successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Bundle not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/bundles/{bundle_id}/products [post]
// @Security BearerAuth
func AddProductsToBundleHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}

	// Decode and validate the request body (array of products)
	req, ok := DecodeRequestBody[[]dtos.BundleProducts](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	// Add products to bundle in database
	err := models.AddProductsToBundle(models.DB, *req, c.Param("bundle_id"))
	if err != nil {
		// Return error if adding products fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product(s) to bundle with ID " + c.Param("bundle_id"),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response confirming products added
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product(s) added to bundle with ID " + c.Param("bundle_id") + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product(s) added to bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
