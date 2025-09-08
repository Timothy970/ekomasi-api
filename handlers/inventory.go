package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// List all inventories
//
// @Summary List all inventories
// @Description List all inventories
// @Tags Inventories
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/inventories [get]
func ListInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	inventories, totalItems, err := models.ListInventory(page, size)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(size)))
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
		Payload: dtos.InventoryListResponse{
			Meta:        meta,
			Inventories: inventories,
		},
		Message:   "Inventories",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Add a new inventory
//
// @Summary Add a new inventory
// @Description Add a new inventory
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/inventories [post]
func CreateInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateInventoryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	if err := models.CreateInventory(*req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
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
		Message:   "Inventory created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get inventory by ID
//
// @Summary List inventory by ID
// @Description List inventory by ID
// @Tags Inventories
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/inventories/{inventory_id} [get]
func GetInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	id := mux.Vars(r)["inventory_id"]

	inv, err := models.GetInventory(id)
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

	dto := dtos.InventoryDTO{
		InventoryID:       inv.InventoryID,
		ProductID:         inv.ProductID,
		VariantID:         inv.VariantID,
		Quantity:          inv.Quantity,
		LowStockThreshold: inv.LowStockThreshold,
		LastUpdated:       inv.LastUpdated,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   dto,
		Message:   "Inventory fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Update inventory
//
// @Summary Update inventory by ID
// @Description Update inventory by ID
// @Tags Inventories
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/inventories/{inventory_id} [patch]
func UpdateInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateInventoryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if req.LowStockThreshold == nil && req.Quantity == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   "Request cannot be empty",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	if err := models.UpdateInventory(id, req.Quantity, req.LowStockThreshold); err != nil {
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
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Inventory updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete an inventory
//
// @Summary Delete inventory by ID
// @Description Delete inventory by ID
// @Tags Inventories
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/inventories/{inventory_id} [delete]
func DeleteInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	if err := models.DeleteInventory(id); err != nil {
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
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Inventory deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func GetInventoryTurnover(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	groupBy := "weekly"
	periodStr := r.URL.Query().Get("period")
	if periodStr != "" {
		groupBy = periodStr
	}
	start, end, _ := ParseDateRange(r)
	data, err := models.GetInventoryTurnover(start, end, groupBy)
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

	resp := dtos.InventoryTurnoverResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
			Type  string    `json:"type"`
		}{Start: start, End: end, Type: groupBy},
		Data: data,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   resp,
		Message:   "Summary inventory turnover report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func GetInventoryTurnoverByProduct(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	productID := mux.Vars(r)["product_id"]
	groupBy := "weekly"
	periodStr := r.URL.Query().Get("period")
	if periodStr != "" {
		groupBy = periodStr
	}
	start, end, _ := ParseDateRange(r)

	data, err := models.GetInventoryTurnoverByProduct(productID, start, end, groupBy)
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

	resp := dtos.InventoryTurnoverResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
			Type  string    `json:"type"`
		}{Start: start, End: end, Type: groupBy},
		Data: data,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   resp,
		Message:   "Product inventory turnover report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
