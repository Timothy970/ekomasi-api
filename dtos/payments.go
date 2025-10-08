package dtos

import "time"

type MpesaRequest struct {
	Phone       string `json:"phone_number" validate:"required"`
	Amount      int    `json:"amount" validate:"required, min=1"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
	OrderID     string `json:"order_id"`
	DeliveryID  string `json:"delivery_id"`
	Type        string `json:"type"`
}
type STKCallbackRequest struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        int    `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  struct {
				Item []struct {
					Name  string      `json:"Name"`
					Value interface{} `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}
type VoucherRequest struct {
	VoucherCode string `json:"voucher_code"`
	Amount      int    `json:"amount"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
	OrderID     string `json:"order_id"`
	DeliveryID  string `json:"delivery_id"`
}
type Voucher struct {
	BackImage     *string   `json:"back_image"`
	FrontImage    *string   `json:"front_image"`
	VoucherType   string    `json:"voucher_type" validate:"required, oneof=fixed percentage free_shipping price_discount"`
	Amount        float64   `json:"amount" validate:"required"`
	ToName        string    `json:"to_name" validate:"required"`
	ToEmail       string    `json:"to_email" validate:"required"`
	FromName      string    `json:"from_name" validate:"required"`
	DeliveryTime  time.Time `json:"delivery_time" validate:"required"`
	Message       string    `json:"message" validate:"required"`
	Status        *string   `json:"status"`
	InternalNotes string    `json:"internal_notes"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiryDate    string    `json:"expiry_date" validate:"required"`
}
type BuyVoucher struct {
	Amount     float64   `json:"amount" validate:"required"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiryDate time.Time `json:"expiry_date" validate:"required"`
}
type BuyVoucherData struct {
	Amount       float64   `json:"amount" validate:"required"`
	FromName     string    `json:"from_name" validate:"required"`
	ToName       string    `json:"to_name" validate:"required"`
	ToEmail      string    `json:"to_email" validate:"required"`
	Message      string    `json:"message" validate:"required"`
	DeliveryTime time.Time `json:"delivery_time" validate:"required"`
}
type VoucherData struct {
	VoucherID  string    `json:"voucher_id"`
	Code       string    `json:"code"`
	Amount     float64   `json:"amount"`
	Balance    float64   `json:"balance"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiryDate time.Time `json:"expiry_date"`
	From       *string   `json:"from"`
	To         *string   `json:"to"`
	IsReedemed *bool     `json:"is_reedemed"`
}
type VoucherDataUpdate struct {
	Amount     float64   `json:"amount" validate:"required"`
	Balance    *float64  `json:"balance"`
	Status     *string   `json:"status"`
	ExpiryDate time.Time `json:"expiry_date" validate:"required"`
}
type Payment struct {
	PaymentID     string  `json:"payment_id"`
	OrderID       string  `json:"order_id" validate:"required"`
	Amount        float64 `json:"amount" validate:"required"`
	VoucherID     *string `json:"voucher_id,omitempty"`
	Status        string  `json:"status"`
	PaymentMethod string  `json:"payment_method" validate:"required"`
	TransactionID string  `json:"transaction_id" validate:"required"`
	CreatedAt     string  `json:"created_at"`
}
type PaymentUpdate struct {
	Status string `json:"status" validate:"required"`
}
type PaymentListResponse struct {
	Payments []Payment      `json:"payments"`
	Meta     PaginationMeta `json:"pagination"`
}

type Refund struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	OrderID   string    `json:"order_id" validate:"required"`
	Amount    float64   `json:"amount" validate:"required"`
	Reason    string    `json:"reason" validate:"required"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt time.Time `json:"updated_at"`
}
type RefundPayload struct {
	Status string `json:"status"` // expected: "approved", "rejected", or "processed"
}

type PaginatedRefundsResponse struct {
	Refunds []Refund       `json:"refunds"`
	Meta    PaginationMeta `json:"pagination"`
}
type CreateVoucherRequest struct {
	VoucherID        string     `json:"voucher_id"`
	Code             string     `json:"code"`
	VerificationHash string     `json:"verification_hash"`
	Amount           float64    `json:"amount"`
	IsRedeemed       bool       `json:"is_redeemed"`
	CreatedAt        time.Time  `json:"created_at"`
	RedeemedAt       *time.Time `json:"redeemed_at,omitempty"`
}

type RedeemVoucherRequest struct {
	Code string `json:"code" validate:"required"`
}

type VoucherEmailInfo struct {
	VoucherID       string  `json:"voucher_id"`
	PersonalizedMsg string  `json:"personalized_msg"`
	DeliveryTime    string  `json:"delivery_time"`
	VoucherCode     string  `json:"voucher_code"`
	Amount          float64 `json:"amount"`
	ToName          string  `json:"to_name"`
	ToEmail         string  `json:"to_email"`
	FromName        string  `json:"from_name"`
	Message         string  `json:"message"`
	ExpiryDate      string  `json:"expiry_date"`
	Code            string  `json:"code"`
}
