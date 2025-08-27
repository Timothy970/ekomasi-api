package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"math"

	"github.com/teris-io/shortid"
)

func CreateWarehouse(req dtos.CreateWarehouseRequest) (string, error) {
	id, _ := shortid.Generate()
	query := `INSERT INTO warehouses (warehouse_id, name, location, warehouse_details) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, id, req.Name, req.Location, req.WarehouseDetails)
	if err != nil {
		return "", err
	}
	return id, nil
}

func ListWarehouses(page, size int) ([]dtos.Warehouse, dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	// Count total
	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM warehouses`).Scan(&total)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Fetch records
	query := `SELECT warehouse_id, name, location, COALESCE(warehouse_details, '') as warehouse_details  FROM warehouses LIMIT ? OFFSET ?`
	rows, err := DB.Query(query, size, offset)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	var warehouses []dtos.Warehouse
	for rows.Next() {
		var w dtos.Warehouse
		if err := rows.Scan(&w.WarehouseID, &w.Name, &w.Location, &w.WarehouseDetails); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		warehouses = append(warehouses, w)
	}

	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return warehouses, meta, nil
}

func GetWarehouseByID(id string) (*dtos.Warehouse, error) {
	var w dtos.Warehouse
	query := `SELECT warehouse_id, name, location, warehouse_details FROM warehouses WHERE warehouse_id = ?`
	err := DB.QueryRow(query, id).Scan(&w.WarehouseID, &w.Name, &w.Location, &w.WarehouseDetails)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func UpdateWarehouse(id string, req dtos.UpdateWarehouseRequest) error {
	exists, err := RecordExists("warehouses", "warehouse_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("warehouse not found")
	}
	query := `UPDATE warehouses SET name = ?, location = ?, warehouse_details = ? WHERE warehouse_id = ?`
	_, err = DB.Exec(query, req.Name, req.Location, req.WarehouseDetails, id)
	return err
}

func DeleteWarehouse(id string) error {
	exists, err := RecordExists("warehouses", "warehouse_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("warehouse not found")
	}
	query := `DELETE FROM warehouses WHERE warehouse_id = ?`
	_, err = DB.Exec(query, id)
	return err
}
