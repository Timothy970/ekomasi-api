package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/teris-io/shortid"
)

// CreateOrder inserts a new order and associated items
func CreateOrder(req dtos.OrderRequest, totalAmount, totalDiscount string) (string, string, error) {
	orderID, _ := shortid.Generate()
	deliveryID, _ := shortid.Generate()
	isGuest := false
	if req.IsGuestOrder != nil {
		isGuest = *req.IsGuestOrder
	}
	_, err := DB.Exec(`
	INSERT INTO orders (
		order_id, user_id, is_guest_order, status,
		total_amount, total_discount, delivery_id, guest_personal_details, guest_delivery_address
	) VALUES (?, ?, ?, 'pending', ?, ?, ?, ?, ?)
`, orderID, req.UserID, isGuest, totalAmount, totalDiscount, deliveryID,
		req.GuestPersonalDetails, req.GuestDeliveryAddress)

	if err != nil {
		return "", "", err
	}
	return orderID, deliveryID, nil
}

func CreateOrderItem(orderID, productID, variantID, quantity, unitPrice string) (string, error) {
	orderItemID, _ := shortid.Generate()
	_, err := DB.Exec(`
		INSERT INTO order_items (
			order_item_id, order_id, product_id, variant_id,
			quantity, unit_price
		) VALUES (?, ?, ?, ?, ?, ?)
	`, orderItemID, orderID, productID, variantID, quantity, unitPrice)

	if err != nil {
		return "", err
	}
	return orderItemID, nil
}

func CreateDeliveries(orderID, deliveryID string, req dtos.OrderRequest) error {
	_, err := DB.Exec(`
		INSERT INTO deliveries (
			delivery_id, order_id, delivery_charge, status, courier_details, delivery_address
		) VALUES (?, ?, ?, 'Processing', ?, ?)
	`, deliveryID, orderID, req.DeliveryCharge, req.CourierDetails, req.DeliveryAddress)

	return err
}

// func GetOrder(orderID string) (*dtos.Order, error) {
// 	var order dtos.Order
// 	err := DB.QueryRow(`
// 		SELECT order_id, total_amount, total_discount, delivery_id, status, created_at
// 		FROM orders WHERE order_id = ?`, orderID).Scan(
// 		&order.OrderID, &order.TotalAmount, &order.TotalDiscount, &order.DeliveryID, &order.Status, &order.CreatedAt,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

//		order.Items, err = getOrderItems(orderID)
//		if err != nil {
//			return nil, err
//		}
//		return &order, nil
//	}
func GetOrderByUser(orderID, userID string) (*dtos.Order, error) {
	query := `
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
            o.created_at
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.order_id = ? AND o.user_id = ?`
	var ord dtos.Order
	var guestAddrStr, guestDetailsStr string
	var totalAmount float64
	err := DB.QueryRow(query, orderID, userID).Scan(
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
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ord.TotalAmount = totalAmount + *ord.DeliveryCharge
	// Parse guest JSON fields
	if guestAddrStr != "" {
		_ = json.Unmarshal([]byte(guestAddrStr), &ord.GuestDeliveryAddress)
	}
	if guestDetailsStr != "" {
		_ = json.Unmarshal([]byte(guestDetailsStr), &ord.GuestPersonalDetails)
	}

	items, err := getOrderProducts(orderID)
	if err != nil {
		return nil, err
	}
	ord.Items = items
	address, err := GetUserAddresses(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ord.UserAddress = &address
	return &ord, nil
}

func GetAllOrders(status *string) ([]dtos.Order, error) {
	var (
		query string
		rows  *sql.Rows
		err   error
	)
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
            o.user_id
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id`
	if status != nil {
		query = baseQuery + " WHERE o.status = ? ORDER BY o.order_id"
		rows, err = DB.Query(query, *status)
	} else {
		query = baseQuery + " ORDER BY o.order_id"
		rows, err = DB.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []dtos.Order

	for rows.Next() {
		var ord dtos.Order
		var guestAddrStr, guestDetailsStr string
		var userID sql.NullString // since it may be NULL for guests
		var totalAmount float64
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
		); err != nil {
			return nil, err
		}
		ord.TotalAmount = totalAmount + *ord.DeliveryCharge
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

		// Fetch items for this order
		items, err := getOrderProducts(ord.OrderID)
		if err != nil {
			return nil, err
		}
		ord.Items = items

		orders = append(orders, ord)
	}

	return orders, nil
}
func ListOrdersByUser(userID string, page, limit int) ([]dtos.Order, *dtos.PaginationMeta, error) {
	offset := (page - 1) * limit
	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE user_id = ?`, userID).Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	query := `
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
            o.created_at
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.user_id = ?
        ORDER BY o.created_at DESC LIMIT ? OFFSET ?`

	rows, err := DB.Query(query, userID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var orders []dtos.Order

	for rows.Next() {
		var ord dtos.Order
		var guestAddrStr, guestDetailsStr string
		var totalAmount float64
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
		); err != nil {
			return nil, nil, err
		}
		ord.TotalAmount = totalAmount + *ord.DeliveryCharge
		// Parse guest JSON fields
		if guestAddrStr != "" {
			_ = json.Unmarshal([]byte(guestAddrStr), &ord.GuestDeliveryAddress)
		}
		if guestDetailsStr != "" {
			_ = json.Unmarshal([]byte(guestDetailsStr), &ord.GuestPersonalDetails)
		}

		// Fetch items for this order
		items, err := getOrderProducts(ord.OrderID)
		if err != nil {
			return nil, nil, err
		}
		ord.Items = items
		address, err := GetUserAddresses(userID)
		if err != nil {
			return nil, nil, err
		}
		ord.UserAddress = &address
		orders = append(orders, ord)
	}
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

func ListGuestOrders(orderID, email, phone string) (*dtos.Order, error) {
	query := `
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
            o.created_at
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.order_id = ?
          AND o.guest_personal_details LIKE ?
          AND o.guest_personal_details LIKE ?`

	var ord dtos.Order
	var guestAddrStr, guestDetailsStr string
	var totalAmount float64
	err := DB.QueryRow(query, orderID, "%"+email+"%", "%"+phone+"%").Scan(
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
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ord.TotalAmount = totalAmount + *ord.DeliveryCharge
	// Parse guest JSON fields
	if guestAddrStr != "" {
		_ = json.Unmarshal([]byte(guestAddrStr), &ord.GuestDeliveryAddress)
	}
	if guestDetailsStr != "" {
		_ = json.Unmarshal([]byte(guestDetailsStr), &ord.GuestPersonalDetails)
	}

	// Fetch items for this order
	items, err := getOrderProducts(orderID)
	if err != nil {
		return nil, err
	}
	ord.Items = items

	return &ord, nil
}

func UpdateOrderStatus(orderID string, req dtos.UpdateOrderStatusRequest) error {
	// Check if order exists
	if err := isOrderThere(orderID); err != nil {
		return err
	}

	// Update with or without payment method
	if req.PaymentMethod != nil {
		_, err := DB.Exec(
			`UPDATE orders SET status = ?, payment_method = ? WHERE order_id = ?`,
			req.Status, *req.PaymentMethod, orderID,
		)
		if err != nil {
			return err
		}
	} else {
		_, err := DB.Exec(
			`UPDATE orders SET status = ? WHERE order_id = ?`,
			req.Status, orderID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func isOrderThere(id string) error {
	exists, err := RecordExists("orders", "order_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("order not found")
	}
	return nil
}

func GetOrderByID(orderID string) (*dtos.Order, error) {
	query := `
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
            o.created_at
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.order_id = ?`

	var ord dtos.Order
	var totalAmount float64
	var guestAddrStr, guestDetailsStr string

	err := DB.QueryRow(query, orderID).Scan(
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
	)

	ord.TotalAmount = totalAmount + *ord.DeliveryCharge
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Parse guest JSON fields
	if guestAddrStr != "" {
		_ = json.Unmarshal([]byte(guestAddrStr), &ord.GuestDeliveryAddress)
	}
	if guestDetailsStr != "" {
		_ = json.Unmarshal([]byte(guestDetailsStr), &ord.GuestPersonalDetails)
	}

	// Fetch items for this order
	items, err := getOrderProducts(orderID)
	if err != nil {
		return nil, err
	}
	ord.Items = items

	return &ord, nil
}

func getOrderProducts(orderID string) ([]dtos.OrderProduct, error) {
	itemsQuery := `
        SELECT 
            p.product_id,
            p.name,
            p.description,
            p.sku,
            oi.unit_price,
            p.category_id,
            p.stock_quantity,
            p.search_vector,
            p.created_at,
            p.last_updated_at
        FROM order_items oi
        JOIN products p ON oi.product_id = p.product_id
        WHERE oi.order_id = ?`

	rows, err := DB.Query(itemsQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dtos.OrderProduct
	for rows.Next() {
		var item dtos.OrderProduct
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.SKU,
			&item.Price,
			&item.CategoryID,
			&item.StockQuantity,
			&item.SearchVector,
			&item.CreatedAt,
			&item.LastUpdated,
		); err != nil {
			return nil, err
		}

		// Fetch product images
		images, err := fetchProductImages(item.ID)
		if err != nil {
			return nil, err
		}
		item.Images = images

		items = append(items, item)
	}

	return items, nil
}

// admin handler to get all orders with pagination and filtering
// filter by status, time range: today, this week, this month, last month, this year
// search by order id, user
func ListOrdersByAdmin(status, timeRange, orderID, user string, page, limit int) ([]dtos.AdminOrder, *dtos.PaginationMeta, error) {
	offset := (page - 1) * limit

	conds := buildAdminOrderConditions(status, timeRange, orderID, user)

	total, err := getAdminOrderCount(conds)
	if err != nil {
		return nil, nil, err
	}

	query, queryArgs := buildAdminOrderQuery(conds, limit, offset)
	rows, err := DB.Query(query, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	orders, err := scanAdminOrderRows(rows)
	if err != nil {
		return nil, nil, err
	}

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

// Helper to build WHERE conditions and args
type OrderConditions struct {
	Conditions []string
	Args       []interface{}
	JoinUsers  bool
}

func buildAdminOrderConditions(status, timeRange, orderID, user string) OrderConditions {
	var conditions []string
	var args []interface{}
	joinUsers := false

	if status != "" {
		conditions = append(conditions, "o.status = ?")
		args = append(args, status)
	}

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

	if !start.IsZero() && !end.IsZero() {
		conditions = append(conditions, "o.created_at >= ? AND o.created_at < ?")
		args = append(args, start, end)
	}

	if orderID != "" {
		conditions = append(conditions, "o.order_id = ?")
		args = append(args, orderID)
	}

	if user != "" {
		joinUsers = true
		conditions = append(conditions, `(
			(o.user_id IS NULL AND o.guest_personal_details LIKE ?)
			OR
			(o.user_id IS NOT NULL AND (
				u.email LIKE ? OR
				u.first_name LIKE ? OR
				u.last_name LIKE ? OR
				u.phone_number LIKE ?
			))
		)`)

		likeUser := "%" + user + "%"
		args = append(args, likeUser, likeUser, likeUser, likeUser, likeUser)
	}

	return OrderConditions{
		Conditions: conditions,
		Args:       args,
		JoinUsers:  joinUsers,
	}
}

// Helper to get total count
func getAdminOrderCount(conds OrderConditions) (int, error) {
	countQuery := "SELECT COUNT(*) FROM orders o"
	if conds.JoinUsers {
		countQuery += " LEFT JOIN users u ON u.user_id = o.user_id"
	}
	if len(conds.Conditions) > 0 {
		countQuery += " WHERE " + joinConditions(conds.Conditions)
	}
	var total int
	err := DB.QueryRow(countQuery, conds.Args...).Scan(&total)
	return total, err
}

// Helper to build main query and args
func buildAdminOrderQuery(conds OrderConditions, limit, offset int) (string, []interface{}) {
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
			o.created_at
		FROM orders o
		LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id`
	if conds.JoinUsers {
		query += " LEFT JOIN users u ON u.user_id = o.user_id"
	}
	if len(conds.Conditions) > 0 {
		query += " WHERE " + joinConditions(conds.Conditions)
	}
	query += " ORDER BY o.created_at DESC LIMIT ? OFFSET ?"
	args := append(append([]interface{}{}, conds.Args...), limit, offset)
	return query, args
}

// Helper to join conditions with AND
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

// Helper to scan rows
func scanAdminOrderRows(rows *sql.Rows) ([]dtos.AdminOrder, error) {
	var orders []dtos.AdminOrder
	for rows.Next() {
		var ord dtos.AdminOrder
		var guestAddrStr, guestDetailsStr string
		var userID sql.NullString
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
			&ord.DeliveryAddress,
			&guestAddrStr,
			&guestDetailsStr,
			&ord.CreatedAt,
		); err != nil {
			return nil, err
		}
		if guestAddrStr != "" {
			_ = json.Unmarshal([]byte(guestAddrStr), &ord.GuestDeliveryAddress)
		}
		if guestDetailsStr != "" {
			_ = json.Unmarshal([]byte(guestDetailsStr), &ord.GuestPersonalDetails)
		}
		items, err := getOrderProducts(ord.OrderID)
		ord.ItemsCount = len(items)
		if err != nil {
			return nil, err
		}
		ord.Items = items
		var user *dtos.Users
		if userID.Valid {
			user, err = GetUserByUserID(userID.String)
			if err != nil {
				return nil, err
			}
		}
		ord.User = user
		orders = append(orders, ord)
	}
	return orders, nil
}

// get order counts, total amount grouped by status
func GetOrderCountsByStatus() ([]dtos.OrderStatusCount, error) {
	query := `
		SELECT status, COUNT(*) as count, SUM(total_amount) as total_amount
		FROM orders
		GROUP BY status
	`
	rows, err := DB.Query(query)
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

		if err := rows.Scan(&status, &count, &amount); err != nil {
			return nil, err
		}

		counts = append(counts, dtos.OrderStatusCount{
			Status:      status,
			Count:       count,
			TotalAmount: amount,
		})

		totalOrders += count
		totalAmount += amount
	}

	// Append total row
	counts = append(counts, dtos.OrderStatusCount{
		Status:      "Total Orders",
		Count:       totalOrders,
		TotalAmount: totalAmount,
	})

	return counts, nil
}
