// Package models provides data access functions for the Adenzo backend.
//
// This file (mpesa.go) contains M-Pesa payment integration functions including:
//   - Refund processing with order status updates
//   - Product stock restoration after refunds
//   - Inventory management for refunded items
package models

import "adenzo_backend/dtos"

// HandleMpesaMoneyReturnRefunds processes refunds for M-Pesa payments.
//
// This function handles the complete refund workflow including:
//   - Updating order status to "Refunded"
//   - Restocking all refunded items back to inventory
//
// Parameters:
//   - order: dtos.Order containing the order details and items to refund
//
// Returns:
//   - error: Order status update error, stock update error, or nil on success
//
// Workflow:
//   1. Change order status to "Refunded"
//   2. Iterate through all order items
//   3. Restore stock quantity for each item
//
func HandleMpesaMoneyReturnRefunds(order dtos.Order) error {
	// Update order status to refunded
	status := "REFUNDED"
	deliveryStatus := "REFUNDED"
	paymentStatus := "REFUNDED"
	err := UpdateOrderStatus(DB, order.OrderID, dtos.UpdateOrderStatusRequest{
		Status:         &status,
		PaymentStatus:  &paymentStatus,
		DeliveryStatus: &deliveryStatus,
	})
	if err != nil {
		return err
	}

	// Iterate through all order items to restore stock
	for _, item := range order.Items {
		// Restock each item by adding back the quantity that was sold
		err = UpdateProductStock(item.ID, item.StockQuantity)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateProductStock increases product stock quantity after a refund.
//
// This function adds the specified quantity back to the product's inventory.
// It's typically called during refund processing to restore stock levels.
//
// Parameters:
//   - productID: string - The unique identifier of the product to restock
//   - quantity: int - The quantity to add back to stock
//
// Returns:
//   - error: Database execution error or nil on success
//
// SQL Operation:
//   - Uses atomic increment: stock_quantity = stock_quantity + ?
//   - Prevents race conditions with direct SQL increment
func UpdateProductStock(productID string, quantity int) error {
	// Execute atomic stock increment query
	_, err := DB.Exec(`
		UPDATE products SET stock_quantity = stock_quantity + ? WHERE product_id = ?`,
		quantity, productID,
	)
	return err
}
