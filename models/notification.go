package models

import (
	"adenzo_backend/dtos"
	"fmt"
	"math"
)

func CreateNotification(n dtos.Notification) error {
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
func GetNotificationByID(id string) (*dtos.Notification, error) {
	query := `
		SELECT notification_id, sender_id, recipient_id, guest_notification_details,
		       channel, status, content, sent_at, is_bulk
		FROM notifications
		WHERE notification_id = ?
	`

	row := DB.QueryRow(query, id)

	var n dtos.Notification
	err := row.Scan(
		&n.NotificationID, &n.SenderID, &n.RecipientID, &n.GuestNotificationDetails,
		&n.Channel, &n.Status, &n.Content, &n.SentAt, &n.IsBulk,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
func UpdateNotification(status, notificationID string) error {
	exists, err := RecordExists("notifications", "notification_id = ?", notificationID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("notification not found")
	}
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
func DeleteNotification(id string) error {
	exists, err := RecordExists("notifications", "notification_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("notification not found")
	}
	query := `DELETE FROM notifications WHERE notification_id = ?`
	_, err = DB.Exec(query, id)
	return err
}
func ListNotifications(page, limit int) ([]dtos.Notification, *dtos.PaginationMeta, error) {
	offset := (page - 1) * limit

	var totalItems int
	err := DB.QueryRow(`SELECT COUNT(*) FROM notifications`).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

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
func ListLogs(page, limit int) ([]dtos.Log, *dtos.PaginationMeta, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Count total logs
	var totalItems int
	err := DB.QueryRow(`SELECT COUNT(*) FROM logs`).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Query logs with pagination
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

	var logs []dtos.Log
	for rows.Next() {
		var log dtos.Log
		err := rows.Scan(&log.LogID, &log.Level, &log.Message, &log.Timestamp, &log.UserID, &log.Metadata)
		if err != nil {
			return nil, nil, err
		}
		logs = append(logs, log)
	}

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
