package dtos

// type CartItem struct {
// 	ProductID   string  `json:"product_id"`
// 	ProductName string  `json:"product_name"`
// 	Quantity    int     `json:"quantity"`
// 	Price       float64 `json:"price"`
// }
type CartItem struct {
	Product  Product `json:"product"`
	Quantity int     `json:"quantity"`
}
type AddToCartRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
	CartID    string `json:"cart_id" validate:"required"`
}
type CreateCartRequest struct {
	UserID    *string `json:"user_id" validate:"omitempty"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
}

type AddToCartResponse struct {
	Message string `json:"message"`
}

type ViewCartResponse struct {
	CartItems     []CartItem `json:"cart_items"`
	SubTotal      float64    `json:"sub_total"`
	Discount      float64    `json:"discount"`
	TotalAmount   float64    `json:"total"`
	EstimatedTax  float64    `json:"estimated_tax"`
	DeliverCharge float64    `json:"delivery_charge"`
}

type UpdateCartItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
}

type UpdateCartItemResponse struct {
	Message string `json:"message"`
}

type RemoveFromCartRequest struct {
	ProductID string `json:"product_id"`
}

type RemoveFromCartResponse struct {
	Message string `json:"message"`
}

type CouponRequest struct {
	DiscountType string `json:"discount_type" validate:"required"`
	Code         string `json:"code" validate:"required"`
	CartID       string `json:"cart_id" validate:"required"`
	LocationID   *int   `json:"location_id,omitempty"`
	RequestType  string `json:"request_type"`
}
type PromotionData struct {
	Type  string
	Value float64
}

type VoucherCode struct {
	Code   string  `json:"code" validate:"required" `
	Amount float64 `json:"amount" validate:"required"`
}
