package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Riders orders retrieves all orders attached to a rider. This endpoint is used by riders to view their assigned orders.
//
// @Summary      List all orders (rider view)
// @Description  Fetch all orders with optional filtering (status, date, etc.)
// @Tags         Rider
// @Produce      json
// @Param        page             query     int     false  "Page number"
// @Param        size             query     int     false  "Page size"
// @Param        order_status     query     string  false  "Filter by order status"
// @Param        payment_status   query     string  false  "Filter by payment status"
// @Param        delivery_status  query     string  false  "Filter by delivery status"
// @Param        payment_method   query     string  false  "Filter by payment method"
// @Param        start_date       query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end_date         query     string  false  "End date (YYYY-MM-DD)"
// @Param        time_range       query     string  false  "Time range (e.g., 'today', 'week')"
// @Param        order_id         query     string  false  "Filter by Order ID"
// @Param        q                query     string  false  "Search query"
// @Success      200              {object}  map[string]interface{}
// @Failure      401              {object}  dtos.ErrorResponse
// @Failure      500              {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /rider/orders [get]
func RiderListOrders(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Ensure the user has permission to view orders
	user, ok := utils.RequirePermissions(r, w, start, requestSummary, "Orders", "")
	if !ok {
		return
	}

	// Parse pagination and filter parameters
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	orderStatus := r.URL.Query().Get("order_status")
	paymentStatus := r.URL.Query().Get("payment_status")
	deliveryStatus := r.URL.Query().Get("delivery_status")
	paymentMethod := r.URL.Query().Get("payment_method")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	timeRange := r.URL.Query().Get("time_range")
	orderID := r.URL.Query().Get("order_id")
	q := r.URL.Query().Get("q")
	past := r.URL.Query().Get("past")

	params := models.AdminOrderParameters{
		OrderStatus:    orderStatus,
		PaymentStatus:  paymentStatus,
		DeliveryStatus: deliveryStatus,
		PaymentMethod:  paymentMethod,
		TimeRange:      timeRange,
		OrderID:        orderID,
		Q:              q,
		Page:           page,
		Limit:          limit,
		StartDate:      startDate,
		EndDate:        endDate,
		Past:           past,
		RiderID:        user.ID,
	}

	// Fetch orders based on parameters
	orders, pagination, err := models.ListOrdersByAdmin(models.DB, params, "")
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list orders for rider",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with filtered orders
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Orders fetched successfully for rider",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"orders": orders, "pagination": pagination},
		Message:   "All Rider Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Assign an order to a rider
//
// @Summary      Assign order to rider
// @Description  Assign an order to a rider
// @Tags         Rider
// @Produce      json
// @Param        order_id   path      int     true  "Order ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      400        {object}  dtos.ErrorResponse
// @Failure      401        {object}  dtos.ErrorResponse
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/orders/assign-rider [post]
func RiderAssignOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Ensure the user has permission to view orders
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Orders", "orders.update")
	if !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.AssignOrderToRiderRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		return
	}

	if err := models.AssignOrderToRider(models.DB, *req); err != nil {
		log.Printf("Error assigning order to rider: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to assign order to rider",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Order has been assigned to rider successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Order assigned to rider successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Handler function for a rider to update order status
//
// @Summary      Update order status (rider)
// @Description  Allows a rider to update the status of an assigned order
// @Tags         Rider
// @Produce      json
// @Param        order_id     path      int     true  "Order ID"
// @Param        order_status query     string  true  "New order status"
// @Success      200          {object}  map[string]interface{}
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      401          {object}  dtos.ErrorResponse
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /order/rider/status [patch]
func RiderUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Ensure the user has permission to update orders
	user, ok := utils.RequirePermissions(r, w, start, requestSummary, "Orders", "orders.update")
	if !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.UpdateOrderDeliveryStatusRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		return
	}

	if err := models.UpdateOrderByRider(models.DB, *req, user.ID); err != nil {
		log.Printf("Error updating order status by rider: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update order status by rider",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Order status has been updated by rider successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Order status updated by rider successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
