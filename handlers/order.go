package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
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
// @Success      200  {object}  map[string]any
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

	respondWithSuccess := func(payload any) {
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
