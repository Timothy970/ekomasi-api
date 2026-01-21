// Package handlers provides HTTP request handlers for the Adenzo backend API.
// This file contains payment processing, refund management, voucher handling,
// and payment option configuration handlers.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Global message template variable for consistent response formatting
var paymentWithID = "Payment with ID "

// CreatePaymentHandler creates a new payment record in the system.
// This endpoint is restricted to admin users and records payment transactions
// associated with orders or other financial operations.
//
// @Summary Create a new payment record
// @Description Creates a new payment entry in the database for tracking financial transactions
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dtos.Payment true "Payment creation request"
// @Success 201 {object} map[string]interface{} "Payment created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/payments [post]
// @Security BearerAuth
func CreatePaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Payments", "payments.create")
	if !ok {
		return
	}

	// Decode and validate the request body into Payment DTO
	req, ok := DecodeRequestBody[dtos.Payment](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}

	// Attempt to create the payment record in the database
	if err := models.CreatePayment(*req); err != nil {
		// Return error response if payment creation fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to create payment",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached payment data to ensure data consistency
	utils.DeleteCacheByPrefix("payments_")
	utils.DeleteCacheByPrefix("payments_pagination_")

	// Return success response confirming payment creation
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Payment created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetPaymentByIDHandler retrieves a single payment record by its unique identifier.
// This endpoint returns detailed information about a specific payment transaction.
//
// @Summary Get payment by ID
// @Description Retrieves detailed information about a specific payment by its unique identifier
// @Tags Payments
// @Produce json
// @Param payment_id path string true "Payment ID"
// @Success 200 {object} map[string]interface{} "Payment details retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Payment not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/payments/{payment_id} [get]
// @Security BearerAuth
func GetPaymentByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract payment ID from URL path parameters
	id := mux.Vars(r)["payment_id"]

	// Retrieve payment record from database by ID
	payment, err := models.GetPaymentByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get payment by ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentWithID + id + " has been fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   payment,
		Message:   "Payment fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ListPaymentsHandler retrieves a paginated list of all payment records.
// This endpoint is restricted to admin users and implements caching for performance.
//
// @Summary List all payments with pagination
// @Description Retrieves a paginated list of all payment records in the system
// @Tags Admin
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param size query int false "Page size (default: 10)"
// @Success 200 {object} map[string]interface{} "Payments retrieved successfully with pagination"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/payments [get]
// @Security BearerAuth
func ListPaymentsHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Parse pagination parameters from query string
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	// Generate cache keys for payments data and pagination metadata
	cacheKeyPayments := fmt.Sprintf("payments_%d_size_%d", page, limit)
	cacheKeyPagination := fmt.Sprintf("payments_pagination_%d_size_%d", page, limit)

	// Initialize variables for payments data and cached versions
	var payments []dtos.Payment
	var cachedPayemnts []dtos.Payment
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta

	// Attempt to retrieve cached data
	_ = utils.GetCache(cacheKeyPayments, &cachedPayemnts)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)

	// If cache miss, fetch from database
	if len(cachedPayemnts) == 0 {
		var err error
		// Fetch payments from database with pagination
		payments, pagination, err = models.ListPayments(page, limit)
		if err != nil {
			// Return error response if database query fails
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: "Failed to list payments",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Cache the fetched data for future requests
		_ = utils.SetCache(cacheKeyPayments, payments)
		_ = utils.SetCache(cacheKeyPagination, pagination)
	} else {
		// Use cached data if available
		payments = cachedPayemnts
		pagination = cachedPagination
	}

	// Build response with payments and pagination metadata
	resp := dtos.PaymentListResponse{
		Payments: payments,
		Meta:     *pagination,
	}

	// Return success response with payment data
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payments fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Payments fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdatePaymentHandler modifies an existing payment record.
// This endpoint is restricted to admin users and allows updating payment details
// such as status, amount, or other payment attributes.
//
// @Summary Update a payment by ID
// @Description Updates an existing payment record's information
// @Tags Admin
// @Accept json
// @Produce json
// @Param payment_id path string true "Payment ID"
// @Param request body dtos.PaymentUpdate true "Payment update request"
// @Success 200 {object} map[string]interface{} "Payment updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Payment not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/payments/{payment_id} [patch]
// @Security BearerAuth
func UpdatePaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Payments", "payments.update")
	if !ok {
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.PaymentUpdate](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}

	// Extract payment ID from URL path parameters
	paymentID := mux.Vars(r)["payment_id"]

	// Attempt to update the payment in the database
	if err := models.UpdatePayment(*req, paymentID); err != nil {
		// Return error response if update fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to update payment with ID " + paymentID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached payment data to ensure data consistency
	utils.DeleteCacheByPrefix("payments_")
	utils.DeleteCacheByPrefix("payments_pagination_")

	// Return success response confirming payment update
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentWithID + paymentID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeletePaymentHandler removes a payment record from the system.
// This endpoint is restricted to admin users and permanently deletes the specified payment.
//
// @Summary Delete a payment by ID
// @Description Permanently deletes a payment record from the system
// @Tags Admin
// @Produce json
// @Param payment_id path string true "Payment ID"
// @Success 200 {object} map[string]interface{} "Payment deleted successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Payment not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/payments/{payment_id} [delete]
// @Security BearerAuth
func DeletePaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Payments", "payments.delete")
	if !ok {
		return
	}

	// Extract payment ID from URL path parameters
	paymentID := mux.Vars(r)["payment_id"]

	// Attempt to delete the payment from the database
	if err := models.DeletePayment(paymentID); err != nil {
		// Return error response if deletion fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to delete payment with ID " + paymentID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached payment data to ensure data consistency
	utils.DeleteCacheByPrefix("payments_")
	utils.DeleteCacheByPrefix("payments_pagination_")

	// Return success response confirming payment deletion
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentWithID + paymentID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// RequestRefund allows authenticated users to submit a refund request.
// This endpoint creates a refund request record associated with the user's account,
// which can then be reviewed and processed by admin users.
//
// @Summary Request a refund for a payment
// @Description Submits a refund request for a payment transaction. The request will be reviewed by admin staff.
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body dtos.Refund true "Refund request details"
// @Success 201 {object} map[string]interface{} "Refund requested successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - user authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/payments/refund [post]
// @Security BearerAuth
func RequestRefund(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Retrieve authenticated user from context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// Return error if user is not authenticated
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "User not found in context",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.Refund](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}

	// Set initial status as requested for new refund requests
	req.Status = "requested"

	// Create refund request in database associated with user
	err := models.AddRefundRequest(*req, user.ID)
	if err != nil {
		// Return error response if refund request creation fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to request refund from user ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached refund data to ensure data consistency
	utils.DeleteCacheByPrefix("refunds_")
	utils.DeleteCacheByPrefix("refunds_pagination_")

	// Return success response confirming refund request submission
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund requested successfully For user ID " + user.ID,
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Refund requested successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ProcessRefund allows admin users to review and update refund request status.
// This endpoint processes refund requests by approving, rejecting, or marking them as processed.
// Admin access is required.
//
// @Summary Process a refund request
// @Description Updates the status of a refund request. Valid statuses: approved, rejected, processed. Admin access required.
// @Tags Admin
// @Accept json
// @Produce json
// @Param refund_id path string true "Refund ID"
// @Param request body dtos.RefundPayload true "Refund status update"
// @Success 200 {object} map[string]interface{} "Refund status updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid status or request body"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Refund not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/payments/refund/{refund_id} [patch]
// @Security BearerAuth
func ProcessRefund(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Payments", "payments.refund")
	if !ok {
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.RefundPayload](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Extract refund ID from URL path parameters
	refundID := mux.Vars(r)["refund_id"]

	// Validate status field - only approved, rejected, or processed are allowed
	if req.Status != "approved" && req.Status != "rejected" && req.Status != "processed" {
		// Return error if status is not one of the allowed values
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Invalid status passed",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid status passed",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Update refund status in the database
	err := models.ProcessRefund(*req, refundID)
	if err != nil {
		// Return error response if refund processing fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to process refund with ID " + refundID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return success response confirming refund status update
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund with ID " + refundID + " processed successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Refund status updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ListRefundsHandler retrieves a paginated list of all refund requests.
// This endpoint implements caching for performance and returns refunds with pagination metadata.
//
// @Summary List all refund requests with pagination
// @Description Retrieves a paginated list of all refund requests in the system
// @Tags Payments
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param size query int false "Page size (default: 10)"
// @Success 200 {object} map[string]interface{} "Refunds retrieved successfully with pagination"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/refunds [get]
// @Security BearerAuth
func ListRefundsHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Parse pagination parameters from query string
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	// Generate cache keys for refunds data and pagination metadata
	cacheKeyRefunds := fmt.Sprintf("refunds_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("refunds_pagination_%d_size_%d", page, size)

	// Initialize variables for refunds data and cached versions
	var refunds []dtos.Refund
	var cachedRefunds []dtos.Refund
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta

	// Attempt to retrieve cached data
	_ = utils.GetCache(cacheKeyRefunds, &cachedRefunds)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)

	// If cache miss, fetch from database
	if cachedRefunds == nil {
		var err error
		// Fetch refunds from database with pagination
		refunds, pagination, err = models.ListRefunds(page, size)
		if err != nil {
			// Return error response if database query fails
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: "Failed to list refunds",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Cache the fetched data for future requests
		_ = utils.SetCache(cacheKeyRefunds, cachedRefunds)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		// Use cached data if available
		refunds = cachedRefunds
		pagination = cachedPagination
	}

	// Build response with refunds and pagination metadata
	response := dtos.PaginatedRefundsResponse{
		Refunds: refunds,
		Meta:    *pagination,
	}

	// Return success response with refund data
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refunds fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Refunds fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetRefundByIDHandler retrieves a single refund request by its unique identifier.
// This endpoint returns detailed information about a specific refund request.
//
// @Summary Get refund by ID
// @Description Retrieves detailed information about a specific refund request by its ID
// @Tags Payments
// @Produce json
// @Param refund_id path string true "Refund ID"
// @Success 200 {object} map[string]interface{} "Refund details retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Refund not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/refunds/{refund_id} [get]
// @Security BearerAuth
func GetRefundByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract refund ID from URL path parameters and convert to integer
	idStr := mux.Vars(r)["refund_id"]
	id, _ := strconv.Atoi(idStr)

	// Retrieve refund record from database by ID
	refund, err := models.GetRefundByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get refund by ID " + idStr,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund with ID " + idStr + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   refund,
		Message:   "Refund fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetRefundByUserIDHandler retrieves refund requests associated with a specific user.
// This endpoint returns refund information filtered by user ID.
//
// @Summary Get refunds by user ID
// @Description Retrieves refund requests for a specific user identified by user ID
// @Tags Payments
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]interface{} "Refund details retrieved successfully"
// @Failure 404 {object} map[string]interface{} "User or refund not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/refunds/user/{user_id} [get]
// @Security BearerAuth
func GetRefundByUserIDHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract user ID from URL path parameters and convert to integer
	idStr := mux.Vars(r)["user_id"]
	id, _ := strconv.Atoi(idStr)

	// Retrieve refund records from database by user ID
	refund, err := models.GetRefundByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get refund by User ID " + idStr,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund with User ID " + idStr + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   refund,
		Message:   "Refund fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CreateVoucherHandler creates a new voucher for promotional or discount purposes.
// This endpoint is restricted to admin users and allows creation of vouchers that can be
// applied during checkout to provide discounts or special offers.
//
// @Summary Create a new voucher
// @Description Creates a new promotional voucher with discount details and usage rules
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dtos.Voucher true "Voucher creation request"
// @Success 201 {object} map[string]interface{} "Voucher created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/vouchers [post]
// @Security BearerAuth
func CreateVoucherHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Vouchers", "promotions.create")
	if !ok {
		return
	}

	// Retrieve authenticated admin user from context
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// Return error if user is not found in context
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "User not found in context while creating voucher",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.Voucher](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}

	// Create voucher in database associated with admin user
	_, err := models.AddNewVoucher(*req, authuser.ID)
	if err != nil {
		// Return error response if voucher creation fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached voucher data to ensure data consistency
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")

	// Return success response confirming voucher creation
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Voucher created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CreatePaymentOptionHandler creates a new payment option/method configuration.
// This endpoint is restricted to admin users and allows adding payment methods
// such as credit cards, mobile money, bank transfers, etc.
//
// @Summary Create a new payment option
// @Description Creates a new payment method configuration for checkout
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dtos.PaymentOption true "Payment option creation request"
// @Success 201 {object} map[string]interface{} "Payment option created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/payment-options [post]
// @Security BearerAuth
func CreatePaymentOptionHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Payments", "payments.create")
	if !ok {
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.PaymentOption](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}

	// Create payment option in database
	err := models.CreatePaymentOption(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to create payment option",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment option created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Payment option created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ListPaymentOptionsHandler retrieves a paginated and filtered list of payment options.
// This endpoint supports searching and filtering by status to find specific payment methods.
//
// @Summary List all payment options with filtering
// @Description Retrieves a paginated list of payment options with optional search and status filtering
// @Tags Payments
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param size query int false "Page size (default: 10)"
// @Param q query string false "Search query for payment option name"
// @Param status query string false "Filter by payment option status (e.g., active, inactive)"
// @Success 200 {object} map[string]interface{} "Payment options retrieved successfully with pagination"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/payment-options [get]
// @Security BearerAuth
func ListPaymentOptionsHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Parse pagination and filter parameters from query string
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	q := r.URL.Query().Get("q")           // Search query for payment option name
	status := r.URL.Query().Get("status") // Status filter

	// Retrieve filtered payment options from database
	paymentOptions, pagination, err := models.ListPaymentOptions(q, status, page, size)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to list pay bills",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Pay bills fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"payment_options": paymentOptions, "meta": pagination},
		Message:   "Payment options fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetPaymentOptionByIDHandler retrieves a single payment option by its unique identifier.
// This endpoint returns detailed information about a specific payment method configuration.
//
// @Summary Get payment option by ID
// @Description Retrieves detailed information about a specific payment option by its ID
// @Tags Payments
// @Produce json
// @Param payment_option_id path string true "Payment Option ID"
// @Success 200 {object} map[string]interface{} "Payment option details retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Payment option not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/payment-options/{payment_option_id} [get]
// @Security BearerAuth
func GetPaymentOptionByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract payment option ID from URL path parameters
	paymentOptionID := mux.Vars(r)["payment_option_id"]

	// Retrieve payment option from database by ID
	paymentOption, err := models.GetPaymentOptionByID(paymentOptionID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get payment option by ID " + paymentOptionID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment option with ID " + paymentOptionID + " got successfully",
			Code:        http.StatusOK,
		},
		Payload:   paymentOption,
		Message:   "Payment option fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdatePaymentOptionHandler modifies an existing payment option configuration.
// This endpoint is restricted to admin users and allows updating payment method details
// such as name, status, fees, or other configuration parameters.
//
// @Summary Update a payment option by ID
// @Description Updates an existing payment option's configuration
// @Tags Admin
// @Accept json
// @Produce json
// @Param payment_option_id path string true "Payment Option ID"
// @Param request body dtos.PaymentOptionUpdate true "Payment option update request"
// @Success 200 {object} map[string]interface{} "Payment option updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Payment option not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/payment-options/{payment_option_id} [patch]
// @Security BearerAuth
func UpdatePaymentOptionHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Payments", "payments.update")
	if !ok {
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.PaymentOptionUpdate](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}

	// Extract payment option ID from URL path parameters
	paymentOptionID := mux.Vars(r)["payment_option_id"]

	// Attempt to update the payment option in the database
	if err := models.UpdatePaymentOption(paymentOptionID, *req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to update payment option with ID " + paymentOptionID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "payment option With ID" + paymentOptionID + " was updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment option updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeletePaymentOptionHandler removes a payment option from the system.
// This endpoint is restricted to admin users and permanently deletes the specified
// payment method configuration.
//
// @Summary Delete a payment option by ID
// @Description Permanently deletes a payment option configuration from the system
// @Tags Admin
// @Produce json
// @Param payment_option_id path string true "Payment Option ID"
// @Success 200 {object} map[string]interface{} "Payment option deleted successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Payment option not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/payment-options/{payment_option_id} [delete]
// @Security BearerAuth
func DeletePaymentOptionHandler(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Payments", "payments.delete")
	if !ok {
		return
	}

	// Extract payment option ID from URL path parameters
	paymentOptionID := mux.Vars(r)["payment_option_id"]

	// Attempt to delete the payment option from the database
	if err := models.DeletePaymentOption(paymentOptionID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to delete payment option with ID " + paymentOptionID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "payment option With ID" + paymentOptionID + " was deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment option deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
