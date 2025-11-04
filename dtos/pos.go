package dtos

type CashPayment struct {
	Amount  float64 `json:"amount" binding:"required"`
	OrderID string  `json:"order_id" binding:"required"`
}
