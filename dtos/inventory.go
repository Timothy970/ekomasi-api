package dtos

import (
	"time"
)

type InventoryDTO struct {
	InventoryID       string    `json:"inventory_id"`
	ProductID         string    `json:"product_id"`
	VariantID         string    `json:"variant_id"`
	Quantity          int       `json:"quantity"`
	LowStockThreshold int       `json:"low_stock_threshold"`
	LastUpdated       time.Time `json:"last_updated"`
}

type CreateInventoryRequest struct {
	ProductID         string `json:"product_id" validate:"required"`
	VariantID         string `json:"variant_id" validate:"required"`
	Quantity          int    `json:"quantity" validate:"gte=0"`
	LowStockThreshold int    `json:"low_stock_threshold" validate:"gte=0"`
}

type UpdateInventoryRequest struct {
	Quantity          *int `json:"quantity,omitempty"`
	LowStockThreshold *int `json:"low_stock_threshold,omitempty"`
}

type InventoryListResponse struct {
	Meta        PaginationMeta `json:"pagination"`
	Inventories []Inventory    `json:"inventories"`
}

type Inventory struct {
	InventoryID       string   `json:"inventory_id"`
	ProductID         string   `json:"product_id"`
	VariantID         *string  `json:"variant_id"`
	BatchNumber       *string  `json:"batch_number"`
	Quantity          int      `json:"inventory_quantity"`
	LowStockThreshold int      `json:"low_stock_threshold"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	SKU               string   `json:"sku"`
	Tag               *string  `json:"tag"`
	Price             float64  `json:"price"`
	CategoryID        string   `json:"category_id"`
	CategoryName      string   `json:"category_name"`
	StockQuantity     int      `json:"stock_quantity"`
	SearchVector      string   `json:"search_vector"`
	Images            []Image  `json:"urls,omitempty"`
	SupplierInfo      Supplier `json:"supplier_info"`
	ManufacturingDate *string  `json:"manufacturing_date"`
	ExpiryDate        *string  `json:"expiry_date"`
	Warranty          *string  `json:"warranty"`
	PlacedOn          string   `json:"placed_on"`
	BuyingPrice       *float64 `json:"buying_price"`
	StoreID           *string  `json:"store_id"`
	Store             *string  `json:"store"`
	UnitCost          *float64 `json:"unit_cost"`
}

type SingleInventory struct {
	InventoryID       string   `json:"inventory_id"`
	ProductID         string   `json:"product_id"`
	VariantID         *string  `json:"variant_id"`
	BatchNumber       *string  `json:"batch_number"`
	Quantity          int      `json:"inventory_quantity"`
	LowStockThreshold int      `json:"low_stock_threshold"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	SKU               string   `json:"sku"`
	Tag               *string  `json:"tag"`
	Price             float64  `json:"price"`
	CategoryID        string   `json:"category_id"`
	CategoryName      string   `json:"category_name"`
	StockQuantity     int      `json:"stock_quantity"`
	SearchVector      string   `json:"search_vector"`
	Images            []Image  `json:"urls,omitempty"`
	SupplierInfo      Supplier `json:"supplier_info"`
	ManufacturingDate *string  `json:"manufacturing_date"`
	ExpiryDate        *string  `json:"expiry_date"`
	Warranty          *string  `json:"warranty"`
	PlacedOn          string   `json:"placed_on"`
	BuyingPrice       *float64 `json:"buying_price"`
	StoreID           *string  `json:"store_id"`
	Store             *string  `json:"store"`
	UnitCost          *float64 `json:"unit_cost"`

	BatchImages      *[]string  `json:"batch_images"`
	InspectionImages *[]string  `json:"inspection_images"`
	InspectionDate   *time.Time `json:"inspection_date"`
	Inspector        *Users     `json:"inspector"`
	InspectionNotes  *string    `json:"inspection_notes"`
	ConditionID      *string    `json:"condition_id"`
	HandlingNotes    *string    `json:"handling_notes"`
}

type InventoryTurnoverRequest struct {
	GroupBy string // week, month, quarter, year
	Start   time.Time
	End     time.Time
}

type InventoryTurnoverResponse struct {
	Period struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
		Type  string    `json:"type"`
	} `json:"period"`
	Data []InventoryTurnoverItem `json:"data"`
}

type InventoryTurnoverItem struct {
	ProductID     *string `json:"product_id,omitempty"`
	CategoryID    *string `json:"category_id,omitempty"`
	AvgInventory  float64 `json:"average_inventory"`
	COGS          float64 `json:"cogs"`
	TurnoverRatio float64 `json:"turnover_ratio"`
}

type StockEntryRequest struct {
	ProductID         string      `json:"product_id" validate:"required"`
	BatchNumber       string      `json:"batch_number" validate:"required"`
	BatchImages       *[]string   `json:"batch_images"`
	ExpiryDate        string      `json:"expiry_date" validate:"required"`
	ManufacturingDate string      `json:"manufacturing_date" validate:"required"`
	InspectionDate    string      `json:"inspection_date" validate:"required"`
	InspectorID       string      `json:"inspector_id" validate:"required"`
	InspectionNotes   string      `json:"inspection_notes"`
	InspectionImage   *[]string   `json:"inspection_images"`
	QuantityReceived  int         `json:"quantity_received" validate:"gte=1"`
	MinimumStockLevel int         `json:"minimum_stock_level" validate:"gte=0"`
	StoreQuantity     []StoreInfo `json:"store_quantity" validate:"required,dive"`
	SupplierID        *string     `json:"supplier_id"`
	BuyingPrice       float64     `json:"buying_price" validate:"required,gte=0"`
	HandlingNotes     string      `json:"handling_notes"`
	SellingPrice      float64     `json:"selling_price" validate:"required,gte=0"`
}

type StoreInfo struct {
	StoreID  string `json:"store_id" validate:"required"`
	Quantity int    `json:"quantity" validate:"required,gte=0"`
}

type Batch struct {
	InventoryID       string   `json:"inventory_id"`
	BatchNumber       string   `json:"batch_number"`
	Images            []string `json:"images,omitempty"`
	ExpiryDate        string   `json:"expiry_date"`
	ManufacturingDate string   `json:"manufacturing_date"`
}

type Inspection struct {
	BatchID         string   `json:"batch_id"`
	InspectionDate  string   `json:"inspection_date"`
	InspectorID     string   `json:"inspector_id"`
	InspectionNotes string   `json:"inspection_notes"`
	Images          []string `json:"images,omitempty"`
}

type InventoryCondition struct {
	BatchID       string `json:"batch_id"`
	HandlingNotes string `json:"handling_notes"`
}

type InventoryTracking struct {
	ProductID         string  `json:"product_id"`
	Quantity          int     `json:"quantity"`
	LowStockThreshold int     `json:"low_stock_threshold"`
	StoreID           string  `json:"store_id"`
	SupplierID        *string `json:"supplier_id"`
}

type InventoryStockSummary struct {
	TotalStock       int `json:"total_stock"`
	MinimumThreshold int `json:"minimum_threshold"`
	TotalSales       int `json:"total_sales"`
}
type InventoryStockHistory struct {
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Date        time.Time `json:"date"`
}
