package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var warehouseWithID = "Warehouse with ID "

// Create warehouse
//
// @Summary Create warehouse
// @Description Create warehouse
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/warehouses [post]
func CreateWarehouse(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Warehouse")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateWarehouseRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Warehouse") {
		return
	}

	_, err := models.CreateWarehouse(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to create warehouse: " + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("warehouses_")
	utils.DeleteCacheByPrefix("warehouses_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Warehouse created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Warehouse created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary List warehouses
// @Description List warehouses
// @Tags Warehouse
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/warehouses [get]
func ListWarehouses(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Warehouse")
	if !ok {
		return
	}
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyWarehouses := fmt.Sprintf("warehouses_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("warehouses_pagination_%d_size_%d", page, size)
	var warehouses []dtos.Warehouse
	var cachedWarehouses []dtos.Warehouse
	var meta dtos.PaginationMeta
	var cachedPagination dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyWarehouses, &cachedWarehouses)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedWarehouses == nil {
		var err error
		warehouses, meta, err = models.ListWarehouses(page, size)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Warehouse",
					Description: "Failed to list warehouses",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKeyWarehouses, cachedWarehouses)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		warehouses = cachedWarehouses
		meta = cachedPagination
	}

	resp := dtos.ListWarehousesResponse{
		Data: warehouses,
		Meta: meta,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Warehouses fetched successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Warehouses fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get warehouse by ID
// @Summary Get warehouse by ID
// @Description Get warehouse by ID
// @Tags Warehouse
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/warehouses/{warehouse_id} [get]
func GetWarehouse(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Warehouse"); !ok {
		return
	}
	id := mux.Vars(r)["warehouse_id"]
	warehouse, err := models.GetWarehouseByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to fetch warehouse with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if warehouse == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: warehouseWithID + id + " not found",
				Code:        http.StatusNotFound,
			},
			Message:   "Warehouse not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: warehouseWithID + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   warehouse,
		Message:   "Warehouse fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Update warehouse by ID
// @Description Update warehouse by ID
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/warehouses/{warehouse_id} [patch]
func UpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Warehouse")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateWarehouseRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Warehouse") {
		return
	}
	id := mux.Vars(r)["warehouse_id"]
	err := models.UpdateWarehouse(id, *req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to update warehouse with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("warehouses_")
	utils.DeleteCacheByPrefix("warehouses_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: warehouseWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Warehouse updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Delete warehouse by ID
// @Description Delete warehouse by ID
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/warehouses/{warehouse_id} [delete]
func DeleteWarehouse(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Warehouse")
	if !ok {
		return
	}
	id := mux.Vars(r)["warehouse_id"]
	err := models.DeleteWarehouse(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to delete warehouse with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("warehouses_")
	utils.DeleteCacheByPrefix("warehouses_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: warehouseWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Warehouse deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
