package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type PurchaseOrderHandler struct {
	DB *sql.DB
}

var pOrderWithID = "Purchase Order with ID "

// Create Purchase Order
//
// @Summary Purchase Order
// @Description Purchase Order
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/purchase-orders [post]
func CreatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreatePurchaseOrderRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		return
	}
	err := models.AddNewPurchaseOrder(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to add new purchase order",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Purchase order added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Purchase order added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// List Purchase Orders with Pagination
// @Summary List Purchase Order
// @Description List Purchase Order
// @Tags Purchase Orders
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/purchase-orders [get]
func ListPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyPurchaseOrders := fmt.Sprintf("purchase_orders_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("purchase_orders_pagination_%d_size_%d", page, size)
	var orders []dtos.PurchaseOrderResponse
	var cachedOrders []dtos.PurchaseOrderResponse
	var meta *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyPurchaseOrders, &cachedOrders)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedOrders == nil {
		var err error
		orders, meta, err = models.ListPurchaseOrders(page, size)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Orders",
					Description: "Failed to list purchase orders",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKeyPurchaseOrders, cachedOrders)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		orders = cachedOrders
		meta = cachedPagination
	}
	resp := dtos.PaginatedPurchaseOrdersResponse{Data: orders, Meta: *meta}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Purchase orders fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Purchase orders fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// Get Purchase Order by ID
// @Summary Get Purchase Order by ID
// @Description Get Purchase Order by ID
// @Tags Purchase Orders
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/purchase-orders/{po_id} [get]
func GetPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	id := mux.Vars(r)["po_id"]
	order, err := models.GetPurchaseOrderByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch purchase order with ID " + id,
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
			Module:      "Orders",
			Description: pOrderWithID + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   order,
		Message:   "Purchase order fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// Update Purchase Order
// @Summary Update Purchase Order by ID
// @Description Update Purchase Order by ID
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/purchase-orders/{po_id} [patch]
func UpdatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdatePurchaseOrderRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		return
	}
	id := mux.Vars(r)["po_id"]
	err := models.UpdatePurchaseOrder(*req, id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update purchase order with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: pOrderWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Purchase order updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// Delete Purchase Order
// @Summary Purchase Purchase Order by ID
// @Description Purchase Purchase Order by ID
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/purchase-orders/{po_id} [delete]
func DeletePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	id := mux.Vars(r)["po_id"]
	err := models.DeletePurchaseOrder(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to delete purchase order with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: pOrderWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Purchase order deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// add product to purchase order item
// @Summary Add product to purchase order item
// @Description Add product to purchase order item
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/purchase_order_items [post]
func AddPurchaseOrderItem(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.PurchaseOrderItem](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		return
	}

	err := models.AddProductToPurchaseOrder(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to add product to purchase order with ID " + req.PoID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Product added to purchase order with ID " + req.PoID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product added to purchase order successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DELETE /purchase_order_items/{po_item_id}
// @Summary Remove product to purchase order item
// @Description Remove product to purchase order item
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/purchase_order_items/{po_item_id} [delete]
func RemovePurchaseOrderItem(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	vars := mux.Vars(r)
	itemID := vars["po_item_id"]
	err := models.RemoveProductFromPurchaseOrder(itemID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to remove product from purchase order item with ID " + itemID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Product removed from purchase order with ID " + itemID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product removed from purchase order successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
