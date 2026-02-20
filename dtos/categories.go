package dtos

type CreateCategory struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description" validate:"required"`
	ParentID    *string `json:"parent_id"`
	Image       string  `json:"image_url" validate:"required"`
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
	Image            *string        `json:"image_url"`
	Subcategories    []CategoryData `json:"subcategories,omitempty"`
	Products         []ProductData  `json:"products,omitempty"`
}
type ProductData struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	URL   *string `json:"url,omitempty"`
}

type AdminCategoryData struct {
	ID            string  `json:"id"`
	ParentID      *string `json:"parent_id,omitempty"`
	Image         *string `json:"image_url,omitempty"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Items         int     `json:"items"`
	Subcategories int     `json:"subcategories"`
	Description   string  `json:"description"`
}

type CategoryWithSubCategories struct {
	Category    string        `json:"name"`
	CategoryID  string        `json:"category_id"`
	SubCategory []SubCategory `json:"subcategories"`
}

type SubCategory struct {
	Category   string `json:"name"`
	CategoryID string `json:"category_id"`
}
