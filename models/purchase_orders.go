// Package models provides data access functions for purchase order management.
//
// This file handles purchase order operations including:
//   - Purchase order creation and CRUD operations
//   - Purchase order item management (add/remove products)
//   - Supplier validation and association
//   - Purchase order status tracking (pending, approved, received, cancelled)
//   - Paginated listing with associated items
//   - Dynamic query building for partial updates
//
// Purchase orders track inventory procurement from suppliers with line items
// for each product/variant ordered, quantities, and unit costs.
package models

import (
	"ekomasi_backend/dtos"
	"database/sql"
	"errors"

	"github.com/teris-io/shortid"
)

// Error messages for purchase order operations
var nopurcahseorder = "purchase order not found"
var wherepo = "po_id = ?"

// AddNewPurchaseOrder creates a new purchase order for a supplier.
//
// This creates the purchase order header without line items. Use AddProductToPurchaseOrder
// to add individual products to the order.
//
// Parameters:
//   - req: dtos.CreatePurchaseOrderRequest containing:
//   - SupplierID: The supplier from whom products will be purchased
//   - TotalCost: Total expected cost of the purchase order
//
// Returns:
//   - error: "supplier not found", database error, or nil on success
//
// Workflow:
//  1. Validate supplier exists
//  2. Generate unique purchase order ID
//  3. Insert purchase order header (status defaults to "pending")
func AddNewPurchaseOrder(db DBExecutor, req dtos.CreatePurchaseOrderRequest) error {
	// Validate supplier exists
	err := isSupplierThere(db, req.SupplierID)
	if err != nil {
		return err
	}

	// Generate unique purchase order ID
	poID, _ := shortid.Generate()

	// Create purchase order header
	_, err = db.Exec(`
		INSERT INTO purchase_orders (po_id, supplier_id, total_cost)
		VALUES (?, ?, ?)
	`, poID, req.SupplierID, req.TotalCost)
	if err != nil {
		return err
	}

	return nil
}

// fetchPurchaseOrderItems retrieves all line items for a purchase order.
//
// This is an internal helper function used to enrich purchase order responses
// with their associated products, variants, quantities, and costs.
//
// Parameters:
//   - poID: string - The purchase order ID to fetch items for
//
// Returns:
//   - []dtos.PurchaseOrderItems: Array of line items with:
//   - PoItemID, PoID, ProductID, VariantID
//   - Quantity, UnitCost
//   - error: Database error or nil on success
func fetchPurchaseOrderItems(db DBExecutor, poID string) ([]dtos.PurchaseOrderItems, error) {
	// Query all items for this purchase order
	rows, err := db.Query(`
		SELECT po_item_id, po_id, product_id, variant_id, quantity, unit_cost
		FROM purchase_order_items
		WHERE po_id = ?`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan line items
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

// ListPurchaseOrders retrieves all purchase orders with pagination.
//
// Orders are sorted by creation date (newest first) and enriched with their
// associated line items.
//
// Parameters:
//   - page: int - Page number (1-based)
//   - size: int - Number of orders per page
//
// Returns:
//   - []dtos.PurchaseOrderResponse: Array of purchase orders with items
//   - *dtos.PaginationMeta: Pagination metadata (page, size, totals, navigation)
//   - error: Database error or nil on success
func ListPurchaseOrders(db DBExecutor, page, size int) ([]dtos.PurchaseOrderResponse, *dtos.PaginationMeta, error) {
	// Calculate offset for pagination
	offset := (page - 1) * size

	// Get total count for pagination metadata
	var totalItems int
	if err := db.QueryRow(`SELECT COUNT(*) FROM purchase_orders`).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Query purchase orders (newest first)
	rows, err := db.Query(`
		SELECT po_id, supplier_id, status, total_cost, created_at, approved_at
		FROM purchase_orders
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan purchase orders and enrich with line items
	var orders []dtos.PurchaseOrderResponse
	for rows.Next() {
		var po dtos.PurchaseOrderResponse
		if err := rows.Scan(&po.PoID, &po.SupplierID, &po.Status, &po.TotalCost, &po.CreatedAt, &po.ApprovedAt); err != nil {
			return nil, nil, err
		}

		// Fetch line items for this purchase order
		items, err := fetchPurchaseOrderItems(db, po.PoID)
		if err != nil {
			return nil, nil, err
		}
		po.Items = items

		orders = append(orders, po)
	}

	// Build pagination metadata
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

// GetPurchaseOrderByID retrieves a single purchase order by its ID.
//
// Parameters:
//   - id: string - The po_id to retrieve
//
// Returns:
//   - dtos.PurchaseOrderResponse: Purchase order details with line items:
//   - PoID, SupplierID, Status
//   - TotalCost, CreatedAt, ApprovedAt
//   - Items: Array of line items (product, variant, quantity, unit cost)
//   - error: "purchase order not found", database error, or nil on success
func GetPurchaseOrderByID(db DBExecutor, id string) (dtos.PurchaseOrderResponse, error) {
	var po dtos.PurchaseOrderResponse

	// Query purchase order details
	err := db.QueryRow(`
		SELECT po_id, supplier_id, status, total_cost, created_at, approved_at
		FROM purchase_orders WHERE po_id = ?`, id).
		Scan(&po.PoID, &po.SupplierID, &po.Status, &po.TotalCost, &po.CreatedAt, &po.ApprovedAt)

	if err == sql.ErrNoRows {
		return dtos.PurchaseOrderResponse{}, errors.New("purchase order not found")
	}
	if err != nil {
		return dtos.PurchaseOrderResponse{}, err
	}

	// Fetch line items for this purchase order
	items, err := fetchPurchaseOrderItems(db, po.PoID)
	if err != nil {
		return dtos.PurchaseOrderResponse{}, err
	}
	po.Items = items

	return po, nil
}

// isPurchaseOrderThere validates if a purchase order exists in the database.
//
// This is an internal helper function used before purchase order operations.
//
// Parameters:
//   - id: string - The po_id to validate
//
// Returns:
//   - error: "purchase order not found" if ID doesn't exist, database error, or nil if exists
func isPurchaseOrderThere(db DBExecutor, id string) error {
	// Check purchase order existence
	exists, err := RecordExists(db, "purchase_orders", wherepo, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(nopurcahseorder)
	}
	return nil
}

// UpdatePurchaseOrder updates a purchase order's details with partial updates.
//
// This function uses dynamic query building to update only the fields provided.
// Empty/zero values are ignored (not updated).
//
// Parameters:
//   - req: dtos.UpdatePurchaseOrderRequest containing optional fields:
//   - Status: Purchase order status (pending, approved, received, cancelled)
//   - TotalCost: Updated total cost
//   - SupplierID: Change supplier (if needed)
//   - poID: string - The po_id to update
//
// Returns:
//   - error: "purchase order not found", database error, or nil on success
func UpdatePurchaseOrder(db DBExecutor, req dtos.UpdatePurchaseOrderRequest, poID string) error {
	// Validate purchase order exists
	err := isPurchaseOrderThere(db, poID)
	if err != nil {
		return err
	}

	// Build dynamic UPDATE query
	query := "UPDATE purchase_orders SET "
	args := []interface{}{}

	// Add status field if provided
	if req.Status != "" {
		query += "status = ?, "
		args = append(args, req.Status)
	}

	// Add total cost field if non-zero
	// Add total cost field if non-zero
	if req.TotalCost != 0 {
		query += "total_cost = ?, "
		args = append(args, req.TotalCost)
	}

	// Add supplier ID field if provided
	if req.SupplierID != "" {
		query += "supplier_id = ?, "
		args = append(args, req.SupplierID)
	}

	// Remove trailing comma and space
	query = query[:len(query)-2]

	// Add WHERE clause
	query += " WHERE po_id = ?"
	args = append(args, poID)

	// Execute update
	_, err = db.Exec(query, args...)
	if err != nil {
		return err
	}
	return nil
}

// DeletePurchaseOrder permanently removes a purchase order from the database.
//
// Warning: This is a hard delete. It will also cascade delete all associated
//
//	purchase order items if foreign key constraints are configured.
//
// Parameters:
//   - id: string - The po_id to delete
//
// Returns:
//   - error: "purchase order not found", database error, or nil on success
func DeletePurchaseOrder(db DBExecutor, id string) error {
	// Validate purchase order exists
	err := isPurchaseOrderThere(db, id)
	if err != nil {
		return err
	}

	// Hard delete purchase order
	_, err = db.Exec(`DELETE FROM purchase_orders WHERE po_id = ?`, id)
	if err != nil {
		return err
	}
	return nil
}

// AddProductToPurchaseOrder adds a product line item to a purchase order.
//
// This function validates the purchase order, product, and variant all exist
// before creating the line item.
//
// Parameters:
//   - item: dtos.PurchaseOrderItem containing:
//   - PoID: Purchase order to add item to
//   - ProductID: Product being ordered
//   - VariantID: Specific variant of the product
//   - Quantity: Number of units to order
//   - UnitCost: Cost per unit
//
// Returns:
//   - error: "purchase order not found", "product not found", "variant not found",
//     database error, or nil on success
//
// Workflow:
//  1. Validate purchase order exists
//  2. Validate product exists
//  3. Validate variant exists
//  4. Generate unique line item ID
//  5. Insert line item into purchase_order_items
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
