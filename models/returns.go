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
	"adenzo_backend/dtos"
	"fmt"

	"github.com/teris-io/shortid"
)

// CreateReturns creates a new return request for an order.
//
// This function creates the return record with initial "Pending" status and
// associates the returned products with their quantities.
//
// Parameters:
//   - req: dtos.ReturnRequest containing:
//   - OrderID: The order from which products are being returned
//   - Reason: Customer's reason for return (e.g., "defective", "wrong item")
//   - ReturnProducts: Array of products with quantities to return
//   - userID: string - The user requesting the return
//
// Returns:
//   - error: Database error or nil on success
//
// Workflow:
//  1. Generate unique return ID
//  2. Insert return record with "Pending" status
//  3. Associate returned products via insertIntoReturnProducts
func CreateReturns(db DBExecutor, req dtos.ReturnRequest, userID string) error {
	// Generate unique return ID
	returnID, _ := shortid.Generate()

	// Create the return record in the database with initial "Pending" status
	query := `
		INSERT INTO returns (return_id, reason, status, order_id, user_id)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query, returnID, req.Reason, "Pending", req.OrderID, userID)
	if err != nil {
		return err
	}

	// Insert products associated with the return
	err = insertIntoReturnProducts(db, returnID, req)
	if err != nil {
		return err
	}
	return nil
}

// insertIntoReturnProducts associates products with a return request.
//
// This is an internal helper function that creates return_products records
// linking the return to specific products and their quantities.
//
// Parameters:
//   - returnID: string - The return_id to associate products with
//   - req: dtos.ReturnRequest - Contains ReturnProducts array with:
//   - ProductID: Product being returned
//   - Quantity: Number of units to return
//
// Returns:
//   - error: Database error or nil on success
func insertIntoReturnProducts(db DBExecutor, returnID string, req dtos.ReturnRequest) error {
	query := `
		INSERT INTO return_products (return_product_id, return_id, product_id, quantity)
		VALUES (?, ?, ?, ?)
	`
	// Insert each product in the return
	for _, product := range req.ReturnProducts {
		// Generate unique ID for each return product association
		returnProductID, _ := shortid.Generate()
		_, err := db.Exec(query, returnProductID, returnID, product.ProductID, product.Quantity)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateReturnStatus updates the status of a return request.
//
// This function allows admins to approve or reject return requests.
// Typical statuses: "Pending", "Approved", "Rejected".
//
// Parameters:
//   - returnID: string - The return_id to update
//   - statusUpdate: dtos.ReturnStatusUpdate containing:
//   - Status: New status value ("Approved", "Rejected", etc.)
//
// Returns:
//   - error: Database error or nil on success
func UpdateReturnStatus(db DBExecutor, returnID string, statusUpdate dtos.ReturnStatusUpdate) error {
	// Update return status (typically by admin)
	query := `
		UPDATE returns SET status = ? WHERE return_id = ?
	`
	_, err := db.Exec(query, statusUpdate.Status, returnID)
	return err
}

// GetReturnByID retrieves a single return request with product details and refund amount.
//
// This function fetches the return record, associated products, and calculates the
// total refund amount based on original order prices.
//
// Parameters:
//   - returnID: string - The return_id to retrieve
//
// Returns:
//   - dtos.ReturnResponse: Return details containing:
//   - ReturnID, Reason, Status, CreatedAt, OrderID
//   - Products: Array of returned products with quantities
//   - TotalRefund: Calculated refund amount from order prices
//   - error: sql.ErrNoRows if not found, database error, or nil on success
func GetReturnByID(db DBExecutor, returnID string) (dtos.ReturnResponse, error) {
	var ret dtos.ReturnResponse

	// Query return details
	query := `
		SELECT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		WHERE r.return_id = ?
	`
	row := db.QueryRow(query, returnID)
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

// GetProductRefundAmount calculates the refund amount for a returned product.
//
// This function retrieves the unit price from the original order and calculates
// the total refund based on quantity being returned.
//
// Parameters:
//   - productID: string - The product being returned
//   - quantity: int - Number of units being returned
//   - orderID: string - The order from which the product is being returned
//
// Returns:
//   - float64: Total refund amount (unit_price * quantity)
//   - error: sql.ErrNoRows if product not in order, database error, or nil on success
func GetProductRefundAmount(db DBExecutor, productID string, quantity int, orderID string) (float64, error) {
	var price float64

	// Get unit price from order_items (price at time of purchase)
	query := `
		SELECT oi.unit_price
		FROM order_items oi
		WHERE oi.product_id = ? AND oi.order_id = ?
		LIMIT 1
	`
	row := db.QueryRow(query, productID, orderID)
	err := row.Scan(&price)
	if err != nil {
		return 0, err
	}

	// Calculate refund: unit price * quantity
	return price * float64(quantity), nil
}

// GetAllReturns retrieves all return requests with filtering, pagination, and status counts.
//
// This function provides a comprehensive return listing with:
//   - Optional status filtering (Pending, Approved, Rejected, or All)
//   - Optional search query (searches reason, order ID, product name)
//   - Pagination support
//   - Status count analytics (counts by each status + total)
//   - Full product details and refund amounts for each return
//
// Parameters:
//   - page: int - Page number (1-based)
//   - size: int - Number of returns per page
//   - status: string - Filter by status ("Pending", "Approved", "Rejected", "All", or "")
//     Empty string or "All" = no status filter
//   - q: string - Search query for reason, order ID, or product name
//     Empty string = no search filter
//
// Returns:
//   - *dtos.ReturnListResponse: Contains:
//   - Returns: Array of return records with products and refund amounts
//   - CountsByStatus: Status analytics (Pending, Approved, Rejected, Total Returns)
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func GetAllReturns(db DBExecutor, page, size int, status, q string) (*dtos.ReturnListResponse, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	where, args := buildReturnFilters(status, q)

	totalRecords, err := countTotalReturns(db, where, args)
	if err != nil {
		return nil, nil, err
	}

	returns, err := fetchReturnsWithDetails(db, where, args, size, offset)
	if err != nil {
		return nil, nil, err
	}

	counts, err := getReturnStatusCounts(db, totalRecords)
	if err != nil {
		return nil, nil, err
	}

	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalRecords,
		TotalPages: (totalRecords + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    offset+size < totalRecords,
	}
	return &dtos.ReturnListResponse{
		Returns:        returns,
		CountsByStatus: counts,
	}, meta, nil
}

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
	where += " AND r.user_id = ?"
	args = append(args, ownerID)

	// Build query with filters
	selectQuery := `
		SELECT DISTINCT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		LEFT JOIN return_products rp ON r.return_id = rp.return_id
		LEFT JOIN products p ON rp.product_id = p.product_id
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
		WHERE r.return_id = ? AND r.user_id = ?
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
