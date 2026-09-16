// Package models provides data access functions for product return management.
//
// This file handles customer return requests and return processing including:
//   - Return request creation with product associations
//   - Return status management (Pending, Approved, Rejected)
//   - Return listing with filtering and pagination
//   - Refund amount calculation based on order item prices
//   - Return validation (order exists, products in order)
//   - Owner-specific return queries (customer's own returns)
//   - Return analytics (counts by status)
//
// Returns allow customers to request product returns with reasons, and admins
// to approve/reject based on return policies. Refund amounts are calculated
// from original order item unit prices.
package models

import (
	"ekomasi_backend/dtos"
)

// GetAllOwnerReturns retrieves all return requests for a specific user.
//
// This function allows customers to view their own return history with filtering.
// Similar to GetAllReturns but filtered by user ownership (no pagination).
//
// Parameters:
//   - status: string - Filter by status ("Pending", "Approved", "Rejected", "All", or "")
//     Empty string or "All" = no status filter
//   - q: string - Search query for reason, order ID, or product name
//     Empty string = no search filter
//   - ownerID: string - The user_id whose returns to retrieve
//
// Returns:
//   - []dtos.ReturnResponse: Array of returns with products and refund amounts
//   - error: Database error or nil on success
func GetAllOwnerReturns(db DBExecutor, status, q, ownerID string) ([]dtos.ReturnResponse, error) {
	// Build dynamic WHERE clause for filtering
	where := "WHERE 1=1"
	var args []interface{}

	// Add status filter if specified (case-insensitive)
	if status != "" && status != "All" {
		where += " AND LOWER(r.status) LIKE ?"
		args = append(args, "%"+status+"%")
	}

	// Add search query filter
	if q != "" {
		where += `
			AND (
				r.reason LIKE ? 
				OR r.order_id LIKE ?
				OR p.name LIKE ?
			)
		`
		args = append(args, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}

	// Add owner filter (only returns belonging to this user)
	where += " AND o.user_id = ?"
	args = append(args, ownerID)

	// Build query with filters
	selectQuery := `
		SELECT DISTINCT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		LEFT JOIN return_products rp ON r.return_id = rp.return_id
		LEFT JOIN products p ON rp.product_id = p.product_id
		LEFT JOIN orders o ON r.order_id = o.order_id
		` + where + `
		ORDER BY r.created_at DESC
	`

	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var returns []dtos.ReturnResponse

	// Scan return records and enrich with product details
	for rows.Next() {
		var ret dtos.ReturnResponse

		if err := rows.Scan(&ret.ReturnID, &ret.Reason, &ret.Status, &ret.CreatedAt, &ret.OrderID); err != nil {
			return nil, err
		}

		// Fetch products and calculate refund for this return
		products, totalRefund, err := fetchReturnProductsAndRefund(db, ret.ReturnID, ret.OrderID)
		if err != nil {
			return nil, err
		}

		ret.Products = products
		ret.TotalRefund = totalRefund
		returns = append(returns, ret)
	}

	return returns, nil
}

// GetOwnerReturnByID retrieves a single return request with owner verification.
//
// This function allows customers to view their own return details while preventing
// access to other users' returns.
//
// Parameters:
//   - returnID: string - The return_id to retrieve
//   - ownerID: string - The user_id for ownership verification
//
// Returns:
//   - dtos.ReturnResponse: Return details containing:
//   - ReturnID, Reason, Status, CreatedAt, OrderID
//   - Products: Array of returned products with quantities
//   - TotalRefund: Calculated refund amount from order prices
//   - error: sql.ErrNoRows if not found or user doesn't own it, database error, or nil on success
func GetOwnerReturnByID(db DBExecutor, returnID, ownerID string) (dtos.ReturnResponse, error) {
	var ret dtos.ReturnResponse

	// Query return details with owner verification
	query := `
		SELECT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		LEFT JOIN orders o ON r.order_id = o.order_id
		WHERE r.return_id = ? AND o.user_id = ?
	`
	row := db.QueryRow(query, returnID, ownerID)
	err := row.Scan(&ret.ReturnID, &ret.Reason, &ret.Status, &ret.CreatedAt, &ret.OrderID)
	if err != nil {
		return ret, err
	}

	// Get product IDs and quantities associated with the return
	productIDsQuery := `
		SELECT rp.product_id, rp.quantity
		FROM return_products rp
		WHERE rp.return_id = ?
	`
	rows, err := db.Query(productIDsQuery, returnID)
	if err != nil {
		return ret, err
	}
	defer rows.Close()

	var productID string
	var quantity int
	var products []dtos.Product

	// Fetch each product and calculate refund
	for rows.Next() {
		var product *dtos.Product
		err := rows.Scan(&productID, &quantity)
		if err != nil {
			return ret, err
		}

		// Get product details
		product, err = GetProductByID(db, productID)
		product.StockQuantity = quantity // Set to return quantity (not actual stock)
		if err != nil {
			return ret, err
		}

		// Calculate refund for this product based on original order price
		refund, err := GetProductRefundAmount(db, productID, quantity, ret.OrderID)
		if err != nil {
			return ret, err
		}
		ret.TotalRefund += refund // Accumulate total refund
		products = append(products, *product)
	}
	ret.Products = products
	return ret, nil
}

func GetOrderItemsForReturn(db DBExecutor, returnID string) (*dtos.ReturnResponse, error) {
	var ret dtos.ReturnResponse

	// Query return details with owner verification
	query := `
		SELECT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		WHERE r.return_id = ?
	`
	row := db.QueryRow(query, returnID)
	err := row.Scan(&ret.ReturnID, &ret.Reason, &ret.Status, &ret.CreatedAt, &ret.OrderID)
	if err != nil {
		return nil, err
	}

	// Get product IDs and quantities associated with the return
	productIDsQuery := `
		SELECT rp.product_id, rp.quantity
		FROM return_products rp
		WHERE rp.return_id = ?
	`
	rows, err := db.Query(productIDsQuery, returnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var productID string
	var quantity int
	var products []dtos.Product

	// Fetch each product and calculate refund
	for rows.Next() {
		var product *dtos.Product
		err := rows.Scan(&productID, &quantity)
		if err != nil {
			return nil, err
		}

		// Get product details
		product, err = GetProductByID(db, productID)
		product.StockQuantity = quantity // Set to return quantity (not actual stock)
		if err != nil {
			return nil, err
		}

		// Calculate refund for this product based on original order price
		refund, err := GetProductRefundAmount(db, productID, quantity, ret.OrderID)
		if err != nil {
			return nil, err
		}
		ret.TotalRefund += refund // Accumulate total refund
		products = append(products, *product)
	}
	ret.Products = products
	return &ret, nil
}
