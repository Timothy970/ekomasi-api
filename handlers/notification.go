// Package handlers provides HTTP request handlers for notification management.
// This file contains handlers for creating, listing, updating, and deleting notifications,
// as well as background schedulers for automated notifications (order confirmations, low stock alerts).
// Supports multi-channel delivery: email, SMS, WhatsApp, and push notifications via WebSocket.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/utils"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// CreateNotificationHandler creates and sends a notification to a user.
// Admin-only operation for manually triggering notifications via various channels.
// Supports email, SMS, WhatsApp, and push notifications. Essential for customer communication.
//
// @Summary      Create notification
// @Description  Create and send notification to user via specified channel (admin only)
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Param        Authorization   header    string                 true   "Bearer token"
// @Param        notification    body      dtos.Notification      true   "Notification details"
// @Success      201             {object}  map[string]interface{}   "Notification created and sent"
// @Failure      400             {object}  dtos.ErrorResponse     "Invalid request or channel"
// @Failure      401             {object}  dtos.ErrorResponse     "Admin authorization required"
// @Failure      404             {object}  dtos.ErrorResponse     "User not found"
// @Security     BearerAuth
// @Router       /api/admin/notifications [post]
func CreateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify user has admin privileges (only admins can create notifications)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Notifications", "notifications.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with notification details
	req, ok := DecodeRequestBody[dtos.Notification](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}

	// Validate all required fields (recipient ID, channel, content)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Notifications") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Verify recipient user exists in database
	exists, err := models.RecordExists(models.DB, "users", "user_id = ?", req.RecipientID)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to check if user exists with ID " + req.RecipientID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if !exists {
		// Recipient user not found
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "User not found with ID " + req.RecipientID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "user not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Validate notification channel (must be email, SMS, WhatsApp, or push)
	if req.Channel != "email" && req.Channel != "sms" && req.Channel != "whatsapp" && req.Channel != "push" {
		// Invalid channel specified
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Invalid notification type",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Invalid notification type specified",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Send notification via specified channel (email, SMS, WhatsApp, or push)
	err = sendNotification(*req)
	if err != nil {
		// Notification sending failed (service down, invalid phone, etc.)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to send notification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Set timestamp when notification was sent
	req.SentAt = time.Now().Format("2006-01-02 15:04:05")
	// Store notification record in database for audit trail
	if err := models.CreateNotification(*req); err != nil {
		// Database insertion failed (notification was sent but not logged)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to create notification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate notification caches to ensure fresh data in list endpoints
	utils.DeleteCacheByPrefix("notifications_")
	utils.DeleteCacheByPrefix("notifications_pagination_")
	// Return success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notification created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Notification created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// sendNotification routes notification to the appropriate delivery channel.
// Fetches user details and sends via email, SMS, WhatsApp, or push notification.
// Returns error if user doesn't have required contact information for the channel.
func sendNotification(req dtos.Notification) error {
	// Fetch user details for contact information
	user, err := models.GetUserByUserID(models.DB, req.RecipientID)
	if err != nil {
		// User not found (should have been caught earlier)
		return err
	}
	// Verify user has phone number for SMS or WhatsApp channels
	if (req.Channel == "whatsapp" || req.Channel == "sms") && user.Phone == "" {
		return errors.New("user doesn't have a phone number")
	}
	// Route notification to appropriate channel
	switch req.Channel {
	case "sms":
		// Send SMS message
		notification.SendSmsMessages(user.Phone, req.Content)
	case "whatsapp":
		// WhatsApp integration (currently commented out, needs template name)
		//to include template name
		// phoneInt, _ := strconv.Atoi(user.Phone)
		// notification.SendWhatsappMessages(phoneInt, req.Content, "Notification")
	case "email":
		// Send email with "Notification" as subject
		notification.SendEmail(user.Email, "Notification", req.Content)
	case "push":
		// Send real-time push notification via WebSocket
		utils.SendToUser(req.RecipientID, "", "", map[string]interface{}{
			"event":   "Notification",
			"message": "Your payment was successful!",
		})
	}

	return nil
}

// ListNotificationsHandler retrieves all notifications with pagination and caching.
// Admin-only operation for viewing notification history and delivery status.
// Uses Redis caching for performance optimization.
//
// @Summary      List all notifications
// @Description  Retrieve paginated list of all notifications with caching (admin only)
// @Tags         Notifications
// @Produce      json
// @Param        Authorization  header    string                 true   "Bearer token"
// @Param        page           query     int                    false  "Page number (default: 1)"
// @Param        size           query     int                    false  "Page size (default: 10)"
// @Success      200            {object}  map[string]interface{}   "Notifications with pagination"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Failure      500            {object}  dtos.ErrorResponse     "Failed to retrieve notifications"
// @Security     BearerAuth
// @Router       /api/admin/notifications [get]
func ListNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can view all notifications)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Notifications", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Generate unique cache keys for notifications and pagination metadata
	cacheKeyNotification := fmt.Sprintf("notifications_%d_size_%d", page, limit)
	cacheKeyPagination := fmt.Sprintf("notifications_pagination_%d_size_%d", page, limit)
	var notifications []dtos.Notification
	var cachedNotifications []dtos.Notification
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	// Attempt to retrieve notifications and pagination from Redis cache
	_ = utils.GetCache(cacheKeyNotification, &cachedNotifications)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	// If cache miss, fetch from database
	if cachedNotifications == nil {
		var err error
		// Fetch paginated notifications from database
		notifications, pagination, err = models.ListNotifications(page, limit)
		if err != nil {
			// Database query failed
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Notifications",
					Description: "Failed to list notifications",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	} else {
		// Cache hit - use cached data
		notifications = cachedNotifications
		pagination = cachedPagination
	}
	// Construct response with notifications and pagination metadata
	resp := dtos.NotificationListResponse{
		Notifications: notifications,
		Meta:          *pagination,
	}

	// Return success response with notifications list
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notifications fetched successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Notifications fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

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
func UpdateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can update notifications)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Notifications", "notifications.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract notification ID from URL path parameters
	id := mux.Vars(r)["notification_id"]
	// Decode and parse JSON request body with updated status
	req, ok := DecodeRequestBody[dtos.UpdateNotification](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}

	// Validate all required fields (status)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Notifications") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Update notification status in database
	if err := models.UpdateNotification(req.Status, id); err != nil {
		// Update failed (notification not found or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to update notification with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate notification caches to ensure fresh data
	utils.DeleteCacheByPrefix("notifications_")
	utils.DeleteCacheByPrefix("notifications_pagination_")
	// Return success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notification with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Notification updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func DeleteNotificationHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can delete notifications)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Notifications", "notifications.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract notification ID from URL path parameters
	id := mux.Vars(r)["notification_id"]
	// Delete notification from database
	if err := models.DeleteNotification(id); err != nil {
		// Deletion failed (notification not found or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to delete notification with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate notification caches to ensure fresh data
	utils.DeleteCacheByPrefix("notifications_")
	utils.DeleteCacheByPrefix("notifications_pagination_")
	// Return success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notification with ID " + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Notification deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func ListLogsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can view system logs)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	// Fetch logs from database with pagination
	logs, meta, err := models.ListLogs(page, limit)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Logs",
				Description: "Failed to list logs",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Construct response with logs and pagination metadata
	resp := dtos.LogListResponse{
		Logs: logs,
		Meta: *meta,
	}

	// Return success response with logs list
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Logs",
			Description: "Logs retrieved successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Logs",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// Returns error only for retryable failures (e.g., network issues).
// Permanent failures (e.g., no email) are handled internally and marked as failed.
func processSingleOrder(orderID string, emailType string) error {
	// Fetch order details from database
	order, err := models.GetOrderByID(models.DB, orderID)
	if err != nil {
		// Cannot fetch order - likely permanent failure (bad ID)
		// Skip and don't retry to avoid blocking other orders
		log.Printf("WARNING: Skipping order %s, cannot fetch details: %v", orderID, err)
		return nil
	}

	// Step 1: Get customer email and name
	userEmail, customerName, err := getCustomerDetails(order)
	if err != nil {
		// User ID present but user record missing/corrupt
		log.Printf("WARNING: Could not resolve user details for order %s: %v", orderID, err)
		// Continue processing, userEmail will be empty string
	}

	// Step 2: Validate email exists
	if userEmail == "" {
		log.Printf("INFO: No email found for order %s. Marking as 'failed'.", orderID)
		// Permanent failure - mark as failed so we don't retry
		if err := models.MarkOrderNotificationSent(models.DB, orderID, "failed"); err != nil {
			// Failed to mark as failed - return error to retry status update
			log.Printf("ERROR: Failed to mark order %s as 'failed': %v", orderID, err)
			return fmt.Errorf("marking order %s as failed: %w", orderID, err)
		}
		return nil // Successfully marked as failed, do not retry
	}

	// Step 3: Build email content and send
	subject := "Your Order Update"
	orderDetails := buildEmailData(order, customerName)
	htmlBody := utils.GenerateOrderConfirmationHTML(orderDetails, emailType)

	// Send email to customer
	if err := notification.SendEmail(userEmail, subject, htmlBody); err != nil {
		// Temporary retryable error (e.g., email gateway down)
		// Return error so order remains pending for next retry
		return fmt.Errorf("sending email for order %s: %w", orderID, err)
	}

	// Step 4: Mark notification as sent in database
	if err := models.MarkOrderNotificationSent(models.DB, orderID, "sent"); err != nil {
		// Email sent but failed to update status - return error to retry update
		// Risk of duplicate email, but better than losing track of sent status
		log.Printf("ERROR: Email sent for order %s, but failed to mark as 'sent': %v", orderID, err)
		return fmt.Errorf("marking order %s as sent: %w", orderID, err)
	}

	log.Printf("Successfully sent order notification for order ID %s", orderID)
	return nil
}

// getCustomerDetails extracts customer email and display name from order.
// Handles both registered users and guest checkouts.
// Returns empty email if neither user nor guest details available.
func getCustomerDetails(order *dtos.Order) (email string, name string, err error) {
	if order.UserID != nil {
		// Registered user - fetch from users table
		user, err := models.GetUserByUserID(models.DB, *order.UserID)
		if err != nil {
			// User ID present but user record not found
			return "", "", fmt.Errorf("getting user %s: %w", *order.UserID, err)
		}

		email = user.Email
		if user.FirstName != "" || user.LastName != "" {
			// Construct full name from first and last name
			name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		} else {
			// Fallback if name not available
			name = "Valued Customer"
		}
		return email, name, nil
	}

	if (order.GuestPersonalDetails != dtos.GuestPersonalDetails{}) {
		var email, name string
		// Guest user - use guest checkout details with nil checks
		if order.GuestPersonalDetails.Email != nil {
			email = *order.GuestPersonalDetails.Email
		} else {
			email = ""
		}
		var firstName, lastName string
		if order.GuestPersonalDetails.FirstName != nil {
			firstName = *order.GuestPersonalDetails.FirstName
		} else {
			firstName = ""
		}
		if order.GuestPersonalDetails.LastName != nil {
			lastName = *order.GuestPersonalDetails.LastName
		} else {
			lastName = ""
		}
		name = fmt.Sprintf("%s %s", firstName, lastName)
		return email, name, nil
	}

	// No UserID and no guest details - email will be empty
	return "", "Valued Customer", nil
}

// buildEmailData transforms order model to email DTO with safe pointer handling.
// Maps order items, handles nil pointers for delivery charge and address.
// Returns complete email data structure for template generation.
func buildEmailData(order *dtos.Order, customerName string) dtos.OrderEmailData {
	// Map order items to email notification format
	orderItems := make([]dtos.OrderNotificationItemRequest, len(order.Items))
	for i, product := range order.Items {
		// Create item DTO with product details
		orderItems[i] = dtos.OrderNotificationItemRequest{
			ProductName: product.Name,
			UnitPrice:   product.Price,
			Quantity:    product.StockQuantity,
		}
	}

	// Safely dereference delivery charge pointer to avoid nil panic
	var shippingFee float64
	if order.DeliveryCharge != nil {
		shippingFee = *order.DeliveryCharge
	}

	// Safely dereference delivery address pointer to avoid nil panic
	var deliveryAddress string
	if order.DeliveryAddress != nil {
		deliveryAddress = *order.DeliveryAddress
	}

	// Return complete email data structure
	return dtos.OrderEmailData{
		OrderID:         order.OrderID,
		CustomerName:    customerName,
		OrderDate:       order.CreatedAt.Format("2006-01-02 15:04:05"),
		ShippingFee:     shippingFee,
		Discount:        order.TotalDiscount,
		TotalAmount:     order.TotalAmount,
		DeliveryAddress: deliveryAddress,
		OrderItems:      orderItems,
	}
}

// StartLowStockEmailScheduler initializes background scheduler for low stock alerts.
// Runs at specified interval but only sends emails at configured hour of day.
// Essential for proactive inventory management.
func StartLowStockEmailScheduler(interval time.Duration, hourOfDay int) {
	// Create ticker that checks at specified interval
	ticker := time.NewTicker(interval)
	// Run scheduler in background goroutine
	go func() {
		for range ticker.C {
			log.Println("Low stock email scheduler tick")
			// Only process emails at specified hour (e.g., 9 AM daily)
			currentHour := time.Now().Hour()
			if currentHour == hourOfDay {
				// Time to send low stock alerts
				processLowStockEmails()
			}
		}
	}()
}

// Alternative implementation (commented out): runs on every tick without hour check
// func StartLowStockEmailScheduler(interval time.Duration) {
// 	log.Printf("Low stock scheduler working at interval %v....", interval)
// 	ticker := time.NewTicker(interval)
// 	go func() {
// 		for range ticker.C {
// 			log.Println("Low stock email scheduler tick - processing low stock emails")
// 			processLowStockEmails()
// 		}
// 	}()
// }

// processLowStockEmails checks inventory and sends alerts for low stock products.
// Fetches products below reorder threshold and emails admin team.
// Critical for preventing stockouts and lost sales.
func processLowStockEmails() {
	// Fetch all products with stock below reorder point
	lowStockProducts, err := models.GetLowStockProducts()
	if err != nil {
		// Critical error: cannot fetch inventory data
		log.Printf("CRITICAL: Failed to fetch low stock products: %v", err)
		return
	}
	if len(lowStockProducts) == 0 {
		// No low stock products - no alerts needed
		log.Printf("No low stock products found.")
		return
	}
	log.Printf("Preparing to send low stock alert emails for ...")
	// Prepare email data with store info and low stock products
	storeName := "Adenzo Store"
	emailData := dtos.LowStockEmailData{
		StoreName: storeName,
		AlertDate: time.Now().Format("2006-01-02"),
		Products:  lowStockProducts,
	}
	// Generate HTML email body from template
	htmlBody := utils.GenerateLowStockAlertHTML(emailData)
	subject := "Low Stock Alert"
	// Send to admin/inventory management team
	emails := []string{
		"timothy.kimani@roamtech.com",
		"mbithe.taabu@roamtech.com",
	}
	for _, email := range emails {
		// Send alert to each admin email
		if err := notification.SendEmail(email, subject, htmlBody); err != nil {
			log.Printf("ERROR: Failed to send low stock email to %s: %v", email, err)
		} else {
			log.Printf("Low stock email sent successfully to %s", email)
		}
	}
}
