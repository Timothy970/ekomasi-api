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
	UserID *string `json:"user_id"`
}

type AddToCartResponse struct {
	Message string `json:"message"`
}

type ViewCartResponse struct {
	CartItems    []CartItem `json:"cart_items"`
	Total        float64    `json:"total"`
	Discount     float64    `json:"discount"`
	Final        float64    `json:"final"`
	EstimatedTax float64    `json:"estimated_tax"`
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
	Type   string `json:"type" validate:"required"`
	Code   string `json:"code" validate:"required"`
	CartID string `json:"cart_id" validate:"required"`
}
type PromotionData struct {
	Type  string
	Value float64
}
