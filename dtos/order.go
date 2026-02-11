package dtos

import (
	"encoding/json"
	"time"
)

type Order struct {
	OrderID              string               `json:"order_id"`
	TotalAmount          float64              `json:"total_amount"`
	TotalDiscount        float64              `json:"total_discount"`
	SubTotal             float64              `json:"sub_total"`
	EstimatedTax         float64              `json:"estimated_tax"`
	IsGuestOrder         bool                 `json:"is_guest_order"`
	DeliveryID           string               `json:"delivery_id"`
	OrderStatus          *string              `json:"order_status"`
	DeliveryStatus       *string              `json:"delivery_status"`
	PaymentMethod        string               `json:"payment_method"`
	PaymentStatus        *string              `json:"payment_status"`
	DeliveryCharge       *float64             `json:"delivery_charge"`
	DeliveryAddress      *string              `json:"delivery_address"`
	GuestDeliveryAddress GuestDeliveryAddress `json:"guest_delivery_address"`
	GuestPersonalDetails GuestPersonalDetails `json:"guest_personal_details"`
	CreatedAt            time.Time            `json:"created_at"`
	Items                []OrderProduct       `json:"items"`
	UserAddress          *[]UserAddress       `json:"user_address"`
	UserID               *string              `json:"user_id,omitempty"`
}
type AdminOrder struct {
	OrderID              string               `json:"order_id"`
	TotalAmount          float64              `json:"total_amount"`
	TotalDiscount        float64              `json:"total_discount"`
	EstimatedTax         float64              `json:"estimated_tax"`
	SubTotal             float64              `json:"sub_total"`
	IsGuestOrder         bool                 `json:"is_guest_order"`
	DeliveryID           string               `json:"delivery_id"`
	OrderStatus          string               `json:"order_status"`
	PaymentStatus        string               `json:"payment_status"`
	DeliveryStatus       *string              `json:"delivery_status"`
	PaymentMethod        string               `json:"payment_method"`
	DeliveryCharge       *float64             `json:"delivery_charge"`
	DeliveryAddress      *string              `json:"delivery_address"`
	DeliveredAt          *time.Time           `json:"delivered_at"`
	GuestDeliveryAddress GuestDeliveryAddress `json:"guest_delivery_address"`
	GuestPersonalDetails GuestPersonalDetails `json:"guest_personal_details"`
	CreatedAt            time.Time            `json:"created_at"`
	ItemsCount           int                  `json:"items_count"`
	Items                []OrderProduct       `json:"items"`
	User                 *Users               `json:"user"`
	Rider                *Rider               `json:"rider"`
}
type GuestPersonalDetails struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

type GuestDeliveryAddress struct {
	Apartment  string `json:"apartment"`
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type Rider struct {
	UserID    string  `json:"user_id"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
}

type OrderProduct struct {
	ID            string           `json:"product_id"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	SKU           string           `json:"sku"`
	Price         float64          `json:"price"`
	CategoryID    *string          `json:"category_id"`
	StockQuantity int              `json:"stock_quantity"`
	SearchVector  string           `json:"search_vector"`
	CreatedAt     time.Time        `json:"created_at"`
	LastUpdated   time.Time        `json:"last_updated"`
	Images        []Image          `json:"urls,omitempty"`
	Warranty      *ProductWarranty `json:"warranty"`
	IsReviewed    bool             `json:"is_reviewed,omitempty"`
	ReviewID      string           `json:"review_id,omitempty"`
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
	Status         *string `json:"order_status"`
	PaymentMethod  *string `json:"payment_method"`
	PaymentStatus  *string `json:"payment_status"`
	DeliveryStatus *string `json:"delivery_status"`
	DeliveredAt    *string `json:"delivered_at"`
}
type OrderResponse struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	CustomerID  int     `json:"customer_id"`
	TotalAmount float64 `json:"total_amount"`
	Status      string  `json:"order_status"`
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
	UserID               *string            `json:"user_id"`
	IsGuestOrder         *bool              `json:"is_guest_order"`
	GuestPersonalDetails *json.RawMessage   `json:"guest_personal_details,omitempty"`
	GuestDeliveryAddress *json.RawMessage   `json:"guest_delivery_address,omitempty"`
	CourierDetails       *string            `json:"courier_details,omitempty"`
	OrderItems           []OrderItemRequest `json:"order_items" validate:"required,dive"`
	DeliveryCharge       float64            `json:"delivery_charge" validate:"required"`
	DeliveryAddress      string             `json:"delivery_address" validate:"required"`
}
type CreateOrderPayload struct {
	IsGuestOrder         *bool              `json:"is_guest_order"`
	GuestPersonalDetails *json.RawMessage   `json:"guest_personal_details,omitempty"`
	GuestDeliveryAddress *json.RawMessage   `json:"guest_delivery_address,omitempty"`
	OrderItems           []OrderItemPayload `json:"order_items" validate:"required,dive"`
	DeliveryAddressID    *int64             `json:"location_id,omitempty"`
	PromoCode            *string            `json:"promo_code,omitempty"`
	StoreID              *string            `json:"store_id,omitempty"`
}
type OrderItemPayload struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required"`
}

type OrderStatusCount struct {
	Status      string  `json:"status"`
	Count       int     `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

type OrderNotificationItemRequest struct {
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity" validate:"required"`
	UnitPrice   float64 `json:"unit_price" validate:"required"`
}

type OrderEmailData struct {
	OrderID         string
	CustomerName    string
	OrderDate       string
	OrderItems      []OrderNotificationItemRequest
	Subtotal        float64
	ShippingFee     float64
	Discount        float64
	TotalAmount     float64
	DeliveryAddress string
}

type AssignOrderToRiderRequest struct {
	OrderID string `json:"order_id" validate:"required"`
	RiderID string `json:"rider_id" validate:"required"`
}

type UpdateOrderDeliveryStatusRequest struct {
	OrderID        string `json:"order_id" validate:"required"`
	DeliveryStatus string `json:"status" validate:"required"`
}
