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

var noCategoryID = "Category ID is required"

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
			log.Printf("Failed to get categories: %v", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
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
		Code:      http.StatusOK,
		Payload:   categories,
		Message:   "Categories fetched successfully",
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
	if id == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   noCategoryID,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	category, err := models.GetCategoryByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to fetch category",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	if category == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   "Category not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   category,
		Message:   "Categories fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
