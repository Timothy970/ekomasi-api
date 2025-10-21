package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"

	"github.com/teris-io/shortid"
)

func CreateRestockNotification(rn dtos.RestockNotificationRequest) error {
	err := isUserThere(rn.UserID)
	if err != nil {
		return err
	}
	err = IsProductThere(rn.ProductID)

	if err != nil {
		return err
	}
	err = isRestockNotification(rn.UserID, rn.ProductID)
	if err != nil {
		return err
	}
	if rn.VariantID != nil {
		err = isVariantThere(*rn.VariantID)
		if err != nil {
			return err
		}
	}
	notificationID, _ := shortid.Generate()
	query := `
		INSERT INTO restock_notifications (notification_id, user_id, product_id, variant_id)
		VALUES (?, ?, ?, ?)
	`
	_, err = DB.Exec(query, notificationID, rn.UserID, rn.ProductID, rn.VariantID)
	return err
}

func ListRestockNotificationsByUser(userID string) ([]dtos.RestockNotification, error) {
	err := isUserThere(userID)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT notification_id, user_id, product_id, variant_id, created_at
		FROM restock_notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func DeleteRestockNotification(notificationID, userID string) error {
	exists, err := RecordExists("restock_notifications", "notification_id = ?", notificationID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("notification not found")
	}
	err = isUserThere(userID)
	if err != nil {
		return err
	}
	query := `
		DELETE FROM restock_notifications
		WHERE notification_id = ? AND user_id = ?
	`
	_, err = DB.Exec(query, notificationID, userID)
	return err
}

func GetNotificationsByProduct(productID string, variantID *string) ([]dtos.RestockNotification, error) {
	query := `
		SELECT notification_id, user_id, product_id, variant_id, created_at
		FROM restock_notifications
		WHERE product_id = ?
	`
	var rows *sql.Rows
	var err error
	if variantID != nil {
		query += " AND variant_id = ?"
		rows, err = DB.Query(query, productID, variantID)
	} else {
		rows, err = DB.Query(query, productID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
