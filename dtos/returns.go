package dtos

type ReturnRequest struct {
	ReturnProducts []ReturnProduct `json:"products" validate:"required,dive"`
	OrderID        string          `json:"order_id" validate:"required"`
	Reason         string          `json:"reason" validate:"required"`
}

type ReturnProduct struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
}

type ReturnResponse struct {
	ReturnID    string    `json:"return_id"`
	OrderID     string    `json:"order_id"`
	Products    []Product `json:"products"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	TotalRefund float64   `json:"total_refund"`
	CreatedAt   string    `json:"created_at"`
}
type ReturnStatusUpdate struct {
	Status      string  `json:"status" validate:"required,oneof=Pending Approved Rejected"`
	PhoneNumber *string `json:"phone_number,omitempty"` // Optional, required if status is Approved
}

type ReturnListResponse struct {
	Returns        []ReturnResponse `json:"returns"`
	CountsByStatus []ReturnsCounts  `json:"count"`
}
type ReturnsCounts struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}
