package dtos

import (
	"database/sql"
	"time"
)

type StockTransferDTO struct {
	TransferID      string    `json:"transfer_id"`
	ProductID       string    `json:"product_id" validate:"required"`
	VariantID       *string   `json:"variant_id,omitempty"`
	FromWarehouseID string    `json:"from_warehouse_id" validate:"required"`
	ToWarehouseID   string    `json:"to_warehouse_id" validate:"required"`
	Quantity        int       `json:"quantity" validate:"required"`
	TransferDate    time.Time `json:"transfer_date"`
	TransferDetails string    `json:"transfer_details,omitempty"`
}
type StockTransferUpdateDTO struct {
	TransferID      string    `json:"transfer_id"`
	ProductID       string    `json:"product_id" validate:"required"`
	VariantID       *string   `json:"variant_id,omitempty"`
	FromWarehouseID string    `json:"from_warehouse_id" validate:"required"`
	ToWarehouseID   string    `json:"to_warehouse_id" validate:"required"`
	Quantity        int       `json:"quantity" validate:"required"`
	TransferDate    time.Time `json:"transfer_date"`
	TransferDetails string    `json:"transfer_details,omitempty"`
	Status          string    `json:"status" validate:"required"`
}

type StockTransferListResponse struct {
	Meta           PaginationMeta     `json:"pagination"`
	StockTransfers []StockTransferDTO `json:"stock_transfers"`
}

type StockTransfer struct {
	TransferID      string
	ProductID       string
	VariantID       sql.NullString
	FromWarehouseID string
	ToWarehouseID   string
	Quantity        int
	TransferDate    time.Time
	TransferDetails sql.NullString
}

type StockTransferResponseDTO struct {
	TransferID        string    `json:"transfer_id"`
	ProductID         string    `json:"product_id" validate:"required"`
	ProductName       string    `json:"product_name"`
	VariantID         *string   `json:"variant_id,omitempty"`
	FromWarehouseID   string    `json:"from_warehouse_id" validate:"required"`
	FromWarehouseName string    `json:"from_warehouse_name"`
	ToWarehouseID     string    `json:"to_warehouse_id" validate:"required"`
	ToWarehouseName   string    `json:"to_warehouse_name"`
	Quantity          int       `json:"quantity" validate:"required"`
	TransferDate      time.Time `json:"transfer_date"`
	TransferDetails   string    `json:"transfer_details,omitempty"`
	Status            string    `json:"status"`
}
