package handlers

import (
	"github.com/gin-gonic/gin"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
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
func RiderListOrders(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure the user has permission to view orders
	user, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "")
	if !ok {
		return
	}

	// Parse pagination and filter parameters
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	orderStatus := c.Query("order_status")
	paymentStatus := c.Query("payment_status")
	deliveryStatus := c.Query("delivery_status")
	paymentMethod := c.Query("payment_method")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	timeRange := c.Query("time_range")
	orderID := c.Query("order_id")
	q := c.Query("q")
	past := c.Query("past")

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
	orders, pagination, err := models.ListOrdersByAdmin(models.DB, params)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list orders for rider",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with filtered orders
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Orders fetched successfully for rider",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"orders": orders, "pagination": pagination},
		Message:   "All Rider Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
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
func RiderAssignOrder(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure the user has permission to view orders
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.update")
	if !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.AssignOrderToRiderRequest](c, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Orders") {
		return
	}

	if err := models.AssignOrderToRider(models.DB, *req); err != nil {
		log.Printf("Error assigning order to rider: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to assign order to rider",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary})
		return
	}

	//send notification to rider about new order assignment
	handleSendOrderAssignmentNotification(*req)
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Order has been assigned to rider successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Order assigned to rider successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary})
}

// helper function to send order assignment notification to rider
func handleSendOrderAssignmentNotification(req dtos.AssignOrderToRiderRequest) {
	user, err := models.GetUserByUserID(models.DB, req.RiderID)
	if err != nil {
		log.Printf("Error fetching rider details for notification: %v", err)
		return
	}
	riderName := user.FirstName + " " + user.LastName
	if riderName == " " {
		if user.Email != "" {
			riderName = user.Email
		} else {
			riderName = user.Phone
		}

	}
	order, err := models.GetOrderByID(models.DB, req.OrderID)
	if err != nil {
		log.Printf("Error fetching order details for notification: %v", err)
		return
	}
	var customerName string
	if order.GuestPersonalDetails.FirstName != nil && order.GuestPersonalDetails.LastName != nil {
		customerName = *order.GuestPersonalDetails.FirstName + " " + *order.GuestPersonalDetails.LastName
	}
	if customerName == "" || customerName == " " {
		if order.GuestPersonalDetails.Email != nil {
			customerName = *order.GuestPersonalDetails.Email
		} else if order.GuestPersonalDetails.Phone != nil {
			customerName = *order.GuestPersonalDetails.Phone
		}
	}
	country := ""
	if order.GuestDeliveryAddress.Country != nil {
		country = *order.GuestDeliveryAddress.Country
	}
	city := ""
	if order.GuestDeliveryAddress.City != nil {
		city = *order.GuestDeliveryAddress.City
	}
	street := ""
	if order.GuestDeliveryAddress.Street != nil {
		street = *order.GuestDeliveryAddress.Street
	}
	deliveryAddress := country + " , " + city + " , " + street + " , " + street
	if deliveryAddress == " ,  ,  , " {
		deliveryAddress = *order.DeliveryAddress
	}
	createdAtStr := order.CreatedAt.Format("2006-01-02 15:04:05")
	htmlContent := utils.GenerateOrderAssignmentEmailContent(req.OrderID, riderName, customerName, deliveryAddress, createdAtStr)
	subject := "New Order Assignment: Order #%d" + req.OrderID
	if user.Email != "" {
		notification.SendEmail(user.Email, subject, htmlContent)
	}
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
func RiderUpdateOrderStatus(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure the user has permission to update orders
	user, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.update")
	if !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.UpdateOrderDeliveryStatusRequest](c, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Orders") {
		return
	}

	if err := models.UpdateOrderByRider(models.DB, *req, user.ID); err != nil {
		log.Printf("Error updating order status by rider: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update order status by rider",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Order status has been updated by rider successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Order status updated by rider successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary})
}
