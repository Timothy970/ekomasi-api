package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var (
	categorySuccess = "Categories fetched successfully"
)

func GetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	var categories []dtos.CategoryData
	var cachedCategories []dtos.CategoryData
	_ = utils.GetCache("category_data", &cachedCategories)
	if cachedCategories == nil {
		var err error
		categories, err = models.GetAllCategories()
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Categories",
					Description: "Failed to fetch categories",
					Code:        http.StatusNotFound,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache("category_data", categories)
	} else {
		categories = cachedCategories
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "All categories fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   categories,
		Message:   categorySuccess,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// GetCategoryByIDHandler handles fetching a single category by ID
// @Summary      Get a category by ID
// @Description  Retrieve a category using its unique ID
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Category ID"
// @Success      200  {object}  dtos.Category
// @Failure      400  {object}  map[string]string "Invalid category ID"
// @Failure      404  {object}  map[string]string "Category not found"
// @Failure      500  {object}  map[string]string "Internal server error"
// @Router       /api/categories/{id} [get]
func GetCategoryByIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	id := mux.Vars(r)["id"]

	category, err := models.GetCategoryByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to fetch category with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to fetch category",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	if category == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Category with ID " + id + " not found",
				Code:        http.StatusNotFound,
			},
			Message:   "Category not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category with ID " + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   category,
		Message:   "Category fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Admin Handler to get categries with pagination
// To return name, type,items,subactegories,description
func AdminGetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Categories"); !ok {
		return
	}
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	categoryName := r.URL.Query().Get("category_name")
	categories, pagination, err := models.GetAdminCategories(page, limit, categoryName)
	if err != nil {
		log.Printf("Failed to get categories: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to get categories",
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
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: categorySuccess,
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"categories": categories, "pagination": pagination},
		Message:   "Category fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func GetCategoriesWithSubCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	categories, err := models.GetCategoriesWithSubCategories()
	if err != nil {
		log.Printf("Failed to get categories: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to get categories",
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
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Categories with subcategories fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   categories,
		Message:   categorySuccess,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
