// Package handlers provides HTTP request handlers for product bundle management.
// This file contains handlers for CRUD operations on product bundles, which are
// collections of multiple products sold together as a package at a special price.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// parseBundleProducts parses the products JSON string from form data
func parseBundleProducts(productsStr string) []dtos.BundleProducts {
	if productsStr == "" {
		return []dtos.BundleProducts{}
	}

	var products []dtos.BundleProducts
	if err := json.Unmarshal([]byte(productsStr), &products); err != nil {
		return []dtos.BundleProducts{}
	}
	return products
}

// parseKeepSelling parses the keep_selling field with default true
func parseKeepSelling(keepSellingStr string) *bool {
	ks := strings.ToLower(keepSellingStr)
	defaultValue := true

	if ks == "" {
		return &defaultValue
	}

	switch ks {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	}

	return &defaultValue
}

// parseCompareAtPrice parses the compare_at_price field
func parseCompareAtPrice(priceStr string) *float64 {
	cp, _ := strconv.ParseFloat(priceStr, 64)
	if cp == 0 {
		return nil
	}
	return &cp
}

// parseBundlePrice parses the bundle_price field
func parseBundlePrice(priceStr string) float64 {
	price, _ := strconv.ParseFloat(priceStr, 64)
	return price
}

// parseStockQuantity parses the stock_quantity field
func parseStockQuantity(quantityStr string) int {
	quantity, _ := strconv.Atoi(quantityStr)
	return quantity
}

// GetBundleProductsHandler retrieves a paginated list of all product bundles.
// Product bundles are collections of products sold together at a special price.
func GetBundleProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	bundles, pagination, err := models.GetBundleProducts(models.DB, limit, page)
	if err != nil {
		// Return error response if database query fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch bundles",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return success response with bundles and pagination metadata
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Bundles fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"bundles": bundles, "pagination": pagination},
		Message:   "Bundles fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// GetBundleByIDProductsHandler retrieves a single product bundle by its unique identifier.
// This endpoint returns detailed information about a specific bundle including all products it contains.
//
// @Summary Get product bundle by ID
// @Description Retrieves detailed information about a specific product bundle by its ID
// @Tags Product Bundles
// @Produce json
// @Param bundle_id path string true "Bundle ID"
// @Success 200 {object} map[string]interface{} "Bundle retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Bundle not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/bundles/{bundle_id} [get]
// @Security BearerAuth
func GetBundleByIDProductsHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract bundle ID from URL path parameters
	bundleID := mux.Vars(r)["bundle_id"]

	// Retrieve bundle from database by ID
	bundle, err := models.GetBundleByIDProducts(models.DB, bundleID)
	if err != nil {
		// Return error response if bundle not found or query fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch bundle",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return success response with bundle details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Bundle fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   bundle,
		Message:   "Bundle fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// CreateBundleHandler creates a new product bundle with image upload.
// This endpoint is restricted to admin users and allows creating bundles with multiple products,
// pricing, images, and inventory settings.
//
// @Summary Create a new product bundle
// @Description Creates a new product bundle with products, pricing, image, and stock quantity
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Param bundle_name formData string true "Bundle name"
// @Param bundle_description formData string false "Bundle description"
// @Param bundle_price formData number true "Bundle price"
// @Param image formData file true "Bundle image"
// @Param products formData string true "JSON array of bundle products with IDs and quantities"
// @Param keep_selling formData boolean false "Continue selling when out of stock (default: true)"
// @Param compare_at_price formData number false "Original price before discount"
// @Param stock_quantity formData int false "Stock quantity"
// @Success 201 {object} map[string]interface{} "Bundle created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/bundles [post]
// @Security BearerAuth
func CreateBundleHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}

	// Retrieve authenticated admin user from context
	authuser, _ := middleware.UserFromContext(r.Context())

	// Parse multipart form data (max 20MB for image upload)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		// Return error if form parsing fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse form data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Extract image file from multipart form (required field)
	file, header, err := r.FormFile("image")
	if err != nil {
		// Return error if image is not provided
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Image is required when creating a bundle",
				Code:        http.StatusBadRequest,
			},
			Message:   "Image is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	defer file.Close()

	// Upload image to Google Cloud Storage
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		// Return error if image upload fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Parse form values
	bundlePrice, _ := strconv.ParseFloat(r.FormValue("bundle_price"), 64)
	stockQuantity, _ := strconv.Atoi(r.FormValue("stock_quantity"))

	// Construct bundle object from form data
	req := &dtos.Bundle{
		Name:           r.FormValue("bundle_name"),
		Description:    r.FormValue("bundle_description"),
		Price:          bundlePrice,
		Image:          url,
		Products:       parseBundleProducts(r.FormValue("products")),
		KeepSelling:    parseKeepSelling(r.FormValue("keep_selling")),
		CompareAtPrice: parseCompareAtPrice(r.FormValue("compare_at_price")),
		StockQuantity:  stockQuantity,
	}

	// Validate the bundle data according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}

	// Create bundle in database associated with admin user
	err = models.CreateBundle(models.DB, *req, authuser.ID)
	if err != nil {
		// Return error if bundle creation fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create product bundle",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached bundle data to ensure consistency
	utils.DeleteCacheByPrefix("products_bundles_page_")
	utils.DeleteCacheByPrefix("bundles_pagination_page_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product bundle created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Created product bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

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
// @Router /api/admin/bundles/{bundle_id} [put]
// @Security BearerAuth
func UpdateBundleHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.update")
	if !ok {
		return
	}

	// Extract bundle ID from URL path parameters
	bundleID := mux.Vars(r)["bundle_id"]

	// Parse multipart form (max 20MB for optional image upload)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		// Return error if form parsing fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse form data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to parse form: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Handle optional image upload (if new image provided, upload it)
	var imageURL string
	if file, header, err := r.FormFile("image"); err == nil {
		defer file.Close()
		// Upload new image to Google Cloud Storage
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			// Return error if image upload fails
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to upload image to storage :" + err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   "Failed to upload image",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return
		}
		imageURL = url
	} else {
		// If no file uploaded, use existing image URL from form value
		imageURL = r.FormValue("image")
	}

	// Construct bundle object from form data using helper functions
	req := &dtos.Bundle{
		Name:           r.FormValue("bundle_name"),
		Description:    r.FormValue("bundle_description"),
		Price:          parseBundlePrice(r.FormValue("bundle_price")),
		Image:          imageURL,
		Products:       parseBundleProducts(r.FormValue("products")),
		KeepSelling:    parseKeepSelling(r.FormValue("keep_selling")),
		CompareAtPrice: parseCompareAtPrice(r.FormValue("compare_at_price")),
		StockQuantity:  parseStockQuantity(r.FormValue("stock_quantity")),
	}

	// Validate the updated bundle data
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}

	// Update bundle in database
	err := models.UpdateBundle(models.DB, *req, bundleID)
	if err != nil {
		// Return error if bundle update fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product bundle with ID " + bundleID,
				Code:        http.StatusInternalServerError,
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
			Description: "Product bundle with ID " + bundleID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product bundle updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func DeleteBundleHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.delete")
	if !ok {
		return
	}

	// Extract bundle ID from URL path parameters
	bundleID := mux.Vars(r)["bundle_id"]

	// Delete bundle from database
	if err := models.DeleteBundle(models.DB, bundleID); err != nil {
		// Return error if bundle deletion fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete product bundle with ID " + bundleID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return success response confirming deletion
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product bundle with ID " + bundleID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product bundle deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func AddProductsToBundleHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}

	// Decode and validate the request body (array of products)
	req, ok := DecodeRequestBody[[]dtos.BundleProducts](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}

	// Add products to bundle in database
	err := models.AddProductsToBundle(models.DB, *req, mux.Vars(r)["bundle_id"])
	if err != nil {
		// Return error if adding products fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product(s) to bundle with ID " + mux.Vars(r)["bundle_id"],
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return success response confirming products added
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product(s) added to bundle with ID " + mux.Vars(r)["bundle_id"] + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product(s) added to bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

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
// @Success 200 {object} map[string]interface{} "Products removed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Bundle not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/bundles/{bundle_id}/products [delete]
// @Security BearerAuth
func RemoveProductsFromBundleHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.delete")
	if !ok {
		return
	}

	// Extract bundle ID from URL path parameters
	bundleID := mux.Vars(r)["bundle_id"]

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.AddProductsToBundle](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}

	// Remove products from bundle in database
	if err := models.RemoveProductsFromBundle(models.DB, *req, bundleID); err != nil {
		// Return error if removing products fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to remove product(s) from bundle with ID " + bundleID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to remove product(s) from bundle",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached bundle data to ensure consistency
	utils.DeleteCacheByPrefix("products_bundles_page_")
	utils.DeleteCacheByPrefix("bundles_pagination_page_")

	// Return success response confirming products removed
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product(s) removed from bundle with ID " + bundleID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product(s) removed from bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
