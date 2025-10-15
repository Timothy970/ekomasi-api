package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Create deals eg today's deal, flash sale, etc
// Get deals
// Update deals
// Delete deals
// Add and remove products from deals
// Get deals with their products

func CreateDealHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateDeal](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	_, err := models.CreateDeal(*req)
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
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Deal created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func GetDealsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	deals, err := models.GetAllDeals()
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
		Payload:   deals,
		Message:   "Deals fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func UpdateDealHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	dealID := mux.Vars(r)["deal_id"]
	req, ok := DecodeRequestBody[dtos.Deal](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.UpdateDeal(dealID, *req)
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
		Message:   "Deal updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func DeleteDealHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	dealID := mux.Vars(r)["deal_id"]
	err := models.DeleteDeal(dealID)
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
		Message:   "Deal deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func AddProductToDealHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.ProductDeal](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.AddProductToDeal(req.ID, req.ProductID, nil, nil)
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
		Message:   "Product added to deal successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func RemoveProductFromDealHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.ProductDeal](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.RemoveProductFromDeal(req.ID, req.ProductID)
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
		Message:   "Product removed from deal successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func GetDealWithProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	dealID := mux.Vars(r)["deal_id"]
	deals, pagination, err := models.GetDealWithProducts(dealID, page, limit)
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
		Payload:   map[string]any{"deals": deals, "pagination": pagination},
		Message:   "Deal with products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func CreateDealProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	req, err := parseDealProductRequest(r)
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

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	if err := validateProductsExist(req.Products, w, r, start, requestSummary); err != nil {
		return
	}

	startDate, endDate, err := parseDuration(req.Duration)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	dealID, err := createDeal(req.Title, startDate, endDate)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	if err := addProductsToDeal(dealID, req.Products, w, r, start, requestSummary); err != nil {
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Message:   fmt.Sprintf("%s Deal created successfully", req.Title),
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
func parseDealProductRequest(r *http.Request) (*dtos.FlashDealProducts, error) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, fmt.Errorf("image is required")
	}
	defer file.Close()

	// Upload to GCS (placeholder)
	url := "jjjjjj"
	// url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	products, err := parseProducts(r.FormValue("products"))
	if err != nil {
		return nil, err
	}

	return &dtos.FlashDealProducts{
		Title:    r.FormValue("title"),
		Image:    url,
		Duration: r.FormValue("duration"),
		Products: products,
	}, nil
}
func parseProducts(productsStr string) ([]dtos.ProductsDeal, error) {
	if productsStr == "" {
		return []dtos.ProductsDeal{}, nil
	}

	var products []dtos.ProductsDeal
	if err := json.Unmarshal([]byte(productsStr), &products); err != nil {
		return nil, fmt.Errorf("invalid products format: %w", err)
	}
	return products, nil
}
func validateProductsExist(products []dtos.ProductsDeal, w http.ResponseWriter, r *http.Request, start time.Time, requestSummary string) error {
	for _, p := range products {
		err := models.IsProductThere(p.ProductID)
		if err != nil {
			if err.Error() == "product not found" {
				err = fmt.Errorf("product with ID %s not found", p.ProductID)
			}
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return err
		}
	}
	return nil
}
func parseDuration(duration string) (time.Time, time.Time, error) {
	var startStr, endStr string
	if _, err := fmt.Sscanf(duration, "%s to %s", &startStr, &endStr); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format. expected 'YYYY-MM-DD to YYYY-MM-DD'")
	}
	return models.StringToTime(startStr), models.StringToTime(endStr), nil
}
func createDeal(title string, startDate, endDate time.Time) (string, error) {
	dealData := dtos.CreateDeal{
		Name:      title,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return models.CreateDeal(dealData)
}

func addProductsToDeal(dealID string, products []dtos.ProductsDeal, w http.ResponseWriter, r *http.Request, start time.Time, requestSummary string) error {
	for _, p := range products {
		discount := float64(p.Discount)
		if err := models.AddProductToDeal(dealID, p.ProductID, &p.DiscountType, &discount); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return err
		}
	}
	return nil
}
