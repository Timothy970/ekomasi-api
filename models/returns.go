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
