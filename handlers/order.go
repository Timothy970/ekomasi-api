package handlers

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"

	"github.com/gorilla/mux"
)

var (
	InvalidOrderID = "Invalid order ID"
	orderWithID    = "Order with ID "
)

func checkStockAvailability(productID string, quantity int) error {
	product, err := models.GetProductByID(productID)
	if err != nil {
		return err
	}
	//check if product stock is sufficient
	if product.StockQuantity < quantity {
		return fmt.Errorf("insufficient stock for product ID %s", productID)
	}
	return nil
}

func processOrderItems(items []dtos.OrderItemRequest) (totalAmount, totalDiscount float64, freeShipping bool, err error) {
	for _, item := range items {
		itemTotal := float64(item.Quantity) * item.UnitPrice
		totalAmount += itemTotal
		err := checkStockAvailability(item.ProductID, item.Quantity)
		if err != nil {
			return 0, 0, false, err
		}
		promo, err := models.GetProductPromotionData(item.ProductID)
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

func createOrderItems(orderID string, items []dtos.OrderItemRequest) error {
	for _, item := range items {
		var variantID string
		if item.VariantID != nil {
			variantID = *item.VariantID
		} else {
			variantID = ""
		}
		if _, err := models.CreateOrderItem(orderID, item.ProductID, variantID, item.Quantity, item.UnitPrice); err != nil {
			return err
		}
	}
	return nil
}

func respondInternalServerError(w http.ResponseWriter, r *http.Request, requestSummary string, start time.Time, msg string) {
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: msg,
			Code:        http.StatusInternalServerError,
		},
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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

// ViewOrderAdminHandler returns details for ANY order (admin access only)
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
func ViewOrderAdminHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	orderID := r.URL.Query().Get("order_id")
	statusParam := r.URL.Query().Get("status")
	var status *string
	if statusParam != "" {
		status = &statusParam
	}
	functionName := utils.GetCurrentFuncName()
	respondWithError := func(code int, message string) {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: message,
				Code:        code,
			},
			Message:   message,
			TimeTaken: time.Since(start),
			Function:  functionName,
			Request:   r,
			RawBody:   requestSummary,
		})
	}

	respondWithSuccess := func(payload interface{}) {
		utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Order retrieved successfully",
				Code:        http.StatusOK,
			},
			Payload:   payload,
			Message:   "Success",
			TimeTaken: time.Since(start),
			Function:  functionName,
			Request:   r,
			RawBody:   requestSummary,
		})
	}

	if orderID != "" {
		order, err := models.GetOrderByID(orderID)
		if err != nil {
			log.Printf("%s", err)
			respondWithError(http.StatusInternalServerError, "Could not fetch order")
			return
		}
		respondWithSuccess(order)
		return
	}

	orders, err := models.GetAllOrders(status)
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

// UpdateOrderStatusHandler updates an order status (admin)
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
// @Router       /admin/orders/{id}/status [put]
func UpdateOrderStatusHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Invalid order ID provided",
				Code:        http.StatusBadRequest,
			},
			Message:   InvalidOrderID,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	req, ok := DecodeRequestBody[dtos.UpdateOrderStatusRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	if err := models.UpdateOrderStatus(orderID, *req); err != nil {
		log.Printf("%s", err)

		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update order status for order ID " + orderID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if req.DeliveryStatus != nil && strings.ToLower(*req.DeliveryStatus) == "delivered" {
		//send order receipt email to customer
		if err := processSingleOrder(orderID, "order_receipt"); err != nil {
			log.Printf("Failed to send order receipt for order ID %s: %v", orderID, err)
		}
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Order status for order ID " + orderID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Order status updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ListOrders returns all orders for the authenticated user
// @Summary      List orders
// @Description  Fetch all orders for current user
// @Tags         Orders
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Failure      500  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/orders [get]
func ListOrders(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to retrieve user from context",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	orders, pagination, err := models.ListOrdersByUser(user.ID, page, limit)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list orders for user ID " + user.ID,
				Code:        http.StatusInternalServerError,
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
			Description: "Orders fetched successfully for user ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"orders": orders, "pagination": pagination},
		Message:   "List Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ViewOrder returns details of a specific order for the authenticated user
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
func ViewOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Invalid order ID",
				Code:        http.StatusBadRequest,
			},
			Message:   InvalidOrderID,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to retrieve user from context",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	order, err := models.GetOrderByUser(orderID, user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch order with ID " + orderID + " for user ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if order == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Order not found with order ID " + orderID + " and user ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   "Order not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: orderWithID + orderID + " fetched successfully for user ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   order,
		Message:   "Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func ViewOrderPOS(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Invalid order ID provided: " + orderID,
				Code:        http.StatusBadRequest,
			},
			Message:   InvalidOrderID,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	order, err := models.GetOrderByID(orderID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch order with ID " + orderID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if order == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: orderWithID + orderID + " not found",
				Code:        http.StatusNotFound,
			},
			Message:   "Order not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: orderWithID + orderID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   order,
		Message:   "Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// List Guest orders by order id, email and phone number
// @Summary      List order for guest
// @Description  Fetch all order for guest user
// @Tags         Orders
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Failure      500  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/guest-orders/{order_id}/{email}/{phone_number} [get]
func ListGuestOrders(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	orderID := mux.Vars(r)["order_id"]
	email := mux.Vars(r)["email"]
	phone := mux.Vars(r)["phone_number"]

	orders, err := models.ListGuestOrders(orderID, email, phone)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list guest orders for order ID " + orderID + " with email " + email + " and phone " + phone,
				Code:        http.StatusInternalServerError,
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
			Description: "Guest orders fetched successfully for order ID " + orderID + " with email " + email + " and phone " + phone,
			Code:        http.StatusOK,
		},
		Payload:   orders,
		Message:   "List Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// List all orders for admin
func AdminListOrders(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure the user is an admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
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
	orders, pagination, err := models.ListOrdersByAdmin(params)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to list orders for admin",
				Code:        http.StatusInternalServerError,
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
			Description: "Orders fetched successfully for admin",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"orders": orders, "pagination": pagination},
		Message:   "All Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// StreamOrdersCSV handles CSV export with streaming for large datasets
func StreamOrdersCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=orders_export.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writeCSVHeader(writer); err != nil {
		writeCSVError(w, "Failed to write CSV header: "+err.Error())
		return
	}
	page := 1
	limit := 1000
	orderStatus := r.URL.Query().Get("order_status")
	paymentStatus := r.URL.Query().Get("payment_status")
	deliveryStatus := r.URL.Query().Get("delivery_status")
	paymentMethod := r.URL.Query().Get("payment_method")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	timeRange := r.URL.Query().Get("time_range")
	orderID := r.URL.Query().Get("order_id")
	q := r.URL.Query().Get("q")
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
		orders, meta, err := models.ListOrdersByAdmin(params)
		if err != nil {
			writeCSVError(w, "Failed to fetch orders: "+err.Error())
			return
		}

		if err := writeOrdersBatch(writer, orders); err != nil {
			writeCSVError(w, "Failed to write CSV record: "+err.Error())
			return
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			writeCSVError(w, "Failed to flush CSV data: "+err.Error())
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

func writeCSVError(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusInternalServerError)
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
	} else {
		name = order.GuestPersonalDetails.FirstName + " " + order.GuestPersonalDetails.LastName
		email = order.GuestPersonalDetails.Email
		phone = order.GuestPersonalDetails.Phone
	}
	return
}

func formatDeliveryAddress(order dtos.AdminOrder) string {
	if order.DeliveryAddress != nil {
		return *order.DeliveryAddress
	}
	return fmt.Sprintf("Apartment %s, Street %s, City %s, State %s, Postal Code %s, Country %s",
		order.GuestDeliveryAddress.Apartment, order.GuestDeliveryAddress.Street,
		order.GuestDeliveryAddress.City, order.GuestDeliveryAddress.State,
		order.GuestDeliveryAddress.PostalCode, order.GuestDeliveryAddress.Country)
}

func formatOrderItems(items []dtos.OrderProduct) string {
	var formatted []string
	for _, item := range items {
		formatted = append(formatted, fmt.Sprintf("%s (Qty: %d)", item.Name, item.StockQuantity))
	}
	return strings.Join(formatted, "; ")
}

// Get order counts grouped by status
func GetOrderCountsByStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure the user is an admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	counts, err := models.GetOrderCountsByStatus()
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to get order counts by status",
				Code:        http.StatusInternalServerError,
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
			Description: "Order counts by status retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   counts,
		Message:   "Orders count by status",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func HoldOrderHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	orderID := mux.Vars(r)["order_id"]
	err := models.HoldOrder(orderID)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: fmt.Sprintf("Failed to hold order: %s", err.Error()),
				Code:        http.StatusInternalServerError,
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
			Description: "Order held successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Order held successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func ReleaseOrderHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	orderID := mux.Vars(r)["order_id"]
	err := models.ReleaseOrder(orderID)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: fmt.Sprintf("Failed to release order: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	order, err := models.GetOrderByID(orderID)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: fmt.Sprintf("Failed to fetch order after release: %s", err.Error()),
				Code:        http.StatusInternalServerError,
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
			Description: "Order released successfully",
			Code:        http.StatusOK,
		},
		Payload:   order,
		Message:   "Order recalled successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// New function to create a new order
// CreateOrderHandler handles the creation of a new order
// Removes the delivery charge from the payload for security purposes
//Removed product price from the payload to avoid manipulation
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

func NewCreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	module := "Orders"

	// Decode request body
	req, ok := DecodeRequestBody[dtos.CreateOrderPayload](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate request payload
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, module) {
		return
	}
	// Fetch order items using product IDs
	orderItems, err := getOrderItems(req.OrderItems)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      module,
				Description: "Failed to retrieve order items",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Determine user ID (guest or authenticated)
	var userID *string
	if req.IsGuestOrder == nil || !*req.IsGuestOrder {
		ok, authUser := middleware.GetTokenAndAuthenticatedUser(w, r)
		if !ok {
			return
		}
		userID = &authUser.ID
	}

	// Build order payload
	order := dtos.OrderRequest{
		OrderItems:           orderItems,
		IsGuestOrder:         req.IsGuestOrder,
		GuestPersonalDetails: req.GuestPersonalDetails,
		GuestDeliveryAddress: req.GuestDeliveryAddress,
		UserID:               userID,
	}

	// Fetch delivery charge
	if req.DeliveryAddressID != nil && *req.DeliveryAddressID != 0 {
		order.DeliveryCharge, err = getOrderDeliveryCharge(int(*req.DeliveryAddressID))
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      module,
					Description: fmt.Sprintf("Failed to get delivery charge for ID %d: %v", req.DeliveryAddressID, err),
					Code:        http.StatusUnauthorized,
				},
				Message:   "Failed to get delivery charge",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
	} else {
		order.DeliveryCharge = 0
	}
	// Process order items: discounts, totals, and stock checks
	totalAmount, totalDiscount, freeShipping, err := processOrderItems(order.OrderItems)
	if err != nil {
		log.Printf("[%s] Error processing order items: %v", module, err)
		respondInternalServerError(w, r, requestSummary, start, err.Error())
		return
	}
	if freeShipping {
		order.DeliveryCharge = 0
	}
	totalAmount += order.DeliveryCharge

	// Apply promo code if present
	if req.PromoCode != nil && *req.PromoCode != "" {
		promoCodeType := models.GetDiscountCodeType(*req.PromoCode)

		totalAmount, totalDiscount, err = applyPromoCodeToOrder(totalAmount, totalDiscount, *req.PromoCode, promoCodeType)
		if err != nil {
			log.Printf("[%s] Error applying promo code: %v", module, err)
			respondInternalServerError(w, r, requestSummary, start, err.Error())
			return
		}
	}

	// Create order and delivery records
	finalAmount := totalAmount - totalDiscount

	orderID, deliveryID, err := models.CreateOrder(order, utils.ToString(finalAmount), utils.ToString(totalDiscount))
	if err != nil {
		log.Printf("[%s] Error creating order: %v", module, err)
		respondInternalServerError(w, r, requestSummary, start, err.Error())
		return
	}

	if err := createOrderItems(orderID, order.OrderItems); err != nil {
		log.Printf("[%s] Error creating order items: %v", module, err)
		respondInternalServerError(w, r, requestSummary, start, err.Error())
		return
	}

	if err := models.CreateDeliveries(orderID, deliveryID, order, req.StoreID); err != nil {
		log.Printf("[%s] Error creating delivery: %v", module, err)
		respondInternalServerError(w, r, requestSummary, start, err.Error())
		return
	}

	// Deduct stock quantities
	if err := deductStock(order.OrderItems); err != nil {
		log.Printf("[%s] Error deducting stock: %v", module, err)
		respondInternalServerError(w, r, requestSummary, start, err.Error())
		return
	}

	// Success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
		RawBody:   requestSummary,
	})
}

func getOrderDeliveryCharge(locationID int) (float64, error) {
	//assume for at store pickup location id is 111111
	if locationID == 111111 {
		return 0, nil
	}
	location, err := models.GetLocationByID(locationID)
	if err != nil {
		return 0, err
	}
	return location.Charge, nil
}

func getOrderItems(items []dtos.OrderItemPayload) ([]dtos.OrderItemRequest, error) {
	var orderItems []dtos.OrderItemRequest
	for _, item := range items {
		product, err := models.GetProductByID(item.ProductID)
		if err != nil {
			return nil, err
		}
		orderItems = append(orderItems, dtos.OrderItemRequest{
			ProductID: product.ID,
			// VariantID: product.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: product.Price,
		})
	}
	return orderItems, nil
}

func deductStock(orderItems []dtos.OrderItemRequest) error {
	for _, item := range orderItems {
		err := models.DeductProductStock(item.ProductID, item.Quantity)
		if err != nil {
			return err
		}
	}
	return nil
}

func DownloadOrderInvoicePDF(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure the user is an admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		return
	}
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	orderID := mux.Vars(r)["order_id"]
	params := models.AdminOrderParameters{
		OrderID: orderID,
		Page:    page,
		Limit:   limit,
	}
	orders, _, err := models.ListOrdersByAdmin(params)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to get order to download invoice PDF",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if len(orders) == 0 {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Order not found for invoice PDF download",
				Code:        http.StatusNotFound,
			},
			Message:   "Order not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	pdfBytes, err := utils.GenerateInvoicePDF(orders[0])
	if err != nil {
		fmt.Println("Failed to generate invoice PDF error :", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to generate invoice PDF " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to generate invoice PDF",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=invoice_"+orderID+".pdf")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)

}
