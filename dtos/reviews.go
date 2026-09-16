package dtos

import "time"

type ReviewResponse struct {
	ID        string    `json:"review_id"`
	User      string    `json:"user"`
	Score     int       `json:"score"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

type ReviewRequest struct {
	UserID  string `json:"user_id" validate:"required"`
	Score   int    `json:"score" validate:"required,min=1,max=5"`
	Details string `json:"details" validate:"required"`
}
type UpdateReview struct {
	Score   int     `json:"score,omitempty" validate:"omitempty,min=1,max=5"`
	Details string  `json:"details,omitempty"`
	Status  *string `json:"status"`
}
type DetailedReviewResponse struct {
	Reviews      []ReviewResponse `json:"reviews"`
	AverageScore float64          `json:"average_score"`
	ScoreCounts  []ScoreCount     `json:"score_counts"`
}
type ScoreCount struct {
	Score int `json:"score"`
	Count int `json:"count"`
}

type AddReview struct {
	ProductID string `json:"product_id" validate:"required"`
	UserID    string `json:"user_id"`
	Score     int    `json:"score" validate:"required"`
	Details   string `json:"details" validate:"required"`
}
