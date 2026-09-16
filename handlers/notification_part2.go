// Package handlers provides HTTP request handlers for notification management.
// This file contains handlers for creating, listing, updating, and deleting notifications,
// as well as background schedulers for automated notifications (order confirmations, low stock alerts).
// Supports multi-channel delivery: email, SMS, WhatsApp, and push notifications via WebSocket.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// UpdateNotificationHandler updates notification status (read/unread).
// Admin-only operation for managing notification states.
//
// @Summary      Update notification
// @Description  Update notification status (admin only)
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Param        Authorization      header    string                      true   "Bearer token"
// @Param        notification_id    path      string                      true   "Notification ID"
// @Param        notification       body      dtos.UpdateNotification     true   "Updated status"
// @Success      200                {object}  map[string]interface{}        "Notification updated"
// @Failure      400                {object}  dtos.ErrorResponse          "Invalid request"
// @Failure      401                {object}  dtos.ErrorResponse          "Admin authorization required"
// @Failure      404                {object}  dtos.ErrorResponse          "Notification not found"
// @Security     BearerAuth
// @Router       /api/admin/notifications/{notification_id} [patch]
func UpdateNotificationHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can update notifications)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Notifications", "notifications.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract notification ID from URL path parameters
	id := c.Param("notification_id")
	// Decode and parse JSON request body with updated status
	req, ok := DecodeRequestBody[dtos.UpdateNotification](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}

	// Validate all required fields (status)
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Notifications") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Update notification status in database
	if err := models.UpdateNotification(req.Status, id); err != nil {
		// Update failed (notification not found or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to update notification with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate notification caches to ensure fresh data
	utils.DeleteCacheByPrefix("notifications_")
	utils.DeleteCacheByPrefix("notifications_pagination_")
	// Return success response
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notification with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Notification updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteNotificationHandler removes a notification from the system.
// Admin-only operation for cleaning up old or unwanted notifications.
//
// @Summary      Delete notification
// @Description  Delete notification by ID (admin only)
// @Tags         Notifications
// @Produce      json
// @Param        Authorization      header    string                 true  "Bearer token"
// @Param        notification_id    path      string                 true  "Notification ID"
// @Success      200                {object}  map[string]interface{}   "Notification deleted"
// @Failure      401                {object}  dtos.ErrorResponse     "Admin authorization required"
// @Failure      404                {object}  dtos.ErrorResponse     "Notification not found"
// @Security     BearerAuth
// @Router       /api/admin/notifications/{notification_id} [delete]
func DeleteNotificationHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can delete notifications)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Notifications", "notifications.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract notification ID from URL path parameters
	id := c.Param("notification_id")
	// Delete notification from database
	if err := models.DeleteNotification(id); err != nil {
		// Deletion failed (notification not found or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to delete notification with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate notification caches to ensure fresh data
	utils.DeleteCacheByPrefix("notifications_")
	utils.DeleteCacheByPrefix("notifications_pagination_")
	// Return success response
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notification with ID " + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Notification deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListLogsHandler retrieves system logs with pagination.
// Admin-only operation for monitoring system activity and debugging.
// Returns logs with timestamps, modules, and error details.
//
// @Summary      List system logs
// @Description  Retrieve paginated system logs for monitoring and debugging (admin only)
// @Tags         Logs
// @Produce      json
// @Param        Authorization  header    string                 true   "Bearer token"
// @Param        page           query     int                    false  "Page number"
// @Param        limit          query     int                    false  "Page size"
// @Success      200            {object}  map[string]interface{}   "Logs with pagination"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Failure      500            {object}  dtos.ErrorResponse     "Failed to retrieve logs"
// @Security     BearerAuth
// @Router       /api/admin/logs [get]
func ListLogsHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can view system logs)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	// Fetch logs from database with pagination
	logs, meta, err := models.ListLogs(page, limit)
	if err != nil {
		// Database query failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Logs",
				Description: "Failed to list logs",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Construct response with logs and pagination metadata
	resp := dtos.LogListResponse{
		Logs: logs,
		Meta: *meta,
	}

	// Return success response with logs list
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Logs",
			Description: "Logs retrieved successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Logs",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// StartOrderNotificationScheduler initializes background scheduler for order confirmation emails.
// Runs at specified interval to process pending order notifications.
// Essential for reliable order confirmation delivery.
func StartOrderNotificationScheduler(interval time.Duration) {
	// Create ticker that fires at specified interval (e.g., every 5 minutes)
	ticker := time.NewTicker(interval)
	// Run scheduler in background goroutine
	go func() {
		for range ticker.C {
			// Process all pending order notifications on each tick
			processPendingOrderNotifications()
		}
	}()
}

// processPendingOrderNotifications is the main entry point to process pending orders.
// Fetches all orders awaiting notification and processes them individually.
// Handles errors gracefully to avoid blocking entire batch.
func processPendingOrderNotifications() {
	// Fetch all orders that need confirmation emails sent
	pendingOrders, err := models.GetPendingOrderNotifications()
	if err != nil {
		// Critical error: can't get pending orders list
		log.Printf("CRITICAL: Failed to fetch pending order notifications: %v", err)
		return
	}

	log.Printf("Processing %d pending order notifications...", len(pendingOrders))

	// Process each order individually to avoid blocking entire batch on single failure
	for _, orderID := range pendingOrders {
		// Process single order with all notification logic encapsulated
		if err := processSingleOrder(orderID, "order_confirmation"); err != nil {
			// Retryable failure (e.g., email service down)
			// Log error and continue to next order, leaving this one pending for retry
			log.Printf("ERROR: Failed to process notification for order %s (will retry): %v", orderID, err)
		}
	}
}

// processSingleOrder handles complete notification flow for one order.
// Fetches order details, resolves customer contact info, generates email, and sends.
