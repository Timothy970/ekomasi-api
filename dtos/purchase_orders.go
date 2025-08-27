package dtos

import "time"

type CreatePurchaseOrderRequest struct {
	SupplierID string  `json:"supplier_id" validate:"required"`
	TotalCost  float64 `json:"total_cost" validate:"required,gt=0"`
}

type UpdatePurchaseOrderRequest struct {
	Status    string  `json:"status" validate:"required,oneof=pending approved cancelled"`
	TotalCost float64 `json:"total_cost" validate:"omitempty,gt=0"`
}

type PurchaseOrderResponse struct {
	PoID       string     `json:"po_id"`
	SupplierID string     `json:"supplier_id"`
	Status     string     `json:"status"`
	TotalCost  float64    `json:"total_cost"`
	CreatedAt  time.Time  `json:"created_at"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
}

type PaginatedPurchaseOrdersResponse struct {
	Data []PurchaseOrderResponse `json:"data"`
	Meta PaginationMeta          `json:"pagination"`
}

type PurchaseOrder struct {
	PoID       string     `db:"po_id"`
	SupplierID string     `db:"supplier_id"`
	Status     string     `db:"status"`
	TotalCost  float64    `db:"total_cost"`
	CreatedAt  time.Time  `db:"created_at"`
	ApprovedAt *time.Time `db:"approved_at"`
}

type PurchaseOrderItem struct {
	PoID      string  `json:"po_id" validate:"required"`
	ProductID string  `json:"product_id" validate:"required"`
	VariantID string  `json:"variant_id" validate:"required"`
	Quantity  int     `json:"quantity" validate:"required"`
	UnitCost  float64 `json:"unit_cost" validate:"required"`
}
