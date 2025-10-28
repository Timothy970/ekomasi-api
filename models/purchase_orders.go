package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"

	"github.com/teris-io/shortid"
)

var nopurcahseorder = "purchase order not found"
var wherepo = "po_id = ?"

func AddNewPurchaseOrder(req dtos.CreatePurchaseOrderRequest) error {
	err := isSupplierThere(req.SupplierID)
	if err != nil {
		return err
	}
	poID, _ := shortid.Generate()

	_, err = DB.Exec(`
		INSERT INTO purchase_orders (po_id, supplier_id, total_cost)
		VALUES (?, ?, ?)
	`, poID, req.SupplierID, req.TotalCost)
	if err != nil {
		return err
	}

	return nil
}
func fetchPurchaseOrderItems(poID string) ([]dtos.PurchaseOrderItems, error) {
	rows, err := DB.Query(`
		SELECT po_item_id, po_id, product_id, variant_id, quantity, unit_cost
		FROM purchase_order_items
		WHERE po_id = ?`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dtos.PurchaseOrderItems
	for rows.Next() {
		var item dtos.PurchaseOrderItems
		if err := rows.Scan(
			&item.PoItemID, &item.PoID, &item.ProductID, &item.VariantID,
			&item.Quantity, &item.UnitCost,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func ListPurchaseOrders(page, size int) ([]dtos.PurchaseOrderResponse, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	// Get total count
	var totalItems int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM purchase_orders`).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	rows, err := DB.Query(`
		SELECT po_id, supplier_id, status, total_cost, created_at, approved_at
		FROM purchase_orders
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var orders []dtos.PurchaseOrderResponse
	for rows.Next() {
		var po dtos.PurchaseOrderResponse
		if err := rows.Scan(&po.PoID, &po.SupplierID, &po.Status, &po.TotalCost, &po.CreatedAt, &po.ApprovedAt); err != nil {
			return nil, nil, err
		}

		// fetch items
		items, err := fetchPurchaseOrderItems(po.PoID)
		if err != nil {
			return nil, nil, err
		}
		po.Items = items

		orders = append(orders, po)
	}

	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: (totalItems + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < totalItems,
	}
	return orders, meta, nil
}

func GetPurchaseOrderByID(id string) (dtos.PurchaseOrderResponse, error) {
	var po dtos.PurchaseOrderResponse
	err := DB.QueryRow(`
		SELECT po_id, supplier_id, status, total_cost, created_at, approved_at
		FROM purchase_orders WHERE po_id = ?`, id).
		Scan(&po.PoID, &po.SupplierID, &po.Status, &po.TotalCost, &po.CreatedAt, &po.ApprovedAt)

	if err == sql.ErrNoRows {
		return dtos.PurchaseOrderResponse{}, errors.New("purchase order not found")
	}
	if err != nil {
		return dtos.PurchaseOrderResponse{}, err
	}

	// fetch items
	items, err := fetchPurchaseOrderItems(po.PoID)
	if err != nil {
		return dtos.PurchaseOrderResponse{}, err
	}
	po.Items = items

	return po, nil
}

func isPurchaseOrderThere(id string) error {
	exists, err := RecordExists("purchase_orders", wherepo, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(nopurcahseorder)
	}
	return nil
}
func UpdatePurchaseOrder(req dtos.UpdatePurchaseOrderRequest, poID string) error {
	err := isPurchaseOrderThere(poID)
	if err != nil {
		return err
	}
	query := "UPDATE purchase_orders SET "
	args := []interface{}{}
	if req.Status != "" {
		query += "status = ?, "
		args = append(args, req.Status)
	}
	if req.TotalCost != 0 {
		query += "total_cost = ?, "
		args = append(args, req.TotalCost)
	}
	if req.SupplierID != "" {
		query += "supplier_id = ?, "
		args = append(args, req.SupplierID)
	}
	query = query[:len(query)-2] // remove trailing comma
	query += " WHERE po_id = ?"
	args = append(args, poID)

	_, err = DB.Exec(query, args...)
	if err != nil {
		return err
	}
	return nil
}
func DeletePurchaseOrder(id string) error {
	err := isPurchaseOrderThere(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM purchase_orders WHERE po_id = ?`, id)
	if err != nil {
		return err
	}
	return nil
}

func AddProductToPurchaseOrder(item dtos.PurchaseOrderItem) error {
	err := isPurchaseOrderThere(item.PoID)
	if err != nil {
		return err
	}
	err = IsProductThere(item.ProductID)
	if err != nil {
		return err
	}
	err = isVariantThere(item.VariantID)
	if err != nil {
		return err
	}
	itemID, _ := shortid.Generate()

	query := `
		INSERT INTO purchase_order_items 
		(po_item_id, po_id, product_id, variant_id, quantity, unit_cost)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	if _, err = DB.Exec(query, itemID, item.PoID, item.ProductID, item.VariantID, item.Quantity, item.UnitCost); err != nil {
		return err
	}
	return nil
}

func RemoveProductFromPurchaseOrder(itemID string) error {

	query := `DELETE FROM purchase_order_items WHERE po_item_id = ?`

	result, err := DB.Exec(query, itemID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("no item found with given ID")
	}
	return nil
}
