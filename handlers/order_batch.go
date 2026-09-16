package handlers

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

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
// @Success      200        {object}  map[string]any
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
