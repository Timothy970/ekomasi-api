package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"

	"github.com/gorilla/mux"
)

var InvalidOrderID = "Invalid order ID"

// CreateOrder creates a new order for the authenticated user
// @Summary      Create order
// @Description  Submit a new order
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        body  body      dtos.OrderRequest true "Order payload"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/orders [post]
func CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	req, ok := DecodeRequestBody[dtos.OrderRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	totalAmount, totalDiscount, applyFreeShipping, err := processOrderItems(req.OrderItems)
	if err != nil {
		log.Printf("Error processing order items: %v", err)
		respondInternalServerError(w, r, requestSummary, start, err.Error())
		return
	}

	if applyFreeShipping {
		fmt.Println("Free shipping applied")
		req.DeliveryCharge = 0
	}

	orderID, deliveryID, err := models.CreateOrder(*req, utils.ToString(totalAmount), utils.ToString(totalDiscount))
	if err != nil {
		log.Printf("Error creating order:::%v", err)
		respondInternalServerError(w, r, requestSummary, start, err.Error())
		return
	}

	if err := createOrderItems(orderID, req.OrderItems); err != nil {
		log.Printf("Error when creating order items:::%v", err)
		respondInternalServerError(w, r, requestSummary, start, err.Error())
		return
	}

	err = models.CreateDeliveries(orderID, deliveryID, *req)
	if err != nil {
		log.Printf("Error when creating delivery:::%v", err)
		respondInternalServerError(w, r, requestSummary, start, err.Error())
		return
	}

	finalAmount := totalAmount + req.DeliveryCharge - totalDiscount
	//send sms and email notification
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusCreated,
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
func processOrderItems(items []dtos.OrderItemRequest) (totalAmount, totalDiscount float64, freeShipping bool, err error) {
	for _, item := range items {
		itemTotal := float64(item.Quantity) * item.UnitPrice
		totalAmount += itemTotal

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
		itemTotal := float64(item.Quantity) * item.UnitPrice
		var variantID string
		if item.VariantID != nil {
			variantID = *item.VariantID
		} else {
			variantID = ""
		}
		if _, err := models.CreateOrderItem(orderID, item.ProductID, variantID, utils.ToString(itemTotal), utils.ToString(itemTotal)); err != nil {
			return err
		}
	}
	return nil
}

func respondInternalServerError(w http.ResponseWriter, r *http.Request, requestSummary string, start time.Time, msg string) {
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		Code:      http.StatusInternalServerError,
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
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
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
			Code:      code,
			Message:   message,
			TimeTaken: time.Since(start),
			Function:  functionName,
			Request:   r,
			RawBody:   requestSummary,
		})
	}

	respondWithSuccess := func(payload interface{}) {
		utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
			Code:      http.StatusOK,
			Payload:   payload,
			Message:   "Success",
			TimeTaken: time.Since(start),
			Function:  functionName,
			Request:   r,
			RawBody:   requestSummary,
		})
	}

	if orderID != "" {
		order, err := models.GetOrder(orderID)
		if err != nil {
			log.Printf("%s", err)
			respondWithError(http.StatusInternalServerError, "Could not fetch order")
			return
		}
		if order == nil {
			respondWithError(http.StatusNotFound, "Order not found")
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
			Code:      http.StatusBadRequest,
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

	if err := models.UpdateOrderStatus(orderID, req.Status); err != nil {
		log.Printf("%s", err)
		msg := ""
		if err.Error() == "order not found" {
			msg = "order not found"
		} else {
			msg = "Failed to update order status"
		}
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   msg,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
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
			Code:      http.StatusInternalServerError,
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	orders, err := models.ListOrdersByUser(user.ID)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   orders,
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
			Code:      http.StatusBadRequest,
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
			Code:      http.StatusInternalServerError,
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
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if order == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   "Order not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
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
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   orders,
		Message:   "List Orders",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
