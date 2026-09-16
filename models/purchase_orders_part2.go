package models

import (
	"ekomasi_backend/dtos"
	"errors"

	"github.com/teris-io/shortid"
)

func AddProductToPurchaseOrder(db DBExecutor, item dtos.PurchaseOrderItem) error {
	// Validate purchase order exists
	err := isPurchaseOrderThere(db, item.PoID)
	if err != nil {
		return err
	}

	// Validate product exists
	err = IsProductThere(db, item.ProductID)
	if err != nil {
		return err
	}

	// Validate variant exists
	err = isVariantThere(db, item.VariantID)
	if err != nil {
		return err
	}

	// Generate unique line item ID
	itemID, _ := shortid.Generate()

	// Insert line item
	query := `
		INSERT INTO purchase_order_items 
		(po_item_id, po_id, product_id, variant_id, quantity, unit_cost)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	if _, err = db.Exec(query, itemID, item.PoID, item.ProductID, item.VariantID, item.Quantity, item.UnitCost); err != nil {
		return err
	}
	return nil
}

// RemoveProductFromPurchaseOrder removes a line item from a purchase order.
//
// Parameters:
//   - itemID: string - The po_item_id to remove
//
// Returns:
//   - error: "no item found with given ID", database error, or nil on success
func RemoveProductFromPurchaseOrder(db DBExecutor, itemID string) error {
	// Delete line item
	query := `DELETE FROM purchase_order_items WHERE po_item_id = ?`

	result, err := db.Exec(query, itemID)
	if err != nil {
		return err
	}

	// Check if item was found and deleted
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("no item found with given ID")
	}
	return nil
}
