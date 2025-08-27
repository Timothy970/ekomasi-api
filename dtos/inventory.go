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
