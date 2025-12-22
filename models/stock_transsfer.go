package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"math"

	"github.com/teris-io/shortid"
)

func CreateStockTransfer(st dtos.StockTransferDTO) error {
	err := IsProductThere(st.ProductID)
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
	err = validateProductAndWarehouse(st)
	if err != nil {
		return err
	}

	transferID, _ := shortid.Generate()
	query := `
		INSERT INTO stock_transfers 
		(transfer_id, product_id, variant_id, from_warehouse_id, to_warehouse_id, quantity, transfer_details) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, transferID, st.ProductID, st.VariantID, st.FromWarehouseID, st.ToWarehouseID, st.Quantity, st.TransferDetails)

	// update inventory for from warehouse
	_, err = DB.Exec(`
		UPDATE inventory SET quantity = quantity - ? 
		WHERE product_id = ? AND warehouse_id = ? AND quantity >= ?`,
		st.Quantity, st.ProductID, st.FromWarehouseID, st.Quantity)

	// create new inventory for to warehouse
	//first get inventory record for to warehouse
	toInventoryQuery := `SELECT low_stock_threshold, supplier FROM inventory WHERE product_id = ? AND warehouse_id = ?`
	var lowStockThreshold sql.NullInt64
	var supplier sql.NullString
	err = DB.QueryRow(toInventoryQuery, st.ProductID, st.ToWarehouseID).Scan(&lowStockThreshold, &supplier)

	//insert record
	_, err = DB.Exec(`INSERT INTO inventory (product_id, warehouse_id, quantity, low_stock_threshold, supplier) VALUES (?, ?, ?, ?, ?)`, st.ProductID, st.ToWarehouseID, st.Quantity, lowStockThreshold, supplier)
	if err != nil {
		return err
	}

	return nil
}

func validateProductAndWarehouse(st dtos.StockTransferDTO) error {
	//check if product exists in from warehouse
	query := `SELECT COUNT(*) FROM inventory WHERE product_id = ? AND warehouse_id = ?`
	var count int
	err := DB.QueryRow(query, st.ProductID, st.FromWarehouseID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("product does not exist in from warehouse")
	}
	//check if product quantity is sufficient in from warehouse
	var availableQty int
	err = DB.QueryRow("SELECT quantity FROM inventory WHERE product_id = ? AND warehouse_id = ?", st.ProductID, st.FromWarehouseID).Scan(&availableQty)
	if err != nil {
		return err
	}
	if availableQty < st.Quantity {
		return errors.New("insufficient product quantity in from warehouse")
	}
	return nil
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

func ListStockTransfers(page, size int, searchParam string) ([]dtos.StockTransferResponseDTO, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	// Arguments for count query
	var countArgs []interface{}
	countQuery := `
		SELECT COUNT(*)
		FROM stock_transfers st
		JOIN products p ON st.product_id = p.product_id
		JOIN warehouses fw ON st.from_warehouse_id = fw.warehouse_id
		JOIN warehouses tw ON st.to_warehouse_id = tw.warehouse_id
	`

	// Arguments for select query
	var selectArgs []interface{}
	selectQuery := `
		SELECT 
			st.transfer_id,
			st.product_id,
			st.variant_id,
			st.from_warehouse_id,
			st.to_warehouse_id,
			st.quantity,
			st.transfer_date,
			st.transfer_details,
			p.name AS product_name,
			fw.name AS from_warehouse_name,
			tw.name AS to_warehouse_name
		FROM stock_transfers st
		JOIN products p ON st.product_id = p.product_id
		JOIN warehouses fw ON st.from_warehouse_id = fw.warehouse_id
		JOIN warehouses tw ON st.to_warehouse_id = tw.warehouse_id
	`

	if searchParam != "" {
		searchLike := "%" + searchParam + "%"
		countQuery += `
			WHERE p.name LIKE ? OR fw.name LIKE ? OR tw.name LIKE ?
		`
		selectQuery += `
			WHERE p.name LIKE ? OR fw.name LIKE ? OR tw.name LIKE ?
		`
		// Add search parameters to both argument slices separately
		countArgs = append(countArgs, searchLike, searchLike, searchLike)
		selectArgs = append(selectArgs, searchLike, searchLike, searchLike)
	}

	selectQuery += `
		ORDER BY st.transfer_date DESC
		LIMIT ? OFFSET ?
	`
	selectArgs = append(selectArgs, size, offset)

	// Run count query
	var total int
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, nil, err
	}

	// Run select query
	rows, err := DB.Query(selectQuery, selectArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var transfers []dtos.StockTransferResponseDTO
	for rows.Next() {
		var st dtos.StockTransferResponseDTO
		if err := rows.Scan(
			&st.TransferID,
			&st.ProductID,
			&st.VariantID,
			&st.FromWarehouseID,
			&st.ToWarehouseID,
			&st.Quantity,
			&st.TransferDate,
			&st.TransferDetails,
			&st.ProductName,
			&st.FromWarehouseName,
			&st.ToWarehouseName,
		); err != nil {
			return nil, nil, err
		}
		transfers = append(transfers, st)
	}

	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page < int(math.Ceil(float64(total)/float64(size))),
	}

	return transfers, meta, nil
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
