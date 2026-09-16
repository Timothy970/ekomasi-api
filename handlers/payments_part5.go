// Package handlers provides HTTP request handlers for the Ekomasi backend API.
// This file contains payment processing, refund management, voucher handling,
// and payment option configuration handlers.
package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DeletePaymentOptionHandler removes a payment option from the system.
// This endpoint is restricted to admin users and permanently deletes the specified
// payment method configuration.
//
// @Summary Delete a payment option by ID
// @Description Permanently deletes a payment option configuration from the system
// @Tags Admin
// @Produce json
// @Param payment_option_id path string true "Payment Option ID"
// @Success 200 {object} map[string]any "Payment option deleted successfully"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 404 {object} map[string]any "Payment option not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/payment-options/{payment_option_id} [delete]
// @Security BearerAuth
func DeletePaymentOptionHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Payments", "payments.delete")
	if !ok {
		return
	}

	// Extract payment option ID from URL path parameters
	paymentOptionID := c.Param("payment_option_id")

	// Attempt to delete the payment option from the database
	if err := models.DeletePaymentOption(models.DB, paymentOptionID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to delete payment option with ID " + paymentOptionID,
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
			Description: "payment option With ID" + paymentOptionID + " was deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment option deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
