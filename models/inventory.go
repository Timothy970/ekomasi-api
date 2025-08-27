package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"

	"github.com/teris-io/shortid"
)

var noinventory = "inventory not found"

func ListInventory(page, size int) ([]dtos.Inventory, int, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	offset := (page - 1) * size

	var totalItems int
	err := DB.QueryRow(`SELECT COUNT(*) FROM inventory`).Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	rows, err := DB.Query(`
		SELECT inventory_id, product_id, variant_id, quantity, low_stock_threshold, last_updated
		FROM inventory
		ORDER BY last_updated DESC
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var inventories []dtos.Inventory
	for rows.Next() {
		var inv dtos.Inventory
		if err := rows.Scan(&inv.InventoryID, &inv.ProductID, &inv.VariantID, &inv.Quantity, &inv.LowStockThreshold, &inv.LastUpdated); err != nil {
			return nil, 0, err
		}
		inventories = append(inventories, inv)
	}

	return inventories, totalItems, nil
}

func CreateInventory(inv dtos.CreateInventoryRequest) error {
	err := isProductThere(inv.ProductID)
	if err != nil {
		return err
	}
	err = isVariantThere(inv.VariantID)
	if err != nil {
		return err
	}
	inventoryID, _ := shortid.Generate()

	_, err = DB.Exec(`
		INSERT INTO inventory (inventory_id, product_id, variant_id, quantity, low_stock_threshold)
		VALUES (?, ?, ?, ?, ?)`,
		inventoryID, inv.ProductID, inv.VariantID, inv.Quantity, inv.LowStockThreshold)
	return err
}

func GetInventory(inventoryID string) (*dtos.Inventory, error) {
	var inv dtos.Inventory
	err := DB.QueryRow(`
		SELECT inventory_id, product_id, variant_id, quantity, low_stock_threshold, last_updated
		FROM inventory
		WHERE inventory_id = ?`, inventoryID).
		Scan(&inv.InventoryID, &inv.ProductID, &inv.VariantID, &inv.Quantity, &inv.LowStockThreshold, &inv.LastUpdated)

	if err == sql.ErrNoRows {
		return nil, errors.New(noinventory)
	}
	return &inv, err
}

func UpdateInventory(inventoryID string, quantity, threshold *int) error {
	exists, err := RecordExists("inventory", "inventory_id = ?", inventoryID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noinventory)
	}
	query := "UPDATE inventory SET "
	args := []interface{}{}

	if quantity != nil {
		query += "quantity = ?, "
		args = append(args, *quantity)
	}
	if threshold != nil {
		query += "low_stock_threshold = ?, "
		args = append(args, *threshold)
	}

	// Remove trailing comma
	query = query[:len(query)-2]
	query += " WHERE inventory_id = ?"
	args = append(args, inventoryID)

	_, err = DB.Exec(query, args...)
	return err
}

func DeleteInventory(inventoryID string) error {
	exists, err := RecordExists("inventory", "inventory_id = ?", inventoryID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noinventory)
	}
	_, err = DB.Exec(`DELETE FROM inventory WHERE inventory_id = ?`, inventoryID)
	return err
}
