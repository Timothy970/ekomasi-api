package dtos

import "time"

type CreateProduct struct {
	ID            string   `json:"product_id,omitempty"`
	Name          string   `json:"name" validate:"required"`
	Description   string   `json:"description" validate:"required"`
	SKU           string   `json:"sku" validate:"required"`
	Price         *float64 `json:"price"`
	CategoryID    string   `json:"category_id" validate:"required"`
	StockQuantity int      `json:"stock_quantity" validate:"gte=0"`
	SearchVector  string   `json:"search_vector" validate:"required"`
	Tag           *string  `json:"tag,omitempty"`
	LowStockAlert int      `json:"low_stock_quantity_warning" validate:"gte=0"`
	BuyingPrice   *float64 `json:"buying_price"`
	Details       []string `json:"details,omitempty"`
	Barcode       string   `json:"barcode"`
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
	ID               string            `json:"product_id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	SKU              string            `json:"sku"`
	Tag              *string           `json:"tag"`
	Price            float64           `json:"price"`
	CategoryID       string            `json:"category_id"`
	CategoryName     string            `json:"category_name"`
	StockQuantity    int               `json:"stock_quantity"`
	MaxStockQuantity int               `json:"max_stock_quantity"`
	SearchVector     string            `json:"search_vector"`
	IsInTodaysDeals  bool              `json:"in_today_deal"`
	CreatedBy        string            `json:"created_by"`
	CreatedAt        time.Time         `json:"created_at"`
	LastUpdated      time.Time         `json:"last_updated"`
	Images           []Image           `json:"urls,omitempty"`
	ProductVariants  []ProductVariants `json:"product_variants"`
	Warranty         *ProductWarranty  `json:"warranty"`
	InWishlist       *bool             `json:"liked_by_user,omitempty"`
	Details          []string          `json:"details,omitempty"`
	Features         []ProductFeature  `json:"features,omitempty"`
	Products         []Product         `json:"products"`
	BuyingPrice      *float64          `json:"buying_price,omitempty"`
}

type Bundle struct {
	Name           string           `json:"bundle_name" validate:"required"`
	Description    string           `json:"bundle_description" validate:"required"`
	Price          float64          `json:"bundle_price" validate:"required,min=0"`
	Image          string           `json:"bundle_image" validate:"required"`
	Products       []BundleProducts `json:"products" validate:"required"`
	CompareAtPrice *float64         `json:"compare_at_price"`
	StockQuantity  int              `json:"stock_quantity" validate:"required,gte=0"`
}

type BundleProducts struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,gte=1"`
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
	Warranty        *ProductWarranty  `json:"warranty"`
	//only visible when user is authenticated
	InWishlist       *bool              `json:"liked_by_user,omitempty"`
	Details          []string           `json:"details,omitempty"`
	Features         []ProductFeature   `json:"features,omitempty"`
	Discount         *float64           `json:"discount"`
	DiscountType     *string            `json:"discount_type"`
	Weight           *string            `json:"weight"`
	WeightLimit      *string            `json:"weight_limit"`
	Dimensions       *string            `json:"dimensions"`
	Manufacturer     *string            `json:"manufacturer"`
	Tax              *ProductTax        `json:"tax"`
	VariantSelection []VariantSelection `json:"variant_selection,omitempty"`
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
	StartDate    string
	EndDate      string
}

type VariantFilter struct {
	Type  string
	Value string
}

type ProductFeature struct {
	ID                    string     `json:"feature_id"`
	ProductID             string     `json:"product_id"`
	Header                string     `json:"header" validate:"required"`
	Image                 *string    `json:"image"`
	Description           string     `json:"description" validate:"required"`
	ImagePosition         string     `json:"image-position" validate:"required"`
	ProductSpecifications *[]string  `json:"product_specifications"`
	TopSection            *[]Section `json:"top_section"`
	DesignType            *string    `json:"design_type" validate:"required"`
	Images                *[]string  `json:"images"`
}

type Section struct {
	Title       string `json:"title"`
	Description string `json:"description"`
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
	ProductID         string             `json:"product_id" validate:"required"`
	Brand             string             `json:"brand"` //brand id for the variant brand
	CategoryID        string             `json:"category_id" validate:"required"`
	Dimensions        string             `json:"dimensions"`
	Manufacturer      string             `json:"manufacturer"`
	WarrantyType      string             `json:"warranty_type"`                       //warranty type id
	WarrantyPeriod    int                `json:"warranty_period" validate:"required"` //in months
	Weight            float64            `json:"weight"`
	WeightLimit       float64            `json:"weight_limit"`
	VariantSelections []VariantSelection `json:"variant_selections" validate:"omitempty,dive"`
}

type VariantSelection struct {
	VariantIDs      []string `json:"variant_ids" validate:"required,min=1"`
	Name            string   `json:"name" validate:"required"`
	SKU             string   `json:"sku" validate:"required"`
	AdditionalPrice float64  `json:"additional_price"`
	StockQuantity   int      `json:"stock_quantity"`
}

type ExpensiveCheapProduct struct {
	CheapestProduct  Product `json:"cheapest_product"`
	ExpensiveProduct Product `json:"expensive_product"`
}
type ProductSpecs struct {
	ProductID    string  `json:"product_id"`
	Weight       float64 `json:"weight"`
	WeightLimit  float64 `json:"weight_limit"`
	Dimensions   string  `json:"dimensions"`
	Manufacturer string  `json:"manufacturer"`
}

type BulkUploadProduct struct {
	ProductID               string    `json:"product_id"`
	Name                    string    `json:"name"`
	Description             string    `json:"description"`
	SKU                     string    `json:"sku"`
	Price                   float64   `json:"price"`
	CategoryID              string    `json:"sub_category_id"`
	StockQuantity           int       `json:"stock_quantity"`
	SearchVector            string    `json:"search_vector"`
	Tag                     *string   `json:"tag"`
	LowStockQuantityWarning int       `json:"low_stock_quantity_warning"`
	CreatedByID             string    `json:"created_by_id"`
	BuyingPrice             float64   `json:"buying_price"`
	Weight                  *float64  `json:"weight"`
	WeightLimit             *float64  `json:"weight_limit"`
	Dimensions              *string   `json:"dimensions"`
	Brand                   *string   `json:"brand"`
	Manufacturer            *string   `json:"manufacturer"`
	WarrantyPeriod          *int      `json:"warranty_period"`
	CreatedAt               time.Time `json:"created_at"`
	Barcode                 string    `json:"barcode"`
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

type PublishBulkProduct struct {
	ProductID      string    `json:"product_id" validate:"required"`
	Images         []Image   `json:"images" validate:"required,dive"`
	ProductDetails *[]string `json:"product_details"`
	VideoLink      *string   `json:"video_link"`
}

type Combination struct {
	ProductID       string  `json:"product_id"`
	Name            string  `json:"name"`
	SKU             string  `json:"sku"`
	AdditionalPrice float64 `json:"additional_price"`
}
