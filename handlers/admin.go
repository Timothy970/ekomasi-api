package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"context"
	"database/sql"
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
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
	req := dtos.CreateCategory{
		Image:       url,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		ParentID:    utils.StringPtr(r.FormValue("parent_id")),
	}

	// Validate request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	// Insert category into DB
	category, err := models.AddNewCategory(req)
	if err != nil {
		log.Printf("Error adding new category: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("Failed to add category: %v", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Invalidate categories cache
	ctx := context.Background()
	Redis.Del(ctx, "categories")

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
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
				Message:   "Failed to upload image: " + err.Error(),
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
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	// Get category ID from URL
	id := mux.Vars(r)["category_id"]
	// Update category
	category, err := models.UpdateCategory(id, req)
	if err != nil {
		log.Printf("Error updating category: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Invalidate cache
	ctx := context.Background()
	Redis.Del(ctx, "categories")

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
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
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	id := mux.Vars(r)["category_id"]
	if id == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   noCategoryID,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
	}
	err := models.DeleteCategory(id)
	if err != nil {
		log.Printf("Error deleting product %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cache
	ctx := context.Background()
	Redis.Del(ctx, "categories")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
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
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateProduct](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	product, err := models.AddNewProduct(*req)
	if err != nil {
		log.Printf("Error for adding new product %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cache
	ctx := context.Background()
	Redis.Del(ctx, "products")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
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
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateProduct](r, w, requestSummary, start)
	if !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]
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

	product, err := models.GetProductByID(productID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   productNotFound,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Error fetching product",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		}
		return
	}
	// update the product
	updatedProduct, err := models.UpdateProductByID(product.ID, *req)

	if err != nil {
		log.Printf("Error for updating new product %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate cache
	ctx := context.Background()
	Redis.Del(ctx, "products")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
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
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]
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
	product, err := models.GetProductByID(productID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   productNotFound,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Error fetching product",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		}
		return
	}
	//delete product
	err = models.DeleteProductByID(product.ID)
	if err != nil {
		log.Printf("Error for updating new product %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate cache
	ctx := context.Background()
	Redis.Del(ctx, "products")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
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
			Code:      http.StatusInternalServerError,
			Message:   "Failed to create coupon",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Coupon created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
