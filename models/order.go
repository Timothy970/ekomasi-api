package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"time"

	"github.com/teris-io/shortid"
)

// CreateOrder inserts a new order and associated items
func CreateOrder(req dtos.OrderRequest, totalAmount, totalDiscount string) (string, string, error) {
	orderID, _ := shortid.Generate()
	deliveryID, _ := shortid.Generate()

	_, err := DB.Exec(`
		INSERT INTO orders (
			order_id, user_id, is_guest_order, status,
			total_amount, total_discount, delivery_id
		) VALUES (?, ?, ?, 'pending', ?, ?, ?)
	`, orderID, req.UserID, req.IsGuestOrder, totalAmount, totalDiscount, deliveryID)

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
			delivery_id, order_id, delivery_charge, status, courier_details
		) VALUES (?, ?, ?, 'Pending Payment', ?)
	`, deliveryID, orderID, req.DeliveryCharge, req.CourierDetails)

	return err
}

func GetOrderByID(orderID string) (*dtos.Order, error) {
	var order dtos.Order
	err := DB.QueryRow(`
		SELECT order_id, total_amount, total_discount, delivery_id, status, created_at 
		FROM orders WHERE order_id = ?`, orderID).Scan(
		&order.OrderID, &order.TotalAmount, &order.TotalDiscount, &order.DeliveryID, &order.Status, &order.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	order.Items, err = getOrderItems(orderID)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func GetOrderByUser(orderID, userID string) (*dtos.Order, error) {
	var order dtos.Order
	query := `
		SELECT order_id, total_amount, total_discount, delivery_id, status, created_at 
		FROM orders WHERE order_id = ? AND user_id = ?`
	err := DB.QueryRow(query, orderID, userID).Scan(
		&order.OrderID, &order.TotalAmount, &order.TotalDiscount, &order.DeliveryID, &order.Status, &order.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	order.Items, err = getOrderItems(order.OrderID)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func GetAllOrders(status *string) ([]*dtos.Order, error) {
	var (
		query string
		rows  *sql.Rows
		err   error
	)

	baseQuery := `
		SELECT 
			o.order_id, o.total_amount, o.total_discount, o.delivery_id, o.status, o.created_at,
			oi.product_id, oi.quantity, oi.unit_price
		FROM orders o
		LEFT JOIN order_items oi ON o.order_id = oi.order_id`

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

	return mapOrdersWithItems(rows)
}

func ListOrdersByUser(userID string) ([]dtos.Order, error) {
	rows, err := DB.Query(`
		SELECT 
			o.order_id, o.total_amount, o.total_discount, o.delivery_id, o.status, o.created_at,
			oi.product_id, oi.quantity, oi.unit_price
		FROM orders o
		LEFT JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.user_id = ?
		ORDER BY o.order_id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders, err := mapOrdersWithItems(rows)
	if err != nil {
		return nil, err
	}

	// Convert []*dtos.Order to []dtos.Order
	result := make([]dtos.Order, len(orders))
	for i, o := range orders {
		result[i] = *o
	}
	return result, nil
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

func mapOrdersWithItems(rows *sql.Rows) ([]*dtos.Order, error) {
	ordersMap := make(map[string]*dtos.Order)

	for rows.Next() {
		var (
			orderID       string
			totalAmount   float64
			totalDiscount float64
			deliveryID    string
			status        string
			createdAt     time.Time
			productID     sql.NullString
			quantity      sql.NullInt64
			unitPrice     sql.NullFloat64
		)

		if err := rows.Scan(
			&orderID, &totalAmount, &totalDiscount, &deliveryID, &status, &createdAt,
			&productID, &quantity, &unitPrice,
		); err != nil {
			return nil, err
		}

		order, exists := ordersMap[orderID]
		if !exists {
			order = &dtos.Order{
				OrderID:       orderID,
				TotalAmount:   totalAmount,
				TotalDiscount: totalDiscount,
				DeliveryID:    deliveryID,
				Status:        status,
				CreatedAt:     createdAt,
				Items:         []dtos.OrderItem{},
			}
			ordersMap[orderID] = order
		}

		if productID.Valid && quantity.Valid && unitPrice.Valid {
			order.Items = append(order.Items, dtos.OrderItem{
				ProductID: productID.String,
				Quantity:  float64(quantity.Int64),
				UnitPrice: unitPrice.Float64,
			})
		}
	}

	var orders []*dtos.Order
	for _, o := range ordersMap {
		orders = append(orders, o)
	}
	return orders, nil
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
