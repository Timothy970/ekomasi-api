package models

import (
	"ekomasi_backend/dtos"
	"strings"
	"time"
)

func ListOrdersByAdmin(db DBExecutor, params AdminOrderParameters) ([]dtos.AdminOrder, *dtos.PaginationMeta, error) {
	// Calculate pagination offset
	offset := (params.Page - 1) * params.Limit

	// Build WHERE conditions and determine required joins
	conds := buildAdminOrderConditions(params)

	// Get total count for pagination
	total, err := getAdminOrderCount(db, conds)
	if err != nil {
		return nil, nil, err
	}

	// Build and execute main query
	query, queryArgs := buildAdminOrderQuery(conds, params.Limit, offset)
	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan and enrich order rows
	orders, err := scanAdminOrderRows(db, rows)
	if err != nil {
		return nil, nil, err
	}

	// Build pagination metadata
	pagination := &dtos.PaginationMeta{
		HasNext:    offset+params.Limit < total,
		HasPrev:    params.Page > 1,
		Page:       params.Page,
		Size:       params.Limit,
		TotalItems: total,
		TotalPages: (total + params.Limit - 1) / params.Limit,
	}

	return orders, pagination, nil
}

// OrderConditions holds dynamically built WHERE conditions and required JOINs.
//
// This internal structure is used by admin order filtering to track:
//   - SQL WHERE clause conditions
//   - Corresponding query arguments
//   - Whether users table JOIN is needed (for user search)
//   - Whether deliveries table JOIN is needed (for delivery search/filter)
//   - Whether rider_orders table JOIN is needed (for rider filtering)
//
// Fields:
//   - Conditions: []string - Array of WHERE clause fragments (e.g., "o.status LIKE ?")
//   - Args: []any - Corresponding query arguments
//   - JoinUsers: bool - true if users table JOIN required
//   - JoinDeliveries: bool - true if deliveries table JOIN required
//   - JoinRiderOrders: bool - true if rider_orders table JOIN required
//   - JoinRiderUsers: bool - true if rider users table JOIN required
type OrderConditions struct {
	Conditions      []string
	Args            []any
	JoinUsers       bool
	JoinDeliveries  bool
	JoinRiderOrders bool
	JoinRiderUsers  bool
}

// calculateTimeRange returns start and end times for predefined time ranges.
//
// This helper function reduces cognitive complexity by isolating time range logic.
//
// Parameters:
//   - timeRange: string - One of: "today", "this_week", "this_month", "last_month", "this_year"
//
// Returns:
//   - time.Time: Start time (zero value if invalid range)
//   - time.Time: End time (zero value if invalid range)
func calculateTimeRange(timeRange string) (time.Time, time.Time) {
	now := time.Now()
	var start, end time.Time

	switch timeRange {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(24 * time.Hour)
	case "this_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(weekday - 1))
		end = start.AddDate(0, 0, 7)
	case "this_month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
	case "last_month":
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
	case "this_year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(1, 0, 0)
	}

	return start, end
}

// buildAdminOrderConditions constructs WHERE conditions from filter parameters.
//
// This is an internal helper that builds dynamic SQL conditions based on
// provided filters and determines which tables need to be joined.
//
// Parameters:
//   - params: AdminOrderParameters with filter criteria
//
// Returns:
//   - OrderConditions: Structure containing:
//   - SQL condition strings
//   - Query arguments
//   - Join requirements
//
// Time Range Logic:
//   - "today": 00:00:00 today to 00:00:00 tomorrow
//   - "this_week": Monday 00:00:00 to next Monday 00:00:00
//   - "this_month": 1st of month to 1st of next month
//   - "last_month": 1st of last month to 1st of this month
//   - "this_year": January 1st to January 1st next year
//   - Custom: StartDate and EndDate override TimeRange
//
// Search Strategy:
//   - Searches across multiple fields with LIKE
//   - Supports name search in "first last" and "last first" order
//   - Automatically adds wildcards
func buildAdminOrderConditions(params AdminOrderParameters) OrderConditions {
	var conditions []string
	var args []any
	joinUsers := false
	joinDeliveries := false
	// Always join rider tables to include rider info in response
	joinRiderOrders := true
	joinRiderUsers := true

	// Filter by rider ID when provided
	if params.RiderID != "" {
		conditions = append(conditions, "ro.rider_id = ?")
		args = append(args, params.RiderID)
	}

	// Handle "past" parameter for rider orders
	if params.RiderID != "" && params.Past == "" {
		// Show only non-delivered/non-completed orders
		conditions = append(conditions, "o.status NOT IN ('delivered', 'completed')")
	} else if params.RiderID != "" && params.Past != "" {
		// Show only delivered/completed orders
		conditions = append(conditions, "o.status IN ('delivered', 'completed')")
	}

	// Filter by user ID when provided
	if params.UserID != "" {
		joinUsers = true
		conditions = append(conditions, "o.user_id = ?")
		args = append(args, params.UserID)
	}

	// Filter by order status
	if params.OrderStatus != "" {
		conditions = append(conditions, "o.status LIKE ?")
		args = append(args, params.OrderStatus)
	}

	// Filter by payment status
	if params.PaymentStatus != "" {
		conditions = append(conditions, "o.payment_status LIKE ?")
		args = append(args, params.PaymentStatus)
	}

	// Filter by delivery status (requires deliveries JOIN)
	if params.DeliveryStatus != "" {
		joinDeliveries = true
		conditions = append(conditions, "d.status LIKE ?")
		args = append(args, params.DeliveryStatus)
	}

	// Filter by payment method
	if params.PaymentMethod != "" {
		conditions = append(conditions, "o.payment_method LIKE ?")
		args = append(args, params.PaymentMethod)
	}

	// Calculate and apply time range filter
	start, end := calculateTimeRange(params.TimeRange)
	if !start.IsZero() && !end.IsZero() {
		conditions = append(conditions, "o.created_at >= ? AND o.created_at < ?")
		args = append(args, start, end)
	}

	// Exact order ID match
	if params.OrderID != "" {
		conditions = append(conditions, "o.order_id = ?")
		args = append(args, params.OrderID)
	}

	// Full-text search across multiple fields (requires users and deliveries JOINs)
	if params.Q != "" {
		joinUsers = true
		joinDeliveries = true

		// Split search query into words for name matching
		words := strings.Fields(params.Q)

		// Search across guest details, IDs, addresses, payment method, and user info
		conditions = append(conditions, `(
		o.guest_personal_details LIKE ? OR
		o.order_id LIKE ? OR
		o.delivery_id LIKE ? OR 
		d.delivery_address LIKE ? OR
		o.guest_delivery_address LIKE ? OR
		o.payment_method LIKE ? OR
		u.email LIKE ? OR
		u.phone_number LIKE ? OR
		(
			(u.first_name LIKE ? AND u.last_name LIKE ?)
			OR
			(u.first_name LIKE ? AND u.last_name LIKE ?)
		)
	)`)

		// Add wildcard pattern for most fields
		likePattern := "%" + params.Q + "%"

		args = append(args,
			likePattern, likePattern, likePattern, likePattern,
			likePattern, likePattern, likePattern, likePattern,
		)

		// Handle name search in both "first last" and "last first" order
		if len(words) > 1 {
			args = append(
				args,
				"%"+words[0]+"%", "%"+words[1]+"%",
				"%"+words[1]+"%", "%"+words[0]+"%",
			)
		} else {
			// Single word - search in both first and last name
			args = append(
				args,
				"%"+params.Q+"%", "%"+params.Q+"%",
				"%"+params.Q+"%", "%"+params.Q+"%",
			)
		}
	}

	// Custom date range (overrides TimeRange)
	if params.StartDate != "" && params.EndDate != "" {
		startDate := StringToTime(params.StartDate)
		endDate := StringToTime(params.EndDate)
		conditions = append(conditions, "o.created_at >= ? AND o.created_at <= ?")
		args = append(args, startDate, endDate)
	}

	return OrderConditions{
		Conditions:      conditions,
		Args:            args,
		JoinUsers:       joinUsers,
		JoinDeliveries:  joinDeliveries,
		JoinRiderOrders: joinRiderOrders,
		JoinRiderUsers:  joinRiderUsers,
	}
}

// getAdminOrderCount returns the total count of orders matching the conditions.
//
// This is an internal helper for pagination that builds a COUNT query
// with appropriate JOINs based on filter requirements.
//
// Parameters:
//   - conds: OrderConditions with WHERE clause and JOIN requirements
//
// Returns:
//   - int: Total number of matching orders
//   - error: Database error or nil on success
func getAdminOrderCount(db DBExecutor, conds OrderConditions) (int, error) {
	// Build count query with required joins
	countQuery := "SELECT COUNT(*) FROM orders o"
	if conds.JoinRiderOrders {
		countQuery += " LEFT JOIN rider_orders ro ON o.order_id = ro.order_id"
	}
	if conds.JoinUsers {
		countQuery += " LEFT JOIN users u ON u.user_id = o.user_id"
	}
	if conds.JoinDeliveries {
		countQuery += " LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id"
	}
	if len(conds.Conditions) > 0 {
		countQuery += " WHERE " + joinConditions(conds.Conditions)
	}

	var total int
	err := db.QueryRow(countQuery, conds.Args...).Scan(&total)
	return total, err
}

// buildAdminOrderQuery constructs the main SELECT query with conditions and pagination.
//
// This is an internal helper that builds the complete SQL query for fetching
// admin orders with all required fields and joins.
//
// Parameters:
//   - conds: OrderConditions with WHERE clause and JOIN requirements
//   - limit: int - Number of orders to fetch
//   - offset: int - Number of orders to skip (for pagination)
//
// Returns:
//   - string: Complete SQL query
//   - []any: Query arguments (conditions + limit + offset)
