package dtos

import "time"

// TenantPaymentGatewayConfig defines tenant-level configuration for card payment gateways
type TenantPaymentGatewayConfig struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	GatewayName   string    `json:"gateway_name" validate:"required"` // 'paystack', 'flutterwave', 'stripe'
	IsEnabled     bool      `json:"is_enabled"`
	PublicKey     string    `json:"public_key"`
	SecretKey     string    `json:"secret_key,omitempty"`
	EncryptionKey string    `json:"encryption_key,omitempty"`
	WebhookSecret string    `json:"webhook_secret,omitempty"`
	MerchantID    string    `json:"merchant_id,omitempty"`
	Currency      string    `json:"currency"`
	Mode          string    `json:"mode"` // 'test' or 'live'
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// PaymentInitializeRequest holds payload for initiating a card/online payment
type PaymentInitializeRequest struct {
	OrderID     string  `json:"order_id" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Currency    string  `json:"currency"`
	Email       string  `json:"email" validate:"required,email"`
	GatewayName string  `json:"gateway_name" validate:"required"` // 'paystack', 'flutterwave', 'stripe'
	CallbackURL string  `json:"callback_url"`
}

// PaymentInitializeResponse holds checkout authorization URL returned by payment gateway
type PaymentInitializeResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code,omitempty"`
	Reference        string `json:"reference"`
	GatewayName      string `json:"gateway_name"`
	Status           bool   `json:"status"`
	Message          string `json:"message"`
}

// PaymentVerifyResponse holds transaction verification status from payment gateway
type PaymentVerifyResponse struct {
	Reference     string    `json:"reference"`
	GatewayName   string    `json:"gateway_name"`
	Status        string    `json:"status"` // 'SUCCESS', 'FAILED', 'PENDING'
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	PaidAt        time.Time `json:"paid_at"`
	Channel       string    `json:"channel"`
	CustomerEmail string    `json:"customer_email"`
}
