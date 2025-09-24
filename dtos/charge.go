package dtos

type Charge struct {
	ID    string  `json:"charge_id"`
	Type  string  `json:"charge_name" validate:"required"`
	Value float64 `json:"charge_value" validate:"required"`
}
type PromoCodeStatusRequest struct {
	IsActive bool `json:"is_active" validate:"required"`
}
