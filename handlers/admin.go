package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var uploadImageError = "Failed to upload image"

// This provides the code for products admin, products categories, products reviews and product bundles functionalities that require admin authorization

// CreateCategoryHandler creates a new product category.
// It supports optional image upload.
// This endpoint is restricted to administrators.
//
// @Summary      Add a new category
// @Description  Create a new product category with an optional image
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        name         formData  string  true  "Category Name"
// @Param        description  formData  string  true  "Category Description"
// @Param        parent_id    formData  string  false "Parent Category ID"
// @Param        image        formData  file    false "Category Image"
// @Success      201          {object}  dtos.Category
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/categories [post]
func CreateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Categories", "categories.create"); !ok {
		return
	}

	// Upload image if present
	url, err := utils.ParseAndUploadFile(r, "image", 20)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to upload image : " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Build DTO
	req := dtos.CreateCategory{
		Image:       url,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		ParentID:    utils.StringPtr(r.FormValue("parent_id")),
	}
	// Validate request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Categories") {
		return
	}

	// Insert category into DB
	category, err := models.AddNewCategory(models.DB, req)
	if err != nil {
		log.Printf("Error adding new category: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to add new category",
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

	// Invalidate categories cache
	_ = utils.DeleteCache("category_data")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   category,
		Message:   "Category created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// UpdateCategoryHandler updates an existing category.
// It supports updating name, description, parent ID, and image.
// This endpoint is restricted to administrators.
//
// @Summary      Update a category
// @Description  Update an existing product category
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        category_id  path      string  true  "Category ID"
// @Param        name         formData  string  false "Category Name"
// @Param        description  formData  string  false "Category Description"
// @Param        parent_id    formData  string  false "Parent Category ID"
// @Param        image        formData  file    false "Category Image"
// @Success      200          {object}  dtos.Category
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/categories/{category_id} [patch]
func UpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Categories", "categories.update"); !ok {
		return
	}

	// Parse multipart form (20 MB max)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to parse form: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
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
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Categories",
					Description: err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   uploadImageError,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return
		}
		imageURL = url
	}

	// Build DTO (image is optional)
	req := dtos.UpdateCategoryPayload{
		Image:       &imageURL,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		ParentID:    utils.StringPtr(r.FormValue("parent_id")),
	}

	// Validate request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Categories") {
		return
	}

	// Get category ID from URL
	id := mux.Vars(r)["category_id"]
	// Update category
	category, err := models.UpdateCategory(models.DB, id, req)
	if err != nil {
		log.Printf("Error updating category: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to update category with id " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Invalidate cache
	_ = utils.DeleteCache("category_data")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   category,
		Message:   "Category updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// DeleteCategoryHandler deletes an existing category.
// This endpoint is restricted to administrators.
//
// @Summary      Delete a category
// @Description  Delete a category by its ID
// @Tags         Admin
// @Produce      json
// @Param        category_id  path      string  true  "Category ID"
// @Success      200          {object}  map[string]interface{}
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/category/{category_id} [delete]
func DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Categories", "categories.delete")
	if !ok {
		return
	}
	id := mux.Vars(r)["category_id"]
	err := models.DeleteCategory(models.DB, id)
	if err != nil {
		log.Printf("Error deleting product %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to delete category with id " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cache
	_ = utils.DeleteCache("category_data")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category with ID " + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Category deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CreateProductHandler creates a new product.
// This endpoint is restricted to administrators.
//
// @Summary      Add a new product
// @Description  Create a new product with initial stock quantity set to 0
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        product  body      dtos.CreateProduct  true  "Product Details"
// @Success      201      {object}  dtos.Product
// @Failure      400      {object}  dtos.ErrorResponse
// @Failure      401      {object}  dtos.ErrorResponse
// @Failure      409      {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products [post]
func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "User not validated or authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateProduct](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}
	//when adding a new product, stock quantity is always 0
	req.StockQuantity = 0
	product, err := models.AddNewProduct(models.DB, *req, authuser.ID)
	if err != nil {
		log.Printf("Error for adding new product %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add new product",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cache
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	_ = utils.DeleteCache("expensiveandcheapproducts")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   product,
		Message:   "Product created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateProductHandler updates an existing product.
// This endpoint is restricted to administrators.
//
// @Summary      Update product
// @Description  Update an existing product by ID
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        product_id  path      string              true  "Product ID"
// @Param        product     body      dtos.CreateProduct  true  "Product Details"
// @Success      200         {object}  dtos.Product
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      409         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/{product_id} [patch]
func UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.update")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateProduct](r, w, requestSummary, start)
	if !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]
	// update the product
	updatedProduct, err := models.UpdateProductByID(models.DB, productID, *req)

	if err != nil {
		log.Printf("Error for updating new product %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product with id " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate cache
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product with ID " + productID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   updatedProduct,
		Message:   "Product updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteProductHandler deletes a product.
// This endpoint is restricted to administrators.
//
// @Summary      Delete product
// @Description  Delete a product by ID
// @Tags         Admin
// @Produce      json
// @Param        product_id  path      string  true  "Product ID"
// @Success      200         {object}  map[string]interface{}
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      409         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/{product_id} [delete]
func DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.delete")
	if !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]
	//delete product
	err := models.DeleteProductByID(models.DB, productID)
	if err != nil {
		log.Printf("Error for deleting product %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete product with id " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate cache
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product with ID " + productID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// AddCoupon creates a new coupon.
// This endpoint is restricted to administrators.
//
// @Summary      Add a new coupon
// @Description  Create a new promotional coupon
// @Tags         Promotions
// @Accept       json
// @Produce      json
// @Param        coupon  body      dtos.PromoCode  true  "Coupon Details"
// @Success      200     {object}  map[string]interface{}
// @Failure      400     {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/coupons [post]
func AddCoupon(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.PromoCode](r, w, requestSummary, start)
	if !ok {
		return
	}
	// user, ok := middleware.UserFromContext(r.Context())
	// if !ok {
	// 	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
	// 		Code:      http.StatusUnauthorized,
	// 		Message:   notAuthenticated,
	// 		TimeTaken: time.Since(start),
	// 		Function:  utils.GetCurrentFuncName(),
	// 		Request:   r,
	// 		RawBody:   requestSummary})
	// 	return
	// }
	if err := models.CreateCoupon(models.DB, *req); err != nil {
		log.Printf("Error adding item to cart: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to create coupon",
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to create coupon",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Coupon created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Coupon created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UploadImageHandler uploads an image to GCS and returns the URL.
// This endpoint is restricted to administrators.
//
// @Summary      Upload an image
// @Description  Upload an image file to Google Cloud Storage
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image File"
// @Success      200    {object}  map[string]string
// @Failure      400    {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/upload [post]
func UploadImageHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}
	url, err := utils.ParseAndUploadFile(r, "image", 10)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to upload image: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Image with URL " + url + " uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload:   url,
		Message:   "Image uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// UploadImageHandler2 uploads an image to GCS and returns the URL.
// This is a duplicate of UploadImageHandler, likely for testing or legacy reasons.
// This endpoint is restricted to administrators.
//
// @Summary      Upload an image (Alternative)
// @Description  Upload an image file to Google Cloud Storage
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image File"
// @Success      200    {object}  map[string]string
// @Failure      400    {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/upload2 [post]
func UploadImageHandler2(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}
	url, err := utils.ParseAndUploadFile(r, "image", 10)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to upload image: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Image with URL " + url + " uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload:   url,
		Message:   "Image uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
