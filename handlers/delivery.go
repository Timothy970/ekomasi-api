package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

var deliveryWithID = "Delivery with ID "

//	Create Delivery
//
// @Summary Create Delivery
// @Description Create Delivery
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/deliveries [post]
func CreateDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.Delivery](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		return
	}
	err := models.CreateNewDelivery(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to create delivery",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("deliveries_")
	utils.DeleteCacheByPrefix("deliveries_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Delivery added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Delivery added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

//	List Deliveries (Paginated)
//
// @Summary List Delivery
// @Description List Delivery
// @Tags Deliveries
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/deliveries [get]
func ListDeliveriesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyDeliveries := fmt.Sprintf("deliveries_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("deliveries_pagination_%d_size_%d", page, size)
	var deliveries []dtos.Delivery
	var cachedDeliveries []dtos.Delivery
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyDeliveries, &cachedDeliveries)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedDeliveries == nil {
		var err error
		deliveries, pagination, err = models.ListDeliveries(page, size)
		if err != nil {
			log.Printf("Error adding new shipping rate: %v", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Orders",
					Description: "Failed to list deliveries",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKeyDeliveries, cachedDeliveries)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		deliveries = cachedDeliveries
		pagination = cachedPagination
	}
	response := map[string]interface{}{
		"deliveries": deliveries,
		"pagination": pagination,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Deliveries listed successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Deliveries fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

//	List Deliveries by user_id
//
// @Summary List Delivery
// @Description List Delivery
// @Tags Deliveries
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/deliveries/user/{user_id} [get]
func ListUserDeliveriesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	userID := mux.Vars(r)["user_id"]

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	deliveries, err := models.ListDeliveriesByUserID(userID, page, size)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list user deliveries for user ID " + userID,
				Code:        http.StatusBadRequest,
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
			Description: "User deliveries for user ID " + userID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   deliveries,
		Message:   "Deliveries fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

//	Get Delivery by ID
//
// @Summary List Delivery
// @Description List Delivery
// @Tags Deliveries
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/deliveries/{delivery_id} [get]
func GetDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	deliveryID := mux.Vars(r)["delivery_id"]
	deliveries, err := models.GetDeliveryByID(deliveryID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to get delivery with ID " + deliveryID,
				Code:        http.StatusBadRequest,
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
			Description: deliveryWithID + deliveryID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   deliveries,
		Message:   "Delivery fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

//	Update Delivery
//
// @Summary Update Delivery
// @Description Update Delivery
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/deliveries/{delivery_id} [patch]
func UpdateDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateDelivery](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		return
	}
	deliveryID := mux.Vars(r)["delivery_id"]
	err := models.UpdateDelivery(req.Status, deliveryID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update delivery with ID " + deliveryID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("deliveries_")
	utils.DeleteCacheByPrefix("deliveries_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: deliveryWithID + deliveryID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Delivery updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

//	Delete Delivery
//
// @Summary Delete Delivery
// @Description Delete Delivery
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/deliveries/{delivery_id} [delete]
func DeleteDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	deliveryID := mux.Vars(r)["delivery_id"]
	err := models.DeleteDelivery(deliveryID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to delete delivery with ID " + deliveryID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("deliveries_")
	utils.DeleteCacheByPrefix("deliveries_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: deliveryWithID + deliveryID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Delivery deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}
