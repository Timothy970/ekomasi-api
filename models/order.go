package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/teris-io/shortid"
)

func CreateOrder(db DBExecutor, req dtos.OrderRequest, totalAmount, totalDiscount string, tenantID int) (string, string, error) {
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
		total_amount, total_discount, delivery_id, guest_personal_details, guest_delivery_address, source, tenant_id
	) VALUES (?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?)
`, orderID, req.UserID, isGuest, totalAmount, totalDiscount, deliveryID,
		guestPersonalDetailsJSON, guestDeliveryAddressJSON, req.OrderSource, tenantID)

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
func CreateOrderItem(db DBExecutor, orderID, productID string, variantSKU *string, quantity int, unitPrice float64) (string, error) {
	// Generate unique ID for this order item
	orderItemID, _ := shortid.Generate()

	// Insert order item with product, variant, quantity, and price
	_, err := db.Exec(`
		INSERT INTO order_items (
			order_item_id, order_id, product_id, variant_sku,
			quantity, unit_price
		) VALUES (?, ?, ?, ?, ?, ?)
	`, orderItemID, orderID, productID, variantSKU, quantity, unitPrice)

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
            o.is_guest_order,
			o.source
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
		&ord.OrderSource,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
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
