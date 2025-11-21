package dtos

import "time"

type CreateProduct struct {
	ID            string   `json:"product_id,omitempty"`
	Name          string   `json:"name" validate:"required"`
	Description   string   `json:"description" validate:"required"`
	SKU           string   `json:"sku" validate:"required"`
	Price         float64  `json:"price" validate:"required,gt=0"`
	CategoryID    string   `json:"category_id" validate:"required"`
	StockQuantity int      `json:"stock_quantity" validate:"required,gte=0"`
	SearchVector  string   `json:"search_vector" validate:"required"`
	Tag           *string  `json:"tag,omitempty"`
	LowStockAlert int      `json:"low_stock_quantity_warning" validate:"gte=0"`
	SellWhenOOS   *bool    `json:"sell_when_out_of_stock"`
	ShowStock     *bool    `json:"show_stock_quantity"`
	BuyingPrice   *float64 `json:"buying_price"`
	Details       []string `json:"details,omitempty"`
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
	BundleImage       *string   `json:"bundle_image"`
	CompareAtPrice    *float64  `json:"compare_at_price"`
	KeepSelling       *bool     `json:"keep_selling_when_out_of_stock"`
	Products          []Product `json:"products"`
}
type Bundle struct {
	Name           string           `json:"bundle_name" validate:"required"`
	Description    string           `json:"bundle_description" validate:"required"`
	Price          float64          `json:"bundle_price" validate:"required,min=0"`
	Image          string           `json:"bundle_image" validate:"required"`
	Products       []BundleProducts `json:"products" validate:"required"`
	KeepSelling    *bool            `json:"keep_selling"`
	CompareAtPrice *float64         `json:"compare_at_price"`
}
type BundleProducts struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,gte=1"`
}
type UpdateBundle struct {
	Name           string   `json:"bundle_name"`
	Description    string   `json:"bundle_description"`
	Price          float64  `json:"bundle_price"`
	Image          *string  `json:"bundle_image"`
	KeepSelling    *bool    `json:"keep_selling"`
	CompareAtPrice *float64 `json:"compare_at_price"`
	ID             string   `json:"bundle_id" validate:"required"`
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
	Tag             *string           `json:"tag"`
	Images          []Image           `json:"urls"`
	ProductVariants []ProductVariants `json:"products_variants"`
	//only visible when user is authenticated
	InWishlist *bool            `json:"liked_by_user,omitempty"`
	Details    []string         `json:"details,omitempty"`
	Features   []ProductFeature `json:"features,omitempty"`
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
	SKU          string
	Tag          string
	MaxPrice     float64
	MinPrice     float64
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
type ProductSpecification struct {
	ProductID        string   `json:"product_id" validate:"required"`
	Age              []string `json:"age"`   //ids of the age variants
	Brand            string   `json:"brand"` //brand id for the variant brand
	CategoryID       string   `json:"category_id"`
	Color            []string `json:"color"` //color variants ids
	Dimensions       string   `json:"dimensions"`
	DiscountType     string   `json:"discount_type"` // type id
	ExpiryDate       string   `json:"expiry_date"`
	ManufacturerDate string   `json:"manufacture_date"`
	Manufacturer     string   `json:"manufacturer"`
	Material         []string `json:"material"` //material variant ids
	Size             []string `json:"size"`     //size variant ids
	Tax              string   `json:"tax"`      //charge id
	WarrantyType     string   `json:"warranty_type"`
	WarrantyPeriod   int      `json:"warranty_period"` //in months
	Weight           int      `json:"weight"`
	WeightLimit      int      `json:"weight_limit"`
}
type ExpensiveCheapProduct struct {
	CheapestProduct  Product `json:"cheapest_product"`
	ExpensiveProduct Product `json:"expensive_product"`
}
type ProductSpecs struct {
	ProductID    string `json:"product_id"`
	Weight       int    `json:"weight"`
	WeightLimit  int    `json:"weight_limit"`
	Dimensions   string `json:"dimensions"`
	Manufacturer string `json:"manufacturer"`
}

type BulkUploadProduct struct {
	ProductID               string  `json:"product_id"`
	Name                    string  `json:"name"`
	Description             string  `json:"description"`
	SKU                     string  `json:"sku"`
	Price                   float64 `json:"price"`
	CategoryID              string  `json:"category_id"`
	StockQuantity           int     `json:"stock_quantity"`
	SearchVector            string  `json:"search_vector"`
	Tag                     string  `json:"tag"`
	LowStockQuantityWarning int     `json:"low_stock_quantity_warning"`
	SellWhenOutOfStock      bool    `json:"sell_when_out_of_stock"`
	ShowStockQuantity       bool    `json:"show_stock_quantity"`
	CreatedByID             string  `json:"created_by_id"`
	BuyingPrice             float64 `json:"buying_price"`
	Image                   string  `json:"image"`
}
type VoucherDesign struct {
	DesignID   string  `json:"design_id"`
	URL        string  `json:"url" validate:"required"`
	Created_At string  `json:"created_at"`
	Name       *string `json:"name"  validate:"required"`
	Status     *string `json:"status" validate:"required"`
}
type LowStockEmailData struct {
	StoreName string
	AlertDate string
	Products  []Product
}
