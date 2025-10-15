package dtos

import (
	"encoding/json"
	"strconv"
	"time"
)

type CreateDeal struct {
	Name        string    `json:"name" validate:"required"`
	Description string    `json:"description" validate:"required"`
	Discount    *float64  `json:"discount"`
	StartDate   time.Time `json:"start_date" validate:"required"`
	EndDate     time.Time `json:"end_date" validate:"required"`
}
type Deal struct {
	DealID    string        `json:"deal_id"`
	Name      string        `json:"name" validate:"required"`
	StartDate time.Time     `json:"start_date"`
	EndDate   time.Time     `json:"end_date"`
	Products  []DealProduct `json:"products"`
}

type DealWithProducts struct {
	DealID    string        `json:"deal_id"`
	Name      string        `json:"name"`
	StartDate time.Time     `json:"start_date"`
	EndDate   time.Time     `json:"end_date"`
	Products  []DealProduct `json:"products"`
}

type ProductDeal struct {
	ID        string `json:"id" validate:"required"`
	ProductID string `json:"product_id" validate:"required"`
}

type FlashDealProducts struct {
	Title    string         `json:"title"`
	Image    string         `json:"image"`
	Duration string         `json:"duration"`
	Products []ProductsDeal `json:"products" validate:"dive"`
}
type ProductsDeal struct {
	ProductID    string        `json:"product_id" validate:"required"`
	Discount     FloatOrString `json:"discount" validate:"required"`
	DiscountType string        `json:"discount_type" validate:"required,oneof=percentage fixed"`
}

type FloatOrString float64

func (f *FloatOrString) UnmarshalJSON(data []byte) error {
	var num float64
	// Try to unmarshal as number
	if err := json.Unmarshal(data, &num); err == nil {
		*f = FloatOrString(num)
		return nil
	}

	// Try to unmarshal as string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if str == "" {
			*f = 0
			return nil
		}
		n, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		*f = FloatOrString(n)
		return nil
	}

	return nil
}
