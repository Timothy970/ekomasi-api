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

// This provides the code for products admin, products categories, products reviews and product bundles functionalities that require admin authorization

// Function for creating Products Categories ### POST /products/categories
// Add new category
// CreateCategoryHandler creates a new category
// @Summary Add a new category
// @Description Add new category
// @Tags Admin
// @Accept json
// @Produce json
// @Param product body dtos.CreateCategory true "Add a new Category"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/products/categories [post]
func CreateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Categories"); !ok {
		return
	}

	url, err := utils.ParseAndUploadFile(r, "image", 20)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to upload file: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
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
	category, err := models.AddNewCategory(req)
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

// Update new category
// UpdateProductHandler updates an existing category
// @Summary Update a category
// @Description Update a category
// @Tags Admin
// @Accept json
// @Produce json
// @Param product body dtos.CreateProduct true "Updated Category"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/products/categories/{category_id} [PATCH]
func UpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Categories"); !ok {
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
					Description: "Failed to upload image: " + err.Error(),
					Code:        http.StatusInternalServerError,
				},
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return
		}
		imageURL = url
	}

	// Build DTO (image is optional)
	req := dtos.CreateCategory{
		Image:       imageURL,
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
	category, err := models.UpdateCategory(id, req)
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

// Delete category
// DeletesProductHandler deletes an existing category
// @Summary Delete a category
// @Description Delete a category
// @Tags Admin
// @Accept json
// @Produce json
// @Param product body map[string]string true "Delete Category"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/products/category/{category_id} [delete]
func DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Categories")
	if !ok {
		return
	}
	id := mux.Vars(r)["category_id"]
	err := models.DeleteCategory(id)
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
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Category deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Add new product
// CreateProductHandler creates a new product
// @Summary Add a new product
// @Description Add new product
// @Tags Admin
// @Accept json
// @Produce json
// @Param product body dtos.CreateProduct true "Add a new Product"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/amin/products [post]
func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
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
	product, err := models.AddNewProduct(*req, authuser.ID)
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

// UpdateProductHandler updates an existing product
// @Summary Update product
// @Description Update product
// @Tags Admin
// @Accept json
// @Produce json
// @Param product body dtos.CreateProduct true "Update a Product"
// @Param        product_id  query     string  true  "Product ID"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/products/{products_id} [PATCH]
func UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateProduct](r, w, requestSummary, start)
	if !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]
	// update the product
	updatedProduct, err := models.UpdateProductByID(productID, *req)

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

// delete a product
// DeleteProductHandler deletes a product
// @Summary Delete product
// @Description Delete product
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/products/{product_id} [put]
func DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]
	//delete product
	err := models.DeleteProductByID(productID)
	if err != nil {
		log.Printf("Error for updating new product %s", err)
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
	if err := models.CreateCoupon(*req); err != nil {
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

// func to upload an image to gcs and return the url
func UploadImageHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	url, err := utils.ParseAndUploadFile(r, "image", 10)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   url,
		Message:   "Image uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
