package dtos

type CashPayment struct {
	Amount  float64 `json:"amount" validate:"required"`
	OrderID string  `json:"order_id" validate:"required"`
	// PromoCode *string `json:"promo_code"`
}

type CreditPayment struct {
	OrderID string `json:"order_id" validate:"required"`
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
	OrderID        string          `json:"order_id" validate:"required"`
	PaymentMethods []PaymentMethod `json:"payments" validate:"required,dive,required"`
}

type VoucherPayment struct {
	VoucherCode string `json:"voucher_code" validate:"required"`
	OrderID     string `json:"order_id" validate:"required"`
	// PromoCode *string `json:"promo_code"`
}
