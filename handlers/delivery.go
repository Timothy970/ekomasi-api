package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var deliveryWithID = "Delivery with ID "

// CreateDeliveryHandler creates a new delivery record.
// This endpoint is restricted to administrators.
//
// @Summary      Create Delivery
// @Description  Create a new delivery record
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        delivery  body      dtos.Delivery  true  "Delivery Details"
// @Success      200       {object}  map[string]interface{}
// @Failure      400       {object}  dtos.ErrorResponse
// @Failure      409       {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/deliveries [post]
func CreateDeliveryHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.create")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.Delivery](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Orders") {
		return
	}
	err := models.CreateNewDelivery(*req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to create delivery",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("deliveries_")
	utils.DeleteCacheByPrefix("deliveries_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Delivery added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Delivery added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListDeliveriesHandler retrieves a paginated list of deliveries.
//
// @Summary      List Deliveries
// @Description  Retrieve a list of deliveries with pagination
// @Tags         Deliveries
// @Produce      json
// @Param        page  query     int     false  "Page number"
// @Param        size  query     int     false  "Page size"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      409   {object}  dtos.ErrorResponse
// @Router       /api/deliveries [get]
func ListDeliveriesHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	page, size := parsePagination(c.Query("page"), c.Query("size"))
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
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Orders",
					Description: "Failed to list deliveries",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Deliveries listed successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Deliveries fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// ListUserDeliveriesHandler retrieves deliveries for a specific user.
//
// @Summary      List User Deliveries
// @Description  Retrieve a list of deliveries for a specific user ID
// @Tags         Deliveries
// @Produce      json
// @Param        user_id  path      string  true   "User ID"
// @Param        page     query     int     false  "Page number"
// @Param        size     query     int     false  "Page size"
// @Success      200      {object}  []dtos.Delivery
// @Failure      400      {object}  dtos.ErrorResponse
// @Failure      409      {object}  dtos.ErrorResponse
// @Router       /api/deliveries/user/{user_id} [get]
func ListUserDeliveriesHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	userID := c.Param("user_id")

	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	deliveries, err := models.ListDeliveriesByUserID(userID, page, size)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list user deliveries for user ID " + userID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "User deliveries for user ID " + userID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   deliveries,
		Message:   "Deliveries fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// GetDeliveryHandler retrieves a delivery by ID.
//
// @Summary      Get Delivery
// @Description  Retrieve a delivery by its unique ID
// @Tags         Deliveries
// @Produce      json
// @Param        delivery_id  path      string  true  "Delivery ID"
// @Success      200          {object}  dtos.Delivery
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Router       /api/deliveries/{delivery_id} [get]
func GetDeliveryHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	deliveryID := c.Param("delivery_id")
	deliveries, err := models.GetDeliveryByID(deliveryID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to get delivery with ID " + deliveryID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: deliveryWithID + deliveryID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   deliveries,
		Message:   "Delivery fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// UpdateDeliveryHandler updates an existing delivery.
// This endpoint is restricted to administrators.
//
// @Summary      Update Delivery
// @Description  Update an existing delivery by ID
// @Tags         Admin
// @Produce      json
// @Param        delivery_id  path      string                true  "Delivery ID"
// @Param        delivery     body      dtos.UpdateDelivery   true  "Delivery Details"
// @Success      200          {object}  map[string]interface{}
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/deliveries/{delivery_id} [patch]
func UpdateDeliveryHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.update")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateDelivery](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Orders") {
		return
	}
	deliveryID := c.Param("delivery_id")
	err := models.UpdateDelivery(req.Status, deliveryID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update delivery with ID " + deliveryID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("deliveries_")
	utils.DeleteCacheByPrefix("deliveries_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: deliveryWithID + deliveryID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Delivery updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// DeleteDeliveryHandler deletes a delivery.
// This endpoint is restricted to administrators.
//
// @Summary      Delete Delivery
// @Description  Delete a delivery by ID
// @Tags         Admin
// @Produce      json
// @Param        delivery_id  path      string  true  "Delivery ID"
// @Success      200          {object}  map[string]interface{}
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/deliveries/{delivery_id} [delete]
func DeleteDeliveryHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.delete")
	if !ok {
		return
	}
	deliveryID := c.Param("delivery_id")
	err := models.DeleteDelivery(deliveryID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to delete delivery with ID " + deliveryID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("deliveries_")
	utils.DeleteCacheByPrefix("deliveries_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: deliveryWithID + deliveryID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Delivery deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}
