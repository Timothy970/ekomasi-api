package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
)

func GetAllOrders(db DBExecutor, tenantID int, status *string) ([]dtos.Order, error) {
	var (
		query string
		rows  *sql.Rows
		err   error
	)

	// Base query with order and delivery join
	baseQuery := `
        SELECT 
            o.order_id,
            o.total_amount,
            o.total_discount,
            o.delivery_id,
            o.status,
            d.status AS delivery_status,
            o.payment_method,
            d.delivery_charge,
            d.delivery_address,
            o.guest_delivery_address,
            o.guest_personal_details,
            o.created_at,
            o.user_id,
			o.is_guest_order,
			o.source
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id`

	// Apply status filter if provided
	if status != nil {
		query = baseQuery + " WHERE o.status = ? AND o.tenant_id = ? ORDER BY o.order_id"
		rows, err = db.Query(query, *status, tenantID)
	} else {
		query = baseQuery + " WHERE o.tenant_id = ? ORDER BY o.order_id"
		rows, err = db.Query(query, tenantID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []dtos.Order

	for rows.Next() {
		var ord dtos.Order
		var guestAddrStr, guestDetailsStr string
		var userID sql.NullString // Nullable for guest orders
		var totalAmount float64

		// Scan order fields including nullable userID
		if err := rows.Scan(
			&ord.OrderID,
			&totalAmount,
			&ord.TotalDiscount,
			&ord.DeliveryID,
			&ord.OrderStatus,
			&ord.DeliveryStatus,
			&ord.PaymentMethod,
			&ord.DeliveryCharge,
			&ord.DeliveryAddress,
			&guestAddrStr,
			&guestDetailsStr,
			&ord.CreatedAt,
			&userID,
			&ord.IsGuestOrder,
			&ord.OrderSource,
		); err != nil {
			return nil, err
		}

		// Calculate tax and subtotal
		ord.TotalAmount = totalAmount
		estimatedTax, _ := GetEstimatedTax(db)
		ord.EstimatedTax = ord.TotalAmount * estimatedTax / 100
		ord.SubTotal = ord.TotalAmount + ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax

		// // Attach user_id if present
		// if userID.Valid {
		// 	ord.UserID = userID.String
		// }

		// Parse guest JSON fields
		if guestAddrStr != "" {
			_ = json.Unmarshal([]byte(guestAddrStr), &ord.GuestDeliveryAddress)
		}
		if guestDetailsStr != "" {
			_ = json.Unmarshal([]byte(guestDetailsStr), &ord.GuestPersonalDetails)
		}

		// Fetch items for this order (empty userID for general order list)
		items, err := getOrderProducts(db, ord.OrderID, "")
		if err != nil {
			return nil, err
		}
		ord.Items = items

		orders = append(orders, ord)
	}

	return orders, nil
}

// ListOrdersByUser retrieves paginated orders for a specific user.
//
// This function fetches all orders belonging to a user with pagination,
// ordered by most recent first. Includes complete order details with items.
//
// Parameters:
//   - userID: string - The user whose orders to retrieve
//   - page: int - Page number (1-indexed)
//   - limit: int - Number of orders per page
//
// Returns:
//   - []dtos.Order: Array of orders with:
//   - Complete order and delivery details
//   - Payment information
//   - Items with product details, images, warranties
//   - Review status for each product
//   - User's address
//   - Guest details (if applicable)
//   - *dtos.PaginationMeta: Pagination metadata (Page, Size, TotalItems, TotalPages, HasPrev, HasNext)
//   - error: Database error or nil on success
func ListOrdersByUser(db DBExecutor, userID string, tenantID int, page, limit int) ([]dtos.Order, *dtos.PaginationMeta, error) {
	// Calculate pagination offset
	offset := (page - 1) * limit

	// Count total orders for this user and tenant
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM orders WHERE user_id = ? AND tenant_id = ?`, userID, tenantID).Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	// Fetch orders with delivery info, ordered by most recent
	query := `
		SELECT 
            o.order_id,
            o.total_amount,
            o.total_discount,
            o.delivery_id,
            o.status,
            d.status AS delivery_status,
            o.payment_method,
			o.payment_status,
            d.delivery_charge,
            d.delivery_address,
            o.guest_delivery_address,
            o.guest_personal_details,
            o.created_at,
			o.is_guest_order,
			o.source
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.user_id = ? AND o.tenant_id = ?
        ORDER BY o.created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, userID, tenantID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var orders []dtos.Order

	for rows.Next() {
		var ord dtos.Order
		var guestAddrStr, guestDetailsStr, paymentStatusStr string
		var totalAmount float64

		// Scan order row
		if err := rows.Scan(
			&ord.OrderID,
			&totalAmount,
			&ord.TotalDiscount,
			&ord.DeliveryID,
			&ord.OrderStatus,
			&ord.DeliveryStatus,
			&ord.PaymentMethod,
			&paymentStatusStr,
			&ord.DeliveryCharge,
			&ord.DeliveryAddress,
			&guestAddrStr,
			&guestDetailsStr,
			&ord.CreatedAt,
			&ord.IsGuestOrder,
			&ord.OrderSource,
		); err != nil {
			return nil, nil, err
		}

		// Calculate tax and subtotal
		ord.TotalAmount = totalAmount
		estimatedTax, _ := GetEstimatedTax(db)
		ord.EstimatedTax = ord.TotalAmount * estimatedTax / 100
		ord.SubTotal = ord.TotalAmount + ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax

		// Parse guest JSON fields if present
		if guestAddrStr != "" {
			_ = json.Unmarshal([]byte(guestAddrStr), &ord.GuestDeliveryAddress)
		}
		if guestDetailsStr != "" {
			_ = json.Unmarshal([]byte(guestDetailsStr), &ord.GuestPersonalDetails)
		}
		if paymentStatusStr != "" {
			ord.PaymentStatus = &paymentStatusStr
		}

		// Fetch items with product details and review status
		items, err := getOrderProducts(db, ord.OrderID, "")
		if err != nil {
			return nil, nil, err
		}
		ord.Items = items

		// Fetch user's address
		address, err := GetUserAddresses(db, userID)
		if err != nil {
			return nil, nil, err
		}
		ord.UserAddress = &address
		orders = append(orders, ord)
	}

	// Build pagination metadata
	pagination := &dtos.PaginationMeta{
		HasNext:    offset+limit < total,
		HasPrev:    page > 1,
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: (total + limit - 1) / limit,
	}

	return orders, pagination, nil
}

// ptrToFloat safely converts a float pointer to float64, returning 0 for nil.
//
// This is a helper function to handle nullable float fields in database queries.
//
// Parameters:
//   - v: *float64 - Pointer to float value (can be nil)
//
// Returns:
//   - float64: The dereferenced value, or 0 if pointer is nil
func ptrToFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// ListGuestOrders retrieves a guest order by ID with email and phone verification.
//
// This function allows guest users to retrieve their order by providing the order ID
// along with matching email and phone number for security verification.
//
// Parameters:
//   - orderID: string - The unique order ID
//   - email: string - Guest's email address (must match guest_personal_details)
//   - phone: string - Guest's phone number (must match guest_personal_details)
//
// Returns:
//   - *dtos.Order: Complete order with items and guest details, or nil if not found/mismatch
//   - error: sql.ErrNoRows if no matching order found, or database error
//
// Security:
//   - Uses LIKE queries on JSON guest_personal_details to verify both email and phone
//   - Returns nil (not found) if credentials don't match
