package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
func CreateCategoryHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Categories", "categories.create"); !ok {
		return
	}

	// Upload image if present
	url, err := utils.ParseAndUploadFile(c.Request, "image", 20)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to upload image : " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Build DTO
	req := dtos.CreateCategory{
		Image:       url,
		Name:        c.Request.FormValue("name"),
		Description: c.Request.FormValue("description"),
		ParentID:    utils.StringPtr(c.Request.FormValue("parent_id")),
	}
	// Validate request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Categories") {
		return
	}

	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	category, err := models.AddNewCategory(models.DB, req, tenantID)
	if err != nil {
		log.Printf("Error adding new category: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to add new category",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Invalidate categories cache
	_ = utils.DeleteCache("category_data")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	// Respond success
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   category,
		Message:   "Category created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func UpdateCategoryHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Categories", "categories.update"); !ok {
		return
	}

	// Parse multipart form (20 MB max)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to parse form: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Handle optional image upload
	var imageURL string
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Categories",
					Description: err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   uploadImageError,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return
		}
		imageURL = url
	}

	// Build DTO (image is optional)
	req := dtos.UpdateCategoryPayload{
		Image:       &imageURL,
		Name:        c.Request.FormValue("name"),
		Description: c.Request.FormValue("description"),
		ParentID:    utils.StringPtr(c.Request.FormValue("parent_id")),
	}

	// Validate request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Categories") {
		return
	}

	// Get category ID from URL
	id := c.Param("category_id")
	// Update category
	category, err := models.UpdateCategory(models.DB, id, req)
	if err != nil {
		log.Printf("Error updating category: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to update category with id " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Invalidate cache
	_ = utils.DeleteCache("category_data")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")

	// Respond success
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   category,
		Message:   "Category updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func DeleteCategoryHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Categories", "categories.delete")
	if !ok {
		return
	}
	id := c.Param("category_id")
	err := models.DeleteCategory(models.DB, id)
	if err != nil {
		log.Printf("Error deleting product %s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to delete category with id " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cache
	_ = utils.DeleteCache("category_data")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category with ID " + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Category deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
