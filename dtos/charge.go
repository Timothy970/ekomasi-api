package dtos

type Charge struct {
	ID    string  `json:"charge_id"`
	Type  string  `json:"charge_name" validate:"required"`
	Value float64 `json:"charge_value" validate:"required"`
}
type PromoCodeStatusRequest struct {
	IsActive bool `json:"is_active" validate:"required"`
}
type AddChargeToProductRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	ChargeID  string `json:"charge_id" validate:"required"`
}
