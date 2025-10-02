package dtos

type CreateDeal struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Discount    *float64 `json:"discount"`
}
type Deal struct {
	DealID      string   `json:"deal_id"`
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Discount    *float64 `json:"discount"`
}

type DealWithProducts struct {
	DealID      string    `json:"deal_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Discount    *float64  `json:"discount"`
	Products    []Product `json:"products"`
}

type ProductDeal struct {
	ID        string `json:"id" validate:"required"`
	ProductID string `json:"product_id" validate:"required"`
}
