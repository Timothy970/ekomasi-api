// Package handlers provides HTTP request handlers for Point of Sale (POS) operations.
// This file contains handlers for product scanning, payment processing (cash, mobile money, voucher),
// split payments, receipt generation, and transaction management for in-store purchases.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProcessVoucherPaymentHandler processes voucher payments for POS transactions.
// This endpoint validates the voucher code, checks sufficient balance, deducts the amount,
// updates the order status, and logs the transaction.
//
// @Summary Process voucher payment for an order
// @Description Processes a voucher payment, validates balance, updates order status, and records usage
// @Tags POS
// @Accept json
// @Produce json
// @Param request body dtos.VoucherPayment true "Voucher payment details"
// @Success 200 {object} map[string]any "Payment processed successfully"
// @Failure 400 {object} map[string]any "Invalid voucher or insufficient balance"
// @Failure 404 {object} map[string]any "Order not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/pos/voucher-payment [post]
// @Security BearerAuth
func ProcessVoucherPaymentHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.VoucherPayment](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	// Retrieve the order from database
	order, err := models.GetOrderByID(models.DB, req.OrderID)
	if err != nil {
		// Return error if order is not found
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Failed to retrieve order: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   orderError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Validate voucher and retrieve its current balance
	voucherBalance, err := models.ValidateVoucher(models.DB, req.VoucherCode)

	if err != nil {
		// Return error if voucher is invalid or expired
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Failed to validate voucher: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Check if voucher has sufficient balance for the order
	if voucherBalance < order.TotalAmount {
		// Return error if voucher balance is insufficient
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: insufficientPaymentError,
				Code:        http.StatusBadRequest,
			},
			Message:   insufficientPaymentError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Prepare order status update with voucher payment details
	paymentMethod := "VOUCHER"
	paymentStatus := "SUCCESS"
	status := "COMPLETED"
	orderStatusData := dtos.UpdateOrderStatusRequest{
		PaymentMethod: &paymentMethod,
		Status:        &status,
		PaymentStatus: &paymentStatus,
	}

	// Start transaction
	tx, err := models.DB.Begin()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "POS",
				Description: "Failed to start transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	defer tx.Rollback()

	// Update order status in database to mark as paid
	err = models.UpdateOrderStatus(tx, order.OrderID, orderStatusData)
	if err != nil {
		// Return error if order status update fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Failed to update order status: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to update order status",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Deduct amount from voucher balance and record usage history
	err = updateVoucherBalanceAndHistory(tx, req.VoucherCode, voucherBalance, order.TotalAmount, order)
	if err != nil {
		// Return error if voucher update fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Failed to update voucher balance and history: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to update voucher balance and history",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Create transaction log entry for audit trail
	logEntry := &dtos.TransactionsList{
		OrderID:              &order.OrderID,
		TransactionReference: ekomasiMpesaPrefix + order.OrderID,
		Amount:               order.TotalAmount,
		Status:               "COMPLETED",
		PaymentMethod:        "VOUCHER",
	}

	// Store transaction log in database (non-critical, log errors only)
	err = models.InsertTransaction(tx, logEntry)
	if err != nil {
		log.Printf("Failed to store transaction log: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "POS",
				Description: "Failed to commit transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response with order ID
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentSuccessMessage,
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"order_id": order.OrderID,
		},
		Message:   paymentSuccessMessage,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})

}

// updateVoucherBalanceAndHistory is a helper function that updates voucher balance
// and records usage history with order item details. This function is used after
// successful voucher payment processing to maintain voucher transaction history.
func updateVoucherBalanceAndHistory(db models.DBExecutor, voucherCode string, voucherBalance float64, orderTotalAmount float64, order *dtos.Order) error {
	// Build cart items array from order items for history recording
	cartItems := []dtos.CartItem{}
	cartItem := dtos.CartItem{}
	for _, item := range order.Items {
		// Retrieve full product details for each order item
		product, _ := models.GetProductByID(models.DB, item.ID)
		cartItem.Product = *product
		cartItem.Quantity = int(item.StockQuantity)
		cartItems = append(cartItems, cartItem)
	}

	// Deduct order amount from voucher balance
	if err := models.UpdateVoucherBalance(db, voucherCode, voucherBalance-orderTotalAmount); err != nil {
		return err
	}

	// Record voucher usage history with order details
	if err := models.AddVoucherHistory(db, voucherCode, orderTotalAmount, cartItems); err != nil {
		return err
	}
	return nil
}

// ProcessCreditPaymentHandler processes credit payments for POS transactions.
// This endpoint validates the payment amount, updates the order status, calculates change,
// and logs the transaction for auditing purposes.
//
// @Summary Process credit payment for an order
// @Description Processes a credit payment, validates sufficient amount, updates order status, and returns change
// @Tags POS
// @Accept json
// @Produce json
// @Param request body dtos.CreditPayment true "Credit payment details"
// @Success 200 {object} map[string]any "Payment processed successfully with change details"
// @Failure 400 {object} map[string]any "Invalid request or insufficient payment"
// @Failure 404 {object} map[string]any "Order not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/pos/credit-payment [post]
