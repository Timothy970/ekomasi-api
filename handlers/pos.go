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
// @Success 200 {object} map[string]interface{} "Payment processed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid voucher or insufficient balance"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
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
// @Success 200 {object} map[string]interface{} "Payment processed successfully with change details"
// @Failure 400 {object} map[string]interface{} "Invalid request or insufficient payment"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/pos/credit-payment [post]
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
