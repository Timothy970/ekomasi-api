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

var (
	orderError               = "Failed to retrieve order"
	insufficientPaymentError = "Insufficient payment amount"
	paymentSuccessMessage    = "Payment processed successfully"
	ekomasiMpesaPrefix       = "EKOMASI - "
)

// ScanProductsHandler scans and retrieves product information using a barcode.
// This endpoint is used in POS systems to quickly look up product details by scanning
// barcodes during checkout or inventory management.
//
// @Summary Scan product by barcode
// @Description Retrieves product details by scanning a barcode for POS operations
// @Tags POS
// @Produce json
// @Param barcode query string true "Product barcode to scan"
// @Success 200 {object} map[string]interface{} "Product scanned successfully"
// @Failure 400 {object} map[string]interface{} "Barcode missing or invalid"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Router /api/pos/scan [get]
// @Security BearerAuth
func ScanProductsHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract barcode from query parameters
	barcode := c.Query("barcode")

	// Validate that barcode is provided
	if barcode == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to continue sacanning as barcode is empty",
				Code:        http.StatusBadRequest,
			},
			Message:   "Barcode is required and cannot be empty",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Query the database to retrieve product by barcode
	product, err := models.GetProductThroughScanning(models.DB, barcode)
	if err != nil {
		// Return error response if product not found or scan fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: fmt.Sprintf("Failed to scan product: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to scan product",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Return success response with product details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product scanned successfully",
			Code:        http.StatusOK,
		},
		Payload:   product,
		Message:   "Product scanned successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// ProcessCashPaymentHandler processes cash payments for POS transactions.
// This endpoint validates the payment amount, updates the order status, calculates change,
// and logs the transaction for auditing purposes.
//
// @Summary Process cash payment for an order
// @Description Processes a cash payment, validates sufficient amount, updates order status, and returns change
// @Tags POS
// @Accept json
// @Produce json
// @Param request body dtos.CashPayment true "Cash payment details"
// @Success 200 {object} map[string]interface{} "Payment processed successfully with change details"
// @Failure 400 {object} map[string]interface{} "Invalid request or insufficient payment"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/pos/cash-payment [post]
// @Security BearerAuth
func ProcessCashPaymentHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.CashPayment](c, requestSummary, start)
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

	// Verify that the payment amount is sufficient
	if req.Amount < order.TotalAmount {
		// Return error if payment is insufficient
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

	// Prepare order status update data with payment method and status
	paymentMethod := "CASH"
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

	// Calculate change to return to customer
	change := req.Amount - order.TotalAmount
	if change < 0 {
		change = 0 // Ensure change is never negative
	}

	// Create transaction log entry for audit trail
	logEntry := &dtos.TransactionsList{
		OrderID:              &order.OrderID,
		TransactionReference: ekomasiMpesaPrefix + order.OrderID,
		Amount:               order.TotalAmount,
		Status:               "COMPLETED",
		PaymentMethod:        "CASH",
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
			"change":   change,
		},
		Message:   paymentSuccessMessage,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})

}

// ProcessSplitPaymentHandler processes payments split across multiple payment methods.
// This endpoint allows customers to pay using a combination of payment methods such as
// cash, mobile money (M-PESA), and vouchers in a single transaction.
//
// @Summary Process split payment using multiple payment methods
// @Description Processes a payment split across cash, M-PESA, vouchers, or combinations thereof
// @Tags POS
// @Accept json
// @Produce json
// @Param request body dtos.SplitPaymentRequest true "Split payment details with multiple payment methods"
// @Success 200 {object} map[string]interface{} "Payment processed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request, insufficient payment, or unsupported method"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/pos/split-payment [post]
// @Security BearerAuth
func ProcessSplitPaymentHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	req, ok := DecodeRequestBody[dtos.SplitPaymentRequest](c, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Payments") {
		return
	}

	if err := validateSplitPaymentMethods(req.PaymentMethods); err != nil {
		respondWithPaymentError(c, requestSummary, start, "Invalid payment methods", err.Error(), http.StatusBadRequest)
		return
	}

	order, err := models.GetOrderByID(models.DB, req.OrderID)
	if err != nil {
		respondWithPaymentError(c, requestSummary, start, fmt.Sprintf("Failed to retrieve order with ID %s", req.OrderID), orderError, http.StatusBadRequest)
		return
	}

	if err := validateTotalPaymentAmount(req.PaymentMethods, order.TotalAmount); err != nil {
		respondWithPaymentError(c, requestSummary, start, insufficientPaymentError, insufficientPaymentError, http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := models.DB.Begin()
	if err != nil {
		respondWithPaymentError(c, requestSummary, start, "Failed to start transaction", err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Process all valid payment methods
	if err := processAllPaymentMethods(tx, req.PaymentMethods, order, c, requestSummary, start); err != nil {
		// Error response already sent by helper function
		return
	}

	if err := tx.Commit(); err != nil {
		respondWithPaymentError(c, requestSummary, start, "Failed to commit transaction", err.Error(), http.StatusInternalServerError)
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment processed and order status updated successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"order_id": order.OrderID,
			"change":   0.0,
		},
		Message:   paymentSuccessMessage,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
