package handlers

import (
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-redis/redis/v8"
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
// @Success      200          {object}  map[string]any
// @Failure      404          {object}  dtos.ErrorResponse
// @Router       /api/products [get]
func GetProductsHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse pagination & filters from query parameters
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	// Extract optional category name filter
	categoryFilter := c.Query("category")
	// Extract optional product name filter
	productFilter := c.Query("product")
	// Extract optional category ID filter
	categoryID := c.Query("category_id")

	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	// Fetch products from database with all filters applied
	products, pagination, err := models.GetAllProducts(models.DB, tenantID, categoryFilter, productFilter, categoryID, page, limit)

	// Check if database query failed
	if err != nil {
		// Return error response with detailed information
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to retrieve products",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Return successful response with products and pagination metadata
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "All products retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"products":   products,
			"pagination": pagination,
		},
		Message:   "All products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func GetProductByIDHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract product ID from URL path parameters
	productID := c.Param("product_id")

	// Fetch product details from database
	product, err := models.GetProductByID(models.DB, productID)
	if err != nil {
		// Log error and return response
		log.Printf("error fetching product with ID %s: %v", productID, err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetching product with ID " + productID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Error fetching product",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})

		return
	}

	// Check if user is authenticated and update product with wishlist status
	ApplyUserWishlist(c.Request, product.ID, product)

	// Respond with product details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product with ID " + product.ID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   product,
		Message:   "Product fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200         {object}  map[string]any
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/product/image [post]
