// Package handlers provides HTTP request handlers for transaction management.
// This file contains handlers for managing payment transactions in the e-commerce platform,
// including viewing transaction history, retrieving transaction details, and updating payment statuses.
// Transactions track all payment activities related to customer orders.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// GetAllTransactionHandler retrieves a paginated list of all payment transactions.
// Admin-only operation with filtering by status and search query support.
// Essential for financial tracking, reconciliation, and payment auditing.
//
// @Summary      List all transactions
// @Description  Retrieve paginated list of payment transactions with status filtering and search (admin only)
// @Tags         Transactions
// @Produce      json
// @Param        page    query     int                       false  "Page number (default: 1)"
// @Param        size    query     int                       false  "Page size (default: 10)"
// @Param        status  query     string                    false  "Filter by transaction status (e.g., pending, completed, failed)"
// @Param        q       query     string                    false  "Search query"
// @Success      200     {object}  map[string]interface{}    "Transactions with pagination metadata"
// @Failure      400     {object}  dtos.ErrorResponse        "Failed to fetch transactions"
// @Failure      401     {object}  dtos.ErrorResponse        "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/transactions [get]
func GetAllTransactionHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can view all transactions)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Transactions"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Extract optional status filter (e.g., "pending", "completed", "failed")
	status := r.URL.Query().Get("status")
	// Extract optional search query parameter
	q := r.URL.Query().Get("q")
	// Fetch paginated transactions from database with filters
	transactions, pagination, err := models.GetAllTransactions(page, limit, status, q)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch transactions " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return transactions list with pagination metadata for financial tracking
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "All transactions fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"transactions": transactions,
			"pagination":   pagination,
		},
		Message:   "Transactions fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetTransactionByIDHandler retrieves detailed information for a specific payment transaction.
// Returns complete transaction details including amount, status, payment method, and timestamps.
// Admin-only operation for transaction verification and payment reconciliation.
//
// @Summary      Get transaction by ID
// @Description  Retrieve detailed information for a specific payment transaction (admin only)
// @Tags         Transactions
// @Produce      json
// @Param        transaction_id  path      string                 true  "Transaction ID"
// @Success      200             {object}  dtos.Transaction       "Transaction details"
// @Failure      400             {object}  dtos.ErrorResponse     "Failed to fetch transaction"
// @Failure      401             {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/transactions/{transaction_id} [get]
func GetTransactionByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can view transaction details)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Transaction"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract transaction ID from URL path parameters
	transactionID := mux.Vars(r)["transaction_id"]
	// Fetch specific transaction details from database
	transaction, err := models.GetTransactionByID(transactionID)
	if err != nil {
		// Transaction not found or database error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Transaction with id " + transactionID + " failed to fetch: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Return transaction details for verification and reconciliation

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Transaction with id" + transactionID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   transaction,
		Message:   "Transaction fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateTransactionStatusHandler updates the status of a payment transaction.
// Admin-only operation that also synchronizes the payment status with the associated order.
// Critical for payment processing workflows and order fulfillment management.
//
// @Summary      Update transaction status
// @Description  Update payment transaction status and sync with order payment status (admin only)
// @Tags         Transactions
// @Accept       json
// @Produce      json
// @Param        transaction_id  path      string                         true  "Transaction ID"
// @Param        status          body      dtos.UpdateTransactionStatus   true  "New transaction status"
// @Success      200             {object}  dtos.SuccessResponse           "Transaction status updated successfully"
// @Failure      400             {object}  dtos.ErrorResponse             "Invalid request or update failed"
// @Failure      401             {object}  dtos.ErrorResponse             "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/transactions/{transaction_id}/status [patch]
func UpdateTransactionStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can update transaction status)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Transaction"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract transaction ID from URL path parameters
	transactionID := mux.Vars(r)["transaction_id"]
	// Decode and parse JSON request body with new status
	req, ok := DecodeRequestBody[dtos.UpdateTransactionStatus](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate status field in request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Update transaction status in database and retrieve associated order ID
	orderID, err := models.UpdateTransactionStatusByID(transactionID, req.Status)
	if err != nil {
		// Transaction status update failed (invalid transaction, status, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update transaction status " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Synchronize payment status with the associated order for consistency
	err = models.UpdateOrderPaymentStatus(orderID, req.Status)
	if err != nil {
		// Order payment status synchronization failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update order payment status " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Transaction status updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Transaction status updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
