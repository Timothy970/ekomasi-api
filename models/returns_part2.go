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
	"fmt"
)

// buildReturnFilters constructs WHERE clause and arguments for return filtering
func buildReturnFilters(status, q string) (string, []interface{}) {
	where := "WHERE 1=1"
	var args []interface{}

	if status != "" && status != "All" {
		where += " AND LOWER(r.status) LIKE ?"
		args = append(args, "%"+status+"%")
	}

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

	return where, args
}

// countTotalReturns counts total matching records for pagination
func countTotalReturns(db DBExecutor, where string, args []interface{}) (int, error) {
	countQuery := `
		SELECT COUNT(DISTINCT r.return_id)
		FROM returns r
		LEFT JOIN return_products rp ON r.return_id = rp.return_id
		LEFT JOIN products p ON rp.product_id = p.product_id
		` + where

	var totalRecords int
	err := db.QueryRow(countQuery, args...).Scan(&totalRecords)
	return totalRecords, err
}

// fetchReturnsWithDetails retrieves returns with product details and refunds
func fetchReturnsWithDetails(db DBExecutor, where string, args []interface{}, size, offset int) ([]dtos.ReturnResponse, error) {
	selectQuery := `
		SELECT DISTINCT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		LEFT JOIN return_products rp ON r.return_id = rp.return_id
		LEFT JOIN products p ON rp.product_id = p.product_id
		` + where + `
		ORDER BY r.created_at DESC
		LIMIT ? OFFSET ?
	`
	argsWithPagination := append(args, size, offset)

	rows, err := db.Query(selectQuery, argsWithPagination...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var returns []dtos.ReturnResponse

	for rows.Next() {
		var ret dtos.ReturnResponse

		if err := rows.Scan(&ret.ReturnID, &ret.Reason, &ret.Status, &ret.CreatedAt, &ret.OrderID); err != nil {
			return nil, err
		}

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

// getReturnStatusCounts retrieves status counts for analytics
func getReturnStatusCounts(db DBExecutor, totalRecords int) ([]dtos.ReturnsCounts, error) {
	countsQuery := `
	SELECT status, COUNT(*)
	FROM returns
	GROUP BY status
`
	countRows, err := db.Query(countsQuery)
	if err != nil {
		return nil, err
	}
	defer countRows.Close()

	statusMap := map[string]int{
		"Pending":  0,
		"Approved": 0,
		"Rejected": 0,
	}

	for countRows.Next() {
		var s string
		var c int
		if err := countRows.Scan(&s, &c); err != nil {
			return nil, err
		}
		if _, ok := statusMap[s]; ok {
			statusMap[s] = c
		}
	}

	counts := []dtos.ReturnsCounts{
		{Status: "Pending", Count: statusMap["Pending"]},
		{Status: "Approved", Count: statusMap["Approved"]},
		{Status: "Rejected", Count: statusMap["Rejected"]},
		{Status: "Total Returns", Count: totalRecords},
	}

	return counts, nil
}

// fetchReturnProductsAndRefund fetches products and calculates total refund for a return.
//
// This is an internal helper function used by GetAllReturns and other functions to
// enrich return records with product details and refund calculations.
//
// Parameters:
//   - returnID: string - The return_id to fetch products for
//   - orderID: string - The order_id for refund price lookups
//
// Returns:
//   - []dtos.Product: Array of products with quantities (StockQuantity set to return quantity)
//   - float64: Total refund amount for all products
//   - error: Database error or nil on success
func fetchReturnProductsAndRefund(db DBExecutor, returnID, orderID string) ([]dtos.Product, float64, error) {
	// Query return products and quantities
	productIDsQuery := `
		SELECT rp.product_id, rp.quantity
		FROM return_products rp
		WHERE rp.return_id = ?
	`
	productRows, err := db.Query(productIDsQuery, returnID)
	if err != nil {
		return nil, 0, err
	}
	defer productRows.Close()

	var products []dtos.Product
	var totalRefund float64

	// Fetch each product and calculate refund
	for productRows.Next() {
		var productID string
		var quantity int
		if err := productRows.Scan(&productID, &quantity); err != nil {
			return nil, 0, err
		}

		// Get product details
		product, err := GetProductByID(db, productID)
		product.StockQuantity = quantity // Set to return quantity (not actual stock)
		if err != nil {
			return nil, 0, err
		}

		// Calculate refund for this product
		refund, err := GetProductRefundAmount(db, productID, quantity, orderID)
		if err != nil {
			return nil, 0, err
		}
		totalRefund += refund // Accumulate total
		products = append(products, *product)
	}
	return products, totalRefund, nil
}

// DeleteReturn permanently removes a return request from the database.
//
// Warning: This is a hard delete. Consider soft delete (status update) to maintain
//
//	return history for auditing and analytics.
//
// Parameters:
//   - returnID: string - The return_id to delete
//
// Returns:
//   - error: Database error or nil on success
func DeleteReturn(db DBExecutor, returnID string) error {
	// Hard delete return record
	query := `
		DELETE FROM returns WHERE return_id = ?
	`
	_, err := db.Exec(query, returnID)
	return err
}

// ValidateReturnRequest validates a return request before creation.
//
// This function ensures:
//  1. The order exists
//  2. All products being returned are in the original order
//
// Parameters:
//   - req: dtos.ReturnRequest containing:
//   - OrderID: Order to validate
//   - ReturnProducts: Products to validate
//
// Returns:
//   - error: "order not found", "product not found in order", or nil if valid
func ValidateReturnRequest(db DBExecutor, req dtos.ReturnRequest) error {
	// Check if order exists
	err := IsOrderThere(db, req.OrderID)
	if err != nil {
		return err
	}

	// Check if each product exists in the order
	for _, product := range req.ReturnProducts {
		err := IsProductInOrder(db, req.OrderID, product.ProductID)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsProductInOrder validates that a product exists in a specific order.
//
// This function is used during return validation to ensure customers can only
// return products they actually purchased.
//
// Parameters:
//   - orderID: string - The order to check
//   - productID: string - The product to validate
//
// Returns:
//   - error: "product not found", "product with ID X not found in order Y",
//     database error, or nil if product is in order
func IsProductInOrder(db DBExecutor, orderID, productID string) error {
	// First validate product exists globally
	err := IsProductThere(db, productID)
	if err != nil {
		return err
	}

	// Now check if the product is in the specific order
	query := `
		SELECT COUNT(*) FROM order_items
		WHERE order_id = ? AND product_id = ?
	`
	var count int
	err = db.QueryRow(query, orderID, productID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("product with ID %s not found in order %s", productID, orderID)
	}
	return nil
}
