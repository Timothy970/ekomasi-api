package dtos

type CreateCategory struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description" validate:"required"`
	ParentID    *string `json:"parent_id"`
}
type UpdateCategory struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
}
type CreateVariant struct {
	// ProductID       string  `json:"product_id" validate:"required"`
	Name            string  `json:"name" validate:"required"`
	AdditionalPrice float64 `json:"additional_price" validate:"required,gt=0"`
	StockQuantity   float64 `json:"stock_quantity" validate:"required,gt=0"`
}
type UpdateVariant struct {
	VariantID       string  `json:"variant_id" validate:"required"`
	ProductID       string  `json:"product_id" validate:"required"`
	Name            string  `json:"name" validate:"required"`
	AdditionalPrice float64 `json:"additional_price" validate:"required, gt=0"`
	StockQuantity   float64 `json:"stock_quantity" validate:"required, gt=0"`
}

type CategoryData struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	ParentCategoryID *string        `json:"parent_category_id"`
	Description      string         `json:"description"`
	Subcategories    []CategoryData `json:"subcategories,omitempty"`
	Products         []ProductData  `json:"products,omitempty"`
}
type ProductData struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	URL   *string `json:"url,omitempty"`
}
