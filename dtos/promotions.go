package dtos

import "time"

// DTO for effectiveness response
type PromotionEffectiveness struct {
	ProductID string  `json:"product_id"`
	TotalSold int64   `json:"total_sold"`
	Revenue   float64 `json:"revenue"`
}

// DTO for comparison response
type PromotionComparison struct {
	PromotionSales int64   `json:"promotion_sales"`
	PromotionRev   float64 `json:"promotion_revenue"`
	BaselineSales  int64   `json:"baseline_sales"`
	BaselineRev    float64 `json:"baseline_revenue"`
}

// DTO for summary response
type PromotionSummary struct {
	PromotionID   string    `json:"promotion_id"`
	PromotionType string    `json:"promotion_type"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	TotalSold     int64     `json:"total_sold"`
	Revenue       float64   `json:"revenue"`
}

type PromoCodeRequest struct {
	ID            string    `json:"id"`
	Description   *string   `json:"description"`
	DiscountType  string    `json:"discount_type" validate:"required,oneof=PERCENTAGE FIXED"`
	DiscountValue float64   `json:"discount_value" validate:"required"`
	ExpiresAt     time.Time `json:"expires_at" validate:"required"`
	IsActive      *bool     `json:"is_active"`
}

type PromoCodeResponse struct {
	ID            string    `json:"promo_code_id"`
	Code          string    `json:"code" validate:"required"`
	Description   *string   `json:"description"`
	DiscountType  string    `json:"discount_type" validate:"required,oneof=PERCENTAGE FIXED"`
	DiscountValue float64   `json:"discount_value" validate:"required"`
	ExpiresAt     time.Time `json:"expires_at" validate:"required"`
	IsActive      bool      `json:"is_active"`
}

type AddPromotionToProductRequest struct {
	ProductID       string `json:"product_id" validate:"required"`
	PromotionTypeID string `json:"promotion_type_id" validate:"required"`
}
