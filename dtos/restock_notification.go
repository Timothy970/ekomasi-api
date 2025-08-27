package dtos

import "time"

type RestockNotificationRequest struct {
	UserID    string  `json:"user_id" validate:"required"`
	ProductID string  `json:"product_id" validate:"required"`
	VariantID *string `json:"variant_id,omitempty" validate:"omitempty"`
}

type RestockNotificationResponse struct {
	NotificationID string    `json:"notification_id"`
	UserID         string    `json:"user_id"`
	ProductID      string    `json:"product_id"`
	VariantID      *string   `json:"variant_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
type RestockNotification struct {
	NotificationID string    `db:"notification_id"`
	UserID         string    `db:"user_id"`
	ProductID      string    `db:"product_id"`
	VariantID      *string   `db:"variant_id"`
	CreatedAt      time.Time `db:"created_at"`
}
