// Package handlers provides HTTP request handlers for the Ekomasi backend API.
// This file contains payment processing, refund management, voucher handling,
// and payment option configuration handlers.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
func CreatePaymentHandler(c *gin.Context) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Payments", "payments.create")
	if !ok {
		return
	}

	// Decode and validate the request body into Payment DTO
	req, ok := DecodeRequestBody[dtos.Payment](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Payments") {
		return
	}

	// Attempt to create the payment record in the database
	if err := models.CreatePayment(models.DB, *req); err != nil {
		// Return error response if payment creation fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to create payment",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached payment data to ensure data consistency
	utils.DeleteCacheByPrefix("payments_")
	utils.DeleteCacheByPrefix("payments_pagination_")

	// Return success response confirming payment creation
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Payment created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func GetPaymentByIDHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract payment ID from URL path parameters
	id := c.Param("payment_id")

	// Retrieve payment record from database by ID
	payment, err := models.GetPaymentByID(models.DB, id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get payment by ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentWithID + id + " has been fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   payment,
		Message:   "Payment fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func ListPaymentsHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse pagination parameters from query string
	page, limit := parsePagination(c.Query("page"), c.Query("size"))

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
		payments, pagination, err = models.ListPayments(models.DB, page, limit)
		if err != nil {
			// Return error response if database query fails
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: "Failed to list payments",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payments fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Payments fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func UpdatePaymentHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Payments", "payments.update")
	if !ok {
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.PaymentUpdate](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Payments") {
		return
	}

	// Extract payment ID from URL path parameters
	paymentID := c.Param("payment_id")

	// Attempt to update the payment in the database
	if err := models.UpdatePayment(models.DB, *req, paymentID); err != nil {
		// Return error response if update fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to update payment with ID " + paymentID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached payment data to ensure data consistency
	utils.DeleteCacheByPrefix("payments_")
	utils.DeleteCacheByPrefix("payments_pagination_")

	// Return success response confirming payment update
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentWithID + paymentID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
