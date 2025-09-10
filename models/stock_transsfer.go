package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"math"

	"github.com/teris-io/shortid"
)

func CreateStockTransfer(st dtos.StockTransferDTO) error {
	err := isProductThere(st.ProductID)
	if err != nil {
		return err
	}
	err = isWarehouseThere(st.FromWarehouseID, "from")
	if err != nil {
		return err
	}
	err = isWarehouseThere(st.ToWarehouseID, "to")
	if err != nil {
		return err
	}
	transferID, _ := shortid.Generate()
	query := `
		INSERT INTO stock_transfers 
		(transfer_id, product_id, variant_id, from_warehouse_id, to_warehouse_id, quantity, transfer_details) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, transferID, st.ProductID, st.VariantID, st.FromWarehouseID, st.ToWarehouseID, st.Quantity, st.TransferDetails)
	return err
}
func isWarehouseThere(id, from string) error {
	exists, err := RecordExists("warehouses", "warehouse_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		switch from {
		case "from":
			return errors.New("from warehouse not found")
		case "to":
			return errors.New("to warehouse not found")
		default:
			return errors.New("warehouse not found")

		}

	}
	return nil
}
func ListStockTransfers(page, size int) ([]dtos.StockTransferDTO, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size
	var total int

	err := DB.QueryRow("SELECT COUNT(*) FROM stock_transfers").Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	rows, err := DB.Query(`
		SELECT transfer_id, product_id, variant_id, from_warehouse_id, to_warehouse_id, quantity, transfer_date, transfer_details
		FROM stock_transfers
		ORDER BY transfer_date DESC
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var transfers []dtos.StockTransferDTO
	for rows.Next() {
		var st dtos.StockTransferDTO
		if err := rows.Scan(
			&st.TransferID, &st.ProductID, &st.VariantID, &st.FromWarehouseID, &st.ToWarehouseID, &st.Quantity, &st.TransferDate, &st.TransferDetails,
		); err != nil {
			return nil, nil, err
		}
		transfers = append(transfers, st)
	}

	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page < int(math.Ceil(float64(total)/float64(size))),
	}
	return transfers, &meta, nil
}

func GetStockTransferByID(id string) (*dtos.StockTransferDTO, error) {
	var st dtos.StockTransferDTO
	err := DB.QueryRow(`
		SELECT transfer_id, product_id, variant_id, from_warehouse_id, to_warehouse_id, quantity, transfer_date, transfer_details
		FROM stock_transfers
		WHERE transfer_id = ?`, id).
		Scan(&st.TransferID, &st.ProductID, &st.VariantID, &st.FromWarehouseID, &st.ToWarehouseID, &st.Quantity, &st.TransferDate, &st.TransferDetails)
	if err == sql.ErrNoRows {
		return nil, errors.New("stock transfer not found")
	}
	return &st, err
}

func UpdateStockTransfer(quantity int, id string) error {
	exists, err := RecordExists("stock_transfers", "transfer_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("stock transfer not found")
	}
	query := `
		UPDATE stock_transfers 
		SET quantity = ?
		WHERE transfer_id = ?`
	_, err = DB.Exec(query, quantity, id)
	return err
}
