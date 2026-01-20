// Package handlers provides HTTP request handlers for product management and operations.
// This file contains handlers for product CRUD operations, product search and filtering,
// media uploads, product features, variants, and promotional management.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var (
	productIdRequired     = "Product ID is required"
	Redis                 *redis.Client
	productNotFound       = "Product not found"
	invalidTopSectionJSON = "Invalid top_section JSON: "
)

// GetProductsHandler retrieves a list of products with optional filtering and pagination.
//
// @Summary      Get all products
// @Description  Retrieve a list of products with optional filtering by category, product name, and pagination
// @Tags         Products
// @Produce      json
// @Param        page         query     int     false  "Page number"
// @Param        size         query     int     false  "Page size"
// @Param        category     query     string  false  "Category filter"
// @Param        product      query     string  false  "Product name filter"
// @Param        category_id  query     string  false  "Category ID filter"
// @Success      200          {object}  map[string]interface{}
// @Failure      404          {object}  dtos.ErrorResponse
// @Router       /api/products [get]
func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Parse pagination & filters from query parameters
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Extract optional category name filter
	categoryFilter := r.URL.Query().Get("category")
	// Extract optional product name filter
	productFilter := r.URL.Query().Get("product")
	// Extract optional category ID filter
	categoryID := r.URL.Query().Get("category_id")

	// Fetch products from database with all filters applied
	products, pagination, err := models.GetAllProducts(categoryFilter, productFilter, categoryID, page, limit)

	// Check if database query failed
	if err != nil {
		// Return error response with detailed information
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to retrieve products",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Return successful response with products and pagination metadata
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "All products retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"products":   products,
			"pagination": pagination,
		},
		Message:   "All products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// parsePagination extracts and validates page and limit parameters from query strings.
// It provides default values if parameters are missing or invalid.
// Default page is 1 and default limit is 10.
func parsePagination(pageStr, sizeStr string) (int, int) {
	// Initialize default pagination values
	page := 1
	limit := 10

	// Parse page number if provided
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	// Parse limit (page size) if provided
	if sizeStr != "" {
		if l, err := strconv.Atoi(sizeStr); err == nil {
			limit = l
		}
	}
	return page, limit
}

// GetProductByIDHandler retrieves a single product by its ID.
// It also checks if the product is in the user's wishlist if a token is provided.
//
// @Summary      Get product by ID
// @Description  Retrieve detailed information for a specific product
// @Tags         Products
// @Produce      json
// @Param        product_id  path      string  true  "Product ID"
// @Success      200         {object}  dtos.Product
// @Failure      404         {object}  dtos.ErrorResponse
// @Failure      500         {object}  dtos.ErrorResponse
// @Router       /api/products/{product_id} [get]
func GetProductByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract product ID from URL path parameters
	productID := mux.Vars(r)["product_id"]

	// Fetch product details from database
	product, err := models.GetProductByID(productID)
	if err != nil {
		// Log error and return response
		log.Printf("error fetching product with ID %s: %v", productID, err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetching product with ID " + productID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Error fetching product",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})

		return
	}

	// Check if user is authenticated and update product with wishlist status
	ApplyUserWishlist(r, product.ID, product)

	// Respond with product details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product with ID " + product.ID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   product,
		Message:   "Product fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UploadProductImageHandler handles the upload of product images and videos.
// It supports uploading gallery images, thumbnails, and video links.
// This endpoint is restricted to administrators.
//
// @Summary      Upload product media
// @Description  Upload images or video links for a product
// @Tags         Products
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id  formData  string  true  "Product ID"
// @Param        is_primary  formData  bool    false "Is primary image"
// @Param        video_link  formData  string  false "Video URL"
// @Param        gallery     formData  file    false "Gallery images"
// @Param        thumbnail   formData  file    false "Thumbnail image"
// @Param        video       formData  file    false "Video file"
// @Success      200         {object}  map[string]interface{}
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/product/image [post]
func UploadProductImageHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Parse and validate upload request parameters (product_id, is_primary, video_link)
	productID, isPrimary, videoLink, err := parseUploadRequest(r)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse multipart form" + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Track all successfully uploaded files
	var uploadedResults []map[string]string

	// Handle optional video link (if provided in form data)
	if videoLink != "" {
		// Insert video link directly to database without upload
		if err := models.InsertProductImage(productID, videoLink, "video", isPrimary); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to insert video link for product ID " + productID,
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
	}

	fileTypes := []string{"gallery", "thumbnail", "video"}
	// Check and upload files for each type
	for _, fileType := range fileTypes {
		results, err := handleFileUploads(r, productID, fileType, isPrimary)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to upload " + fileType + " for product ID " + productID,
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
		uploadedResults = append(uploadedResults, results...)
	}

	// Ensure at least one file or link was processed
	if len(uploadedResults) == 0 {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "No files uploaded for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   "No files were received. Please upload at least one file using the keys: 'gallery', 'thumbnail', or 'video'.",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Clear product-related caches to ensure data consistency
	clearProductCache()

	// Return success response with uploaded file details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product media for product with ID " + productID + " uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload:   uploadedResults,
		Message:   "Product media uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// UpdateProductImageHandler handles the update of product images.
// It deletes existing media and uploads new media.
// This endpoint is restricted to administrators.
//
// @Summary      Update product media
// @Description  Replace existing product media with new uploads
// @Tags         Products
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id  formData  string  true  "Product ID"
// @Param        is_primary  formData  bool    false "Is primary image"
// @Param        video_link  formData  string  false "Video URL"
// @Param        gallery     formData  file    false "Gallery images"
// @Param        thumbnail   formData  file    false "Thumbnail image"
// @Param        video       formData  file    false "Video file"
// @Success      200         {object}  map[string]interface{}
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/product/image [put]
func UpdateProductImageHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Ensure admin access
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}

	// Parse upload request parameters
	productID, isPrimary, videoLink, err := parseUploadRequest(r)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse multipart form" + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Fetch existing media to delete later
	existingMedia, err := models.GetProductImages(productID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch existing video links for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	var uploadedResults []map[string]string

	// Handle optional video link
	if videoLink != "" {
		if err := models.InsertProductImage(productID, videoLink, "video", isPrimary); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to insert video link for product ID " + productID,
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
	}

	fileTypes := []string{"gallery", "thumbnail", "video"}
	// Check and upload files for each type
	for _, fileType := range fileTypes {
		results, err := handleFileUploads(r, productID, fileType, isPrimary)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to upload " + fileType + " for product ID " + productID,
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
		uploadedResults = append(uploadedResults, results...)
	}

	// Ensure at least one file or link was processed
	if len(uploadedResults) == 0 {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "No files uploaded for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   "No files were received. Please upload at least one file using the keys: 'gallery', 'thumbnail', or 'video'.",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	clearProductCache()
	// Delete old media
	for _, media := range existingMedia {
		err := models.DeleteProductImage(media.ImageID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to delete existing media for product ID " + productID,
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
	}

	// Respond with success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product media for product with ID " + productID + " uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload:   uploadedResults,
		Message:   "Product media uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// handleFileUploads processes file uploads for a specific file type (gallery, thumbnail, video).
// It uploads files to Google Cloud Storage and inserts records into the database.
// Returns a list of uploaded file metadata or an error if any upload fails.
func handleFileUploads(r *http.Request, productID, fileType string, isPrimary bool) ([]map[string]string, error) {
	// Extract files for the specified type from multipart form
	formFiles := r.MultipartForm.File[fileType]
	if len(formFiles) == 0 {
		// No files of this type, return empty list (not an error)
		return nil, nil
	}

	// Track uploaded files
	var uploaded []map[string]string

	// Process each file in the array
	for _, fileHeader := range formFiles {
		// Upload file to Google Cloud Storage
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{fileHeader})
		if err != nil {
			return nil, fmt.Errorf("failed to upload %s: %w", fileType, err)
		}

		// Insert product image record into database
		if err := models.InsertProductImage(productID, url, fileType, isPrimary); err != nil {
			return nil, fmt.Errorf("failed to insert %s into DB: %w", fileType, err)
		}

		// Add uploaded file metadata to result list
		uploaded = append(uploaded, map[string]string{
			"type": fileType,
			"url":  url,
		})
	}
	return uploaded, nil
}

// clearProductCache invalidates all product-related cache entries.
// This ensures data consistency after product modifications by clearing
// cached product lists, pagination data, and category-product associations.
func clearProductCache() {
	// Define all product-related cache key prefixes
	prefixes := []string{
		"products_page_",                 // Product listing pages
		"pagination_page_",               // Pagination metadata
		"categories_products",            // Category-product associations
		"categories_products_pagination", // Category pagination
		"expensiveandcheapproducts",      // Price extremes cache
	}

	// Clear cache for each prefix
	for _, prefix := range prefixes {
		_ = utils.DeleteCacheByPrefix(prefix)
	}
}

// parseUploadRequest extracts and validates product media upload parameters from the request.
// It returns the product ID, whether the media is primary, video link (if provided), and any validation errors.
func parseUploadRequest(r *http.Request) (string, bool, string, error) {
	// Extract product ID from form (required field)
	productID := r.FormValue("product_id")
	if productID == "" {
		return "", false, "", fmt.Errorf("product_id is required")
	}

	// Parse is_primary flag (determines if this is the main product image)
	isPrimary := strings.ToLower(r.FormValue("is_primary")) == "true"

	// Extract optional video link
	videoLink := r.FormValue("video_link")

	return productID, isPrimary, videoLink, nil
}

// GetRelatedProductsHandler retrieves related products based on category and other criteria.
// It utilizes Redis caching for performance.
//
// @Summary      Get related products
// @Description  Retrieve a list of related products for a specific product
// @Tags         Products
// @Produce      json
// @Param        product_id  query     string  true   "Product ID"
// @Param        page        query     int     false  "Page number"
// @Param        size        query     int     false  "Page size"
// @Success      200         {object}  map[string]interface{}
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      404         {object}  dtos.ErrorResponse
// @Router       /api/products/related [get]
func GetRelatedProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	ctx := r.Context()

	// Parse query parameters
	productID := r.URL.Query().Get("product_id")
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	if productID == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: productIdRequired,
				Code:        http.StatusBadRequest,
			},
			Message:   productIdRequired,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Create Redis cache key for related products
	cacheKey := fmt.Sprintf("related_products:%s", productID)

	// Try to retrieve related products from Redis cache first
	cachedVal, err := Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit - unmarshal cached JSON data
		var cachedProducts []dtos.Product
		if err := json.Unmarshal([]byte(cachedVal), &cachedProducts); err == nil {
			// Successfully retrieved from cache, return immediately
			utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Related Products for product ID " + productID + " fetched from cache successfully",
					Code:        http.StatusOK,
				},
				Payload:   cachedProducts,
				Message:   "Related Products",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Optionally log unmarshal error
		log.Printf("Redis cache unmarshal error: %v", err)
	}

	// Cache miss - get the product
	product, err := models.GetProductByID(productID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: productWithID + productID + " not found",
				Code:        http.StatusNotFound,
			},
			Message:   productNotFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch related products
	relatedProducts, pagination, err := models.GetRelatedProducts(product.CategoryID, product.ID, limit, page)
	if err != nil {
		log.Printf("error getting related products %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "No related products for product with ID " + productID,
				Code:        http.StatusNotFound,
			},
			Message:   "No related products found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Cache result
	if jsonBytes, err := json.Marshal(relatedProducts); err == nil {
		Redis.Set(ctx, cacheKey, jsonBytes, time.Hour)
	} else {
		log.Printf("Failed to cache related products: %v", err)
	}

	// Respond with related products
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Related Products for product ID " + productID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"products":   relatedProducts,
			"pagination": pagination,
		},
		Message:   "Related Products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetProductsHandlerBySubCategoryID retrieves products belonging to a specific subcategory.
//
// @Summary      Get products by subcategory
// @Description  Retrieve a list of products for a specific subcategory
// @Tags         Products
// @Produce      json
// @Param        subcategory_id  path      string  true   "Subcategory ID"
// @Param        page            query     int     false  "Page number"
// @Param        size            query     int     false  "Page size"
// @Success      200             {object}  map[string]interface{}
// @Failure      404             {object}  dtos.ErrorResponse
// @Router       /api/products/subcategories/{subcategory_id} [get]
func GetProductsHandlerBySubCategoryID(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Parse pagination parameters from query string
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Extract subcategory ID from URL path
	subCategoryID := mux.Vars(r)["subcategory_id"]

	// Fetch products belonging to this subcategory from database
	products, pagination, err := models.FetchSubcategoryProducts(subCategoryID, page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetch products for subcategory ID " + subCategoryID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with products
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Subcategory products fetched successfully for subcategory ID " + subCategoryID,
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"products":   products,
			"pagination": pagination,
		},
		Message:   "Subcategory products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetCategoryProductsHandlerByCategoryID retrieves products belonging to a specific category.
// It supports search parameters for filtering.
//
// @Summary      Get products by category
// @Description  Retrieve a list of products for a specific category
// @Tags         Products
// @Produce      json
// @Param        category_id  path      string  true   "Category ID"
// @Param        page         query     int     false  "Page number"
// @Param        size         query     int     false  "Page size"
// @Success      200          {object}  map[string]interface{}
// @Failure      404          {object}  dtos.ErrorResponse
// @Router       /api/products/categories-products/{category_id} [get]
func GetCategoryProductsHandlerByCategoryID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Parse search parameters
	searchParams, ok := ParseSearchParams(r, start, requestSummary, w)
	if !ok {
		return // Error response already sent
	}
	categoryID := mux.Vars(r)["category_id"]

	// Fetch from DB
	products, pagination, err := models.GetCategoriesWithSubcategoriesAndProducts(*searchParams, categoryID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetch products for category ID " + categoryID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with products
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Category products fetched successfully for category ID " + categoryID,
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"categories": products,
			"pagination": pagination,
		},
		Message:   "category products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ApplyUserWishlist checks if a product is in the authenticated user's wishlist.
// It updates the product DTO with wishlist status if the user is authenticated.
// Returns the modified product and a boolean indicating if the check was performed.
func ApplyUserWishlist(r *http.Request, productID string, product *dtos.Product) (dtos.Product, bool) {
	// Check if user is authenticated (token present in request)
	authuser, ok := middleware.IsUserTokenPassed(r)
	if !ok {
		// No authentication, return product unchanged
		return *product, false
	}

	// Query database to check if product is in user's wishlist
	userWishlist, err := models.IsProductInUserWishlist(authuser.ID, productID)
	if err != nil {
		// Log error but don't fail the request
		log.Printf("error fetching wishlist products: %v", err)
		return *product, false
	}

	// Update product with wishlist status if true
	if userWishlist {
		product.InWishlist = &userWishlist
	}

	return *product, true
}

// GetCategoryProductsHandler retrieves all products grouped by category.
// It supports search parameters for filtering.
//
// @Summary      Get all category products
// @Description  Retrieve products grouped by category
// @Tags         Products
// @Produce      json
// @Param        page         query     int     false  "Page number"
// @Param        size         query     int     false  "Page size"
// @Success      200          {object}  map[string]interface{}
// @Failure      404          {object}  dtos.ErrorResponse
// @Router       /api/products/categories-products [get]
func GetCategoryProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Parse search parameters
	searchParams, ok := ParseSearchParams(r, start, requestSummary, w)
	if !ok {
		return // Error response already sent
	}
	// Fetch from DB
	products, pagination, err := models.GetCategoriesWithSubcategoriesAndProducts(*searchParams, "")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetch category products",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with products
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Category products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"categories": products,
			"pagination": pagination,
		},
		Message:   "category products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Sort options constants
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
func ParseSearchParams(r *http.Request, start time.Time, requestSummary string, w http.ResponseWriter) (*dtos.SearchParams, bool) {
	// Extract all query parameters
	query := r.URL.Query()

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
	if !validateSortParam(searchParams.SortBy, r, w, start, requestSummary) {
		return nil, false
	}

	// Log search parameters for debugging
	log.Printf("search params: %+v", searchParams)
	return searchParams, true
}

// validateSortParam checks if the provided sort parameter is valid.
// It returns true if valid or empty, false if invalid (with error response sent).
func validateSortParam(sortBy string, r *http.Request, w http.ResponseWriter, start time.Time, requestSummary string) bool {
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
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: fmt.Sprintf("Invalid sort parameter: %s", sortBy),
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("Invalid sort parameter: %s", sortBy),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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
// @Success      200          {object}  map[string]interface{}
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      500          {object}  dtos.ErrorResponse
// @Router       /api/products/search [get]
func SearchProductsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Parse and validate all search parameters from query string
	searchParams, ok := ParseSearchParams(r, start, requestSummary, w)
	if !ok {
		// Parsing failed, error response already sent by ParseSearchParams
		return
	}

	// Execute product search with all filters applied (false = include out of stock)
	products, pagination, err := models.SearchProducts(*searchParams, false)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to search products",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to search products: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Respond with search results
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"products":   products,
			"pagination": pagination,
			"filters": map[string]interface{}{
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
		Request:   r,
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
func AddProductFeatures(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products"); !ok {
		return
	}

	productID := mux.Vars(r)["product_id"]

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	mainImageURL, imageURLs, err := parseFeatureImages(r, w, start)
	if err != nil {
		return
	}

	topSections, productSpecs, err := parseFeatureJSONFields(r, w, start)
	if err != nil {
		return
	}

	designType := r.FormValue("design_type")
	imagePosition := r.FormValue("image_position")
	//default image position to left if not provided
	if imagePosition == "" {
		imagePosition = "left"
	}
	req := dtos.ProductFeature{
		Image:                 &mainImageURL,
		Header:                r.FormValue("header"),
		Description:           r.FormValue("description"),
		ImagePosition:         imagePosition,
		Images:                &imageURLs,
		TopSection:            &topSections,
		ProductSpecifications: &productSpecs,
		DesignType:            &designType,
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}

	feature, err := models.AddProductFeature(req, productID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product feature: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product feature added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   feature,
		Message:   "Product feature added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func parseFeatureImages(r *http.Request, w http.ResponseWriter, start time.Time) (string, []string, error) {
	var mainImageURL string
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		mainImageURL, err = utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
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
			return "", nil, err
		}
	}

	var imageURLs []string
	if r.MultipartForm != nil && r.MultipartForm.File["images"] != nil {
		for _, fh := range r.MultipartForm.File["images"] {
			url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{fh})
			if err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Products",
						Description: "Failed uploading images: " + err.Error(),
						Code:        http.StatusInternalServerError,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
				})
				return "", nil, err
			}
			imageURLs = append(imageURLs, url)
		}
	}
	log.Printf("Uploaded feature images: %v", imageURLs)
	log.Printf("Uploaded main feature image: %s", mainImageURL)
	return mainImageURL, imageURLs, nil
}

func parseFeatureJSONFields(r *http.Request, w http.ResponseWriter, start time.Time) ([]dtos.Section, []string, error) {
	var topSections []dtos.Section
	topSectionStr := r.FormValue("top_section")
	if topSectionStr != "" {
		if err := json.Unmarshal([]byte(topSectionStr), &topSections); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: invalidTopSectionJSON + err.Error(),
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid top_section format",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return nil, nil, err
		}
	}

	var productSpecs []string
	specStr := r.FormValue("product_specifications")
	if specStr != "" {
		if err := json.Unmarshal([]byte(specStr), &productSpecs); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Invalid product_specifications JSON: " + err.Error(),
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid product_specifications format",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return nil, nil, err
		}
	}

	return topSections, productSpecs, nil
}

// UpdateProductFeatureHandler updates an existing product feature.
// This endpoint is restricted to administrators.
//
// @Summary      Update product feature
// @Description  Update features, images, and specifications of a product
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        feature_id              path      string  true  "Feature ID"
// @Param        image                   formData  file    false "Main feature image"
// @Param        images                  formData  file    false "Additional feature images"
// @Param        header                  formData  string  false "Feature header"
// @Param        description             formData  string  false "Feature description"
// @Param        image_position          formData  string  false "Image position"
// @Param        top_section             formData  string  false "Top section JSON"
// @Param        product_specifications  formData  string  false "Product specifications JSON"
// @Param        design_type             formData  string  false "Design type"
// @Success      200                     {object}  dtos.ProductFeature
// @Failure      400                     {object}  dtos.ErrorResponse
// @Failure      500                     {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/features/{feature_id} [put]
func UpdateProductFeatureHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products"); !ok {
		return
	}

	featureID := mux.Vars(r)["feature_id"]

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse request data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	mainImageURL, imageURLs, err := parseFeatureImages(r, w, start)
	if err != nil {
		return
	}

	topSections, productSpecs, err := parseFeatureJSONFields(r, w, start)
	if err != nil {
		return
	}

	designType := r.FormValue("design_type")
	req := dtos.ProductFeature{
		Image:                 &mainImageURL,
		Header:                r.FormValue("header"),
		Description:           r.FormValue("description"),
		ImagePosition:         r.FormValue("image_position"),
		Images:                &imageURLs,
		TopSection:            &topSections,
		ProductSpecifications: &productSpecs,
		DesignType:            &designType,
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}

	feature, err := models.UpdateProductFeature(req, featureID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product feature: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product feature updated successfully for feature ID " + featureID,
			Code:        http.StatusOK,
		},
		Payload:   feature,
		Message:   "Product feature updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// UpdateAllProductFeaturesHandler updates all features for a product.
// This endpoint is restricted to administrators.
//
// @Summary      Update all product features
// @Description  Update all features, images, and specifications of a product
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
// @Success      200                     {object}  dtos.ProductFeature
// @Failure      400                     {object}  dtos.ErrorResponse
// @Failure      500                     {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/{product_id}/features [put]
func UpdateAllProductFeaturesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products"); !ok {
		return
	}

	productID := mux.Vars(r)["product_id"]

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse request data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	mainImageURL, imageURLs, err := parseFeatureImages(r, w, start)
	if err != nil {
		return
	}

	topSections, productSpecs, err := parseFeatureJSONFields(r, w, start)
	if err != nil {
		return
	}

	designType := r.FormValue("design_type")
	req := dtos.ProductFeature{
		Image:                 &mainImageURL,
		Header:                r.FormValue("header"),
		Description:           r.FormValue("description"),
		ImagePosition:         r.FormValue("image_position"),
		Images:                &imageURLs,
		TopSection:            &topSections,
		ProductSpecifications: &productSpecs,
		DesignType:            &designType,
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}

	feature, err := models.UpdateProductFeatures(req, productID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product feature: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product features updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   feature,
		Message:   "Product features updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// GetFeaturesByProductHandler retrieves features for a specific product.
//
// @Summary      Get product features
// @Description  Retrieve features, images, and specifications for a product
// @Tags         Products
// @Produce      json
// @Param        product_id  path      string  true  "Product ID"
// @Success      200         {object}  []dtos.ProductFeature
// @Failure      500         {object}  dtos.ErrorResponse
// @Router       /api/products/{product_id}/features [get]
func GetFeaturesByProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	productID := mux.Vars(r)["product_id"]
	features, err := models.GetProductFeaturesByProductID(productID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch product features: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product features fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   features,
		Message:   "Product features fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// DeleteProductFeatureHandler deletes a product feature.
// This endpoint is restricted to administrators.
//
// @Summary      Delete product feature
// @Description  Delete a specific feature of a product
// @Tags         Admin
// @Produce      json
// @Param        feature_id  path      string  true  "Feature ID"
// @Success      200         {object}  map[string]interface{}
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/features/{feature_id} [delete]
func DeleteProductFeatureHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products"); !ok {
		return
	}

	featureID := mux.Vars(r)["feature_id"]

	err := models.DeleteProductFeature(featureID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete product feature: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to delete feature: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product feature deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product feature deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// HandleProductSpecifications adds specifications, variants, warranty, tax, and discounts to a product.
// This endpoint is restricted to administrators.
//
// @Summary      Add product specifications
// @Description  Add specifications, variants, warranty, tax, and discounts to a product
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        specifications  body      dtos.ProductSpecification  true  "Product Specifications"
// @Success      200             {object}  map[string]interface{}
// @Failure      400             {object}  dtos.ErrorResponse
// @Failure      500             {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/specifications [post]
func HandleProductSpecifications(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure the user is an admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	state := "add"
	req, ok := DecodeRequestBody[dtos.ProductSpecification](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}

	//handle products specifications
	err := handleProductSpecs(*req, state)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product specifications: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//handle products variants
	err = handleProductsVariants(*req, state)
	log.Printf("handleProductsVariants ***** %s", err)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product variants: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//handle product warranty
	err = handleProductsWarranty(*req)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product warranty: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//add tax to a product
	err = attachProductTax(*req)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product tax: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// add discount to a product
	err = attachProductDiscount(*req, state)
	log.Printf("attachProductDiscount ***** %s", err)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product discount: " + err.Error(),
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
			Description: "Product specifications added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product specifications added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// HandleProductSpecificationsUpdate updates specifications, variants, warranty, tax, and discounts for a product.
// This endpoint is restricted to administrators.
//
// @Summary      Update product specifications
// @Description  Update specifications, variants, warranty, tax, and discounts for a product
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        specifications  body      dtos.ProductSpecification  true  "Product Specifications"
// @Success      200             {object}  map[string]interface{}
// @Failure      400             {object}  dtos.ErrorResponse
// @Failure      500             {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/specifications [put]
func HandleProductSpecificationsUpdate(w http.ResponseWriter, r *http.Request) {
	state := "update"
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure the user is an admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.ProductSpecification](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}

	//handle products specifications
	err := handleProductSpecs(*req, state)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product specifications: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//handle products variants
	err = handleProductsVariants(*req, state)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product variants: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//handle product warranty
	err = handleProductsWarranty(*req)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product warranty: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//add tax to a product
	err = attachProductTax(*req)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product tax: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// add discount to a product
	err = attachProductDiscount(*req, state)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product discount: " + err.Error(),
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
			Description: "Product specifications updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product specifications updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func handleProductSpecs(req dtos.ProductSpecification, state string) error {
	data := dtos.ProductSpecs{
		ProductID:    req.ProductID,
		Weight:       req.Weight,
		WeightLimit:  req.WeightLimit,
		Dimensions:   req.Dimensions,
		Manufacturer: req.Manufacturer,
	}
	//if updating first hold existing specs
	specs := []string{}
	var err error
	if state == "update" {
		specs, err = models.HoldProductSpecs(req.ProductID)
		if err != nil {
			return err
		}
	}
	err = models.InsertProductSpecs(data)
	if err != nil {
		return err
	}
	//if updating remove held specs
	if state == "update" {
		for _, specID := range specs {
			err := models.RemoveHeldProductSpecs(specID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
func handleProductsVariants(req dtos.ProductSpecification, state string) error {
	data := dtos.ProductVariantRequest{
		ProductID: req.ProductID,
	}
	noVariantMsg := "variant not found"

	// Map each variant type to its IDs
	variantGroups := map[string][]string{
		"age":      req.Age,
		"brand":    toSlice(req.Brand), // handle single value as slice
		"material": req.Material,
		"color":    req.Color,
		"size":     req.Size,
	}
	// If updating, first hold existing variants
	var variantIDsExisting []string
	var err error
	if state == "update" {
		variantIDsExisting, err = models.HoldProductVariants(req.ProductID)
		if err != nil {
			return err
		}
	}
	// Loop through each variant group
	for variantType, variantIDs := range variantGroups {
		for _, id := range variantIDs {
			if err := addProductVariantWithHandling(id, variantType, data, noVariantMsg); err != nil {
				return err
			}
		}
	}
	// If updating, remove held variants
	if state == "update" {
		for _, variantID := range variantIDsExisting {
			err := models.RemoveHeldProductVariants(variantID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func toSlice(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func addProductVariantWithHandling(variantID, variantType string, data dtos.ProductVariantRequest, notFoundMsg string) error {
	err := models.AddProductVariant(variantID, data)
	if err == nil {
		return nil
	}
	if err.Error() == notFoundMsg {
		return fmt.Errorf("%s variant with variant_id %s not found", variantType, variantID)
	}
	return err
}

// handleProductsWarranty creates or updates warranty information for a product.
// It takes product specification data and creates a warranty record in the database.
func handleProductsWarranty(req dtos.ProductSpecification) error {
	// Build warranty data transfer object from product specification
	data := dtos.AddProductWarrantiesRequest{
		ProductID:         req.ProductID,        // Product identifier
		WarrantyTypeID:    req.WarrantyType,     // Type of warranty (manufacturer, extended, etc.)
		WarrantyPeriod:    req.WarrantyPeriod,   // Duration of warranty coverage
		ManufacturingDate: req.ManufacturerDate, // Product manufacturing date
		ExpiryDate:        req.ExpiryDate,       // Warranty expiration date
	}

	// Insert warranty record into database
	err := models.AddProductWarranties(data)
	return err
}

// attachProductTax associates a tax charge with a product.
// It creates the product-tax relationship in the database.
func attachProductTax(req dtos.ProductSpecification) error {
	// Build charge attachment data
	data := dtos.AddChargeToProductRequest{
		ProductID: req.ProductID, // Product to attach tax to
		ChargeID:  req.Tax,       // Tax/charge identifier
	}
	// Create product-charge association in database
	err := models.AddChargeToProduct(data)
	return err
}

// attachProductDiscount associates a promotional discount with a product.
// When updating (state="update"), it holds existing promotions, adds new ones, then removes held promotions.
// This ensures atomic promotion updates without conflicts.
func attachProductDiscount(req dtos.ProductSpecification, state string) error {
	// Build promotion attachment data
	data := dtos.AddPromotionToProductRequest{
		ProductID:       req.ProductID,    // Product to attach discount to
		PromotionTypeID: req.DiscountType, // Discount/promotion identifier
	}

	// Only proceed if a promotion type is specified
	if data.PromotionTypeID == "" {
		return nil
	}

	if state == "update" {
		return updateProductDiscount(req.ProductID, data)
	}

	return models.AddPromotionToProduct(data)
}

// updateProductDiscount handles the atomic update of product promotions.
// It holds existing promotions, adds new ones, then removes held promotions.
func updateProductDiscount(productID string, data dtos.AddPromotionToProductRequest) error {
	// Temporarily hold current promotions
	promotionIDs, err := models.HoldProductPromotions(productID)
	if err != nil {
		return err
	}

	// Add new promotion to product
	err = models.AddPromotionToProduct(data)
	if err != nil {
		return err
	}

	// Remove old held promotions after new one is added
	for _, promoID := range promotionIDs {
		err := models.RemoveHeldProductPromotions(promoID)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetExpensiveAndCheapProducts retrieves the most expensive and cheapest products.
// It utilizes caching for performance.
//
// @Summary      Get expensive and cheap products
// @Description  Retrieve the most expensive and cheapest products
// @Tags         Products
// @Produce      json
// @Success      200         {object}  dtos.ExpensiveCheapProduct
// @Failure      500         {object}  dtos.ErrorResponse
// @Router       /api/products/expensive-cheap [get]
func GetExpensiveAndCheapProducts(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Define cache key for expensive/cheap products
	cacheKey := "expensiveandcheapproducts"
	var products *dtos.ExpensiveCheapProduct
	var cachedProducts *dtos.ExpensiveCheapProduct

	// Try to retrieve from Redis cache
	_ = utils.GetCache(cacheKey, &cachedProducts)
	if cachedProducts == nil {
		// Cache miss - fetch from database
		var err error
		products, err = models.GetExpensiveAndCheapProducts()
		if err != nil {
			// Database query failed, return error response
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to fetch expensive and cheapest products",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return
		}
		// Cache the fetched products (no expiration set - uses default)
		_ = utils.SetCache(cacheKey, products)
	} else {
		// Cache hit - use cached data
		products = cachedProducts
	}

	// Return successful response with expensive and cheap products
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Expensive and cheapest products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   products,
		Message:   "Expensive and cheapest products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
