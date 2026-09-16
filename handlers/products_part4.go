package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetRelatedProductsHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	ctx := c.Request.Context()

	// Parse query parameters
	productID := c.Query("product_id")
	page, limit := parsePagination(c.Query("page"), c.Query("size"))

	if productID == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: productIdRequired,
				Code:        http.StatusBadRequest,
			},
			Message:   productIdRequired,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
			utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Related Products for product ID " + productID + " fetched from cache successfully",
					Code:        http.StatusOK,
				},
				Payload:   cachedProducts,
				Message:   "Related Products",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		// Optionally log unmarshal error
		log.Printf("Redis cache unmarshal error: %v", err)
	}

	// Cache miss - get the product
	product, err := models.GetProductByID(models.DB, productID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: productWithID + productID + " not found",
				Code:        http.StatusNotFound,
			},
			Message:   productNotFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch related products
	relatedProducts, pagination, err := models.GetRelatedProducts(models.DB, product.CategoryID, product.ID, limit, page)
	if err != nil {
		log.Printf("error getting related products %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "No related products for product with ID " + productID,
				Code:        http.StatusNotFound,
			},
			Message:   "No related products found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Related Products for product ID " + productID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"products":   relatedProducts,
			"pagination": pagination,
		},
		Message:   "Related Products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200             {object}  map[string]any
// @Failure      404             {object}  dtos.ErrorResponse
// @Router       /api/products/subcategories/{subcategory_id} [get]
func GetProductsHandlerBySubCategoryID(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse pagination parameters from query string
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	// Extract subcategory ID from URL path
	subCategoryID := c.Param("subcategory_id")

	// Fetch products belonging to this subcategory from database
	products, pagination, err := models.FetchSubcategoryProducts(models.DB, subCategoryID, page, limit)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetch products for subcategory ID " + subCategoryID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with products
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Subcategory products fetched successfully for subcategory ID " + subCategoryID,
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"products":   products,
			"pagination": pagination,
		},
		Message:   "Subcategory products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200          {object}  map[string]any
// @Failure      404          {object}  dtos.ErrorResponse
// @Router       /api/products/categories-products/{category_id} [get]
func GetCategoryProductsHandlerByCategoryID(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse search parameters
	searchParams, ok := ParseSearchParams(c, start, requestSummary)
	if !ok {
		return // Error response already sent
	}
	categoryID := c.Param("category_id")
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	log.Printf("Fetching categories with tenant id %d", tenantID)
	// Fetch from DB
	products, pagination, err := models.GetCategoriesWithSubcategoriesAndProducts(models.DB, *searchParams, categoryID, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetch products for category ID " + categoryID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with products
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Category products fetched successfully for category ID " + categoryID,
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"categories": products,
			"pagination": pagination,
		},
		Message:   "category products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
	userWishlist, err := models.IsProductInUserWishlist(models.DB, authuser.ID, productID)
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
// @Success      200          {object}  map[string]any
// @Failure      404          {object}  dtos.ErrorResponse
// @Router       /api/products/categories-products [get]
func GetCategoryProductsHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse search parameters
	searchParams, ok := ParseSearchParams(c, start, requestSummary)
	if !ok {
		return // Error response already sent
	}
	// Fetch from DB
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	log.Printf("Fetching categories-products with tenant id %d", tenantID)
	products, pagination, err := models.GetCategoriesWithSubcategoriesAndProducts(models.DB, *searchParams, "", tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetch category products",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with products
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Category products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"categories": products,
			"pagination": pagination,
		},
		Message:   "category products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// Sort options constants
