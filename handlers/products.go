package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var productIdRequired = "Product ID is required"
var Redis *redis.Client
var productNotFound = "Product not found"

// GetProductsHandler retrieves all products

// @Summary All Products Data
// @Description Get all product categories.
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/products [get]
func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Parse pagination & filters
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	categoryFilter := r.URL.Query().Get("category")
	productFilter := r.URL.Query().Get("product")
	categoryID := r.URL.Query().Get("category_id")

	// Decide if we should use cache

	var products []dtos.CategoryWithProducts
	var pagination *dtos.PaginationMeta
	var err error

	log.Printf("not using cache***")
	products, pagination, err = models.GetAllProducts(categoryFilter, productFilter, categoryID, page, limit)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
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

// parsePagination extracts page & limit from query params with defaults.
func parsePagination(pageStr, sizeStr string) (int, int) {
	page := 1
	limit := 10

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}
	if sizeStr != "" {
		if l, err := strconv.Atoi(sizeStr); err == nil {
			limit = l
		}
	}
	return page, limit
}

// getProductsFromCacheOrDB returns products from cache if available, otherwise from DB and updates cache.
func getProductsFromCacheOrDB(page, limit int, categoryFilter, productFilter, categoryID string) ([]dtos.CategoryWithProducts, *dtos.PaginationMeta, error) {
	var cachedProducts []dtos.CategoryWithProducts
	var cachedPagination *dtos.PaginationMeta

	cacheKeyProducts := fmt.Sprintf("products_page_%d_size_%d", page, limit)
	cacheKeyPagination := fmt.Sprintf("pagination_page_%d_size_%d", page, limit)

	_ = utils.GetCache(cacheKeyProducts, &cachedProducts)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)

	// Cache hit
	if cachedProducts != nil {
		return cachedProducts, cachedPagination, nil
	}

	// Cache miss → fetch from DB
	products, pagination, err := models.GetAllProducts(categoryFilter, productFilter, categoryID, page, limit)
	if err != nil {
		return nil, nil, err
	}

	// Update cache (optional: add TTL)
	_ = utils.SetCache(cacheKeyProducts, products)
	_ = utils.SetCache(cacheKeyPagination, pagination)

	return products, pagination, nil
}

// @Summary Product By ID Data
// @Description Get product product by id.
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/products/{product_id} [get]
func GetProductByIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	productID := mux.Vars(r)["product_id"]
	product, err := models.GetProductByID(productID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   "Product no found",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Error fetch product",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		}
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   product,
		Message:   "Product fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UploadProductImageHandler handles the image upload for a product
// @Summary Upload product image
// @Description Upload an image for a product
// @Tags Products
// @Accept multipart/form-data
// @Produce json
// @Param product_id formData string true "Product ID"
// @Param image formData file true "Product Image"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/product/image [post]
func UploadProductImageHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	productID := r.FormValue("product_id")
	if productID == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "product_id is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	isPrimaryStr := r.FormValue("is_primary")
	isPrimary := strings.ToLower(isPrimaryStr) == "true"

	err := r.ParseMultipartForm(20 << 20) // 20 MB
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to parse form: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	formFiles := r.MultipartForm.File["file"]
	if len(formFiles) == 0 {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "No files uploaded",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	uploadedURLs := []string{}

	for _, fileHeader := range formFiles {
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{fileHeader})
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Failed to upload file: " + err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   ""})
			return
		}

		err = models.InsertProductImage(productID, url, isPrimary)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Failed to insert image into DB: " + err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}

		uploadedURLs = append(uploadedURLs, url)
	}
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   uploadedURLs,
		Message:   "Product image(s) uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// get related products
// @Summary Related Products
// @Description Get all related products.
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/products/related [get]
func GetRelatedProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	ctx := r.Context()
	productID := r.URL.Query().Get("product_id")
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	if productID == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   productIdRequired,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Create a cache key
	cacheKey := fmt.Sprintf("related_products:%s", productID)

	// Try to get from Redis first
	cachedVal, err := Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var cachedProducts []dtos.Product
		if err := json.Unmarshal([]byte(cachedVal), &cachedProducts); err == nil {
			utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
				Code:      http.StatusOK,
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
			Code:      http.StatusNotFound,
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
			Code:      http.StatusNotFound,
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
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

//CRUD operations for bundle products

// Get all bundle products
func GetBundleProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	bundleName := r.URL.Query().Get("bundle_name")
	bundleID := r.URL.Query().Get("bundle_id")
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	useCache := bundleID == "" && bundleName == ""
	var bundles []dtos.GetBundleRequest
	var cachedBundles []dtos.GetBundleRequest
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	cacheKeyBundles := fmt.Sprintf("products_bundles_page_%d_size_%d", page, limit)
	cacheKeyPagination := fmt.Sprintf("bundles_pagination_page_%d_size_%d", page, limit)

	if useCache {
		_ = utils.GetCache(cacheKeyBundles, &cachedBundles)
		_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
		if cachedBundles == nil {
			var err error
			bundles, pagination, err = models.GetBundleProducts(bundleID, bundleName, limit, page)
			if err != nil {
				log.Printf("bundles get error::%s", err)
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					Code:      http.StatusNotFound,
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary})
				return
			}
			_ = utils.SetCache(cacheKeyBundles, bundles)
			_ = utils.SetCache(cacheKeyPagination, pagination)
		} else {
			bundles = cachedBundles
			pagination = cachedPagination
		}
	} else {
		var err error
		bundles, pagination, err = models.GetBundleProducts(bundleID, bundleName, limit, page)
		if err != nil {
			log.Printf("bundles get error::%s", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	response := map[string]interface{}{
		"bundles": bundles,
	}
	if pagination != nil {
		response["pagination"] = pagination
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   response,
		Message:   "Bundles fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// Create a new bundle
func CreateBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.Bundle](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.CreateBundle(*req)
	if err != nil {
		log.Printf("create bundle error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to create product bundle",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("products_bundles_page_")
	utils.DeleteCacheByPrefix("bundles_pagination_page_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Created product bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// Update an existing bundle
func UpdateBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateBundle](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.UpdateBundle(*req, req.ID)
	if err != nil {
		log.Printf("update bundle error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Product bundle updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete a bundle
func DeleteBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.DeleteBundle](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.DeleteBundle(req.ID)
	if err != nil {
		log.Printf("delete bundle error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Product bundle deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// add products to a bundle
func AddProductsToBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.AddProductsToBundle](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.AddProductsToBundle(*req, req.ID)
	if err != nil {
		log.Printf("dd product to bundle error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Product(s) added to bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Remove products from a bundle
func RemoveProductsFromBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	bundleID := mux.Vars(r)["bundle_id"]
	req, ok := DecodeRequestBody[dtos.AddProductsToBundle](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.RemoveProductsFromBundle(*req, bundleID)
	if err != nil {
		log.Printf("dd product to bundle error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to remove product(s) from bundle",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("products_bundles_page_")
	utils.DeleteCacheByPrefix("bundles_pagination_page_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Product(s) removed from bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get products by subcategory ID
// @Summary All Products Data
// @Description Get all product categories.
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/products/subcategories/{subcategory_id} [get]
func GetProductsHandlerBySubCategoryID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	subCategoryID := mux.Vars(r)["subcategory_id"]

	// Fetch from DB
	products, pagination, err := models.FetchSubcategoryProducts(subCategoryID, page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Store in cache only if it's a non-filtered, full, unpaginated response
	// if useCache {
	// 	if productBytes, err := json.Marshal(products); err == nil {
	// 		Redis.Set(ctx, "products", productBytes, 10*time.Minute)
	// 	}
	// }
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
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

// Get products by subcategory ID
// @Summary All Products Data
// @Description Get all product categories.
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/products/categories-products/{category_id} [get]
func GetCategoryProductsHandlerByCategoryID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	categoryID := mux.Vars(r)["category_id"]

	// Fetch from DB
	products, pagination, err := models.GetCategoriesWithSubcategoriesAndProducts(page, limit, categoryID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
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

// Get Category products
// @Summary All Products Data
// @Description Get all product categories.
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/products/categories-products [get]
func GetCategoryProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Fetch from DB
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	var products []dtos.CategoryResponse
	var cachedProdcts []dtos.CategoryResponse
	cacheKeyProducts := fmt.Sprintf("categories_products%d_size_%d", page, limit)
	cacheKeyPagination := fmt.Sprintf("categories_products_pagination%d_size_%d", page, limit)
	_ = utils.GetCache(cacheKeyProducts, &cachedProdcts)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedProdcts == nil {
		var err error
		products, pagination, err = models.GetCategoriesWithSubcategoriesAndProducts(page, limit, "")
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKeyPagination, pagination)
		_ = utils.SetCache(cacheKeyProducts, products)
	} else {
		pagination = cachedPagination
		products = cachedProdcts
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
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

func SearchProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Parse query parameters
	query := r.URL.Query()

	searchParams := dtos.SearchParams{
		CategoryName: query.Get("category_name"),
		ProductName:  query.Get("product_name"),
		VariantName:  query.Get("variant_name"),
		VariantValue: query.Get("variant_value"),
		SortBy:       query.Get("sort_by"),
	}

	// Parse pagination
	page, limit := parsePagination(query.Get("page"), query.Get("size"))
	searchParams.Page = page
	searchParams.Limit = limit

	// Validate sort parameter
	if searchParams.SortBy != "" {
		validSorts := map[string]bool{
			SortPriceHighToLow:   true,
			SortPriceLowToHigh:   true,
			SortDateOldToNew:     true,
			SortDateNewToOld:     true,
			SortFeatured:         true,
			SortBestSellers:      true,
			SortAlphabeticallyAZ: true,
			SortAlphabeticallyZA: true,
		}
		if !validSorts[searchParams.SortBy] {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusBadRequest,
				Message:   "Invalid sort parameter",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
	}

	// Perform search
	products, pagination, err := models.SearchProducts(searchParams)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to search products: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
		Payload: map[string]interface{}{
			"products":   products,
			"pagination": pagination,
			"filters": map[string]string{
				"category_name": searchParams.CategoryName,
				"product_name":  searchParams.ProductName,
				"variant_name":  searchParams.VariantName,
				"variant_value": searchParams.VariantValue,
				"sort_by":       searchParams.SortBy,
			},
		},
		Message:   "Products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
