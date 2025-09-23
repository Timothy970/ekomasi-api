package dtos

import "github.com/go-redis/redis/v8"

var Redis *redis.Client

type PromoCode struct {
	Code               string `json:"code"`
	DiscountPercentage int    `json:"discount_percentage"`
	ExpiryDate         string `json:"expiry_date"`
}
type PromoCodeData struct {
	DiscountType  string  `json:"discount_type"`
	DiscountValue float64 `json:"discount_value"`
}
