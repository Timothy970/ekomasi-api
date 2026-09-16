package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"time"
)

func buildAdminOrderQuery(conds OrderConditions, limit, offset int) (string, []any) {
	// Base query with all required order and delivery fields
	query := `
		SELECT 
			o.user_id,
			o.order_id,
			o.total_amount,
			o.total_discount,
			o.delivery_id,
			o.status,
			o.payment_status,
			d.status AS delivery_status,
			o.payment_method,
			d.delivery_charge,
			d.delivery_address,
			o.guest_delivery_address,
			o.guest_personal_details,
			o.created_at,
			o.is_guest_order,
			o.source,
			d.delivered_at,
			ro.rider_id AS rider_user_id,
			rider.first_name AS rider_first_name,
			rider.last_name AS rider_last_name,
			rider.email AS rider_email,
			rider.phone_number AS rider_phone
		FROM orders o
		LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id`

	// Add rider_orders JOIN if needed
	if conds.JoinRiderOrders {
		query += " LEFT JOIN rider_orders ro ON o.order_id = ro.order_id"
	}

	// Add rider users JOIN if needed
	if conds.JoinRiderUsers {
		query += " LEFT JOIN users rider ON ro.rider_id = rider.user_id"
	}

	// Add users JOIN if needed for search
	if conds.JoinUsers {
		query += " LEFT JOIN users u ON u.user_id = o.user_id"
	}

	// Add WHERE clause if conditions exist
	if len(conds.Conditions) > 0 {
		query += " WHERE " + joinConditions(conds.Conditions)
	}

	// Order by most recent first and add pagination
	query += " ORDER BY o.created_at DESC LIMIT ? OFFSET ?"

	// Append pagination args to condition args
	args := append(append([]any{}, conds.Args...), limit, offset)
	return query, args
}

// joinConditions combines SQL condition strings with AND operators.
//
// This is an internal helper that wraps each condition in parentheses
// and joins them with AND.
//
// Parameters:
//   - conditions: []string - Array of SQL condition fragments
//
// Returns:
//   - string: Combined conditions like "(cond1) AND (cond2) AND (cond3)"
func joinConditions(conditions []string) string {
	return "(" + conditions[0] + ")" + func() string {
		if len(conditions) == 1 {
			return ""
		}
		s := ""
		for i := 1; i < len(conditions); i++ {
			s += " AND (" + conditions[i] + ")"
		}
		return s
	}()
}

// scanAdminOrderRows processes rows into AdminOrder DTOs with enrichment.
//
// This is an internal helper that scans query results and enriches orders with:
//   - Parsed guest JSON fields
//   - Order items with product details
//   - User information (if registered user)
//   - Calculated tax and subtotal
//
// Parameters:
//   - rows: *sql.Rows - Result set from admin order query
//
// Returns:
//   - []dtos.AdminOrder: Array of enriched admin orders
//   - error: Scan error, JSON parse error, or database error
func scanAdminOrderRows(db DBExecutor, rows *sql.Rows) ([]dtos.AdminOrder, error) {
	var orders []dtos.AdminOrder
	for rows.Next() {
		ord, err := scanSingleAdminOrder(db, rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, ord)
	}
	return orders, nil
}

// scanSingleAdminOrder scans and enriches a single admin order row.
func scanSingleAdminOrder(db DBExecutor, rows *sql.Rows) (dtos.AdminOrder, error) {
	var ord dtos.AdminOrder
	var guestAddrStr, guestDetailsStr sql.NullString
	var userID, deliveryAddressStr sql.NullString
	var deliveredAt sql.NullTime
	var riderUserID, riderFirstName, riderLastName, riderEmail, riderPhone sql.NullString

	// Scan all order fields including nullable columns and rider details
	if err := rows.Scan(
		&userID,
		&ord.OrderID,
		&ord.TotalAmount,
		&ord.TotalDiscount,
		&ord.DeliveryID,
		&ord.OrderStatus,
		&ord.PaymentStatus,
		&ord.DeliveryStatus,
		&ord.PaymentMethod,
		&ord.DeliveryCharge,
		&deliveryAddressStr,
		&guestAddrStr,
		&guestDetailsStr,
		&ord.CreatedAt,
		&ord.IsGuestOrder,
		&ord.OrderSource,
		&deliveredAt,
		&riderUserID,
		&riderFirstName,
		&riderLastName,
		&riderEmail,
		&riderPhone,
	); err != nil {
		return ord, err
	}

	// Calculate tax and subtotal
	calculateOrderFinancials(db, &ord)

	// Parse and set optional fields
	parseOrderOptionalFields(&ord, guestAddrStr, guestDetailsStr, deliveryAddressStr, deliveredAt)

	// Parse and set rider details
	parseRiderDetails(&ord, riderUserID, riderFirstName, riderLastName, riderEmail, riderPhone)

	// Enrich with items and user data
	if err := enrichOrderWithItemsAndUser(db, &ord, userID); err != nil {
		return ord, err
	}

	return ord, nil
}

// calculateOrderFinancials calculates tax and subtotal for an order.
func calculateOrderFinancials(db DBExecutor, ord *dtos.AdminOrder) {
	estimatedTax, _ := GetEstimatedTax(db)
	ord.EstimatedTax = ord.TotalAmount * estimatedTax / 100
	ord.SubTotal = ord.TotalAmount + ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax
}

// parseOrderOptionalFields parses JSON fields and sets nullable fields.
func parseOrderOptionalFields(ord *dtos.AdminOrder, guestAddrStr, guestDetailsStr, deliveryAddressStr sql.NullString, deliveredAt sql.NullTime) {
	if guestAddrStr.Valid {
		_ = json.Unmarshal([]byte(guestAddrStr.String), &ord.GuestDeliveryAddress)
	}
	if guestDetailsStr.Valid {
		_ = json.Unmarshal([]byte(guestDetailsStr.String), &ord.GuestPersonalDetails)
	}
	if deliveryAddressStr.Valid {
		ord.DeliveryAddress = &deliveryAddressStr.String
	}
	if deliveredAt.Valid {
		ord.DeliveredAt = &deliveredAt.Time
	}
}

// parseRiderDetails parses rider information and sets the Rider field.
func parseRiderDetails(ord *dtos.AdminOrder, riderUserID, riderFirstName, riderLastName, riderEmail, riderPhone sql.NullString) {
	if riderUserID.Valid {
		ord.Rider = &dtos.Rider{
			UserID: riderUserID.String,
		}
		if riderFirstName.Valid {
			ord.Rider.FirstName = &riderFirstName.String
		}
		if riderLastName.Valid {
			ord.Rider.LastName = &riderLastName.String
		}
		if riderEmail.Valid {
			ord.Rider.Email = &riderEmail.String
		}
		if riderPhone.Valid {
			ord.Rider.Phone = &riderPhone.String
		}
	}
}

// enrichOrderWithItemsAndUser fetches and attaches order items and user data.
func enrichOrderWithItemsAndUser(db DBExecutor, ord *dtos.AdminOrder, userID sql.NullString) error {
	// Fetch order items and set count
	items, err := getOrderProducts(db, ord.OrderID, userID.String)
	if err != nil {
		return err
	}
	ord.ItemsCount = len(items)
	ord.Items = items

	// Fetch user details if registered user order
	if userID.Valid {
		user, err := GetUserByUserID(db, userID.String)
		if err != nil {
			return err
		}
		ord.User = user
	}

	return nil
}

// GetOrderCountsByStatus returns order statistics grouped by status.
//
// This function provides order count and total amount aggregated by order status,
// useful for admin dashboards and reporting.
//
// Returns:
//   - []dtos.OrderStatusCount: Array of status statistics including:
//   - Status: Order status ("pending", "completed", etc.)
//   - Count: Number of orders with this status
//   - TotalAmount: Sum of total_amount for orders with this status
//   - Final row: Status="Total Orders" with overall count and amount
//   - error: Database error or nil on success
func GetOrderCountsByStatus(db DBExecutor) ([]dtos.OrderStatusCount, error) {
	// Group orders by status with counts and sums
	query := `
		SELECT status, COUNT(*) as count, SUM(total_amount) as total_amount
		FROM orders
		GROUP BY status
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := []dtos.OrderStatusCount{}
	var totalOrders int
	var totalAmount float64

	for rows.Next() {
		var status string
		var count int
		var amount float64

		// Scan status, count, and sum
		if err := rows.Scan(&status, &count, &amount); err != nil {
			return nil, err
		}

		counts = append(counts, dtos.OrderStatusCount{
			Status:      status,
			Count:       count,
			TotalAmount: amount,
		})

		// Accumulate totals
		totalOrders += count
		totalAmount += amount
	}

	// Append summary row with overall totals
	counts = append(counts, dtos.OrderStatusCount{
		Status:      "Total Orders",
		Count:       totalOrders,
		TotalAmount: totalAmount,
	})

	return counts, nil
}

// HoldOrder marks an order as held for review or fraud prevention.
//
// This function inserts a record into held_orders table to flag orders
// that require manual review before processing.
//
// Parameters:
//   - orderID: string - The order to hold
//
// Returns:
//   - error: Database error or nil on success
//
// Use Cases:
//   - Fraud detection
//   - Payment verification required
//   - Inventory availability check
//   - High-value order review
func HoldOrder(db DBExecutor, orderID string) error {
	query := `INSERT INTO held_orders (order_id, held_at) VALUES (?, ?)`
	_, err := db.Exec(query, orderID, time.Now())
	return err
}

// ReleaseOrder removes an order from held status.
//
// This function deletes the held_orders record, allowing the order
// to proceed with normal processing.
//
// Parameters:
//   - orderID: string - The order to release from hold
//
// Returns:
//   - error: Database error or nil on success
