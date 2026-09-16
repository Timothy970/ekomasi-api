package handlers

import (
	"encoding/csv"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

func ViewOrderPOS(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract order ID from query parameters
	orderID := c.Query("order_id")
	if orderID == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Invalid order ID provided: " + orderID,
				Code:        http.StatusBadRequest,
			},
			Message:   InvalidOrderID,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch order by ID
	order, err := models.GetOrderByID(models.DB, orderID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch order with ID " + orderID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if order == nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: orderWithID + orderID + " not found",
				Code:        http.StatusNotFound,
			},
			Message:   "Order not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with order details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: orderWithID + orderID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   order,
		Message:   "Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListGuestOrders retrieves orders for a guest user based on order ID, email, and phone number.
//
// @Summary      List order for guest
// @Description  Fetch all order for guest user
// @Tags         Orders
// @Produce      json
// @Param        order_id      path      string  true  "Order ID"
// @Param        email         path      string  true  "Email"
// @Param        phone_number  path      string  true  "Phone Number"
// @Success      200           {array}   map[string]any
// @Failure      500           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/guest-orders/{order_id}/{email}/{phone_number} [get]
func ListGuestOrders(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract parameters from path variables
	orderID := c.Param("order_id")
	email := c.Param("email")
	phone := c.Param("phone_number")

	// Fetch guest orders
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	orders, err := models.ListGuestOrders(models.DB, orderID, email, phone, tenantID)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list guest orders for order ID " + orderID + " with email " + email + " and phone " + phone,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with orders
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Guest orders fetched successfully for order ID " + orderID + " with email " + email + " and phone " + phone,
			Code:        http.StatusOK,
		},
		Payload:   orders,
		Message:   "List Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AdminListOrders retrieves all orders with optional filtering for administrators.
// This endpoint is restricted to administrators.
//
// @Summary      List all orders (admin)
// @Description  Fetch all orders with optional filtering (status, date, etc.)
// @Tags         Admin
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
// @Success      200              {object}  map[string]any
// @Failure      401              {object}  dtos.ErrorResponse
// @Failure      500              {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/orders [get]
func AdminListOrders(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure the user has permission to view orders
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "")
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
	userID := c.Query("user_id")

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
		UserID:         userID,
	}

	// Fetch orders based on parameters
	orders, pagination, err := models.ListOrdersByAdmin(models.DB, params)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list orders for admin",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with filtered orders
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Orders fetched successfully for admin",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"orders": orders, "pagination": pagination},
		Message:   "All Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// StreamOrdersCSV exports orders as a CSV file with streaming.
// This endpoint is restricted to administrators.
//
// @Summary      Export orders CSV
// @Description  Stream all orders as a CSV file download
// @Tags         Admin
// @Produce      text/csv
// @Param        order_status     query     string  false  "Filter by order status"
// @Param        payment_status   query     string  false  "Filter by payment status"
// @Param        delivery_status  query     string  false  "Filter by delivery status"
// @Param        payment_method   query     string  false  "Filter by payment method"
// @Param        start_date       query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end_date         query     string  false  "End date (YYYY-MM-DD)"
// @Param        time_range       query     string  false  "Time range"
// @Param        order_id         query     string  false  "Filter by Order ID"
// @Param        q                query     string  false  "Search query"
// @Success      200              {file}    file
// @Failure      500              {string}  string "Internal Server Error"
// @Security     BearerAuth
// @Router       /admin/orders/export/csv [get]
func StreamOrdersCSV(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=orders_export.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	if err := writeCSVHeader(writer); err != nil {
		writeCSVError(c, "Failed to write CSV header: "+err.Error())
		return
	}
	page := 1
	limit := 1000
	orderStatus := c.Query("order_status")
	paymentStatus := c.Query("payment_status")
	deliveryStatus := c.Query("delivery_status")
	paymentMethod := c.Query("payment_method")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	timeRange := c.Query("time_range")
	orderID := c.Query("order_id")
	q := c.Query("q")
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
	}

	for {
		orders, meta, err := models.ListOrdersByAdmin(models.DB, params)
		if err != nil {
			writeCSVError(c, "Failed to fetch orders: "+err.Error())
			return
		}

		if err := writeOrdersBatch(writer, orders); err != nil {
			writeCSVError(c, "Failed to write CSV record: "+err.Error())
			return
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			writeCSVError(c, "Failed to flush CSV data: "+err.Error())
			return
		}

		if meta == nil || page >= meta.TotalPages {
			break
		}
		page++
	}
}

func writeCSVHeader(writer *csv.Writer) error {
	header := []string{
		"Order ID", "Total Amount", "Total Discount", "Delivery ID", "Order Status",
		"Delivery Status", "Payment Method", "Delivery Charge", "Delivery Address",
		"Customer Name", "Customer Email", "Customer Phone", "Items Count", "Created At", "Items",
	}
	return writer.Write(header)
}

func writeCSVError(c *gin.Context, msg string) {
	c.String(http.StatusInternalServerError, msg)
}
