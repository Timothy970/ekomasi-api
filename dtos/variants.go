package dtos

type VariantRequest struct {
	VariantType string  `json:"variant_type" validate:"required,oneof=color size material brand gender age_group availability condition pattern style season fit capacity length width sales_promotion feature"`
	Name        string  `json:"name" validate:"required"`
	HexCode     *string `json:"hex_code,omitempty"` // only relevant for colors
}

type VariantResponse struct {
	VariantID   string  `json:"variant_id"`
	VariantType string  `json:"variant_type"`
	Name        string  `json:"name"`
	HexCode     *string `json:"hex_code,omitempty"`
}
type GroupedVariants struct {
	VariantType string            `json:"variant_type"`
	Variants    []VariantResponse `json:"variants"`
}

type ProductVariantRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	// VariantID       string  `json:"variant_id" validate:"required,uuid4"`
	AdditionalPrice *float64 `json:"additional_price"`
	StockQuantity   *int     `json:"stock_quantity"`
}
type Variant struct {
	VariantID   string `json:"variant_id"`
	VariantType string `json:"variant_type"`
	Name        string `json:"name"`
}
type ProductVariantResponse struct {
	VariantID       string  `json:"variant_id"`
	ProductID       string  `json:"product_id"`
	AdditionalPrice float64 `json:"additional_price"`
	StockQuantity   int     `json:"stock_quantity"`
}
type VariantWithProducts struct {
	VariantID       string    `json:"variant_id"`
	VariantType     string    `json:"variant_type"`
	Name            string    `json:"name"`
	HexCode         *string   `json:"hex_code,omitempty"`
	AdditionalPrice float64   `json:"additional_price"`
	StockQuantity   int       `json:"stock_quantity"`
	Products        []Product `json:"products"`
}
type ProductVariants struct {
	VariantID       string  `json:"variant_id"`
	VariantType     string  `json:"variant_type"`
	Name            string  `json:"name"`
	HexCode         *string `json:"hex_code,omitempty"`
	AdditionalPrice float64 `json:"additional_price"`
	StockQuantity   int     `json:"stock_quantity"`
}
