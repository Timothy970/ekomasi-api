package handlers

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

var (
	InvalidOrderID = "Invalid order ID"
	orderWithID    = "Order with ID "
)

func checkStockAvailability(db models.DBExecutor, productID string, quantity int) error {
	product, err := models.GetProductByID(db, productID)
	if err != nil {
		return err
	}
	//check if product stock is sufficient
	if product.StockQuantity < quantity {
		return fmt.Errorf("insufficient stock for product ID %s", productID)
	}
	return nil
}

func processOrderItems(db models.DBExecutor, items []dtos.OrderItemRequest) (totalAmount, totalDiscount float64, freeShipping bool, err error) {
	for _, item := range items {
		itemTotal := float64(item.Quantity) * item.UnitPrice
		discountValue, discountType, err := models.GetProductDiscount(db, item.ProductID)
		if err != nil {
			log.Printf("Error fetching product discount: %v", err)
			return 0, 0, false, errors.New("failed to fetch product discount")
		}
		if discountType != "" && discountValue > 0 {
			itemTotal = calculateNewPriceWithDiscount(itemTotal, discountValue, discountType)
		}
		totalAmount += itemTotal
		err = checkStockAvailability(db, item.ProductID, item.Quantity)
		if err != nil {
			return 0, 0, false, err
		}
		promo, err := models.GetProductPromotionData(db, item.ProductID)
		if err != nil {
			return 0, 0, false, err
		}

		if promo.Type != "" {
			discount, err := calculateDifferentPromotionTypes(&promo, item)
			if err != nil {
				log.Printf("Discount calculation error for product %s: %v", item.ProductID, err)
			} else {
				totalDiscount += discount
				fmt.Printf("Discount for %s: %.2f\n", item.ProductID, discount)
			}
			if promo.Type == "FreeShipping" {
				freeShipping = true
			}
		}
	}
	return
}

func createOrderItems(db models.DBExecutor, orderID string, items []dtos.OrderItemRequest) error {
	for _, item := range items {
		if _, err := models.CreateOrderItem(db, orderID, item.ProductID, item.VariationSKU, item.Quantity, item.UnitPrice); err != nil {
			return err
		}
	}
	return nil
}

func respondInternalServerError(c *gin.Context, requestSummary string, start time.Time, msg string) {
	utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: msg,
			Code:        http.StatusInternalServerError,
		},
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

func calculateDifferentPromotionTypes(promo *dtos.PromotionData, item dtos.OrderItemRequest) (float64, error) {
	subtotal := float64(item.Quantity) * item.UnitPrice
	switch promo.Type {
	case "Percentage":
		return (promo.Value / 100) * subtotal, nil
	case "Fixed":
		if subtotal > promo.Value {
			return promo.Value, nil
		}
		return subtotal, nil
	case "BOGO":
		if item.Quantity > 1 {
			freeItems := item.Quantity / 2
			return float64(freeItems) * item.UnitPrice, nil
		}
		return 0, nil
	case "Tiered":
		switch {
		case item.Quantity >= 10:
			return 0.20 * subtotal, nil
		case item.Quantity >= 5:
			return 0.10 * subtotal, nil
		default:
			return 0, nil
		}
	default:
		return 0, fmt.Errorf("unsupported promotion type: %s", promo.Type)
	}
}

// ViewOrderAdminHandler retrieves details for any order.
// This endpoint is restricted to administrators.
//
// @Summary      View order (admin)
// @Description  Retrieve a single order by ID – admin privileges required
// @Tags         Admin
// @Produce      json
// @Param        id   path      int  true  "Order ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  dtos.ErrorResponse
// @Failure      404  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/orders/{id} [get]
func ViewOrderAdminHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "")
	if !ok {
		return
	}

	// Extract query parameters
	orderID := c.Query("order_id")
	statusParam := c.Query("status")
	var status *string
	if statusParam != "" {
		status = &statusParam
	}

	functionName := utils.GetCurrentFuncName()
	respondWithError := func(code int, message string) {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: message,
				Code:        code,
			},
			Message:   message,
			TimeTaken: time.Since(start),
			Function:  functionName,
			Request:   c.Request,
			RawBody:   requestSummary,
		})
	}

	respondWithSuccess := func(payload interface{}) {
		utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Order retrieved successfully",
				Code:        http.StatusOK,
			},
			Payload:   payload,
			Message:   "Success",
			TimeTaken: time.Since(start),
			Function:  functionName,
			Request:   c.Request,
			RawBody:   requestSummary,
		})
	}

	// If order ID is provided, fetch specific order
	if orderID != "" {
		order, err := models.GetOrderByID(models.DB, orderID)
		if err != nil {
			log.Printf("%s", err)
			respondWithError(http.StatusInternalServerError, "Could not fetch order")
			return
		}
		respondWithSuccess(order)
		return
	}

	// Otherwise, fetch all orders (optionally filtered by status)
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	orders, err := models.GetAllOrders(models.DB, tenantID, status)
	if err != nil {
		log.Printf("%s", err)
		respondWithError(http.StatusInternalServerError, "Could not fetch orders")
		return
	}
	if len(orders) == 0 {
		respondWithError(http.StatusNotFound, "No orders found")
		return
	}
	respondWithSuccess(orders)
}

// UpdateOrderStatusHandler updates the status of an order.
// This endpoint is restricted to administrators.
//
// @Summary      Update order status (admin)
// @Description  Change the status of an order
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        id     path      int  true  "Order ID"
// @Param        body   body      dtos.UpdateOrderStatusRequest true "New status"
// @Success      200    {object}  dtos.GenericResponse
// @Failure      400    {object}  dtos.ErrorResponse
// @Failure      500    {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/orders/{id}/status [patch]
func UpdateOrderStatusHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	//check if user is admin
	user, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.update")
	if !ok {
		return
	}
	// Extract order ID from query parameters
	orderID := c.Query("order_id")
	if orderID == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Invalid order ID provided",
				Code:        http.StatusBadRequest,
			},
			Message:   InvalidOrderID,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	//check if user role is rider
	//if rider check if the order is assigned to the rider

	if strings.ToLower(user.Role) == "rider" {
		//check if the order is assigned to the user as a rider
		assigned, err := models.IsOrderAssignedToRider(models.DB, orderID, user.ID)
		if err != nil {
			log.Printf("%s", err)
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Orders",
					Description: "Failed to check order assignment for order ID " + orderID + " and user ID " + user.ID,
					Code:        http.StatusNotFound,
				},
				Message:   "Failed to check order assignment",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})

			return
		}
		if !assigned {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Orders",
					Description: "User with ID " + user.ID + " is not assigned to order ID " + orderID,
					Code:        http.StatusForbidden,
				},
				Message:   "You do not have permission to update this order",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}

	// Decode request body
	req, ok := DecodeRequestBody[dtos.UpdateOrderStatusRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Update order status in database
	if err := models.UpdateOrderStatus(models.DB, orderID, *req); err != nil {
		log.Printf("%s", err)

		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update order status for order ID " + orderID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Send receipt email if delivered
	if req.DeliveryStatus != nil && strings.ToLower(*req.DeliveryStatus) == "delivered" {
		//send order receipt email to customer
		if err := processSingleOrder(orderID, "order_receipt"); err != nil {
			log.Printf("Failed to send order receipt for order ID %s: %v", orderID, err)
		}
	}

	// Respond with success message
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Order status for order ID " + orderID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Order status updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListOrders retrieves all orders for the authenticated user.
//
// @Summary      List orders
// @Description  Fetch all orders for current user
// @Tags         Orders
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Failure      500  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/orders [get]
func ListOrders(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	user, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to retrieve user from context",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Parse pagination parameters
	page, limit := parsePagination(c.Query("page"), c.Query("size"))

	// Fetch orders for user
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	orders, pagination, err := models.ListOrdersByUser(models.DB, user.ID, tenantID, page, limit)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list orders for user ID " + user.ID,
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
			Description: "Orders fetched successfully for user ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"orders": orders, "pagination": pagination},
		Message:   "List Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ViewOrder retrieves details of a specific order for the authenticated user.
//
// @Summary      View order
// @Description  Fetch a single order by ID (must belong to current user)
// @Tags         Orders
// @Produce      json
// @Param        id   path      int  true  "Order ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  dtos.ErrorResponse
// @Failure      404  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/orders/{id} [get]
func ViewOrder(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract order ID from query parameters
	orderID := c.Query("order_id")
	if orderID == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Invalid order ID",
				Code:        http.StatusBadRequest,
			},
			Message:   InvalidOrderID,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Get authenticated user from context
	user, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to retrieve user from context",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch order for user
	order, err := models.GetOrderByUser(models.DB, orderID, user.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch order with ID " + orderID + " for user ID " + user.ID,
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
				Description: "Order not found with order ID " + orderID + " and user ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   "Order is not found",
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
			Description: orderWithID + orderID + " fetched successfully for user ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   order,
		Message:   "Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ViewOrderPOS retrieves details of a specific order for POS systems.
//
// @Summary      View order (POS)
// @Description  Fetch a single order by ID for POS
// @Tags         Orders
// @Produce      json
// @Param        order_id   query     string  true  "Order ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      400        {object}  dtos.ErrorResponse
// @Failure      404        {object}  dtos.ErrorResponse
// @Failure      500        {object}  dtos.ErrorResponse
// @Router       /api/pos/orders/view [get]
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
// @Success      200           {array}   map[string]interface{}
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
// @Success      200              {object}  map[string]interface{}
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

func writeOrdersBatch(writer *csv.Writer, orders []dtos.AdminOrder) error {
	for _, order := range orders {
		customerName, customerEmail, customerPhone := formatCustomerDetails(order)
		deliveryAddress := formatDeliveryAddress(order)
		deliveryStatus := ""
		if order.DeliveryStatus != nil {
			deliveryStatus = *order.DeliveryStatus
		}
		deliveryCharge := 0.0
		if order.DeliveryCharge != nil {
			deliveryCharge = *order.DeliveryCharge
		}
		itemsStr := formatOrderItems(order.Items)

		record := []string{
			order.OrderID,
			fmt.Sprintf("%.2f", order.TotalAmount),
			fmt.Sprintf("%.2f", order.TotalDiscount),
			order.DeliveryID,
			order.OrderStatus,
			deliveryStatus,
			order.PaymentMethod,
			fmt.Sprintf("%.2f", deliveryCharge),
			deliveryAddress,
			customerName,
			customerEmail,
			customerPhone,
			fmt.Sprintf("%d", order.ItemsCount),
			order.CreatedAt.Format(time.RFC3339),
			itemsStr,
		}

		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func formatCustomerDetails(order dtos.AdminOrder) (name, email, phone string) {
	if order.User != nil {
		name = order.User.FirstName + " " + order.User.LastName
		email = order.User.Email
		phone = order.User.Phone
	} else if !isGuestPersonalDetailsEmpty(order.GuestPersonalDetails) {
		// Name: Prefer FirstName + LastName, else Email, else Phone
		if order.GuestPersonalDetails.FirstName != nil && order.GuestPersonalDetails.LastName != nil {
			name = *order.GuestPersonalDetails.FirstName + " " + *order.GuestPersonalDetails.LastName
		} else if order.GuestPersonalDetails.Email != nil {
			name = *order.GuestPersonalDetails.Email
		} else if order.GuestPersonalDetails.Phone != nil {
			name = *order.GuestPersonalDetails.Phone
		} else {
			name = ""
		}
		if order.GuestPersonalDetails.Email != nil {
			email = *order.GuestPersonalDetails.Email
		} else {
			email = ""
		}
		if order.GuestPersonalDetails.Phone != nil {
			phone = *order.GuestPersonalDetails.Phone
		} else {
			phone = ""
		}
	} else {
		name = ""
		email = ""
		phone = ""
	}
	return
}

func isGuestPersonalDetailsEmpty(details dtos.GuestPersonalDetails) bool {
	return details.FirstName == nil && details.LastName == nil && details.Email == nil && details.Phone == nil
}

func formatDeliveryAddress(order dtos.AdminOrder) string {
	if order.DeliveryAddress != nil {
		return *order.DeliveryAddress
	}
	safeStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	return fmt.Sprintf("Apartment %s, Street %s, City %s, State %s, Postal Code %s, Country %s",
		safeStr(order.GuestDeliveryAddress.Apartment), safeStr(order.GuestDeliveryAddress.Street),
		safeStr(order.GuestDeliveryAddress.City), safeStr(order.GuestDeliveryAddress.State),
		safeStr(order.GuestDeliveryAddress.PostalCode), safeStr(order.GuestDeliveryAddress.Country))
}

func formatOrderItems(items []dtos.OrderProduct) string {
	var formatted []string
	for _, item := range items {
		formatted = append(formatted, fmt.Sprintf("%s (Qty: %d)", item.Name, item.StockQuantity))
	}
	return strings.Join(formatted, "; ")
}

// GetOrderCountsByStatus retrieves order counts grouped by status.
// This endpoint is restricted to administrators.
//
// @Summary      Get order counts by status
// @Description  Retrieve the count of orders for each status
// @Tags         Admin
// @Produce      json
// @Success      200  {object}  map[string]int
// @Failure      401  {object}  dtos.ErrorResponse
// @Failure      500  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/orders/counts [get]
func GetOrderCountsByStatus(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure the user is an admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "")
	if !ok {
		return
	}

	// Fetch order counts
	counts, err := models.GetOrderCountsByStatus(models.DB)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to get order counts by status",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with counts
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Order counts by status retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   counts,
		Message:   "Orders count by status",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// HoldOrderHandler places an order on hold.
//
// @Summary      Hold order
// @Description  Place an order on hold status
// @Tags         Orders
// @Produce      json
// @Param        order_id   path      string  true  "Order ID"
// @Success      200        {object}  dtos.GenericResponse
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/orders/{order_id}/hold [patch]
func HoldOrderHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract order ID from path variables
	orderID := c.Param("order_id")

	// Update order status to hold
	err := models.HoldOrder(models.DB, orderID)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: fmt.Sprintf("Failed to hold order: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Order held successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Order held successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ReleaseOrderHandler releases an order from hold.
//
// @Summary      Release order
// @Description  Release an order from hold status
// @Tags         Orders
// @Produce      json
// @Param        order_id   path      string  true  "Order ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/orders/{order_id}/release [patch]
func ReleaseOrderHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract order ID from path variables
	orderID := c.Param("order_id")

	// Update order status to release hold
	err := models.ReleaseOrder(models.DB, orderID)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: fmt.Sprintf("Failed to release order: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch updated order details
	order, err := models.GetOrderByID(models.DB, orderID)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: fmt.Sprintf("Failed to fetch order after release: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
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
			Description: "Order released successfully",
			Code:        http.StatusOK,
		},
		Payload:   order,
		Message:   "Order recalled successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// NewCreateOrderHandler handles the creation of a new order.
// It processes order items, calculates totals, applies discounts, and manages stock.
//
// @Summary      Create new order
// @Description  Create a new order with order items and delivery details
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        body  body      dtos.CreateOrderPayload  true  "Order details"
// @Success      201   {object}  dtos.GenericResponse
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Router       /api/orders/new [post]
func NewCreateOrderHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	module := "Orders"

	req, ok := DecodeRequestBody[dtos.CreateOrderPayload](c, requestSummary, start)
	if !ok {
		return
	}
	if req.GuestPersonalDetails != nil && req.GuestPersonalDetails.Email != nil && *req.GuestPersonalDetails.Email == "" {
		req.GuestPersonalDetails.Email = nil
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, module) {
		return
	}
	//validate the phone number in the guest personal details if the phone number is provided
	if req.GuestPersonalDetails != nil && req.GuestPersonalDetails.Phone != nil && *req.GuestPersonalDetails.Phone != "" {
		if !utils.IsValidKenyanPhone(*req.GuestPersonalDetails.Phone) {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      module,
					Description: "Invalid phone number format in guest personal details",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid phone number format",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return
		}
	}
	order, err := buildOrderRequest(req, c, requestSummary, start, module, models.DB)
	if err != nil {
		return
	}

	finalAmount, totalDiscount, err := calculateOrderTotals(order, req.PromoCode, module, models.DB)
	if err != nil {
		log.Printf("[%s] Error calculating totals: %v", module, err)
		respondInternalServerError(c, requestSummary, start, err.Error())
		return
	}

	tx, err := models.DB.Begin()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      module,
				Description: "Failed to start transaction: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	defer tx.Rollback() // Rollback if not committed

	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	orderID, deliveryID, err := createOrderAndDelivery(tx, order, req.StoreID, finalAmount, totalDiscount, module, tenantID)
	if err != nil {
		log.Printf("[%s] Error creating order: %v", module, err)
		respondInternalServerError(c, requestSummary, start, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[%s] Error committing transaction: %v", module, err)
		respondInternalServerError(c, requestSummary, start, "Failed to commit transaction")
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      module,
			Description: fmt.Sprintf("Order with ID %s created successfully", orderID),
			Code:        http.StatusCreated,
		},
		Payload: map[string]interface{}{
			"order_id":    orderID,
			"delivery_id": deliveryID,
			"total":       finalAmount,
		},
		Message:   "Order created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

func buildOrderRequest(req *dtos.CreateOrderPayload, c *gin.Context, requestSummary string, start time.Time, module string, db models.DBExecutor) (*dtos.OrderRequest, error) {
	orderItems, err := getOrderItems(db, req.OrderItems)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      module,
				Description: "Failed to retrieve order items",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return nil, err
	}

	userID, err := determineUserID(req, c)
	if err != nil {
		return nil, err
	}

	deliveryCharge, err := fetchDeliveryCharge(req.DeliveryAddressID, c, requestSummary, start, module, db)
	if err != nil {
		return nil, err
	}

	if req.OrderSource == nil || *req.OrderSource == "" {
		defaultSource := "Online"
		req.OrderSource = &defaultSource
	}

	return &dtos.OrderRequest{
		OrderItems:           orderItems,
		IsGuestOrder:         req.IsGuestOrder,
		GuestPersonalDetails: req.GuestPersonalDetails,
		GuestDeliveryAddress: req.GuestDeliveryAddress,
		UserID:               userID,
		DeliveryCharge:       deliveryCharge,
		OrderSource:          req.OrderSource,
	}, nil
}

func determineUserID(req *dtos.CreateOrderPayload, c *gin.Context) (*string, error) {
	if req.IsGuestOrder != nil && *req.IsGuestOrder {
		return nil, nil
	}
	ok, authUser := middleware.GetTokenAndAuthenticatedUser(c.Writer, c.Request)
	if !ok {
		return nil, fmt.Errorf("authentication failed")
	}
	return &authUser.ID, nil
}

func fetchDeliveryCharge(addressID *int64, c *gin.Context, requestSummary string, start time.Time, module string, db models.DBExecutor) (float64, error) {
	if addressID == nil || *addressID == 0 {
		return 0, nil
	}
	charge, err := getOrderDeliveryCharge(db, int(*addressID))
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      module,
				Description: fmt.Sprintf("Failed to get delivery charge for ID %d: %v", addressID, err),
				Code:        http.StatusUnauthorized,
			},
			Message:   "Failed to get delivery charge",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return 0, err
	}
	return charge, nil
}

func calculateOrderTotals(order *dtos.OrderRequest, promoCode *string, module string, db models.DBExecutor) (float64, float64, error) {
	totalAmount, totalDiscount, freeShipping, err := processOrderItems(db, order.OrderItems)
	if err != nil {
		return 0, 0, err
	}

	if freeShipping {
		order.DeliveryCharge = 0
	}
	totalAmount += order.DeliveryCharge

	if promoCode != nil && *promoCode != "" {
		promoCodeType := models.GetDiscountCodeType(db, *promoCode)
		totalAmount, totalDiscount, err = applyPromoCodeToOrder(db, totalAmount, totalDiscount, *promoCode, promoCodeType, *order)
		if err != nil {
			return 0, 0, err
		}
	}

	return totalAmount - totalDiscount, totalDiscount, nil
}

func createOrderAndDelivery(db models.DBExecutor, order *dtos.OrderRequest, storeID *string, finalAmount, totalDiscount float64, module string, tenantID int) (string, string, error) {
	orderID, deliveryID, err := models.CreateOrder(db, *order, utils.ToString(finalAmount), utils.ToString(totalDiscount), tenantID)
	if err != nil {
		return "", "", err
	}

	if err := createOrderItems(db, orderID, order.OrderItems); err != nil {
		return "", "", err
	}

	if err := models.CreateDeliveries(db, orderID, deliveryID, *order, storeID); err != nil {
		return "", "", err
	}

	if err := deductStock(db, order.OrderItems); err != nil {
		return "", "", err
	}

	//if finalAmount is zero, mark order as paid
	if finalAmount == 0 {
		if err := models.MarkOrderAsPaid(db, orderID); err != nil {
			return "", "", err
		}
	}

	return orderID, deliveryID, nil
}

func getOrderDeliveryCharge(db models.DBExecutor, locationID int) (float64, error) {
	//assume for at store pickup location id is 111111
	if locationID == 111111 {
		return 0, nil
	}
	location, err := models.GetLocationByID(db, locationID)
	if err != nil {
		return 0, err
	}
	return location.Charge, nil
}

func getOrderItems(db models.DBExecutor, items []dtos.OrderItemPayload) ([]dtos.OrderItemRequest, error) {
	var orderItems []dtos.OrderItemRequest
	for _, item := range items {
		product, err := models.GetProductByID(db, item.ProductID)
		if err != nil {
			return nil, err
		}
		// is variation sku is not nil, get additional price for the variation and add to the product price
		if item.VariationSKU != nil && *item.VariationSKU != "" {
			additionalPrice, _, err := models.GetVariationPrice(db, *item.VariationSKU)
			if err != nil {
				return nil, err
			}
			product.Price += additionalPrice
		}
		orderItems = append(orderItems, dtos.OrderItemRequest{
			ProductID: product.ID,
			// VariantID: product.VariantID,
			Quantity:     item.Quantity,
			UnitPrice:    product.Price,
			VariationSKU: item.VariationSKU,
		})
	}
	return orderItems, nil
}

func deductStock(db models.DBExecutor, orderItems []dtos.OrderItemRequest) error {
	for _, item := range orderItems {
		if err := models.DeductProductStock(db, item.ProductID, item.Quantity); err != nil {
			return err
		}
	}
	return nil
}

// DownloadOrderInvoicePDF generates and downloads an invoice PDF for a specific order.
// This endpoint is restricted to administrators.
//
// @Summary      Download order invoice
// @Description  Generate and download PDF invoice for an order
// @Tags         Admin
// @Produce      application/pdf
// @Param        order_id  path      string  true  "Order ID"
// @Success      200       {file}    file
// @Failure      404       {object}  dtos.ErrorResponse
// @Failure      500       {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/orders/{order_id}/invoice [get]
func DownloadOrderInvoicePDF(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure the user has permission to view orders
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "")
	if !ok {
		return
	}

	// Extract order ID from path variables
	orderID := c.Param("order_id")

	// Fetch order details
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	params := models.AdminOrderParameters{
		OrderID: orderID,
		Page:    page,
		Limit:   limit,
	}
	orders, _, err := models.ListOrdersByAdmin(models.DB, params)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to get order to download invoice PDF",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if len(orders) == 0 {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Order not found for invoice PDF download",
				Code:        http.StatusNotFound,
			},
			Message:   "Order not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Generate PDF invoice
	pdfBytes, err := utils.GenerateInvoicePDF(orders[0])
	if err != nil {
		fmt.Println("Failed to generate invoice PDF error :", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to generate invoice PDF " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to generate invoice PDF",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
