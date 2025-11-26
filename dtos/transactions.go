package dtos

type TransactionsList struct {
	TransactionID        string  `json:"transaction_id"`
	OrderID              string  `json:"order_id"`
	MpesaReference       *string `json:"mpesa_reference"`
	TransactionReference string  `json:"transaction_reference"`
	PhoneNumber          *string `json:"phone_number"`
	Amount               float64 `json:"amount"`
	AccountNumber        *string `json:"account_number"`
	Status               string  `json:"status"`
	TransactionDate      string  `json:"transaction_date"`
	PaymentMethod        string  `json:"payment_method"`
}

type UpdateTransactionStatus struct {
	Status string `json:"status" validate:"required"`
}
