// Package handlers provides HTTP request handlers for restock notification management.
// This file contains handlers for managing customer restock notifications, allowing customers
// to subscribe to alerts when out-of-stock products become available again. This improves
// customer experience and helps recover potential lost sales.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// RequestRestockNotification allows customers to subscribe to restock alerts.
// When a product or variant becomes available again, subscribed users will be notified.
// This helps capture demand for out-of-stock items and improve customer satisfaction.
//
// @Summary      Create restock notification request
// @Description  Subscribe to receive notification when an out-of-stock product becomes available
// @Tags         Restock Notification
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.RestockNotificationRequest  true  "Restock notification request"
// @Success      200      {object}  dtos.SuccessResponse             "Notification request created successfully"
// @Failure      400      {object}  dtos.ErrorResponse               "Invalid request data"
// @Failure      409      {object}  dtos.ErrorResponse               "Notification already exists"
// @Security     BearerAuth
// @Router       /api/restock-notifications [post]
func RequestRestockNotification(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.RestockNotificationRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request (user ID, product ID, contact method)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Notifications") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	if err := models.CreateRestockNotification(models.DB, *req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to create restock notification",
				Code:        http.StatusBadRequest,
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
			Module:      "Notifications",
			Description: "Restock notification sent",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Restock notification sent",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ListUserRestockNotifications retrieves all active restock notification subscriptions for a user.
// This allows customers to view which products they're waiting for and manage their subscriptions.
//
// @Summary      Get user's restock notification subscriptions
// @Description  Retrieve all active restock notifications for a specific user
// @Tags         Restock Notification
// @Produce      json
// @Param        user_id  path      string                            true  "User ID"
// @Success      200      {array}   dtos.RestockNotification          "List of active restock notifications"
// @Failure      400      {object}  dtos.ErrorResponse                "Invalid user ID or database error"
// @Security     BearerAuth
// @Router       /api/restock-notifications/{user_id} [get]
func ListUserRestockNotifications(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract user ID from URL path parameters
	// The user ID from the path is not directly used for fetching notifications
	// as the authenticated user's ID is preferred for security.
	_ = mux.Vars(r)["user_id"]

	// Extract authenticated user from context for secure listing
	authuser, _ := middleware.UserFromContext(r.Context())
	// Fetch all active restock notifications for this user from database
	notifications, err := models.ListRestockNotificationsByUser(models.DB, authuser.ID)
	// Debug logging to track notification retrieval
	log.Printf("******nots %v", notifications)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to list restock notifications for user",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return successful response with list of all active notification subscriptions
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Restock notifications retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   notifications,
		Message:   "Notifications",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CancelRestockNotification allows users to unsubscribe from a restock notification.
// Users may cancel if they no longer want the product or found it elsewhere.
//
// @Summary      Cancel restock notification subscription
// @Description  Delete a restock notification subscription for a user
// @Tags         Restock Notification
// @Produce      json
// @Param        user_id          path      string                 true  "User ID"
// @Param        notification_id  path      string                 true  "Notification ID"
// @Success      200              {object}  dtos.SuccessResponse   "Notification cancelled successfully"
// @Failure      400              {object}  dtos.ErrorResponse     "Invalid parameters or notification not found"
// @Security     BearerAuth
// @Router       /api/restock-notifications/{user_id}/{notification_id} [delete]
func CancelRestockNotification(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract notification ID and user ID from URL path parameters
	notificationID := mux.Vars(r)["notification_id"]
	_ = mux.Vars(r)["user_id"] // Placeholder for unused path variable

	// Extract authenticated user from context for secure cancellation
	authuser, _ := middleware.UserFromContext(r.Context())

	// Delete the restock notification from database
	// Verifies user owns this notification before deletion for security
	if err := models.DeleteRestockNotification(models.DB, notificationID, authuser.ID); err != nil {
		// Deletion failed (notification not found or user doesn't own it)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to cancel restock notification with ID " + notificationID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return success response - user unsubscribed from restock notification
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Restock notification with ID " + notificationID + " cancelled successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Restock notification cancelled successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// TriggerRestockNotifications sends notifications to all subscribed users when a product is restocked.
// This is typically called automatically by the inventory management system when stock levels increase.
// It notifies waiting customers via email/SMS and cleans up notification records after sending.
//
// @Summary      Trigger restock notifications for a product
// @Description  Send notifications to all users waiting for a product to be restocked
// @Tags         Restock Notification
// @Produce      json
// @Param        product_id  path      string                 true   "Product ID"
// @Param        variant_id  query     string                 false  "Optional variant ID for specific variant"
// @Success      200         {object}  dtos.SuccessResponse   "Notifications triggered with count"
// @Failure      400         {object}  dtos.ErrorResponse     "Invalid product ID or database error"
// @Security     BearerAuth
// @Router       /api/restock-notifications/trigger [post]
func TriggerRestockNotifications(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract product ID from URL path parameters
	productID := mux.Vars(r)["product_id"]
	// Extract optional variant ID from query parameters
	// If provided, only notify users waiting for this specific variant
	variantID := r.URL.Query().Get("variant_id")
	var variantPtr *string
	if variantID != "" {
		// Convert to pointer for optional database query parameter
		variantPtr = &variantID
	}

	// Fetch all pending notifications for this product/variant from database
	notifications, err := models.GetNotificationsByProduct(models.DB, productID, variantPtr)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to get restock notifications for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// TODO1: Send email/SMS to each user in the notifications list
	// After sending, delete notification records to prevent duplicate alerts
	// This should be implemented with a notification service (email, SMS, push)

	// Return success response with count of notifications triggered
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Restock notifications triggered for product ID " + productID,
			Code:        http.StatusOK,
		},
		Payload:   len(notifications),
		Message:   "triggered_count",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}
