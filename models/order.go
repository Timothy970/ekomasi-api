package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"

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

	err := DB.QueryRow(query, orderID, userID).Scan(
		&ord.OrderID,
		&ord.TotalAmount,
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

		if err := rows.Scan(
			&ord.OrderID,
			&ord.TotalAmount,
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
func ListOrdersByUser(userID string) ([]dtos.Order, error) {
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
        ORDER BY o.created_at DESC`

	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []dtos.Order

	for rows.Next() {
		var ord dtos.Order
		var guestAddrStr, guestDetailsStr string

		if err := rows.Scan(
			&ord.OrderID,
			&ord.TotalAmount,
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
		items, err := getOrderProducts(ord.OrderID)
		if err != nil {
			return nil, err
		}
		ord.Items = items
		address, err := GetUserAddresses(userID)
		if err != nil {
			return nil, err
		}
		ord.UserAddress = &address
		orders = append(orders, ord)
	}

	return orders, nil
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

	err := DB.QueryRow(query, orderID, "%"+email+"%", "%"+phone+"%").Scan(
		&ord.OrderID,
		&ord.TotalAmount,
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

func UpdateOrderStatus(orderID, status string) error {
	res, err := DB.Exec(`UPDATE orders SET status = ? WHERE order_id = ?`, status, orderID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("order not found")
	}
	return nil
}

func getOrderItems(orderID string) ([]dtos.OrderItem, error) {
	rows, err := DB.Query(`SELECT product_id, quantity, unit_price FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dtos.OrderItem
	for rows.Next() {
		var item dtos.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.UnitPrice); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
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
	var guestAddrStr, guestDetailsStr string

	err := DB.QueryRow(query, orderID).Scan(
		&ord.OrderID,
		&ord.TotalAmount,
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
