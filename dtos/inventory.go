package dtos

import "time"

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
	InventoryID       string    `json:"inventory_id"`
	ProductID         string    `json:"product_id"`
	VariantID         string    `json:"variant_id"`
	Quantity          int       `json:"quantity"`
	LowStockThreshold int       `json:"low_stock_threshold"`
	LastUpdated       time.Time `json:"last_updated"`
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
