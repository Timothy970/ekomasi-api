// Package handlers provides HTTP request handlers for product bundle management.
// This file contains handlers for CRUD operations on product bundles, which are
// collections of multiple products sold together as a package at a special price.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
func GetBundleProductsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	page, limit := parsePagination(c.Query("page"), c.Query("size"))

	bundles, pagination, err := models.GetBundleProducts(models.DB, limit, page)
	if err != nil {
		// Return error response if database query fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch bundles",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response with bundles and pagination metadata
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Bundles fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"bundles": bundles, "pagination": pagination},
		Message:   "Bundles fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success 200 {object} map[string]any "Bundle retrieved successfully"
// @Failure 404 {object} map[string]any "Bundle not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/bundles/{bundle_id} [get]
// @Security BearerAuth
func GetBundleByIDProductsHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract bundle ID from URL path parameters
	bundleID := c.Param("bundle_id")

	// Retrieve bundle from database by ID
	bundle, err := models.GetBundleByIDProducts(models.DB, bundleID)
	if err != nil {
		// Return error response if bundle not found or query fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch bundle",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response with bundle details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Bundle fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   bundle,
		Message:   "Bundle fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success 201 {object} map[string]any "Bundle created successfully"
// @Failure 400 {object} map[string]any "Invalid request or validation failed"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/bundles [post]
// @Security BearerAuth
func CreateBundleHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}

	// Retrieve authenticated admin user from context
	authuser, _ := middleware.UserFromContext(c.Request.Context())

	// Parse multipart form data (max 20MB for image upload)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		// Return error if form parsing fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse form data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Extract image file from multipart form (required field)
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		// Return error if image is not provided
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Image is required when creating a bundle",
				Code:        http.StatusBadRequest,
			},
			Message:   "Image is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}
	defer file.Close()

	// Upload image to Google Cloud Storage
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		// Return error if image upload fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Parse form values
	bundlePrice, _ := strconv.ParseFloat(c.Request.FormValue("bundle_price"), 64)
	stockQuantity, _ := strconv.Atoi(c.Request.FormValue("stock_quantity"))

	// Construct bundle object from form data
	req := &dtos.Bundle{
		Name:           c.Request.FormValue("bundle_name"),
		Description:    c.Request.FormValue("bundle_description"),
		Price:          bundlePrice,
		Image:          url,
		Products:       parseBundleProducts(c.Request.FormValue("products")),
		CompareAtPrice: parseCompareAtPrice(c.Request.FormValue("compare_at_price")),
		StockQuantity:  stockQuantity,
	}

	// Validate the bundle data according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	// Create bundle in database associated with admin user
	err = models.CreateBundle(models.DB, *req, authuser.ID)
	if err != nil {
		// Return error if bundle creation fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create product bundle",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached bundle data to ensure consistency
	utils.DeleteCacheByPrefix("products_bundles_page_")
	utils.DeleteCacheByPrefix("bundles_pagination_page_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product bundle created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Created product bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}
