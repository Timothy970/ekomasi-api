package dtos

import "time"

type ReviewResponse struct {
	ID        string    `json:"review_id"`
	ProductID string    `json:"product_id"`
	UserID    string    `json:"user_id"`
	Score     int       `json:"score"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

type ReviewRequest struct {
	UserID  string `json:"user_id" validate:"required"`
	Score   int    `json:"score" validate:"required"`
	Details string `json:"details" validate:"required"`
}
type UpdateReview struct {
	Score   int    `json:"score,omitempty"`
	Details string `json:"details,omitempty"`
	Status  string `json:"status" validate:"required"`
}
