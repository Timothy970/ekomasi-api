package dtos

type CashPayment struct {
	Amount  float64 `json:"amount" binding:"required"`
	OrderID string  `json:"order_id" binding:"required"`
	// PromoCode *string `json:"promo_code"`
}

type PaymentMethod struct {
	ID          string  `json:"id"`
	Type        string  `json:"type" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,min=1"`
	VoucherCode *string `json:"voucherCode"`
	PhoneNumber *string `json:"phoneNumber,omitempty"`
}

type SplitPaymentRequest struct {
	OrderID        string          `json:"order_id" binding:"required"`
	PaymentMethods []PaymentMethod `json:"payments" binding:"required,dive,required"`
}

type VoucherPayment struct {
	VoucherCode string `json:"voucher_code" binding:"required"`
	OrderID     string `json:"order_id" binding:"required"`
	// PromoCode *string `json:"promo_code"`
}
