// Package models provides data access functions for restock notification management.
//
// This file handles customer restock notification subscriptions allowing:
//   - Notification subscription for out-of-stock products
//   - Optional variant-specific notifications
//   - User subscription management (list, delete)
//   - Product-based notification retrieval for triggering alerts
//   - Duplicate subscription prevention
//
// When products are restocked, the system can query subscribed users and send
// notifications via email/SMS to recover potential lost sales.
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"

	"github.com/teris-io/shortid"
)

// CreateRestockNotification creates a restock notification subscription for a user.
//
// This function allows customers to subscribe to alerts when an out-of-stock product
// (or specific variant) becomes available again. It validates all entities exist and
// prevents duplicate subscriptions.
//
// Parameters:
//   - rn: dtos.RestockNotificationRequest containing:
//   - UserID: The user subscribing to the notification
//   - ProductID: The out-of-stock product to monitor
//   - VariantID: Optional specific variant (e.g., size, color) - nil for any variant
//
// Returns:
//   - error: "user not found", "product not found", "variant not found",
//     "notification for this product already exists", database error, or nil on success
//
// Workflow:
//  1. Validate user exists
//  2. Validate product exists
//  3. Check for duplicate subscription (user already subscribed to this product)
//  4. If variant specified, validate variant exists
//  5. Generate unique notification ID
//  6. Insert subscription record
func CreateRestockNotification(db DBExecutor, rn dtos.RestockNotificationRequest) error {
	// Validate user exists
	err := isUserThere(db, rn.UserID)
	if err != nil {
		return err
	}

	// Validate product exists
	err = IsProductThere(db, rn.ProductID)

	if err != nil {
		return err
	}

	// Check for duplicate subscription (prevent user from subscribing twice)
	err = isRestockNotification(db, rn.UserID, rn.ProductID)
	if err != nil {
		return err
	}

	// If variant specified, validate variant exists
	if rn.VariantID != nil {
		err = isVariantThere(db, *rn.VariantID)
		if err != nil {
			return err
		}
	}

	// Generate unique notification subscription ID
	notificationID, _ := shortid.Generate()

	// Insert restock notification subscription
	query := `
		INSERT INTO restock_notifications (notification_id, user_id, product_id, variant_id)
		VALUES (?, ?, ?, ?)
	`
	_, err = db.Exec(query, notificationID, rn.UserID, rn.ProductID, rn.VariantID)
	return err
}

// ListRestockNotificationsByUser retrieves all active restock notification subscriptions for a user.
//
// This allows users to view which products they're waiting for and manage their subscriptions.
// Results are ordered by creation date (newest first).
//
// Parameters:
//   - userID: string - The user_id to retrieve subscriptions for
//
// Returns:
//   - []dtos.RestockNotification: Array of active subscriptions containing:
//   - NotificationID: Unique subscription identifier
//   - UserID: Subscribing user
//   - ProductID: Product being monitored
//   - VariantID: Optional specific variant (nil if monitoring all variants)
//   - CreatedAt: Subscription creation timestamp
//   - error: "user not found", database error, or nil on success
func ListRestockNotificationsByUser(db DBExecutor, userID string) ([]dtos.RestockNotification, error) {
	// Validate user exists
	err := isUserThere(db, userID)
	if err != nil {
		return nil, err
	}

	// Query all active subscriptions for this user
	query := `
		SELECT notification_id, user_id, product_id, variant_id, created_at
		FROM restock_notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan notification subscriptions
	var notifications []dtos.RestockNotification
	for rows.Next() {
		var rn dtos.RestockNotification
		if err := rows.Scan(&rn.NotificationID, &rn.UserID, &rn.ProductID, &rn.VariantID, &rn.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, rn)
	}
	return notifications, nil
}

// DeleteRestockNotification cancels a restock notification subscription.
//
// This allows users to unsubscribe from notifications if they no longer want the product
// or found it elsewhere. Validates notification exists and user owns it before deletion.
//
// Parameters:
//   - notificationID: string - The notification_id to cancel
//   - userID: string - The user_id attempting to cancel (for ownership verification)
//
// Returns:
//   - error: "notification not found", "user not found", database error, or nil on success
func DeleteRestockNotification(db DBExecutor, notificationID, userID string) error {
	// Validate notification exists
	exists, err := RecordExists(db, "restock_notifications", "notification_id = ?", notificationID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("notification not found")
	}

	// Validate user exists
	err = isUserThere(db, userID)
	if err != nil {
		return err
	}

	// Delete notification (only if user owns it)
	query := `
		DELETE FROM restock_notifications
		WHERE notification_id = ? AND user_id = ?
	`
	_, err = db.Exec(query, notificationID, userID)
	return err
}

// GetNotificationsByProduct retrieves all subscriptions for a specific product/variant.
//
// This function is used by the inventory system to find which users should be notified
// when a product is restocked. Supports both general product notifications and
// variant-specific notifications.
//
// Parameters:
//   - productID: string - The product_id that has been restocked
//   - variantID: *string - Optional specific variant ID:
//   - If nil: Returns all users waiting for ANY variant of this product
//   - If specified: Returns only users waiting for this specific variant
//
// Returns:
//   - []dtos.RestockNotification: Array of subscriptions to notify containing:
//   - NotificationID: Unique subscription identifier
//   - UserID: User to notify
//   - ProductID: The restocked product
//   - VariantID: The specific variant (if applicable)
//   - CreatedAt: Subscription creation timestamp
//   - error: Database error or nil on success
//
// Usage:
//   - Variant restocked: GetNotificationsByProduct("prod123", &"var456") - notify variant-specific subscribers
//   - General product restocked: GetNotificationsByProduct("prod123", nil) - notify all subscribers
func GetNotificationsByProduct(db DBExecutor, productID string, variantID *string) ([]dtos.RestockNotification, error) {
	// Build base query for product subscriptions
	query := `
		SELECT notification_id, user_id, product_id, variant_id, created_at
		FROM restock_notifications
		WHERE product_id = ?
	`
	var rows *sql.Rows
	var err error

	// Add variant filter if specific variant restocked
	if variantID != nil {
		query += " AND variant_id = ?"
		rows, err = db.Query(query, productID, variantID)
	} else {
		// No variant filter - all subscribers for this product
		rows, err = db.Query(query, productID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan notification subscriptions
	var notifications []dtos.RestockNotification
	for rows.Next() {
		var rn dtos.RestockNotification
		if err := rows.Scan(&rn.NotificationID, &rn.UserID, &rn.ProductID, &rn.VariantID, &rn.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, rn)
	}
	return notifications, nil
}
