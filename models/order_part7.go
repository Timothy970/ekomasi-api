package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"

	"github.com/teris-io/shortid"
)

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
	if err != nil {
		return err
	}
	// update product_variant_combinations table if the product has variants
	_, err = db.Exec(`
	UPDATE product_variant_combinations  SET stock_quantity = stock_quantity - ?
	WHERE product_id = ?`, quantity, productID)
	if err != nil {
		return err
	}
	return nil
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

// helper function to see if order belongs to a rider
func IsOrderAssignedToRider(db DBExecutor, orderID, riderID string) (bool, error) {
	var existingRiderID sql.NullString
	err := db.QueryRow(`SELECT rider_id FROM rider_orders WHERE order_id = ?`, orderID).Scan(&existingRiderID)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if existingRiderID.Valid && existingRiderID.String == riderID {
		return true, nil
	}
	return false, nil
}
