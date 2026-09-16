package handlers

import (
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

	finalAmount, totalDiscount, err := calculateOrderTotals(order, req.PromoCode, models.DB)
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
	orderID, deliveryID, err := createOrderAndDelivery(tx, order, req.StoreID, finalAmount, totalDiscount, tenantID)
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
		Payload: map[string]any{
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

func calculateOrderTotals(order *dtos.OrderRequest, promoCode *string, db models.DBExecutor) (float64, float64, error) {
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

func createOrderAndDelivery(db models.DBExecutor, order *dtos.OrderRequest, storeID *string, finalAmount, totalDiscount float64, tenantID int) (string, string, error) {
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
