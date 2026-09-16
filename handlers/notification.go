// Package handlers provides HTTP request handlers for notification management.
// This file contains handlers for creating, listing, updating, and deleting notifications,
// as well as background schedulers for automated notifications (order confirmations, low stock alerts).
// Supports multi-channel delivery: email, SMS, WhatsApp, and push notifications via WebSocket.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
// @Success      201             {object}  map[string]any   "Notification created and sent"
// @Failure      400             {object}  dtos.ErrorResponse     "Invalid request or channel"
// @Failure      401             {object}  dtos.ErrorResponse     "Admin authorization required"
// @Failure      404             {object}  dtos.ErrorResponse     "User not found"
// @Security     BearerAuth
// @Router       /api/admin/notifications [post]
func CreateNotificationHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify user has admin privileges (only admins can create notifications)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Notifications", "notifications.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with notification details
	req, ok := DecodeRequestBody[dtos.Notification](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}

	// Validate all required fields (recipient ID, channel, content)
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Notifications") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Verify recipient user exists in database
	exists, err := models.RecordExists(models.DB, "users", "user_id = ?", req.RecipientID)
	if err != nil {
		// Database query failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to check if user exists with ID " + req.RecipientID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if !exists {
		// Recipient user not found
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "User not found with ID " + req.RecipientID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "user not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Validate notification channel (must be email, SMS, WhatsApp, or push)
	if req.Channel != "email" && req.Channel != "sms" && req.Channel != "whatsapp" && req.Channel != "push" {
		// Invalid channel specified
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Invalid notification type",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Invalid notification type specified",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Send notification via specified channel (email, SMS, WhatsApp, or push)
	err = sendNotification(*req)
	if err != nil {
		// Notification sending failed (service down, invalid phone, etc.)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to send notification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Set timestamp when notification was sent
	req.SentAt = time.Now().Format("2006-01-02 15:04:05")
	// Store notification record in database for audit trail
	if err := models.CreateNotification(*req); err != nil {
		// Database insertion failed (notification was sent but not logged)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to create notification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate notification caches to ensure fresh data in list endpoints
	utils.DeleteCacheByPrefix("notifications_")
	utils.DeleteCacheByPrefix("notifications_pagination_")
	// Return success response
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notification created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Notification created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
		utils.SendToUser(req.RecipientID, map[string]any{
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
// @Success      200            {object}  map[string]any   "Notifications with pagination"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Failure      500            {object}  dtos.ErrorResponse     "Failed to retrieve notifications"
// @Security     BearerAuth
// @Router       /api/admin/notifications [get]
func ListNotificationsHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can view all notifications)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Notifications", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
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
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Notifications",
					Description: "Failed to list notifications",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notifications fetched successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Notifications fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
