// Package handlers provides HTTP request handlers for Point of Sale (POS) operations.
// This file contains handlers for product scanning, payment processing (cash, mobile money, voucher),
// split payments, receipt generation, and transaction management for in-store purchases.
package handlers

import (
	"bytes"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// validateTotalPaymentAmount validates that the total payment amount is sufficient
func validateTotalPaymentAmount(methods []dtos.PaymentMethod, orderTotal float64) error {
	totalAmount := 0.0
	for _, method := range methods {
		totalAmount += method.Amount
	}
	if totalAmount < orderTotal {
		return fmt.Errorf("%s", insufficientPaymentError)
	}
	return nil
}

// processAllPaymentMethods processes each payment method in the split payment
func processAllPaymentMethods(db models.DBExecutor, methods []dtos.PaymentMethod, order *dtos.Order, c *gin.Context, requestSummary string, start time.Time) error {
	for _, paymentMethod := range methods {
		if err := processSinglePaymentMethod(db, paymentMethod, order, c, requestSummary, start); err != nil {
			return err
		}
	}
	return nil
}

// processSinglePaymentMethod processes a single payment method
func processSinglePaymentMethod(db models.DBExecutor, paymentMethod dtos.PaymentMethod, order *dtos.Order, c *gin.Context, requestSummary string, start time.Time) error {
	switch strings.ToLower(paymentMethod.Type) {
	case "cash":
		return handleCashPaymentMethod(db, order, c, requestSummary, start)
	case "mpesa":
		return handleMpesaPaymentMethod(db, paymentMethod, order, c, requestSummary, start)
	case "voucher":
		return handleVoucherPaymentMethod(db, paymentMethod, order, c, requestSummary, start)
	default:
		respondWithPaymentError(c, requestSummary, start, fmt.Sprintf("Unsupported payment method: %s", paymentMethod.Type), fmt.Sprintf("Unsupported payment method: %s", paymentMethod.Type), http.StatusBadRequest)
		return fmt.Errorf("unsupported payment method: %s", paymentMethod.Type)
	}
}

// handleCashPaymentMethod handles cash payment processing
func handleCashPaymentMethod(db models.DBExecutor, order *dtos.Order, c *gin.Context, requestSummary string, start time.Time) error {
	if err := processCashPayment(db, order); err != nil {
		respondWithPaymentError(c, requestSummary, start, fmt.Sprintf("Failed to process cash payment: %s", err.Error()), "Failed to process cash payment", http.StatusBadRequest)
		return err
	}
	return nil
}

// handleMpesaPaymentMethod handles M-PESA payment processing
func handleMpesaPaymentMethod(db models.DBExecutor, paymentMethod dtos.PaymentMethod, order *dtos.Order, c *gin.Context, requestSummary string, start time.Time) error {
	mpesaReq := &dtos.MpesaRequest{
		OrderID:     order.OrderID,
		Phone:       *paymentMethod.PhoneNumber,
		Amount:      int(paymentMethod.Amount),
		DeliveryID:  order.DeliveryID,
		Reference:   ekomasiMpesaPrefix + order.OrderID,
		Description: fmt.Sprintf("Payment for order %s", order.OrderID),
	}

	client, err := NewMpesaClient()
	if err != nil {
		respondWithPaymentError(c, requestSummary, start, "Failed to initialize MPESA client", fmt.Sprintf("Failed to initialize MPESA client: %s", err), http.StatusInternalServerError)
		return err
	}

	response, err := client.LipaNaMpesaOnline(*mpesaReq)
	if err != nil {
		respondWithPaymentError(c, requestSummary, start, "Failed to initiate MPESA payment", err.Error(), http.StatusBadRequest)
		return err
	}

	if err = models.StoreStkResponse(db, response, *mpesaReq); err != nil {
		respondWithPaymentError(c, requestSummary, start, "Failed to store MPESA payment request", err.Error(), http.StatusInternalServerError)
		return err
	}

	if err = storeTransactionLog(db, *mpesaReq); err != nil {
		log.Printf("Failed to store transaction's log: %v", err)
	}

	return nil
}

// handleVoucherPaymentMethod handles voucher payment processing
func handleVoucherPaymentMethod(db models.DBExecutor, paymentMethod dtos.PaymentMethod, order *dtos.Order, c *gin.Context, requestSummary string, start time.Time) error {
	if err := processVoucherPayment(db, order, *paymentMethod.VoucherCode); err != nil {
		respondWithPaymentError(c, requestSummary, start, fmt.Sprintf("Failed to process voucher payment: %s", err.Error()), err.Error(), http.StatusBadRequest)
		return err
	}
	return nil
}

// respondWithPaymentError is a helper function to reduce code duplication for error responses
func respondWithPaymentError(c *gin.Context, requestSummary string, start time.Time, description, message string, code int) {
	utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: description,
			Code:        code,
		},
		Message:   message,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
	})
}

// processCashPayment is a helper function that processes cash payments for orders.
// It updates the order status to mark payment as successful and logs the transaction.
// This function is used internally by split payment processing.
func processCashPayment(db models.DBExecutor, order *dtos.Order) error {
	// Prepare order status update with cash payment details
	method := "CASH"
	paymentStatus := "SUCCESS"
	status := "COMPLETED"
	orderStatusData := dtos.UpdateOrderStatusRequest{
		PaymentMethod: &method,
		Status:        &status,
		PaymentStatus: &paymentStatus,
	}

	// Update order status to mark as paid with cash
	if err := models.UpdateOrderStatus(db, order.OrderID, orderStatusData); err != nil {
		return err
	}

	// Create transaction log entry for audit purposes
	logEntry := &dtos.TransactionsList{
		OrderID:              &order.OrderID,
		TransactionReference: ekomasiMpesaPrefix + order.OrderID,
		Amount:               order.TotalAmount,
		Status:               "COMPLETED",
		PaymentMethod:        "CASH",
	}

	// Store transaction log in database (non-critical, log errors only)
	if err := models.InsertTransaction(db, logEntry); err != nil {
		log.Printf("Failed to store log for the transaction: %v", err)
	}

	return nil
}

// processVoucherPayment is a helper function that processes voucher payments for orders.
// It validates the voucher, checks sufficient balance, updates order status,
// deducts from voucher balance, and logs the transaction.
func processVoucherPayment(db models.DBExecutor, order *dtos.Order, voucherCode string) error {
	// Validate voucher and retrieve its current balance
	voucherBalance, err := models.ValidateVoucher(db, voucherCode)
	if err != nil {
		return fmt.Errorf("failed to validate voucher: %w", err)
	}

	// Check if voucher has sufficient balance for the order
	if voucherBalance < order.TotalAmount {
		return fmt.Errorf("insufficient voucher balance")
	}

	// Prepare order status update with voucher payment details
	method := "VOUCHER"
	paymentStatus := "SUCCESS"
	status := "COMPLETED"
	orderStatusData := dtos.UpdateOrderStatusRequest{
		PaymentMethod: &method,
		Status:        &status,
		PaymentStatus: &paymentStatus,
	}

	// Update order status to mark as paid with voucher
	if err := models.UpdateOrderStatus(db, order.OrderID, orderStatusData); err != nil {
		return fmt.Errorf("failed to delete POS session with ID %s: %w", "unknown", err)
	}

	// Deduct amount from voucher balance and record usage history
	if err := updateVoucherBalanceAndHistory(db, voucherCode, voucherBalance, order.TotalAmount, order); err != nil {
		return fmt.Errorf("failed to update voucher balance and history: %w", err)
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
	if err := models.InsertTransaction(db, logEntry); err != nil {
		log.Printf("Failed to store transaction log: %v", err)
	}

	return nil
}

// validateSplitPaymentMethods validates that each payment method in a split payment
// has all required fields. For mobile money payments, phone numbers are required.
// For voucher payments, voucher codes are required.
func validateSplitPaymentMethods(methods []dtos.PaymentMethod) error {
	// Iterate through each payment method to validate required fields
	for _, method := range methods {
		switch strings.ToLower(method.Type) {
		case "mpesa", "airtel":
			// Validate that phone number is provided for mobile money payments
			if method.PhoneNumber == nil || *method.PhoneNumber == "" {
				return fmt.Errorf("phone number is required for mobile money payments")
			}

		case "voucher":
			// Validate that voucher code is provided for voucher payments
			if method.VoucherCode == nil || *method.VoucherCode == "" {
				return fmt.Errorf("voucher code is required for voucher payments")
			}
		}
	}
	return nil
}

// PrintReceiptHandler handles receipt printing operations for POS transactions.
// This endpoint would typically trigger a print job to a connected receipt printer.
// Implementation pending based on printer hardware integration.
//
// @Summary Print receipt for an order
// @Description Triggers receipt printing for a completed order
// @Tags POS
// @Produce json
// @Param order_id path string true "Order ID"
// @Success 200 {object} map[string]interface{} "Receipt sent to printer"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "Printer error"
// @Router /api/pos/receipt/print/{order_id} [post]
// @Security BearerAuth
func PrintReceiptHandler(c *gin.Context) {
	// TODO1: Implement receipt printing logic based on printer hardware
}

// DownloadReceiptHandler generates and downloads a PDF receipt for a completed order.
// This endpoint retrieves order details, generates a formatted PDF receipt,
// and returns it as a downloadable file.
//
// @Summary Download receipt PDF for an order
// @Description Generates and downloads a PDF receipt for the specified order
// @Tags POS
// @Produce application/pdf
// @Param order_id path string true "Order ID"
// @Success 200 {file} application/pdf "Receipt PDF file"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "PDF generation failed"
// @Router /api/pos/receipt/download/{order_id} [get]
// @Security BearerAuth
func DownloadReceiptHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract order ID from URL path parameters
	orderID := c.Param("order_id")

	// Retrieve order details from database
	order, err := models.GetOrderByID(models.DB, orderID)
	if err != nil {
		// Return error if order is not found
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
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

	// Generate PDF receipt from order data
	receiptData, err := models.GenerateReceiptPDF(*order)
	if err != nil {
		// Return error if PDF generation fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Receipts",
				Description: fmt.Sprintf("Failed to generate receipt: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to generate receipt",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Set response headers for PDF download
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"receipt_%s.pdf\"", orderID))

	// Write PDF to buffer for transmission
	var buf bytes.Buffer
	err = receiptData.Output(&buf)
	if err != nil {
		// Return error if PDF output fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Receipts",
				Description: fmt.Sprintf("Failed to output PDF: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to output PDF",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Write PDF bytes to response
	c.Writer.Write(buf.Bytes())
}
