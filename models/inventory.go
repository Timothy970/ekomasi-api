package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

var noinventory = "inventory not found"

func ListInventory(page, size int, categoryID, stock, storeID string) ([]dtos.Inventory, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size
	countQuery := `SELECT COUNT(*) FROM inventory inv JOIN products prd ON inv.product_id = prd.product_id`
	var filters []string
	var args []interface{}

	if categoryID != "" {
		filters = append(filters, "prd.category_id = ?")
		args = append(args, categoryID)
	}
	if stock != "" {
		switch strings.ToLower(stock) {
		case "in":
			filters = append(filters, "inv.quantity > 0")
		case "out":
			filters = append(filters, "inv.quantity = 0")
		case "low":
			filters = append(filters, "inv.quantity <= inv.low_stock_threshold")
		}
	}
	if storeID != "" {
		filters = append(filters, "inv.warehouse_id = ?")
		args = append(args, storeID)
	}

	// Only add WHERE if there are filters
	countSQL := countQuery
	if len(filters) > 0 {
		countSQL += " WHERE " + strings.Join(filters, " AND ")
	}
	var totalItems int
	err := DB.QueryRow(countSQL, args...).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	query := `SELECT inv.inventory_id, prd.product_id, inv.variant_id, inv.quantity, inv.low_stock_threshold, prd.name, prd.description, prd.sku, prd.tag, prd.price, prd.category_id, cat.name, prd.stock_quantity, prd.search_vector
        FROM inventory inv
        JOIN products prd ON inv.product_id = prd.product_id
        JOIN categories cat ON prd.category_id = cat.category_id`
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += " ORDER BY inv.last_updated DESC LIMIT ? OFFSET ?"

	rows, err := DB.Query(query, append(args, size, offset)...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var inventories []dtos.Inventory
	for rows.Next() {
		var inv dtos.Inventory
		if err := rows.Scan(&inv.InventoryID, &inv.ProductID, &inv.VariantID, &inv.Quantity, &inv.LowStockThreshold, &inv.Name, &inv.Description, &inv.SKU, &inv.Tag, &inv.Price, &inv.CategoryID, &inv.CategoryName, &inv.StockQuantity, &inv.SearchVector); err != nil {
			return nil, nil, err
		}
		inv.Images, _ = fetchProductImages(inv.ProductID)
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
	err := isInventoryThere(inventoryID)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT 
			inv.inventory_id, prd.product_id, inv.variant_id, inv.quantity, inv.low_stock_threshold,
			prd.name, prd.description, prd.sku, prd.tag, prd.price,
			prd.category_id, cat.name, prd.stock_quantity, prd.search_vector
		FROM inventory inv
		JOIN products prd ON inv.product_id = prd.product_id
		JOIN categories cat ON prd.category_id = cat.category_id
		WHERE inv.inventory_id = ?
		ORDER BY inv.last_updated DESC
	`

	row := DB.QueryRow(query, inventoryID)

	var inv dtos.Inventory
	if err := row.Scan(
		&inv.InventoryID, &inv.ProductID, &inv.VariantID, &inv.Quantity, &inv.LowStockThreshold,
		&inv.Name, &inv.Description, &inv.SKU, &inv.Tag, &inv.Price,
		&inv.CategoryID, &inv.CategoryName, &inv.StockQuantity, &inv.SearchVector,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New(noinventory)
		}
		return nil, err
	}

	// Fetch images for the product
	if imgs, err := fetchProductImages(inv.ProductID); err == nil {
		inv.Images = imgs
	}

	return &inv, nil
}

func UpdateInventory(inventoryID string, quantity, threshold *int) error {
	err := isInventoryThere(inventoryID)
	if err != nil {
		return err
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

func StoreBatchDetails(req dtos.Batch) (string, error) {
	err := isInventoryThere(req.InventoryID)
	if err != nil {
		return "", err
	}
	batchID, _ := shortid.Generate()
	var imagesData []byte

	if req.Images != nil {
		imagesData, err = json.Marshal(req.Images)
		if err != nil {
			fmt.Println("Error:", err)
			return "", err
		}
	}

	_, err = DB.Exec(`
		INSERT INTO inventory_batches (batch_id, inventory_id, batch_number, images, expiry_date, manufacturing_date)
		VALUES (?, ?, ?, ?, ?, ?)`,
		batchID, req.InventoryID, req.BatchNumber, imagesData, req.ExpiryDate, req.ManufacturingDate)
	if err != nil {
		log.Printf("Error inserting batch details: %v", err)
	}
	return batchID, err
}

func isInventoryThere(inventoryID string) error {
	exists, err := RecordExists("inventory", "inventory_id = ?", inventoryID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("inventory not found")
	}
	return nil
}
func isBatchThere(batchID string) error {
	exists, err := RecordExists("inventory_batches", "batch_id = ?", batchID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("batch not found")
	}
	return nil
}
func StoreInspectionDetails(req dtos.Inspection) error {
	err := isBatchThere(req.BatchID)
	if err != nil {
		return err
	}
	err = isUserThere(req.InspectorID)
	if err != nil {
		if err.Error() == "user not found" {
			return fmt.Errorf("inspector not found")
		}
		return err
	}
	var imagesData []byte

	if req.Images != nil {
		imagesData, err = json.Marshal(req.Images)
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}
	}
	inspectionID, _ := shortid.Generate()
	_, err = DB.Exec(`
		INSERT INTO batch_inspections (inspection_id, batch_id, inspection_date, inspector_id, inspection_notes, images)
		VALUES (?, ?, ?, ?, ?, ?)`,
		inspectionID, req.BatchID, req.InspectionDate, req.InspectorID, req.InspectionNotes, imagesData)
	return err
}
func isConditionThere(conditionID string) error {
	exists, err := RecordExists("batch_conditions", "condition_id = ?", conditionID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("condition not found")
	}
	return nil
}

func StoreHandlingNotes(req dtos.InventoryCondition) error {
	err := isBatchThere(req.BatchID)
	if err != nil {
		return err
	}
	err = isConditionThere(req.ConditionID)
	if err != nil {
		return err
	}
	notesID, _ := shortid.Generate()
	_, err = DB.Exec(`
		INSERT INTO inventory_handling_notes (handling_note_id, batch_id, handling_notes, condition_id)
		VALUES (?, ?, ?, ?)`,
		notesID, req.BatchID, req.HandlingNotes, req.ConditionID)
	return err
}

func StoreInventoryTracking(req dtos.InventoryTracking) (string, error) {
	err := isProductThere(req.ProductID)
	if err != nil {
		return "", err
	}
	inventoryID, _ := shortid.Generate()
	_, err = DB.Exec(`
		INSERT INTO inventory (inventory_id, product_id, quantity, low_stock_threshold, warehouse_id)
		VALUES (?, ?, ?, ?, ?)`,
		inventoryID, req.ProductID, req.Quantity, req.LowStockThreshold, req.StoreID)
	if err != nil {
		return "", err
	}
	return inventoryID, nil
}
