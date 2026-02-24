// Package models provides data access functions for the Adenzo backend.
//
// This file (order.go) contains order management functions including:
//   - Order creation with guest support (CreateOrder, CreateOrderItem, CreateDeliveries)
//   - Order retrieval (GetOrderByUser, GetAllOrders, GetOrderByID, ListOrdersByUser, ListGuestOrders)
//   - Order status management (UpdateOrderStatus, HoldOrder, ReleaseOrder)
//   - Admin order listing with advanced filtering (ListOrdersByAdmin)
//   - Order statistics and reporting (GetOrderCountsByStatus)
//   - Inventory management (DeductProductStock, UpdateProductStock)
//   - Notification tracking (MarkOrderNotificationSent)
//   - Order financial updates (UpdateOrderTotals)
package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

// CreateOrder inserts a new order with guest support into the database.
//
// This function creates the core order record including user/guest identification,
// totals, and associated delivery information.
//
// Parameters:
//   - req: dtos.OrderRequest containing:
//   - UserID: ID of the ordering user (can be empty for guest orders)
//   - IsGuestOrder: Pointer to boolean indicating if this is a guest order
//   - GuestPersonalDetails: JSON string with guest contact info (name, email, phone)
//   - GuestDeliveryAddress: JSON string with guest delivery address
//   - DeliveryCharge: Delivery fee amount
//   - DeliveryAddress: Registered user's delivery address
//   - totalAmount: string - Total order amount including all items, taxes, delivery
//   - totalDiscount: string - Total discount applied to the order
//
// Returns:
//   - string: Generated order_id
//   - string: Generated delivery_id
//   - error: Database error or nil on success
func CreateOrder(db DBExecutor, req dtos.OrderRequest, totalAmount, totalDiscount string) (string, string, error) {
	// Generate unique IDs for order and delivery
	orderID, _ := shortid.Generate()
	deliveryID, _ := shortid.Generate()

	// Determine if this is a guest order
	isGuest := false
	if req.IsGuestOrder != nil {
		isGuest = *req.IsGuestOrder
	}
	guestPersonalDetailsJSON, _ := json.Marshal(req.GuestPersonalDetails)
	guestDeliveryAddressJSON, _ := json.Marshal(req.GuestDeliveryAddress)

	// Insert order with status defaulted to 'pending'
	_, err := db.Exec(`
	INSERT INTO orders (
		order_id, user_id, is_guest_order, status,
		total_amount, total_discount, delivery_id, guest_personal_details, guest_delivery_address
	) VALUES (?, ?, ?, 'pending', ?, ?, ?, ?, ?)
`, orderID, req.UserID, isGuest, totalAmount, totalDiscount, deliveryID,
		guestPersonalDetailsJSON, guestDeliveryAddressJSON)

	if err != nil {
		return "", "", err
	}
	return orderID, deliveryID, nil
}

// CreateOrderItem inserts a single order item (product) into an order.
//
// This function creates an order line item linking a product/variant to an order
// with quantity and pricing information.
//
// Parameters:
//   - orderID: string - The parent order ID this item belongs to
//   - productID: string - The product being ordered
//   - variantID: string - The specific product variant (size, color, etc.)
//   - quantity: int - Number of units ordered
//   - unitPrice: float64 - Price per unit at the time of order
//
// Returns:
//   - string: Generated order_item_id
//   - error: Database error or nil on success
func CreateOrderItem(db DBExecutor, orderID, productID, variantID string, quantity int, unitPrice float64) (string, error) {
	// Generate unique ID for this order item
	orderItemID, _ := shortid.Generate()

	// Insert order item with product, variant, quantity, and price
	_, err := db.Exec(`
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

// CreateDeliveries creates a delivery record for an order.
//
// This function handles both standard deliveries and warehouse pickup scenarios.
// For warehouse pickups, it validates the warehouse exists and sets appropriate
// courier details and address.
//
// Parameters:
//   - orderID: string - The parent order ID
//   - deliveryID: string - Pre-generated delivery ID from CreateOrder
//   - req: dtos.OrderRequest containing:
//   - DeliveryCharge: Delivery fee amount
//   - DeliveryAddress: Customer's delivery address
//   - storeID: *string - Optional warehouse ID for pickup orders
//
// Returns:
//   - error: "warehouse not found" if storeID is invalid, or database error
//
// Behavior:
//   - If storeID is provided: Validates warehouse, sets pickup details
//   - If storeID is nil/empty: Creates standard delivery with "To be assigned to rider"
//   - Initial delivery status: "Processing"
func CreateDeliveries(db DBExecutor, orderID, deliveryID string, req dtos.OrderRequest, storeID *string) error {
	// Default courier assignment
	courierDetails := "To be assigned to rider"

	// Handle warehouse pickup scenario
	if storeID != nil && *storeID != "" {
		// Validate warehouse exists
		exists, err := RecordExists(db, "warehouses", "warehouse_id = ?", *storeID)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("warehouse not found")
		}

		// Fetch warehouse name for display
		var warehouseName string
		err = db.QueryRow("SELECT name FROM warehouses WHERE warehouse_id = ?", *storeID).Scan(&warehouseName)
		if err != nil {
			return err
		}

		// Set pickup-specific details
		courierDetails = fmt.Sprintf("To be collected from warehouse: %s", warehouseName)
		req.DeliveryAddress = fmt.Sprintf("Warehouse : %s", warehouseName)
	}

	// Insert delivery record with status "Processing"
	_, err := db.Exec(`
		INSERT INTO deliveries (
			delivery_id, order_id, delivery_charge, status, courier_details, delivery_address
		) VALUES (?, ?, ?, 'Processing', ?, ?)
	`, deliveryID, orderID, req.DeliveryCharge, courierDetails, req.DeliveryAddress)

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
//
// GetOrderByUser retrieves a complete order with all details for a specific user.
//
// This function fetches order information including items, delivery details, user address,
// and calculates tax and subtotal. Supports both guest and registered user orders.
//
// Parameters:
//   - orderID: string - The unique order ID to retrieve
//   - userID: string - The user ID to verify order ownership
//
// Returns:
//   - *dtos.Order: Complete order details including:
//   - Order info (ID, totals, discounts, status, timestamps)
//   - Delivery info (status, charge, address)
//   - Payment info (method, status)
//   - Items: Array of products with images, warranties, and review status
//   - UserAddress: Registered user's address (if not guest)
//   - GuestPersonalDetails: Guest contact info (if guest order)
//   - GuestDeliveryAddress: Guest delivery address (if guest order)
//   - Calculated fields: EstimatedTax, SubTotal
//   - error: sql.ErrNoRows if order not found, or database error
//
// Financial Calculations:
//   - EstimatedTax = TotalAmount * tax_rate / 100
//   - SubTotal = TotalAmount - TotalDiscount - DeliveryCharge - EstimatedTax
func GetOrderByUser(db DBExecutor, orderID, userID string) (*dtos.Order, error) {
	// Query order with LEFT JOIN to deliveries for comprehensive data
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
            o.is_guest_order
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.order_id = ? AND o.user_id = ?`
	var ord dtos.Order
	var guestAddrStr, guestDetailsStr, paymentStatusStr string
	var totalAmount float64

	// Scan order and delivery fields
	err := db.QueryRow(query, orderID, userID).Scan(
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
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Calculate estimated tax from system settings
	estimatedTax, _ := GetEstimatedTax(db)
	ord.EstimatedTax = ord.TotalAmount * estimatedTax / 100
	ord.TotalAmount = totalAmount

	// Calculate subtotal (amount before tax, delivery, discounts)
	ord.SubTotal = ord.TotalAmount - ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax

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

	// Fetch order items with product details, images, and warranties
	items, err := getOrderProducts(db, orderID, userID)
	if err != nil {
		return nil, err
	}
	ord.Items = items

	// Fetch user's address for registered users
	address, err := GetUserAddresses(db, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ord.UserAddress = &address
	return &ord, nil
}

// GetAllOrders retrieves all orders with optional status filtering.
//
// This function fetches all orders in the system, optionally filtered by order status.
// Each order includes complete details with items, delivery info, and guest data if applicable.
//
// Parameters:
//   - status: *string - Optional status filter ("pending", "processing", "completed", "cancelled", etc.)
//     If nil, returns all orders regardless of status
//
// Returns:
//   - []dtos.Order: Array of orders with:
//   - Complete order and delivery information
//   - Items array with product details
//   - Guest personal and delivery details (for guest orders)
//   - Calculated tax and subtotal
//   - error: Database error or nil on success
func GetAllOrders(db DBExecutor, status *string) ([]dtos.Order, error) {
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
			o.is_guest_order
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id`

	// Apply status filter if provided
	if status != nil {
		query = baseQuery + " WHERE o.status = ? ORDER BY o.order_id"
		rows, err = db.Query(query, *status)
	} else {
		query = baseQuery + " ORDER BY o.order_id"
		rows, err = db.Query(query)
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
		); err != nil {
			return nil, err
		}

		// Calculate tax and subtotal
		estimatedTax, _ := GetEstimatedTax(db)
		ord.EstimatedTax = ord.TotalAmount * estimatedTax / 100
		ord.TotalAmount = totalAmount
		ord.SubTotal = ord.TotalAmount - ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax

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
func ListOrdersByUser(db DBExecutor, userID string, page, limit int) ([]dtos.Order, *dtos.PaginationMeta, error) {
	// Calculate pagination offset
	offset := (page - 1) * limit

	// Count total orders for this user
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM orders WHERE user_id = ?`, userID).Scan(&total)
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
			o.is_guest_order
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.user_id = ?
        ORDER BY o.created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, userID, limit, offset)
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
		); err != nil {
			return nil, nil, err
		}

		// Calculate tax and subtotal
		estimatedTax, _ := GetEstimatedTax(db)
		ord.EstimatedTax = ord.TotalAmount * estimatedTax / 100
		ord.TotalAmount = totalAmount
		ord.SubTotal = ord.TotalAmount - ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax

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
func ListGuestOrders(db DBExecutor, orderID, email, phone string) (*dtos.Order, error) {
	// Query with email and phone verification
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
			o.is_guest_order
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.order_id = ?
          AND o.guest_personal_details LIKE ?
          AND o.guest_personal_details LIKE ?`

	var ord dtos.Order
	var guestAddrStr, guestDetailsStr, paymentStatusStr string
	var totalAmount float64

	// Search for email and phone in JSON guest details
	err := db.QueryRow(query, orderID, "%"+email+"%", "%"+phone+"%").Scan(
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
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Calculate tax and subtotal
	estimatedTax, _ := GetEstimatedTax(db)
	ord.EstimatedTax = ord.TotalAmount * estimatedTax / 100
	ord.TotalAmount = totalAmount
	ord.SubTotal = ord.TotalAmount - ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax

	// Parse guest JSON fields
	if guestAddrStr != "" {
		_ = json.Unmarshal([]byte(guestAddrStr), &ord.GuestDeliveryAddress)
	}
	if guestDetailsStr != "" {
		_ = json.Unmarshal([]byte(guestDetailsStr), &ord.GuestPersonalDetails)
	}
	if paymentStatusStr != "" {
		ord.PaymentStatus = &paymentStatusStr
	}

	// Fetch items for this order
	items, err := getOrderProducts(db, orderID, "")
	if err != nil {
		return nil, err
	}
	ord.Items = items

	return &ord, nil
}

// UpdateOrderStatus updates order and delivery status fields dynamically.
//
// This function provides flexible updates to order and delivery records,
// updating only the fields provided in the request.
//
// Parameters:
//   - orderID: string - The order ID to update
//   - req: dtos.UpdateOrderStatusRequest containing optional fields:
//   - Status: *string - New order status ("pending", "processing", "completed", "cancelled", etc.)
//   - PaymentMethod: *string - Payment method ("card", "mpesa", "cash", etc.)
//   - PaymentStatus: *string - Payment status ("pending", "paid", "failed", etc.)
//   - DeliveryStatus: *string - Delivery status ("Processing", "Shipped", "Delivered", etc.)
//   - DeliveredAt: *string - Delivery timestamp (ISO format string)
//
// Returns:
//   - error: "order not found" if orderID doesn't exist, or database error
//
// Behavior:
//   - Only updates fields that are non-nil in the request
//   - Automatically sets delivered_at timestamp when DeliveryStatus is "delivered"
//   - Updates orders table for: Status, PaymentMethod, PaymentStatus
//   - Updates deliveries table for: DeliveryStatus, DeliveredAt
//   - Skips database operations if no fields provided for a table
func UpdateOrderStatus(db DBExecutor, orderID string, req dtos.UpdateOrderStatusRequest) error {
	// Validate order existence
	if err := IsOrderThere(db, orderID); err != nil {
		return err
	}

	// Build dynamic UPDATE fields for orders table
	setParts := []string{}
	args := []interface{}{}

	if req.Status != nil {
		setParts = append(setParts, "status = ?")
		args = append(args, *req.Status)
	}
	if req.PaymentMethod != nil {
		setParts = append(setParts, "payment_method = ?")
		args = append(args, *req.PaymentMethod)
	}
	if req.PaymentStatus != nil {
		setParts = append(setParts, "payment_status = ?")
		args = append(args, *req.PaymentStatus)
	}

	// Only run ORDER update if something is actually being updated
	if len(setParts) > 0 {
		query := fmt.Sprintf("UPDATE orders SET %s WHERE order_id = ?", strings.Join(setParts, ", "))
		args = append(args, orderID)

		if _, err := db.Exec(query, args...); err != nil {
			return err
		}
	}

	// Build dynamic UPDATE fields for deliveries table
	deliveryParts := []string{}
	deliveryArgs := []interface{}{}

	// Auto-set delivered_at when status changes to "delivered"
	if req.DeliveryStatus != nil && strings.ToLower(*req.DeliveryStatus) == "delivered" {
		now := time.Now()
		deliveryParts = append(deliveryParts, "delivered_at = ?")
		deliveryArgs = append(deliveryArgs, now)
	}
	// Manual delivered_at override
	if req.DeliveredAt != nil {
		deliveryParts = append(deliveryParts, "delivered_at = ?")
		deliveryTime := StringToTime(*req.DeliveredAt)
		deliveryArgs = append(deliveryArgs, deliveryTime)
	}
	if req.DeliveryStatus != nil {
		deliveryParts = append(deliveryParts, "status = ?")
		deliveryArgs = append(deliveryArgs, *req.DeliveryStatus)
		// if _, err := DB.Exec(
		// 	"UPDATE deliveries SET status = ? WHERE order_id = ?",
		// 	*req.DeliveryStatus,
		// 	orderID,
		// ); err != nil {
		// 	return err
		// }
	}

	// Only run DELIVERY update if something is actually being updated
	if len(deliveryParts) > 0 {
		query := fmt.Sprintf("UPDATE deliveries SET %s WHERE order_id = ?", strings.Join(deliveryParts, ", "))
		deliveryArgs = append(deliveryArgs, orderID)

		if _, err := db.Exec(query, deliveryArgs...); err != nil {
			return err
		}
	}

	return nil
}

// IsOrderThere validates that an order exists in the database.
//
// This is a helper function used before updates or deletions to ensure
// the order exists and provide clear error messages.
//
// Parameters:
//   - id: string - The order_id to validate
//
// Returns:
//   - error: "order not found" if ID doesn't exist, database error, or nil if exists
func IsOrderThere(db DBExecutor, id string) error {
	exists, err := RecordExists(db, "orders", "order_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("order not found")
	}
	return nil
}

// GetOrderByID retrieves a complete order by ID without user ownership check.
//
// This function is typically used by admins to fetch any order. It includes
// complete order details, items, and user address (if not a guest order).
//
// Parameters:
//   - orderID: string - The unique order ID to retrieve
//
// Returns:
//   - *dtos.Order: Complete order with:
//   - Order and delivery information
//   - Payment details
//   - Items with product details, images, warranties
//   - User address (for registered users)
//   - Guest details (for guest orders)
//   - Calculated tax and subtotal
//   - error: "order not found", sql.ErrNoRows, or database error
func GetOrderByID(db DBExecutor, orderID string) (*dtos.Order, error) {
	// Validate order exists
	err := IsOrderThere(db, orderID)
	if err != nil {
		return nil, err
	}

	var userID sql.NullString
	// Query with LEFT JOIN to deliveries
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
            o.created_at,
            o.user_id,
			o.is_guest_order
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.order_id = ?`

	var ord dtos.Order
	var totalAmount float64
	var guestAddrStr, guestDetailsStr string

	// Scan order fields including nullable user_id
	err = DB.QueryRow(query, orderID).Scan(
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
	)

	// Calculate tax and subtotal
	estimatedTax, _ := GetEstimatedTax(DB)
	ord.EstimatedTax = ord.TotalAmount * estimatedTax / 100
	ord.TotalAmount = totalAmount
	ord.SubTotal = ord.TotalAmount - ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax

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

	// Fetch user address if this is a registered user order
	if userID.Valid {
		address, err := GetUserAddresses(db, userID.String)
		if err != nil {
			return nil, err
		}
		ord.UserAddress = &address
		ord.UserID = &userID.String

	}

	// Fetch items with product details
	items, err := getOrderProducts(db, orderID, userID.String)
	if err != nil {
		return nil, err
	}
	ord.Items = items

	return &ord, nil
}

// getOrderProducts is an internal helper that fetches order items with product details.
//
// This function retrieves all products in an order with complete information including
// images, warranties, and review status (if userID provided).
//
// Parameters:
//   - orderID: string - The order whose items to fetch
//   - userID: string - User ID for checking review status (empty string to skip review checks)
//
// Returns:
//   - []dtos.OrderProduct: Array of order items with:
//   - Product details (ID, Name, Description, SKU, CategoryID, Price)
//   - StockQuantity: Actually contains the ordered quantity (not current stock)
//   - Images: Array of product images
//   - Warranty: Product warranty information
//   - IsReviewed: Boolean if user has reviewed this product
//   - ReviewID: The user's review ID if reviewed
//   - error: Database error or nil on success
func getOrderProducts(db DBExecutor, orderID string, userID string) ([]dtos.OrderProduct, error) {
	// Query joins order_items with products to get complete product info
	itemsQuery := `
        SELECT 
            p.product_id,
            p.name,
            p.description,
            p.sku,
            oi.unit_price,
            p.category_id,
            oi.quantity,
            p.search_vector,
            p.created_at,
            p.last_updated_at
        FROM order_items oi
        JOIN products p ON oi.product_id = p.product_id
        WHERE oi.order_id = ?`

	rows, err := db.Query(itemsQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dtos.OrderProduct
	for rows.Next() {
		var item dtos.OrderProduct

		// Scan product details and ordered quantity
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.SKU,
			&item.Price,
			&item.CategoryID,
			&item.StockQuantity, // Actually ordered quantity from oi.quantity
			&item.SearchVector,
			&item.CreatedAt,
			&item.LastUpdated,
		); err != nil {
			return nil, err
		}

		// Fetch product images
		images, err := fetchProductImages(db, item.ID)
		if err != nil {
			return nil, err
		}
		item.Images = images

		// Fetch product warranty
		warranty, err := FetchProductWarranties(db, item.ID)
		if err != nil {
			return nil, err
		}
		item.Warranty = &warranty

		// Check if user has reviewed this product
		if userID != "" {
			item.IsReviewed, item.ReviewID = checkIfReviewed(db, item.ID, userID)
		}

		items = append(items, item)
	}

	return items, nil
}

// checkIfReviewed is an internal helper to check if a user has reviewed a product.
//
// This function queries the product_reviews table to determine if the user
// has already submitted a review for the given product.
//
// Parameters:
//   - productID: string - The product to check
//   - userID: string - The user to check
//
// Returns:
//   - bool: true if user has reviewed the product, false otherwise
//   - string: The review_id if reviewed, empty string otherwise
func checkIfReviewed(db DBExecutor, productID, userID string) (bool, string) {
	query := `SELECT review_id FROM product_reviews WHERE product_id = ? AND user_id = ?`
	var reviewID string
	err := db.QueryRow(query, productID, userID).Scan(&reviewID)
	if err != nil {
		return false, ""
	}
	if reviewID != "" {
		return true, reviewID
	}
	return false, ""
}

// AdminOrderParameters defines filtering and pagination options for admin order listing.
//
// This structure supports comprehensive order filtering with multiple criteria,
// time range selections, full-text search, and pagination.
//
// Fields:
//   - OrderStatus: string - Filter by order status ("pending", "completed", etc.)
//   - PaymentStatus: string - Filter by payment status ("paid", "pending", etc.)
//   - DeliveryStatus: string - Filter by delivery status ("Delivered", "Shipped", etc.)
//   - PaymentMethod: string - Filter by payment method ("card", "mpesa", "cash", etc.)
//   - TimeRange: string - Predefined time range:
//   - "today": Orders from today only
//   - "this_week": Orders from current week (Monday-Sunday)
//   - "this_month": Orders from current month
//   - "last_month": Orders from previous month
//   - "this_year": Orders from current year
//   - OrderID: string - Exact order ID match
//   - Q: string - Full-text search across:
//   - Guest personal details
//   - Order ID, Delivery ID
//   - Delivery addresses (guest and registered)
//   - Payment method
//   - User email, phone
//   - User names (supports "first last" and "last first" order)
//   - StartDate: string - Custom date range start (ISO format)
//   - EndDate: string - Custom date range end (ISO format)
//   - Page: int - Page number (1-indexed)
//   - Limit: int - Orders per page
type AdminOrderParameters struct {
	OrderStatus    string
	PaymentStatus  string
	DeliveryStatus string
	PaymentMethod  string
	TimeRange      string
	OrderID        string
	Q              string
	Page           int
	Limit          int
	StartDate      string
	EndDate        string
	Past           string
	RiderID        string
	UserID         string
}

// ListOrdersByAdmin retrieves paginated orders with advanced filtering for admin dashboards.
//
// This function provides comprehensive order management capabilities including:
//   - Multiple status filters (order, payment, delivery)
//   - Predefined and custom time ranges
//   - Full-text search across orders, users, and addresses
//   - Pagination with metadata
//
// Parameters:
//   - params: AdminOrderParameters with filter and pagination options
//
// Returns:
//   - []dtos.AdminOrder: Array of orders with:
//   - Complete order and delivery information
//   - User details (if registered user)
//   - Items array with full product details
//   - ItemsCount: Number of items in order
//   - DeliveredAt: Actual delivery timestamp
//   - Guest details (if guest order)
//   - *dtos.PaginationMeta: Pagination with HasNext, HasPrev, TotalItems, TotalPages
//   - error: Database error or nil on success
//
// Search Behavior:
//   - Searches across guest details, order IDs, addresses, payment method, user info
//   - Supports name search in both "first last" and "last first" order
//   - Uses LIKE queries with wildcards for flexible matching
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
//   - Args: []interface{} - Corresponding query arguments
//   - JoinUsers: bool - true if users table JOIN required
//   - JoinDeliveries: bool - true if deliveries table JOIN required
//   - JoinRiderOrders: bool - true if rider_orders table JOIN required
//   - JoinRiderUsers: bool - true if rider users table JOIN required
type OrderConditions struct {
	Conditions      []string
	Args            []interface{}
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
	var args []interface{}
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
//   - []interface{}: Query arguments (conditions + limit + offset)
func buildAdminOrderQuery(conds OrderConditions, limit, offset int) (string, []interface{}) {
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
	args := append(append([]interface{}{}, conds.Args...), limit, offset)
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
	ord.SubTotal = ord.TotalAmount - ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax
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
func ReleaseOrder(db DBExecutor, orderID string) error {
	query := `DELETE FROM held_orders WHERE order_id = ?`
	_, err := db.Exec(query, orderID)
	return err
}

// DeductProductStock decreases product stock quantity after order confirmation.
//
// This function subtracts the ordered quantity from the product's stock.
// Used when orders are confirmed/paid to reflect sold inventory.
//
// Parameters:
//   - productID: string - The product whose stock to decrease
//   - quantity: int - Amount to deduct from stock
//
// Returns:
//   - error: Database error or nil on success
func DeductProductStock(db DBExecutor, productID string, quantity int) error {
	// Atomic stock decrement
	query := `
		UPDATE products
		SET stock_quantity = stock_quantity - ?
		WHERE product_id = ?`
	_, err := db.Exec(query, quantity, productID)
	return err
}

// MarkOrderNotificationSent updates order notification tracking status.
//
// This function marks that a notification (email, SMS, etc.) has been sent
// for an order to prevent duplicate notifications.
//
// Parameters:
//   - orderID: string - The order whose notification status to update
//   - status: string - New notification status ("sent", "failed", etc.)
//
// Returns:
//   - error: Database error or nil on success
func MarkOrderNotificationSent(db DBExecutor, orderID, status string) error {
	query := `UPDATE order_notifications SET status = ? WHERE order_id = ?`
	_, err := db.Exec(query, status, orderID)
	return err
}

// UpdateOrderTotals updates the financial totals for an order.
//
// This function modifies the total_amount and total_discount fields,
// useful when order items are modified or pricing adjustments are made.
//
// Parameters:
//   - order: *dtos.Order containing:
//   - OrderID: The order to update
//   - TotalAmount: New total order amount
//   - TotalDiscount: New total discount applied
//
// Returns:
//   - error: Database error or nil on success
func UpdateOrderTotals(db DBExecutor, order *dtos.Order) error {
	query := `UPDATE orders SET total_amount = ?, total_discount = ? WHERE order_id = ?`
	_, err := db.Exec(query, order.TotalAmount, order.TotalDiscount, order.OrderID)
	return err
}

// Mark order as paid
// Parameters:
// - orderID: string - The order to mark as paid
// Returns:
// - error: Database error or nil on success
func MarkOrderAsPaid(db DBExecutor, orderID string) error {
	query := `UPDATE orders SET payment_status = 'PAID' WHERE order_id = ?`
	_, err := db.Exec(query, orderID)
	return err
}

// Assign order to a rider
// Parameters:
// - orderID: string - The order to assign
// - riderID: string - The rider to assign the order to
// Returns:
// - error: Database error or nil on success
func AssignOrderToRider(db DBExecutor, req dtos.AssignOrderToRiderRequest) error {
	//check if the rider exists and is active
	var existingUser string
	err := db.QueryRow(`SELECT user_id FROM users WHERE user_id = ? AND status = 'active'`, req.RiderID).Scan(&existingUser)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("rider not found or inactive")
		}
		return err
	}
	//first check if order exists and is not already assigned
	var existingRiderID sql.NullString
	err = db.QueryRow(`SELECT rider_id FROM rider_orders WHERE order_id = ?`, req.OrderID).Scan(&existingRiderID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	//if order already assigned to a rider, update the assignment
	if existingRiderID.Valid {
		_, err = db.Exec(`UPDATE rider_orders SET rider_id = ? WHERE order_id = ?`, req.RiderID, req.OrderID)
	} else {
		riderOrderID, _ := shortid.Generate()
		//if order not assigned, create new assignment
		_, err = db.Exec(`INSERT INTO rider_orders (rider_order_id, order_id, rider_id) VALUES (?, ?, ?)`, riderOrderID, req.OrderID, req.RiderID)
	}
	return err
}

// Model to update order status(mostly for delivery) by a rider
// First check if the order is assigned to the rider, then update the delivery status
// Parameters:
// - orderID: string - The order to update
// - riderID: string - The rider updating the status
// - deliveryStatus: string - The new delivery status
// Returns:
// - error: Database error or nil on success
func UpdateOrderByRider(db DBExecutor, req dtos.UpdateOrderDeliveryStatusRequest, riderID string) error {
	// Check if the order is assigned to the rider
	var existingRiderID sql.NullString
	err := db.QueryRow(`SELECT rider_id FROM rider_orders WHERE order_id = ?`, req.OrderID).Scan(&existingRiderID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if !existingRiderID.Valid || existingRiderID.String != riderID {
		return fmt.Errorf("order not assigned to this rider")
	}
	// Update the delivery status
	query := `UPDATE deliveries SET status = ? WHERE order_id = ?`
	_, err = db.Exec(query, req.DeliveryStatus, req.OrderID)
	return err
}
