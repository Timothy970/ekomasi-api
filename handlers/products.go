package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"database/sql"
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
	//check if token is passed, if so check if products belong to the users wishlist
	ApplyUserWishlist(r, product.ID, product)
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
	reqSummary := utils.GetRequestSummary(r)

	productID, isPrimary, videoLink, err := parseUploadRequest(r)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   reqSummary,
		})
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   reqSummary,
		})
		return
	}

	var uploadedResults []map[string]string

	// handle optional video link
	if videoLink != "" {
		if err := models.InsertProductImage(productID, videoLink, "video", isPrimary); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusBadRequest,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   reqSummary,
			})
			return
		}
	}

	fileTypes := []string{"gallery", "thumbnail", "video"}
	for _, fileType := range fileTypes {
		results, err := handleFileUploads(r, productID, fileType, isPrimary)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusBadRequest,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   reqSummary,
			})
			return
		}
		uploadedResults = append(uploadedResults, results...)
	}

	if len(uploadedResults) == 0 {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "No files uploaded",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   reqSummary,
		})
		return
	}

	clearProductCache()

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   uploadedResults,
		Message:   "Product media uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   reqSummary,
	})
}
func handleFileUploads(r *http.Request, productID, fileType string, isPrimary bool) ([]map[string]string, error) {
	formFiles := r.MultipartForm.File[fileType]
	if len(formFiles) == 0 {
		return nil, nil
	}

	var uploaded []map[string]string
	for _, fileHeader := range formFiles {
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{fileHeader})
		if err != nil {
			return nil, fmt.Errorf("failed to upload %s: %w", fileType, err)
		}

		if err := models.InsertProductImage(productID, url, fileType, isPrimary); err != nil {
			return nil, fmt.Errorf("failed to insert %s into DB: %w", fileType, err)
		}

		uploaded = append(uploaded, map[string]string{
			"type": fileType,
			"url":  url,
		})
	}
	return uploaded, nil
}
func clearProductCache() {
	prefixes := []string{
		"products_page_",
		"pagination_page_",
		"categories_products",
		"categories_products_pagination",
		"expensiveandcheapproducts",
	}
	for _, prefix := range prefixes {
		_ = utils.DeleteCacheByPrefix(prefix)
	}
}

func parseUploadRequest(r *http.Request) (string, bool, string, error) {
	productID := r.FormValue("product_id")
	if productID == "" {
		return "", false, "", fmt.Errorf("product_id is required")
	}
	isPrimary := strings.ToLower(r.FormValue("is_primary")) == "true"
	videoLink := r.FormValue("video_link")
	return productID, isPrimary, videoLink, nil
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
	searchParams, ok := ParseSearchParams(r, start, requestSummary, w)
	if !ok {
		return // Error response already sent
	}
	categoryID := mux.Vars(r)["category_id"]

	// Fetch from DB
	products, pagination, err := models.GetCategoriesWithSubcategoriesAndProducts(*searchParams, categoryID)
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
func ApplyUserWishlist(r *http.Request, productID string, product *dtos.Product) (dtos.Product, bool) {
	authuser, ok := middleware.IsUserTokenPassed(r)
	if !ok {
		return *product, false
	}

	userWishlist, err := models.IsProductInUserWishlist(authuser.ID, productID)
	if err != nil {
		log.Printf("error fetching wishlist products: %v", err)
		return *product, false
	}

	if userWishlist {
		product.InWishlist = &userWishlist
	}

	return *product, true
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
	searchParams, ok := ParseSearchParams(r, start, requestSummary, w)
	if !ok {
		return // Error response already sent
	}
	// Fetch from DB
	products, pagination, err := models.GetCategoriesWithSubcategoriesAndProducts(*searchParams, "")
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
func ParseSearchParams(r *http.Request, start time.Time, requestSummary string, w http.ResponseWriter) (*dtos.SearchParams, bool) {
	query := r.URL.Query()

	// Parse variants (e.g., variant=color---red&variant=size---L)
	var variants []dtos.VariantFilter
	for _, variantParam := range query["variant"] {
		if variantParam == "" {
			continue
		}
		parts := strings.Split(variantParam, "---")
		if len(parts) == 2 {
			variants = append(variants, dtos.VariantFilter{
				Type:  parts[0],
				Value: parts[1],
			})
		}
	}

	// Build search params
	searchParams := &dtos.SearchParams{
		Q:            query.Get("q"),
		CategoryName: query.Get("category_name"),
		ProductName:  query.Get("product_name"),
		Variants:     variants,
		SortBy:       query.Get("sort_by"),
		SKU:          query.Get("sku"),
		Tag:          query.Get("tag"),
		MaxPrice:     getFloatQueryParam(query, "maxPrice"),
		MinPrice:     getFloatQueryParam(query, "minPrice"),
	}

	// Parse pagination
	searchParams.Page, searchParams.Limit = parsePagination(query.Get("page"), query.Get("size"))

	// Validate sort parameter
	if !validateSortParam(searchParams.SortBy, r, w, start, requestSummary) {
		return nil, false
	}

	log.Printf("search params: %+v", searchParams)
	return searchParams, true
}
func validateSortParam(sortBy string, r *http.Request, w http.ResponseWriter, start time.Time, requestSummary string) bool {
	if sortBy == "" {
		return true
	}

	validSorts := map[string]bool{
		"price_high_to_low":  true,
		"price_low_to_high":  true,
		"date_old_to_new":    true,
		"date_new_to_old":    true,
		"featured":           true,
		"best_sellers":       true,
		"alphabetically_a_z": true,
		"alphabetically_z_a": true,
	}

	if !validSorts[sortBy] {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
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

func SearchProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	searchParams, ok := ParseSearchParams(r, start, requestSummary, w)
	if !ok {
		return // Error response already sent
	}
	// Perform search
	products, pagination, err := models.SearchProducts(*searchParams, false)
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
func getFloatQueryParam(query url.Values, key string) float64 {
	valueStr := query.Get(key)
	if valueStr == "" {
		return 0
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0
	}
	return value
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

func HandleProductSpecifications(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure the user is an admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.ProductSpecification](r, w, requestSummary, start)
	if !ok {
		return
	}
	//handle products specifications
	err := handleProductSpecs(*req)
	log.Printf("handleProductSpecs ***** %s", err)
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
	//handle products variants
	err = handleProductsVariants(*req)
	log.Printf("handleProductsVariants ***** %s", err)

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
	//handle product warranty
	err = handleProductsWarranty(*req)
	log.Printf("handleProductsWarranty ***** %s", err)

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
	//add tax to a product
	err = attachProductTax(*req)
	log.Printf("attachProductTax ***** %s", err)

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
	// add discount to a product
	err = attachProductDiscount(*req)
	log.Printf("attachProductDiscount ***** %s", err)

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
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Product specifications added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func handleProductSpecs(req dtos.ProductSpecification) error {
	data := dtos.ProductSpecs{
		ProductID:    req.ProductID,
		Weight:       req.Weight,
		WeightLimit:  req.WeightLimit,
		Dimensions:   req.Dimensions,
		Manufacturer: req.Manufacturer,
	}
	err := models.InsertProductSpecs(data)
	return err
}
func handleProductsVariants(req dtos.ProductSpecification) error {
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

	// Loop through each variant group
	for variantType, variantIDs := range variantGroups {
		for _, id := range variantIDs {
			if err := addProductVariantWithHandling(id, variantType, data, noVariantMsg); err != nil {
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

func handleProductsWarranty(req dtos.ProductSpecification) error {
	data := dtos.AddProductWarrantiesRequest{
		ProductID:         req.ProductID,
		WarrantyTypeID:    req.WarrantyType,
		WarrantyPeriod:    req.WarrantyPeriod,
		ManufacturingDate: req.ManufacturerDate,
		ExpiryDate:        req.ExpiryDate,
	}
	err := models.AddProductWarranties(data)
	return err
}

func attachProductTax(req dtos.ProductSpecification) error {
	data := dtos.AddChargeToProductRequest{
		ProductID: req.ProductID,
		ChargeID:  req.Tax,
	}
	err := models.AddChargeToProduct(data)
	return err
}
func attachProductDiscount(req dtos.ProductSpecification) error {
	data := dtos.AddPromotionToProductRequest{
		ProductID:       req.ProductID,
		PromotionTypeID: req.DiscountType,
	}
	if data.PromotionTypeID != "" {
		err := models.AddPromotionToProduct(data)
		if err != nil {
			return nil
		}
	}
	return nil
}

// Function to return the most expensive and most cheapest product
func GetExpensiveAndCheapProducts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	cacheKey := "expensiveandcheapproducts"
	var products *dtos.ExpensiveCheapProduct
	var cachedProducts *dtos.ExpensiveCheapProduct
	_ = utils.GetCache(cacheKey, &cachedProducts)
	if cachedProducts == nil {
		var err error
		products, err = models.GetExpensiveAndCheapProducts()
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
		_ = utils.SetCache(cacheKey, products)
	} else {
		products = cachedProducts
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   products,
		Message:   "Expensive and cheapest products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
