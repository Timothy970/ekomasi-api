package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	categorySuccess = "Categories fetched successfully"
)

// GetCategoriesHandler retrieves all product categories.
// It utilizes caching for performance.
//
// @Summary      Get all categories
// @Description  Retrieve a list of all product categories
// @Tags         Categories
// @Produce      json
// @Success      200  {object}  []dtos.CategoryData
// @Failure      404  {object}  dtos.ErrorResponse
// @Router       /api/categories [get]
func GetCategoriesHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	var categories []dtos.CategoryData
	var cachedCategories []dtos.CategoryData
	cacheKey := fmt.Sprintf("category_data:tenant:%d", tenantID)
	_ = utils.GetCache(cacheKey, &cachedCategories)
	if cachedCategories == nil {
		var err error
		categories, err = models.GetAllCategories(models.DB, tenantID)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Categories",
					Description: "Failed to fetch categories",
					Code:        http.StatusNotFound,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKey, categories)
	} else {
		categories = cachedCategories
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "All categories fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   categories,
		Message:   categorySuccess,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// GetCategoryByIDHandler handles fetching a single category by ID.
//
// @Summary      Get a category by ID
// @Description  Retrieve a category using its unique ID
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Category ID"
// @Success      200  {object}  dtos.Category
// @Failure      400  {object}  dtos.ErrorResponse
// @Failure      404  {object}  dtos.ErrorResponse
// @Failure      500  {object}  dtos.ErrorResponse
// @Router       /api/categories/{id} [get]
func GetCategoryByIDHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	id := c.Param("id")

	category, err := models.GetCategoryByID(models.DB, id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to fetch category with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to fetch category",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	if category == nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Category with ID " + id + " not found",
				Code:        http.StatusNotFound,
			},
			Message:   "Category not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category with ID " + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   category,
		Message:   "Category fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AdminGetCategoriesHandler retrieves categories with pagination for admin view.
// It returns detailed category information including subcategories and descriptions.
// This endpoint is restricted to administrators.
//
// @Summary      Get categories (Admin)
// @Description  Retrieve paginated categories with details for admin
// @Tags         Admin
// @Produce      json
// @Param        page  query     int     false  "Page number"
// @Param        size  query     int     false  "Page size"
// @Param        q     query     string  false  "Search query"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/categories [get]
func AdminGetCategoriesHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Categories", ""); !ok {
		return
	}
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	categoryName := c.Query("q")
	categoryType := c.Query("type")
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	categories, pagination, err := models.GetAdminCategories(models.DB, tenantID, page, limit, categoryName, categoryType)
	if err != nil {
		log.Printf("Failed to get categories: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to get categories",
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: categorySuccess,
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"categories": categories, "pagination": pagination},
		Message:   "Category fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetCategoriesWithSubCategoriesHandler retrieves all categories including their subcategories.
//
// @Summary      Get categories with subcategories
// @Description  Retrieve a hierarchical list of categories and their subcategories
// @Tags         Categories
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  dtos.ErrorResponse
// @Router       /api/categories/tree [get]
func GetCategoriesWithSubCategoriesHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)

	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	categories, err := models.GetCategoriesWithSubCategories(models.DB, tenantID)
	if err != nil {
		log.Printf("Failed to get categories: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to get categories",
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Categories with subcategories fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   categories,
		Message:   categorySuccess,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
