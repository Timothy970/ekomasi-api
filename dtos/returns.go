package dtos

type ReturnRequest struct {
	Products []string `json:"products" validate:"required"`
	Quantity int      `json:"quantity" validate:"required,min=1"`
	OrderID  string   `json:"order_id" validate:"required"`
	Reason   string   `json:"reason" validate:"required"`
}

type ReturnResponse struct {
	ReturnID    string    `json:"return_id"`
	OrderID     string    `json:"order_id"`
	Products    []Product `json:"products"`
	Quantity    int       `json:"quantity"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	TotalRefund float64   `json:"total_refund"`
	CreatedAt   string    `json:"created_at"`
}
type ReturnStatusUpdate struct {
	Status string `json:"status" validate:"required,oneof=Pending Approved Rejected"`
}
