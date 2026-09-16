// Package handlers provides HTTP request handlers for the Ekomasi backend API.
// This file contains payment processing, refund management, voucher handling,
// and payment option configuration handlers.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
func CreatePaymentOptionHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Payments", "payments.create")
	if !ok {
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.PaymentOption](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Payments") {
		return
	}

	// Create payment option in database
	err := models.CreatePaymentOption(models.DB, *req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to create payment option",
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
			Description: "Payment option created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Payment option created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func ListPaymentOptionsHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse pagination and filter parameters from query string
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	q := c.Query("q")           // Search query for payment option name
	status := c.Query("status") // Status filter

	// Retrieve filtered payment options from database
	paymentOptions, pagination, err := models.ListPaymentOptions(models.DB, q, status, page, size)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to list pay bills",
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
			Description: "Pay bills fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"payment_options": paymentOptions, "meta": pagination},
		Message:   "Payment options fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func GetPaymentOptionByIDHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract payment option ID from URL path parameters
	paymentOptionID := c.Param("payment_option_id")

	// Retrieve payment option from database by ID
	paymentOption, err := models.GetPaymentOptionByID(models.DB, paymentOptionID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get payment option by ID " + paymentOptionID,
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
			Description: "Payment option with ID " + paymentOptionID + " got successfully",
			Code:        http.StatusOK,
		},
		Payload:   paymentOption,
		Message:   "Payment option fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func UpdatePaymentOptionHandler(c *gin.Context) {
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
	req, ok := DecodeRequestBody[dtos.PaymentOptionUpdate](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Payments") {
		return
	}

	// Extract payment option ID from URL path parameters
	paymentOptionID := c.Param("payment_option_id")

	// Attempt to update the payment option in the database
	if err := models.UpdatePaymentOption(models.DB, paymentOptionID, *req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to update payment option with ID " + paymentOptionID,
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
			Description: "payment option With ID" + paymentOptionID + " was updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment option updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
