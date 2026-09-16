// Package models provides data access functions for the Ekomasi backend.
//
// This file (notification.go) contains notification and logging management functions including:
//   - Notification CRUD operations (create, read, update, delete)
//   - Notification listing with pagination
//   - System log retrieval with pagination
//   - Low stock product monitoring for inventory alerts
//   - Support for bulk notifications and guest notifications
package models

import (
	"ekomasi_backend/dtos"
	"fmt"
	"math"
)

// CreateNotification inserts a new notification into the database.
//
// This function creates notifications for users or guests across various channels
// (email, SMS, push, etc.) with support for both individual and bulk notifications.
//
// Parameters:
//   - n: dtos.Notification containing:
//   - NotificationID: Unique identifier for the notification
//   - SenderID: ID of the user/system sending the notification
//   - RecipientID: ID of the recipient user (can be empty for guest notifications)
//   - GuestNotificationDetails: Contact details for guest recipients (email/phone)
//   - Channel: Delivery channel ("email", "sms", "push", etc.)
//   - Status: Current status ("pending", "sent", "failed", etc.)
//   - Content: Notification message content
//   - SentAt: Timestamp when notification was created/scheduled
//   - IsBulk: Boolean indicating if this is part of a bulk notification campaign
//
// Returns:
//   - error: Database execution error or nil on success
func CreateNotification(n dtos.Notification) error {
	// Insert notification with all fields
	query := `
		INSERT INTO notifications (
			notification_id, sender_id, recipient_id, guest_notification_details,
			channel, status, content, sent_at, is_bulk
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := DB.Exec(query,
		n.NotificationID, n.SenderID, n.RecipientID, n.GuestNotificationDetails,
		n.Channel, n.Status, n.Content, n.SentAt, n.IsBulk,
	)
	return err
}

// GetNotificationByID retrieves a single notification by its ID.
//
// This function fetches complete notification details including sender,
// recipient, channel, status, and content information.
//
// Parameters:
//   - id: string - The unique notification_id to retrieve
//
// Returns:
//   - *dtos.Notification: Pointer to notification with all fields populated
//   - error: sql.ErrNoRows if notification not found, or database error
func GetNotificationByID(id string) (*dtos.Notification, error) {
	// Query all notification fields
	query := `
		SELECT notification_id, sender_id, recipient_id, guest_notification_details,
		       channel, status, content, sent_at, is_bulk
		FROM notifications
		WHERE notification_id = ?
	`

	row := DB.QueryRow(query, id)

	var n dtos.Notification
	// Scan all fields into notification struct
	err := row.Scan(
		&n.NotificationID, &n.SenderID, &n.RecipientID, &n.GuestNotificationDetails,
		&n.Channel, &n.Status, &n.Content, &n.SentAt, &n.IsBulk,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// UpdateNotification updates the status of an existing notification.
//
// This function modifies only the notification status, typically used to mark
// notifications as "sent", "read", "failed", etc.
//
// Parameters:
//   - status: string - New status value ("pending", "sent", "read", "failed", etc.)
//   - notificationID: string - The unique notification_id to update
//
// Returns:
//   - error: "notification not found" if ID doesn't exist, or database error
func UpdateNotification(status, notificationID string) error {
	// Validate notification exists before updating
	exists, err := RecordExists(DB, "notifications", "notification_id = ?", notificationID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("notification not found")
	}

	// Update only the status field
	query := `
		UPDATE notifications
		SET status = ?
		WHERE notification_id = ?
	`
	_, err = DB.Exec(query,
		status, notificationID,
	)
	return err
}

// DeleteNotification permanently removes a notification from the database.
//
// This function validates the notification exists before attempting deletion
// to provide clear error messages.
//
// Parameters:
//   - id: string - The unique notification_id to delete
//
// Returns:
//   - error: "notification not found" if ID doesn't exist, or database error
//
// Warning: This is a permanent delete operation with no soft-delete or recovery.
func DeleteNotification(id string) error {
	// Validate notification exists before deleting
	exists, err := RecordExists(DB, "notifications", "notification_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("notification not found")
	}

	// Permanent deletion
	query := `DELETE FROM notifications WHERE notification_id = ?`
	_, err = DB.Exec(query, id)
	return err
}

// ListNotifications retrieves paginated notifications ordered by most recent first.
//
// This function returns all notifications with pagination metadata, useful for
// displaying notification feeds, admin panels, or notification history.
//
// Parameters:
//   - page: int - Page number (1-indexed)
//   - limit: int - Number of notifications per page
//
// Returns:
//   - []dtos.Notification: Array of notifications ordered by sent_at DESC
//   - *dtos.PaginationMeta: Pagination metadata with:
//   - Page: Current page number
//   - Size: Items per page
//   - TotalItems: Total notification count
//   - TotalPages: Total pages available
//   - HasPrev: Boolean indicating if previous page exists
//   - HasNext: Boolean indicating if next page exists
//   - error: Database error or nil on success
func ListNotifications(page, limit int) ([]dtos.Notification, *dtos.PaginationMeta, error) {
	// Calculate pagination offset
	offset := (page - 1) * limit

	// Count total notifications for pagination metadata
	var totalItems int
	err := DB.QueryRow(`SELECT COUNT(*) FROM notifications`).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Fetch notifications ordered by most recent first
	query := `
		SELECT notification_id, sender_id, recipient_id, guest_notification_details,
		       channel, status, content, sent_at, is_bulk
		FROM notifications
		ORDER BY sent_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := DB.Query(query, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan all notification rows
	var notifications []dtos.Notification
	for rows.Next() {
		var n dtos.Notification
		err := rows.Scan(
			&n.NotificationID, &n.SenderID, &n.RecipientID, &n.GuestNotificationDetails,
			&n.Channel, &n.Status, &n.Content, &n.SentAt, &n.IsBulk,
		)
		if err != nil {
			return nil, nil, err
		}
		notifications = append(notifications, n)
	}

	// Build pagination metadata
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return notifications, meta, nil
}

// ListLogs retrieves paginated system logs ordered by most recent first.
//
// This function returns all system logs with automatic input validation and
// pagination metadata. Useful for admin dashboards, debugging, and audit trails.
//
// Parameters:
//   - page: int - Page number (automatically adjusted to minimum 1 if invalid)
//   - limit: int - Logs per page (automatically adjusted to minimum 10 if invalid)
//
// Returns:
//   - []dtos.Log: Array of logs containing:
//   - LogID: Unique log identifier
//   - Level: Log level ("info", "warning", "error", etc.)
//   - Message: Log message content
//   - Timestamp: When the log was created
//   - UserID: Associated user (if applicable)
//   - Metadata: Additional JSON metadata
//   - *dtos.PaginationMeta: Pagination metadata (Page, Size, TotalItems, TotalPages, HasPrev, HasNext)
//   - error: Database error or nil on success
func ListLogs(page, limit int) ([]dtos.Log, *dtos.PaginationMeta, error) {
	// Validate and correct page number (minimum 1)
	if page < 1 {
		page = 1
	}
	// Validate and correct limit (minimum 10)
	if limit < 1 {
		limit = 10
	}
	// Calculate pagination offset
	offset := (page - 1) * limit

	// Count total logs for pagination metadata
	var totalItems int
	err := DB.QueryRow(`SELECT COUNT(*) FROM logs`).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Fetch logs ordered by most recent first
	query := `
		SELECT log_id, level, message, timestamp, user_id, metadata
		FROM logs
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := DB.Query(query, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan all log rows
	var logs []dtos.Log
	for rows.Next() {
		var log dtos.Log
		err := rows.Scan(&log.LogID, &log.Level, &log.Message, &log.Timestamp, &log.UserID, &log.Metadata)
		if err != nil {
			return nil, nil, err
		}
		logs = append(logs, log)
	}

	// Build pagination metadata
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return logs, meta, nil
}

// GetLowStockProducts retrieves products with stock at or below warning threshold.
//
// This function identifies products that need restocking by comparing current
// stock_quantity against the low_stock_quantity_warning threshold. Limited to
// 30 products to prevent overwhelming inventory teams.
//
// Returns:
//   - []dtos.Product: Array of up to 30 products with:
//   - Full product details including stock levels
//   - Products where: stock_quantity <= low_stock_quantity_warning
//   - error: Database error or nil on success
//
// Use Cases:
//   - Automated low stock alerts/notifications
//   - Inventory dashboard warnings
//   - Reorder trigger system
//   - Stock monitoring reports
//
// Limited to 30 products to avoid performance issues with bulk operations.
func GetLowStockProducts() ([]dtos.Product, error) {
	// Query product IDs where stock is at or below warning threshold
	// Limited to 30 to prevent overwhelming notifications
	//ordered by stock quantity ascending to prioritize lowest stock first
	selectQuery := ` 
	SELECT product_id from products where stock_quantity <= low_stock_quantity_warning
	ORDER BY stock_quantity ASC
	LIMIT 30
`
	rows, err := DB.Query(selectQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.Product
	// Iterate through low stock product IDs
	for rows.Next() {
		var productID string
		if err := rows.Scan(&productID); err != nil {
			return nil, err
		}

		// Fetch full product details for each low stock item
		product, err := GetProductByID(DB, productID)
		if err != nil {
			return nil, err
		}
		products = append(products, *product)
	}
	return products, nil
}
