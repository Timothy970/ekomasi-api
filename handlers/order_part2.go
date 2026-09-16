package handlers

import (
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
// @Success      200  {array}   map[string]any
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
// @Success      200  {object}  map[string]any
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
// @Success      200        {object}  map[string]any
// @Failure      400        {object}  dtos.ErrorResponse
// @Failure      404        {object}  dtos.ErrorResponse
// @Failure      500        {object}  dtos.ErrorResponse
// @Router       /api/pos/orders/view [get]
