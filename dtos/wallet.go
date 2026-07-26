package dtos

import "time"

// CustomerWalletDTO represents customer wallet & store credit balance
type CustomerWalletDTO struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	TenantID    string    `json:"tenant_id"`
	Balance     float64   `json:"balance"`
	StoreCredit float64   `json:"store_credit"`
	Total       float64   `json:"total"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WalletDepositRequest holds payload for depositing funds into user wallet
type WalletDepositRequest struct {
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	GatewayName string  `json:"gateway_name" validate:"required"` // 'paystack', 'flutterwave', 'stripe', 'mpesa'
	Phone       string  `json:"phone,omitempty"`
	Email       string  `json:"email,omitempty"`
	CallbackURL string  `json:"callback_url,omitempty"`
}

// WalletTransactionDTO represents wallet ledger entries
type WalletTransactionDTO struct {
	ID           string    `json:"id"`
	WalletID     string    `json:"wallet_id"`
	Type         string    `json:"type"` // 'deposit', 'withdrawal', 'purchase_payment', 'store_credit_refund', 'cashback'
	Amount       float64   `json:"amount"`
	BalanceAfter float64   `json:"balance_after"`
	Reference    string    `json:"reference"`
	GatewayName  string    `json:"gateway_name"`
	Status       string    `json:"status"` // 'PENDING', 'COMPLETED', 'FAILED'
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

// PayWithWalletRequest holds payload for paying for an order using wallet or store credit
type PayWithWalletRequest struct {
	OrderID         string  `json:"order_id" validate:"required"`
	UseStoreCredit  bool    `json:"use_store_credit"`
	AmountToDeduct  float64 `json:"amount_to_deduct" validate:"required,gt=0"`
}
