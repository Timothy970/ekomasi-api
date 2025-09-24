package dtos

import "time"

type CreateProduct struct {
	ID            string  `json:"product_id,omitempty"`
	Name          string  `json:"name" validate:"required"`
	Description   string  `json:"description" validate:"required"`
	SKU           string  `json:"sku" validate:"required"`
	Price         float64 `json:"price" validate:"required,gt=0"`
	CategoryID    string  `json:"category_id" validate:"required"`
	StockQuantity int     `json:"stock_quantity" validate:"required,gte=0"`
	SearchVector  string  `json:"search_vector" validate:"required"`
}

// AddToCartWithVariantsRequest represents the request to add item with specific variants to cart
type AddToCartWithVariantsRequest struct {
	UserID    int `json:"user_id"`
	ProductID int `json:"product_id"`
	VariantID int `json:"variant_id"`
	Quantity  int `json:"quantity"`
}

// AddToCartWithVariantsResponse defines the response body when adding to cart with variants
type AddToCartWithVariantsResponse struct {
	Message string `json:"message"`
}

// CartResponse represents the response for the view cart endpoint
type CartResponse struct {
	Items    []CartItem `json:"items"`
	Subtotal float64    `json:"subtotal"`
}

// CartUpdateRequest represents the request body for updating the cart
type CartUpdateRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// WishlistRequest struct represents the request body for creating or updating a wishlist
type WishlistRequest struct {
	Name       string   `json:"name"`
	ProductIDs []string `json:"product_ids"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}
type GetBundleRequest struct {
	BundleID          string    `json:"bundle_id"`
	BundleName        string    `json:"bundle_name"`
	BundleDescription string    `json:"bundle_description"`
	BundlePrice       float64   `json:"bundle_price"`
	Products          []Product `json:"products"`
}
type Bundle struct {
	Name        string  `json:"bundle_name"`
	Description string  `json:"bundle_description"`
	Price       float64 `json:"bundle_price"`
}
type UpdateBundle struct {
	ID          string  `json:"bundle_id" validate:"required"`
	Name        string  `json:"bundle_name" validate:"required"`
	Description string  `json:"bundle_description" validate:"required"`
	Price       float64 `json:"bundle_price" validate:"required"`
}
type DeleteBundle struct {
	ID string `json:"bundle_id" validate:"required"`
}
type AddProductsToBundle struct {
	ID         string   `json:"bundle_id" validate:"required"`
	ProductIDs []string `json:"product_ids" validate:"required"`
}

type Variants struct {
	ID              string  `json:"variant_id"`
	ProductID       string  `json:"product_id"`
	Name            string  `json:"name"`
	AdditionalPrice float64 `json:"additional_price"`
	StockQuantity   float64 `json:"stock_quantity"`
}

type PaginationMeta struct {
	Page       int  `json:"page"`
	Size       int  `json:"size"`
	TotalItems int  `json:"total_items"`
	TotalPages int  `json:"total_pages"`
	HasPrev    bool `json:"has_prev"`
	HasNext    bool `json:"has_next"`
}

type SubcategoryProducts struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	ImageURL               string    `json:"image_url"`
	ParentID               string    `json:"parent_id"`
	ParentCategoryName     string    `json:"parent_category_name"`
	ParentCategoryImageURL string    `json:"parent_category_image_url"`
	Products               []Product `json:"products"`
}

type SubcategoryResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ParentID    string `json:"parent_id"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
}

type CategoryResponse struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	ParentID      *string               `json:"parent_id"`
	ImageURL      string                `json:"image_url"`
	Description   string                `json:"description"`
	Subcategories []SubcategoryResponse `json:"subcategories"`
	Products      []CategoryProduct     `json:"products"`
}

type PaginatedCategoriesResponse struct {
	Categories []CategoryResponse `json:"categories"`
	Meta       PaginationMeta     `json:"pagination"`
}
type CategoryProduct struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	SKU             string            `json:"sku"`
	Price           float64           `json:"price"`
	CategoryID      string            `json:"parent_category_id"` // parent category
	SubcategoryID   string            `json:"subcategory_id"`     // child category
	StockQuantity   int               `json:"stock_quantity"`
	SearchVector    string            `json:"search_vector"`
	CreatedAt       time.Time         `json:"created_at"`
	LastUpdated     time.Time         `json:"last_updated"`
	Images          []Image           `json:"urls"`
	ProductVariants []ProductVariants `json:"products_variants"`
}

// SearchParams represents the search parameters
type SearchParams struct {
	Q            string // Search query for category name OR product name
	CategoryName string
	ProductName  string
	Variants     []VariantFilter // Changed from single variant to slice
	SortBy       string
	Page         int
	Limit        int
}

type VariantFilter struct {
	Type  string
	Value string
}

type ProductFeature struct {
	ID            string `json:"feature_id"`
	ProductID     string `json:"product_id"`
	Header        string `json:"header" validate:"required"`
	Image         string `json:"image" validate:"required"`
	Description   string `json:"description" validate:"required"`
	ImagePosition string `json:"image-position" validate:"required"`
}
type UpdateProductFeature struct {
	ID            string `json:"feature_id"`
	ProductID     string `json:"product_id"`
	Header        string `json:"header" validate:"required"`
	Image         string `json:"image"`
	Description   string `json:"description" validate:"required"`
	ImagePosition string `json:"image-position" validate:"required"`
}
