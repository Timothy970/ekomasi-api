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

func ListInventory(page, size int, categoryID, stock, storeID, search string) ([]dtos.Inventory, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	baseCount := `
		SELECT COUNT(*) 
		FROM inventory inv
		JOIN products prd ON inv.product_id = prd.product_id
		JOIN categories cat ON prd.category_id = cat.category_id
	`

	var filters []string
	var args []interface{}

	// Filter by category
	if categoryID != "" {
		filters = append(filters, "prd.category_id = ?")
		args = append(args, categoryID)
	}

	// Filter by stock level
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

	// Filter by warehouse
	if storeID != "" {
		filters = append(filters, "inv.warehouse_id = ?")
		args = append(args, storeID)
	}

	// Search field
	if search != "" {
		filters = append(filters, "(prd.name LIKE ? OR cat.name LIKE ? OR inv.inventory_id LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Build COUNT query
	countSQL := baseCount
	if len(filters) > 0 {
		countSQL += " WHERE " + strings.Join(filters, " AND ")
	}

	var totalItems int
	if err := DB.QueryRow(countSQL, args...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Build SELECT query
	selectSQL := `
		SELECT 
			inv.inventory_id, inv.warehouse_id, prd.product_id, inv.variant_id,
			inv.quantity, inv.low_stock_threshold, prd.name, prd.description,
			prd.sku, prd.tag, prd.price, prd.category_id, 
			cat.name, prd.stock_quantity, prd.search_vector
		FROM inventory inv
		JOIN products prd ON inv.product_id = prd.product_id
		JOIN categories cat ON prd.category_id = cat.category_id
	`

	if len(filters) > 0 {
		selectSQL += " WHERE " + strings.Join(filters, " AND ")
	}

	selectSQL += " ORDER BY inv.last_updated DESC LIMIT ? OFFSET ?"

	rows, err := DB.Query(selectSQL, append(args, size, offset)...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var inventories []dtos.Inventory

	for rows.Next() {
		var inv dtos.Inventory
		err := rows.Scan(
			&inv.InventoryID, &inv.StoreID, &inv.ProductID, &inv.VariantID,
			&inv.Quantity, &inv.LowStockThreshold, &inv.Name, &inv.Description,
			&inv.SKU, &inv.Tag, &inv.Price, &inv.CategoryID,
			&inv.CategoryName, &inv.StockQuantity, &inv.SearchVector,
		)
		if err != nil {
			return nil, nil, err
		}

		inv.Images, _ = fetchProductImages(inv.ProductID)
		inv.SupplierInfo, _ = fetchSupplierByInventoryID(inv.InventoryID)

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
	err := IsProductThere(inv.ProductID)
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
func GetInventory(inventoryID string) (*dtos.SingleInventory, error) {
	err := isInventoryThere(inventoryID)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT 
			inv.inventory_id, inv.warehouse_id, prd.product_id, inv.variant_id, inv.quantity, inv.low_stock_threshold,
			prd.name, prd.description, prd.sku, prd.tag, prd.price,
			prd.category_id, cat.name, prd.stock_quantity, prd.search_vector, invbatch.batch_number, invbatch.expiry_date, invbatch.manufacturing_date, prdWarranty.warranty_period, inv.last_updated, prd.buying_price, bacthinsp.inspection_date, bacthinsp.inspector_id, bacthinsp.inspection_notes, bacthinsp.images, invbatch.images, invhandlingnotes.handling_notes, invhandlingnotes.condition_id
		FROM inventory inv
		JOIN products prd ON inv.product_id = prd.product_id
		JOIN categories cat ON prd.category_id = cat.category_id
		LEFT JOIN inventory_batches invbatch ON inv.inventory_id = invbatch.inventory_id
		LEFT JOIN product_warranties prdWarranty ON prd.product_id = prdWarranty.product_id
		LEFT JOIN batch_inspections bacthinsp ON invbatch.batch_id = bacthinsp.batch_id
		LEFT JOIN inventory_handling_notes invhandlingnotes ON invbatch.batch_id = invhandlingnotes.batch_id
		WHERE inv.inventory_id = ?
		ORDER BY inv.last_updated DESC
	`

	row := DB.QueryRow(query, inventoryID)

	var inv dtos.SingleInventory
	var inspectionDate, inspectorID, inspectionImagesJSON, batchImagesJSON sql.NullString
	var buyingPrice sql.NullFloat64

	if err := row.Scan(
		&inv.InventoryID, &inv.StoreID, &inv.ProductID, &inv.VariantID, &inv.Quantity, &inv.LowStockThreshold,
		&inv.Name, &inv.Description, &inv.SKU, &inv.Tag, &inv.Price,
		&inv.CategoryID, &inv.CategoryName, &inv.StockQuantity, &inv.SearchVector, &inv.BatchNumber, &inv.ExpiryDate, &inv.ManufacturingDate, &inv.Warranty, &inv.PlacedOn, &buyingPrice, &inspectionDate, &inspectorID, &inv.InspectionNotes, &inspectionImagesJSON, &batchImagesJSON, &inv.HandlingNotes, &inv.ConditionID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New(noinventory)
		}
		return nil, err
	}
	if inspectionDate.Valid {
		inspDate := StringToTime(inspectionDate.String)
		inv.InspectionDate = &inspDate
	}
	if inspectorID.Valid {
		inv.Inspector, err = GetUserByUserID(inspectorID.String)
		if err != nil {
			return nil, err
		}
	}
	if inspectionImagesJSON.Valid {
		var imgs []string
		if err := json.Unmarshal([]byte(inspectionImagesJSON.String), &imgs); err == nil {
			inv.InspectionImages = &imgs
		}
	}

	if batchImagesJSON.Valid {
		var imgs []string
		if err := json.Unmarshal([]byte(batchImagesJSON.String), &imgs); err == nil {
			inv.BatchImages = &imgs
		}
	}
	if buyingPrice.Valid {
		inv.BuyingPrice = &buyingPrice.Float64
	}
	// Fetch images for the product
	if imgs, err := fetchProductImages(inv.ProductID); err == nil {
		inv.Images = imgs
	}
	//get supplier info using inventoryID
	inv.SupplierInfo, _ = fetchSupplierByInventoryID(inventoryID)
	return &inv, nil
}
func fetchSupplierByInventoryID(inventoryID string) (dtos.Supplier, error) {
	var supplier dtos.Supplier
	query := `
		SELECT s.supplier_id, s.name, s.contact_email, s.contact_phone, s.extra_details
		FROM suppliers s
		JOIN inventory inv ON s.supplier_id = inv.supplier_id
		WHERE inv.inventory_id = ?
	`
	err := DB.QueryRow(query, inventoryID).Scan(&supplier.SupplierID, &supplier.Name, &supplier.ContactEmail, &supplier.ContactPhone, &supplier.ExtraDetails)
	return supplier, err
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
	err := IsProductThere(productID)
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
	err := IsProductThere(req.ProductID)
	if err != nil {
		return "", err
	}
	err = isSupplierThere(req.SupplierID)
	if err != nil {
		return "", err
	}
	exists, err := RecordExists("warehouses", "warehouse_id = ?", req.StoreID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("warehouse not found")
	}
	inventoryID, _ := shortid.Generate()
	_, err = DB.Exec(`
		INSERT INTO inventory (inventory_id, product_id, quantity, low_stock_threshold, warehouse_id, supplier_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		inventoryID, req.ProductID, req.Quantity, req.LowStockThreshold, req.StoreID, req.SupplierID)
	if err != nil {
		return "", err
	}
	return inventoryID, nil
}

//get stock summary using inventoryID and optioanal storeID filter

func GetInventoryStockSummary(inventoryID, storeID string) (*dtos.InventoryStockSummary, error) {
	// Check if inventory exists
	if err := isInventoryThere(inventoryID); err != nil {
		return nil, err
	}

	// Struct to hold results
	var summary dtos.InventoryStockSummary

	// Query to get product ID, total stock, and low stock threshold
	query := `
		SELECT 
			product_id,
			SUM(quantity) AS total_stock,
			MIN(low_stock_threshold) AS min_threshold
		FROM inventory
		WHERE inventory_id = ?
	`

	args := []interface{}{inventoryID}

	// Add optional store filter
	if storeID != "" {
		query += " AND warehouse_id = ?"
		args = append(args, storeID)
	}

	query += " GROUP BY product_id"

	row := DB.QueryRow(query, args...)
	var productID string
	if err := row.Scan(&productID, &summary.TotalStock, &summary.MinimumThreshold); err != nil {
		return nil, err
	}

	// Query total sales from order_items
	salesQuery := `
		SELECT COALESCE(SUM(quantity), 0)
		FROM order_items
		WHERE product_id = ?
	`
	salesArgs := []interface{}{productID}

	// If storeID is provided, filter by store
	if storeID != "" {
		salesQuery += " AND store_id = ?"
		salesArgs = append(salesArgs, storeID)
	}

	if err := DB.QueryRow(salesQuery, salesArgs...).Scan(&summary.TotalSales); err != nil {
		return nil, err
	}

	return &summary, nil
}

func GetInventoryStockHistory(inventoryID string, page, size int) (*[]dtos.InventoryStockHistory, *dtos.PaginationMeta, error) {
	// Check if the inventory record exists
	if err := isInventoryThere(inventoryID); err != nil {
		return nil, nil, err
	}

	offset := (page - 1) * size

	// Step 1: Get product_id for the given inventory
	var productID string
	productQuery := `SELECT product_id FROM inventory WHERE inventory_id = ?`
	if err := DB.QueryRow(productQuery, inventoryID).Scan(&productID); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch product_id: %v", err)
	}

	// Step 2: Fetch product name and buying price
	var productName string
	var buyingPrice float64
	productInfoQuery := `SELECT name, buying_price FROM products WHERE product_id = ?`
	if err := DB.QueryRow(productInfoQuery, productID).Scan(&productName, &buyingPrice); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch product details: %v", err)
	}

	// Step 3: Count total records for pagination
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM inventory WHERE product_id = ?`
	if err := DB.QueryRow(countQuery, productID).Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to count inventory history: %v", err)
	}

	// Step 4: Fetch paginated inventory history
	query := `
		SELECT quantity, last_updated
		FROM inventory
		WHERE product_id = ?
		ORDER BY last_updated DESC
		LIMIT ? OFFSET ?
	`

	rows, err := DB.Query(query, productID, size, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch inventory history: %v", err)
	}
	defer rows.Close()

	var history []dtos.InventoryStockHistory

	for rows.Next() {
		var record dtos.InventoryStockHistory
		var quantity int
		var lastUpdated time.Time

		if err := rows.Scan(&quantity, &lastUpdated); err != nil {
			return nil, nil, err
		}

		record.Description = fmt.Sprintf("%s × %d", productName, quantity)
		record.Amount = float64(quantity) * buyingPrice
		record.Date = lastUpdated

		history = append(history, record)
	}

	// Step 5: Prepare pagination metadata
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalCount,
		TotalPages: int(math.Ceil(float64(totalCount) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page < int(math.Ceil(float64(totalCount)/float64(size))),
	}

	return &history, meta, nil
}
