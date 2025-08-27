package dtos

import "github.com/go-redis/redis/v8"

var Redis *redis.Client

type PromoCode struct {
	Code               string `json:"code"`
	DiscountPercentage int    `json:"discount_percentage"`
	ExpiryDate         string `json:"expiry_date"`
}
