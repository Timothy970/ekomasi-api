package dtos

import "encoding/json"

type Notification struct {
	NotificationID           string `json:"notification_id"`
	SenderID                 string `json:"sender_id"`
	RecipientID              string `json:"recipient_id" validate:"required"`
	GuestNotificationDetails string `json:"guest_notification_details"`
	Channel                  string `json:"channel"`
	Status                   string `json:"status"`
	Content                  string `json:"content"`
	SentAt                   string `json:"sent_at"`
	IsBulk                   bool   `json:"is_bulk"`
}
type UpdateNotification struct {
	Status string `json:"status" validate:"required"`
}

type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Meta          PaginationMeta `json:"meta"`
}
type Log struct {
	LogID     string          `json:"log_id"`
	Level     string          `json:"level"`
	Message   string          `json:"message"`
	Timestamp string          `json:"timestamp"`
	UserID    *string         `json:"user_id"`  // Nullable
	Metadata  json.RawMessage `json:"metadata"` // JSON
}

type LogListResponse struct {
	Logs []Log          `json:"logs"`
	Meta PaginationMeta `json:"meta"`
}

type SendTemplateRequest struct {
	ChatID       string         `json:"chatId"`
	TemplateName string         `json:"templateName"`
	Vars         map[string]any `json:"vars"`
}

type SendTextRequest struct {
	ChatID string `json:"chatId"`
	Text   string `json:"text"`
}
