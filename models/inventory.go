package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

var noinventory = "inventory not found"

func ListInventory(page, size int) ([]dtos.Inventory, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	var totalItems int
	err := DB.QueryRow(`SELECT COUNT(*) FROM inventory`).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	rows, err := DB.Query(`
		SELECT inventory_id, product_id, variant_id, quantity, low_stock_threshold, last_updated
		FROM inventory
		ORDER BY last_updated DESC
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var inventories []dtos.Inventory
	for rows.Next() {
		var inv dtos.Inventory
		if err := rows.Scan(&inv.InventoryID, &inv.ProductID, &inv.VariantID, &inv.Quantity, &inv.LowStockThreshold, &inv.LastUpdated); err != nil {
			return nil, nil, err
		}
		inventories = append(inventories, inv)
	}
	totalPages := int(math.Ceil(float64(totalItems) / float64(size)))
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
	return inventories, &meta, nil
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

// helper to build period grouping
func getPeriodExpr(groupBy string) string {
	switch strings.ToLower(groupBy) {
	case "week":
		return "YEARWEEK(o.created_at)"
	case "month":
		return "DATE_FORMAT(o.created_at, '%Y-%m')"
	case "quarter":
		return "CONCAT(YEAR(o.created_at), '-Q', QUARTER(o.created_at))"
	case "year":
		return "YEAR(o.created_at)"
	default:
		return "DATE_FORMAT(o.created_at, '%Y-%m-%d')" // daily fallback
	}
}

// turnover by product or category
func GetInventoryTurnover(start, end time.Time, groupBy string) ([]dtos.InventoryTurnoverItem, error) {
	periodExpr := getPeriodExpr(groupBy)

	query := fmt.Sprintf(`
        SELECT 
            oi.product_id,
            p.category_id,
            AVG(poi.unit_cost * poi.quantity) AS avg_inventory,
            SUM(oi.unit_price * oi.quantity) AS cogs
        FROM orders o
        JOIN order_items oi ON o.order_id = oi.order_id
        JOIN products p ON oi.product_id = p.product_id
        LEFT JOIN purchase_order_items poi ON poi.product_id = oi.product_id
        LEFT JOIN purchase_orders po ON poi.po_id = po.po_id
        WHERE o.created_at BETWEEN ? AND ?
        GROUP BY %s, oi.product_id, p.category_id
    `, periodExpr)

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []dtos.InventoryTurnoverItem{}
	for rows.Next() {
		var item dtos.InventoryTurnoverItem
		var avgInv, cogs sql.NullFloat64
		var productID sql.NullString
		var categoryID sql.NullString

		if err := rows.Scan(&productID, &categoryID, &avgInv, &cogs); err != nil {
			return nil, err
		}

		if productID.Valid {
			item.ProductID = &productID.String
		}
		if categoryID.Valid {
			item.CategoryID = &categoryID.String
		}
		item.AvgInventory = avgInv.Float64
		item.COGS = cogs.Float64
		if item.AvgInventory > 0 {
			item.TurnoverRatio = item.COGS / item.AvgInventory
		}
		results = append(results, item)
	}
	return results, nil
}

func GetInventoryTurnoverByProduct(productID string, start, end time.Time, groupBy string) ([]dtos.InventoryTurnoverItem, error) {
	err := isProductThere(productID)
	if err != nil {
		return nil, err
	}
	periodExpr := getPeriodExpr(groupBy)

	query := fmt.Sprintf(`
        SELECT 
            oi.product_id,
            AVG(poi.unit_cost * poi.quantity) AS avg_inventory,
            SUM(oi.unit_price * oi.quantity) AS cogs
        FROM orders o
        JOIN order_items oi ON o.order_id = oi.order_id
        LEFT JOIN purchase_order_items poi ON poi.product_id = oi.product_id
        LEFT JOIN purchase_orders po ON poi.po_id = po.po_id
        WHERE o.created_at BETWEEN ? AND ? AND oi.product_id = ?
        GROUP BY %s, oi.product_id
    `, periodExpr)

	rows, err := DB.Query(query, start, end, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []dtos.InventoryTurnoverItem{}
	for rows.Next() {
		var item dtos.InventoryTurnoverItem
		var avgInv, cogs sql.NullFloat64
		var productID sql.NullString

		if err := rows.Scan(&productID, &avgInv, &cogs); err != nil {
			return nil, err
		}

		if productID.Valid {
			item.ProductID = &productID.String
		}
		item.AvgInventory = avgInv.Float64
		item.COGS = cogs.Float64
		if item.AvgInventory > 0 {
			item.TurnoverRatio = item.COGS / item.AvgInventory
		}
		results = append(results, item)
	}
	return results, nil
}
