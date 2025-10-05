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
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// The possible types the frontend can send
	fileTypes := []string{"gallery", "thumbnail", "video"}

	uploadedResults := []map[string]string{}

	for _, fileType := range fileTypes {
		formFiles := r.MultipartForm.File[fileType]
		if len(formFiles) == 0 {
			continue
		}

		for _, fileHeader := range formFiles {
			url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{fileHeader})
			if err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					Code:      http.StatusInternalServerError,
					Message:   "Failed to upload " + fileType + ": " + err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   ""})
				return
			}

			// Now save with type
			err = models.InsertProductImage(productID, url, fileType, isPrimary)
			if err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					Code:      http.StatusInternalServerError,
					Message:   "Failed to insert " + fileType + " into DB: " + err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary})
				return
			}

			uploadedResults = append(uploadedResults, map[string]string{
				"type": fileType,
				"url":  url,
			})
		}
	}

	if len(uploadedResults) == 0 {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "No files uploaded",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// clear cache
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   uploadedResults,
		Message:   "Product media uploaded successfully",
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
	// bundleName := r.URL.Query().Get("bundle_name")
	bundleID := r.URL.Query().Get("bundle_id")
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	bundles, pagination, err := models.GetBundleProducts(bundleID, limit, page)
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   map[string]any{"bundles": bundles, "pagination": pagination},
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
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	// Get image file
	file, header, err := r.FormFile("image")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "Image is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	defer file.Close()
	// Upload image to GCS
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	req := &dtos.Bundle{
		Name:        r.FormValue("bundle_name"),
		Description: r.FormValue("bundle_description"),
		Price:       func() float64 { p, _ := strconv.ParseFloat(r.FormValue("bundle_price"), 64); return p }(),
		Image:       url,
		CategoryID:  r.FormValue("category_id"),
		Products: func() []dtos.BundleProducts {
			productsStr := r.FormValue("products")
			if productsStr == "" {
				return []dtos.BundleProducts{}
			}

			var products []dtos.BundleProducts
			if err := json.Unmarshal([]byte(productsStr), &products); err != nil {
				// you may want to handle error properly instead of swallowing it
				return []dtos.BundleProducts{}
			}
			return products
		}(),
		KeepSelling: func() *bool {
			ks := strings.ToLower(r.FormValue("keep_selling"))
			switch ks {
			case "true":
				b := true
				return &b
			case "false":
				b := false
				return &b
			}
			return nil
		}(),
		CompareAtPrice: func() *float64 {
			cp, _ := strconv.ParseFloat(r.FormValue("compare_at_price"), 64)
			if cp == 0 {
				return nil
			}
			return &cp
		}(),
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err = models.CreateBundle(*req)
	if err != nil {
		log.Printf("create bundle error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
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
	// Parse multipart form (20 MB max)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "Failed to parse form: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Handle optional image upload
	var imageURL string
	if file, header, err := r.FormFile("image"); err == nil {
		defer file.Close()
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return
		}
		imageURL = url
	}

	req := &dtos.UpdateBundle{
		Name:        r.FormValue("bundle_name"),
		Description: r.FormValue("bundle_description"),
		Price:       func() float64 { p, _ := strconv.ParseFloat(r.FormValue("bundle_price"), 64); return p }(),
		Image:       &imageURL,
		CategoryID:  r.FormValue("category_id"),
		KeepSelling: func() *bool {
			ks := strings.ToLower(r.FormValue("keep_selling"))
			switch ks {
			case "true":
				b := true
				return &b
			case "false":
				b := false
				return &b
			}
			return nil
		}(),
		CompareAtPrice: func() *float64 {
			cp, _ := strconv.ParseFloat(r.FormValue("compare_at_price"), 64)
			if cp == 0 {
				return nil
			}
			return &cp
		}(),
		ID: r.FormValue("bundle_id"),
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.UpdateBundle(*req)
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
	req, ok := DecodeRequestBody[[]dtos.BundleProducts](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.AddProductsToBundle(*req, mux.Vars(r)["bundle_id"])
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

	// Parse variants (multiple variant parameters)
	var variants []dtos.VariantFilter
	variantParams := query["variant"] // This gets all values for "variant" parameter

	for _, variantParam := range variantParams {
		if variantParam != "" {
			parts := strings.Split(variantParam, "---")
			if len(parts) == 2 {
				variants = append(variants, dtos.VariantFilter{
					Type:  parts[0],
					Value: parts[1],
				})
			}
		}
	}

	searchParams := dtos.SearchParams{
		Q:            query.Get("q"), // New search query parameter
		CategoryName: query.Get("category_name"),
		ProductName:  query.Get("product_name"),
		Variants:     variants, // Now supports multiple variants
		SortBy:       query.Get("sort_by"),
		SKU:          query.Get("sku"),
		Tag:          query.Get("tag"),
	}
	log.Printf("search params: %+v", searchParams)
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
	products, pagination, err := models.SearchProducts(searchParams, false)
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
			"filters": map[string]interface{}{
				"q":             searchParams.Q,
				"category_name": searchParams.CategoryName,
				"product_name":  searchParams.ProductName,
				"variants":      searchParams.Variants, // Now shows all variants
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

// add product features
// @Summary Add product features
// @Description Add product an features
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Param product_id formData string true "Product ID"
// @Param image formData file true "Product Image"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/product/image [post]
func AddProductFeatures(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]

	// Parse multipart form (20 MB max)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "Failed to parse form: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Get image file
	file, header, err := r.FormFile("image")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "Image is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	defer file.Close()

	// Upload image to GCS
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to upload image: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Build DTO
	req := dtos.ProductFeature{
		Image:         url,
		Header:        r.FormValue("header"),
		Description:   r.FormValue("description"),
		ImagePosition: r.FormValue("image_position"),
	}
	// Validate request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	//insert into db
	feature, err := models.AddProductFeature(req, productID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   feature,
		Message:   "Product feature added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Update Product feature
func UpdateProductFeatureHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	featureID := mux.Vars(r)["feature_id"]
	// Parse multipart form in case image is sent
	_ = r.ParseMultipartForm(20 << 20)

	var imageURL string
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		imageURL, err = utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Failed to upload image: " + err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return
		}
	}

	req := dtos.UpdateProductFeature{
		Header:        r.FormValue("header"),
		Description:   r.FormValue("description"),
		ImagePosition: r.FormValue("image_position"),
		Image:         imageURL, // empty if not uploaded
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	feature, err := models.UpdateProductFeature(req, featureID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   feature,
		Message:   "Product feature updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
func GetFeaturesByProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	productID := mux.Vars(r)["product_id"]
	features, err := models.GetProductFeaturesByProductID(productID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   features,
		Message:   "Product features fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
func DeleteProductFeatureHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	featureID := mux.Vars(r)["feature_id"]

	err := models.DeleteProductFeature(featureID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to delete feature: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Product feature deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
