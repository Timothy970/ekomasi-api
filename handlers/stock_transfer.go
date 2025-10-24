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

// @Summary Create  Stock transfers
// @Description Create  Stock transfers
// @Tags Stock transfers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/stock_transfers [post]
func CreateStockTransfer(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Warehouse")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.StockTransferDTO](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Warehouse") {
		return
	}
	if req.FromWarehouseID == req.ToWarehouseID {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "From and To warehouse cannot be the same",
				Code:        http.StatusBadRequest,
			},
			Message:   "From and To warehouse cannot be the same",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if err := models.CreateStockTransfer(*req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to create stock transfer",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("transfers_")
	utils.DeleteCacheByPrefix("transfers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Stock transfer created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Stock transfer created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

//	List Stock transfers (Paginated)
//
// @Summary List  Stock transfers
// @Description List  Stock transfers
// @Tags Stock transfers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/stock_transfers [get]
func ListStockTransfers(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Warehouse"); !ok {
		return
	}
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyTransfer := fmt.Sprintf("transfers_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("transfers_pagination_%d_size_%d", page, size)
	var transfers []dtos.StockTransferDTO
	var cachedTransfers []dtos.StockTransferDTO
	var meta *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyTransfer, &cachedTransfers)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedTransfers == nil {
		var err error
		transfers, meta, err = models.ListStockTransfers(page, size)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Warehouse",
					Description: "Failed to list stock transfers",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKeyTransfer, cachedTransfers)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		transfers = cachedTransfers
		meta = cachedPagination
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Stock transfers fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   dtos.StockTransferListResponse{Meta: *meta, StockTransfers: transfers},
		Message:   "Stock transfers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Get  Stock transfer by ID
// @Description Get  Stock transfer by ID
// @Tags Stock transfers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/stock_transfers/{transfer_id} [get]
func GetStockTransfer(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	id := mux.Vars(r)["transfer_id"]

	st, err := models.GetStockTransferByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to fetch stock transfer with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Stock transfer fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   st,
		Message:   "Stock transfer fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Update  Stock transfer by ID
// @Description Update  Stock transfer by ID
// @Tags Stock transfers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/stock_transfers/{transfer_id} [patch]
func UpdateStockTransfer(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Warehouse")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.StockTransferUpdateDTO](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Warehouse") {
		return
	}
	id := mux.Vars(r)["transfer_id"]

	if err := models.UpdateStockTransfer(req.Quantity, id); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to update stock transfer with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("transfers_")
	utils.DeleteCacheByPrefix("transfers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Stock transfer with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Stock transfer updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
