// Package handlers provides HTTP request handlers for the Ekomasi backend API.
// This file contains payment processing, refund management, voucher handling,
// and payment option configuration handlers.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
func DeletePaymentHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Payments", "payments.delete")
	if !ok {
		return
	}

	// Extract payment ID from URL path parameters
	paymentID := c.Param("payment_id")

	// Attempt to delete the payment from the database
	if err := models.DeletePayment(models.DB, paymentID); err != nil {
		// Return error response if deletion fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to delete payment with ID " + paymentID,
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

	// Return success response confirming payment deletion
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentWithID + paymentID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func RequestRefund(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Retrieve authenticated user from context
	user, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		// Return error if user is not authenticated
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "User not found in context",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.Refund](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Payments") {
		return
	}

	// Set initial status as requested for new refund requests
	req.Status = "requested"

	// Create refund request in database associated with user
	if err := models.AddRefundRequest(models.DB, *req, user.ID); err != nil {
		// Return error response if refund request creation fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to request refund from user ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached refund data to ensure data consistency
	utils.DeleteCacheByPrefix("refunds_")
	utils.DeleteCacheByPrefix("refunds_pagination_")

	// Return success response confirming refund request submission
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund requested successfully For user ID " + user.ID,
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Refund requested successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func ProcessRefund(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Payments", "payments.refund")
	if !ok {
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.RefundPayload](c, requestSummary, start)
	if !ok {
		return
	}

	// Extract refund ID from URL path parameters
	refundID := c.Param("refund_id")

	// Validate status field - only approved, rejected, or processed are allowed
	if req.Status != "approved" && req.Status != "rejected" && req.Status != "processed" {
		// Return error if status is not one of the allowed values
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Invalid status passed",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid status passed",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Update refund status in the database
	err := models.ProcessRefund(models.DB, *req, refundID)
	if err != nil {
		// Return error response if refund processing fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to process refund with ID " + refundID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response confirming refund status update
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund with ID " + refundID + " processed successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Refund status updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListRefundsHandler retrieves a paginated list of all refund requests.
// This endpoint implements caching for performance and returns refunds with pagination metadata.
