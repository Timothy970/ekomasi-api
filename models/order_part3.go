package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func ListGuestOrders(db DBExecutor, orderID, email, phone string, tenantID int) (*dtos.Order, error) {
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
			o.is_guest_order,
			o.source
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.order_id = ?
          AND o.guest_personal_details LIKE ?
          AND o.guest_personal_details LIKE ?
          AND o.tenant_id = ?`

	var ord dtos.Order
	var guestAddrStr, guestDetailsStr, paymentStatusStr string
	var totalAmount float64

	// Search for email and phone in JSON guest details
	err := db.QueryRow(query, orderID, "%"+email+"%", "%"+phone+"%", tenantID).Scan(
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
	args := []any{}

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
	deliveryArgs := []any{}

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
