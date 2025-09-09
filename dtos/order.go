package dtos

import (
	"encoding/json"
	"time"
)

type Order struct {
	OrderID       string      `json:"order_id"`
	TotalAmount   float64     `json:"total_amount"`
	TotalDiscount float64     `json:"total_discount"`
	DeliveryID    string      `json:"delivery_id"`
	Status        string      `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	Items         []OrderItem `json:"items"`
}
type UserOrder struct {
	OrderID       string      `json:"order_id"`
	TotalAmount   float64     `json:"total_amount"`
	TotalDiscount float64     `json:"total_discount"`
	DeliveryID    string      `json:"delivery_id"`
	Status        string      `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	Items         []OrderItem `json:"items"`
}

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type CreateOrderRequest struct {
	Items []OrderItem `json:"items"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}
type OrderResponse struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	CustomerID  int     `json:"customer_id"`
	TotalAmount float64 `json:"total_amount"`
	Status      string  `json:"status"`
	OrderDate   string  `json:"order_date"`

	Items []OrderItem `json:"items,omitempty"`
}
type OrderListResponse struct {
	Orders []OrderResponse `json:"orders"`
	Total  int             `json:"total"`
}
type OrderDetailResponse struct {
	Order OrderResponse `json:"order"`
	Items []OrderItem   `json:"items"`
}
type OrderStatusUpdateResponse struct {
	Message string `json:"message"`
}
type OrderErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
type OrderCreateResponse struct {
	OrderID int    `json:"order_id"`
	Message string `json:"message"`
}
type OrderListRequest struct {
	UserID  int `json:"user_id"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}
type OrderListResponseWithPagination struct {
	Orders      []OrderResponse `json:"orders"`
	Total       int             `json:"total"`
	CurrentPage int             `json:"current_page"`
	TotalPages  int             `json:"total_pages"`
	PerPage     int             `json:"per_page"`
}
type OrderCreateRequest struct {
	Items       []OrderItem `json:"items"`
	CustomerID  int         `json:"customer_id"`
	TotalAmount float64     `json:"total_amount"`
	Status      string      `json:"status"`
	OrderDate   string      `json:"order_date"`
}
type OrderUpdateRequest struct {
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	Status      string      `json:"status"`
	OrderDate   string      `json:"order_date"`
}
type OrderUpdateResponse struct {
	Message string `json:"message"`

	Order OrderResponse `json:"order"`
}
type OrderDeleteResponse struct {
	Message string `json:"message"`
	OrderID int    `json:"order_id"`
}
type GenericResponse struct {
	Message string `json:"message"`
}
type OrderItemRequest struct {
	ProductID string  `json:"product_id" validate:"required"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity" validate:"required"`
	UnitPrice float64 `json:"unit_price" validate:"required"`
}

type OrderRequest struct {
	UserID               *string            `json:"user_id,omitempty"`
	IsGuestOrder         *bool              `json:"is_guest_order"`
	GuestPersonalDetails *json.RawMessage   `json:"guest_personal_details,omitempty"`
	GuestDeliveryAddress *json.RawMessage   `json:"guest_delivery_address,omitempty"`
	CourierDetails       *string            `json:"courier_details,omitempty"`
	OrderItems           []OrderItemRequest `json:"order_items" validate:"required,dive"`
	DeliveryCharge       float64            `json:"delivery_charge" validate:"required"`
	DeliveryAddress      string             `json:"delivery_address" validate:"required"`
}
