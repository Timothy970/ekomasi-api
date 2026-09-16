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

// @Security BearerAuth
func ProcessCreditPaymentHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.CreditPayment](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}
	order, err := models.GetOrderByID(models.DB, req.OrderID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Failed to retrieve order with ID %s: %s", req.OrderID, err.Error()),
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

	// Prepare order status update data with payment method and status
	paymentMethod := "CREDIT"
	paymentStatus := "SUCCESS"
	status := "COMPLETED"
	orderStatusData := dtos.UpdateOrderStatusRequest{
		PaymentMethod: &paymentMethod,
		Status:        &status,
		PaymentStatus: &paymentStatus,
	}

	// Update order status in database to mark as paid
	err = models.UpdateOrderStatus(models.DB, order.OrderID, orderStatusData)
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

	// Create transaction log entry for audit trail
	logEntry := &dtos.TransactionsList{
		OrderID:              &order.OrderID,
		TransactionReference: ekomasiMpesaPrefix + order.OrderID,
		Amount:               order.TotalAmount,
		Status:               "COMPLETED",
		PaymentMethod:        "CREDIT",
	}

	// Store transaction log in database for record-keeping
	err = models.InsertTransaction(models.DB, logEntry)
	if err != nil {
		// Log error but don't fail the transaction
		log.Printf("Failed to store log for transaction: %v", err)
	}

	// Return success response with order ID and change amount
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment processed and order status updated successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"order_id": order.OrderID,
			"change":   0.0},
		Message:   paymentSuccessMessage,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})

}
