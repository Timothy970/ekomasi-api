package dtos

import (
	"time"
)

type SocialMedia struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
	Link string `json:"link"`
}

type MenuItem struct {
	Name     string     `json:"name"`
	Link     string     `json:"link"`
	SubMenus []MenuItem `json:"subMenus,omitempty"`
}

type SliderItem struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	ImageURL string `json:"imageUrl"`
	Link     string `json:"link,omitempty"`
}

type FooterSection struct {
	Title string     `json:"title"`
	Links []MenuItem `json:"links"`
}

// type Footer struct {
// 	Sections      []FooterSection `json:"sections"`
// 	CopyrightText string          `json:"copyrightText"`
// 	PaymentIcons  []string        `json:"paymentIcons"`
// }

type HomePageData struct {
	Header   Header       `json:"header"`
	Menu     []MenuItem   `json:"menu"`
	Slider   []SliderItem `json:"slider"`
	Products []Product    `json:"products"`
	Footer   Footer       `json:"footer"`
}

type Header struct {
	TollNumber string        `json:"tollNumber"`
	Socials    []SocialMedia `json:"socials"`
}
type Footer struct {
	CopyrightText  *string
	CompanyAddress *string
	ContactEmail   *string
	PhoneNumber    *string
}

type SocialLink struct {
	Platform  *string
	URL       *string
	IconClass *string
}
type SocialLinkRequest struct {
	ID           int    `json:"id,omitempty"`
	Platform     string `json:"platform" validate:"required"`
	URL          string `json:"url" validate:"required,url"`
	IconClass    string `json:"icon_class" `
	DisplayOrder int    `json:"display_order"`
}

type MenuLink struct {
	Title *string `json:"title"`
	HREF  *string `json:"href"`
}

type Banner struct {
	ID           int
	ImageURL     string
	Text         string
	Heading      string
	ButtonText   string
	ButtonURL    string
	DisplayOrder int
	IsActive     bool
	Type         string
}
type BannerInfo struct {
	Image        *string `form:"banner_image"`
	Text         *string `form:"text,omitempty"`
	Heading      *string `form:"heading,omitempty"`
	ButtonText   *string `form:"button_text,omitempty"`
	ButtonURL    *string `form:"button_url,omitempty"`
	DisplayOrder *int    `form:"display_order,omitempty"`
	IsActive     *bool   `form:"is_active,omitempty"`
}
type UpdateBannerInfo struct {
	ID           int    `json:"image_id"`
	Text         string `json:"text,omitempty"`
	Heading      string `json:"heading,omitempty"`
	ButtonText   string `json:"button_text,omitempty"`
	ButtonURL    string `json:"button_url,omitempty"`
	DisplayOrder int    `json:"display_order,omitempty"`
	IsActive     *bool  `json:"is_active,omitempty"`
}
type CategoryWithProducts struct {
	CategoryID       string                  `json:"category_id"`
	Name             string                  `json:"name"`
	ParentCategoryID *string                 `json:"parent_category_id"`
	Description      string                  `json:"description"`
	Products         []Product               `json:"products,omitempty"`
	Subcategories    []*CategoryWithProducts `json:"subcategories,omitempty"`
}
type Promotion struct {
	ID                   string                  `json:"promotion_id"`
	Name                 string                  `json:"name"`
	StartDate            time.Time               `json:"start_date"`
	EndDate              time.Time               `json:"end_date"`
	IsActive             bool                    `json:"is_active"`
	PromotionType        string                  `json:"promotion_type"`
	Amount               string                  `json:"amount"`
	PromotionDescription string                  `json:"promotion_description"`
	PromotionProducts    []PromotionProductGroup `json:"promotion_products"`
}
type PromotionProductGroup struct {
	ID                 string          `json:"promotion_product_id"`
	PromotionID        string          `json:"promotion_id"`
	ProductID          string          `json:"product_id"`
	DiscountPercentage float64         `json:"discount_percentage"`
	Categories         []CategoryGroup `json:"categories"`
}
type PromotionProduct struct {
	ID                 string  `json:"promotion_product_id"`
	PromotionID        string  `json:"promotion_id"`
	ProductID          string  `json:"product_id"`
	DiscountPercentage float64 `json:"discount_percentage"`
}
type CategoryGroup struct {
	CategoryID       string    `json:"category_id"`
	Name             string    `json:"name"`
	ParentCategoryID *string   `json:"parent_category_id,omitempty"`
	Description      string    `json:"description"`
	Products         []Product `json:"products"`
}
type Category struct {
	ID               string  `json:"category_id"`
	Name             string  `json:"name"`
	ParentCategoryID *string `json:"parent_category_id"`
	Description      string  `json:"description"`
	Image            string  `json:"image_url"`
}

type Product struct {
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
	Weight           *float64          `json:"weight"`
	WeightLimit      *float64          `json:"weight_limit"`
	Dimensions       *string           `json:"dimensions"`
	Manufacturer     *string           `json:"manufacturer"`
	Discount         *float64          `json:"discount"`
	DiscountType     *string           `json:"discount_type"`
	Tax              *ProductTax       `json:"tax"`
	LowStockAlert    int               `json:"low_stock_quantity_warning"`
	//for bundles, it will contain the products in the bundle
	BundleProducts    []Product          `json:"products,omitempty"`
	IsProductFeatured bool               `json:"is_featured"`
	BundleQuantity    int                `json:"bundle_quantity,omitempty"`
	Barcode           *string            `json:"barcode,omitempty"`
	VariantSelection  []VariantSelection `json:"variant_selection,omitempty"`
}

type DiscountType struct {
	ID          string `json:"discount_type_id"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	Value       string `json:"value" validate:"required"`
}

type ProductTax struct {
	ID    string   `json:"tax_id"`
	Name  *string  `json:"tax_name"`
	Value *float64 `json:"tax_value"`
}

type DealProduct struct {
	ID               string            `json:"product_id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	SKU              string            `json:"sku"`
	Tag              *string           `json:"tag"`
	Price            float64           `json:"price"`
	CategoryID       string            `json:"category_id"`
	CategoryName     string            `json:"category_name"`
	StockQuantity    int               `json:"stock_quantity"`
	MaxStockQuantity int               `json:"max_stock_quantiy"`
	SearchVector     string            `json:"search_vector"`
	IsInTodaysDeals  bool              `json:"in_today_deal"`
	CreatedBy        string            `json:"created_by"`
	CreatedAt        time.Time         `json:"created_at"`
	LastUpdated      time.Time         `json:"last_updated"`
	Images           []Image           `json:"urls,omitempty"`
	ProductVariants  []ProductVariants `json:"product_variants"`
	Warranty         *ProductWarranty  `json:"warranty"`
	Discount         *float64          `json:"discount,omitempty"`
	DiscountType     *string           `json:"discount_type,omitempty"`
	Weight           *float64          `json:"weight"`
	WeightLimit      *float64          `json:"weight_limit"`
	Dimensions       *string           `json:"dimensions"`
	Manufacturer     *string           `json:"manufacturer"`
	Tax              *ProductTax       `json:"tax"`
}
type FeaturedProduct struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	Product   Product   `json:"product"`
	CreatedAt time.Time `json:"created_at"`
}

type PromotionProductDetail struct {
	PromotionProduct
	Product Product `json:"product"`
}
type Image struct {
	ImageID   string `json:"image_id"`
	URL       string `json:"url"`
	IsPrimary bool   `json:"is_primary"`
	Type      string `json:"type"`
}
type DeliveryFeedback struct {
	FeedbackID string `json:"feedback_id"`
	DeliveryID string `json:"delivery_id"`
	Score      int    `json:"score"`
	Details    string `json:"details"`
}

type PromotionType struct {
	ID          string `json:"promotion_type_id"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	Value       string `json:"value" validate:"required"`
}
type NewPromotion struct {
	Name            string    `json:"name" validate:"required"`
	PromotionIdType int       `json:"promotion_type_id" validate:"required"`
	StartDate       time.Time `json:"start_date" validate:"required"`
	EndDate         time.Time `json:"end_date" validate:"required"`
}
type DeletePromotion struct {
	PromotionID string `json:"promotion_id"`
}
type EditPromotion struct {
	PromotionID     string    `json:"promotion_id"`
	Name            string    `json:"name"`
	PromotionIdType int       `json:"promotion_type_id"`
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	IsActive        *bool     `json:"is_active"`
}
type AttachProductToPromotion struct {
	PromotionID string   `json:"promotion_id"`
	ProductIDs  []string `json:"product_ids"`
}
type Blog struct {
	BlogID      string  `json:"blog_id"`
	Title       string  `json:"title" validate:"required"`
	Content     string  `json:"content" validate:"required"`
	AuthorID    string  `json:"author_id"`
	Author      *string `json:"author" validate:"required"`
	PublishedAt string  `json:"published_at"`
	IsPublished bool    `json:"is_published"`
	ImageURL    *string `json:"image_url"`
}
type UpdateBlog struct {
	// BlogID      string `json:"blog_id"`
	// Title       string `json:"title" validate:"required"`
	// Content     string `json:"content" validate:"required"`
	// AuthorID    string `json:"author_id" validate:"required"`
	// PublishedAt string `json:"published_at"`
	IsPublished bool `json:"is_published" validate:"required"`
}

type MenuLinkRequest struct {
	Title        string  `json:"title" validate:"required"`
	URL          string  `json:"url" validate:"required"`
	DisplayOrder int     `json:"display_order" validate:"required"`
	ParentID     *string `json:"parent_id"`
}
type BlogRequest struct {
	BlogID         string    `json:"blog_id"`
	AuthorID       string    `json:"author_id"`
	IsPublished    bool      `json:"is_published"`
	BannerImageUrl *string   `json:"banner_image_url"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
	PublishedAt    time.Time `json:"published_at"`
	Status         *string   `json:"status"`
	// Author      struct {
	// 	Name   string  `json:"name" validate:"required"`
	// 	Avatar *string `json:"avatar"`
	// } `json:"author"`
	Author          map[string]any `json:"author"`
	ReadTimeMinutes int            `json:"read_time_minutes" validate:"required"`
	Title           string         `json:"title" validate:"required"`
	Description     *string        `json:"description"`

	Sections []struct {
		Position   int         `json:"position" validate:"required"`
		Banner     *BlogBanner `json:"banner"` // Pointer allows null values
		Paragraphs []Paragraph `json:"paragraphs" dive:"required"`
		Images     []BlogImage `json:"images"`
	} `json:"sections"`

	Tags *[]string `json:"tags"`
}

// type BlogBanner struct {
// 	ImageURL string `json:"image_url"`
// 	Alt      string `json:"alt"`
// 	//other fields can be added as needed

// }
type BlogBanner map[string]any

//	type Paragraph struct {
//		Text string `json:"text"`
//	}
type Paragraph map[string]any

//	type BlogImage struct {
//		ImageURL string `json:"image_url"`
//		Alt      string `json:"alt"`
//	}
type BlogImage map[string]any
