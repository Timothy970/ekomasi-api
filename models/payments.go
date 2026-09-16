// Package models provides data access functions for the Ekomasi backend.
//
// This file (payments.go) contains payment processing and management functions including:
//   - M-Pesa STK Push integration (StoreStkResponse, UpdateStkResponse)
//   - Payment CRUD operations (CreatePayment, GetPaymentByID, ListPayments, UpdatePayment, DeletePayment)
//   - Order/delivery status updates after payment (UpdateDeliveryOrderTables)
//   - Voucher payment processing (UpdateVoucherOrderTables, SetVoucherAsRedeemed, AddVoucherHistory)
//   - Refund management (AddRefundRequest, ProcessRefund, ListRefunds, GetRefundByID)
//   - Payment options configuration (CreatePaymentOption, ListPaymentOptions, UpdatePaymentOption, DeletePaymentOption)
//   - Checkout request tracking (GetCheckoutRequestIDByOrderID)
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"strings"
)

var nopayment = "payment not found"
var paymentid = "payment_id = ?"

// StoreStkResponse stores M-Pesa STK Push response for tracking payment status.
//
// This function records the initial STK Push request details including checkout
// and merchant request IDs for both product orders and voucher purchases.
//
// Parameters:
//   - response: map[string]any containing M-Pesa API response with:
//   - CheckoutRequestID: Unique checkout request identifier from M-Pesa
//   - MerchantRequestID: Merchant's request identifier from M-Pesa
//   - req: dtos.MpesaRequest containing:
//   - OrderID: The order or voucher order ID
//   - DeliveryID: Delivery ID (for product orders only)
//   - Amount: Payment amount
//   - Type: "voucher" or "product" to determine storage logic
//
// Returns:
//   - error: Database error or nil on success
//
// Behavior:
//   - Type "voucher": Stores without delivery_id, sets type="VOUCHER"
//   - Type "product": Stores with delivery_id, sets type="PRODUCT"
//   - Initial status: "PROCESSING"
func StoreStkResponse(db DBExecutor, response map[string]any, req dtos.MpesaRequest) error {
	// Extract M-Pesa response IDs safely from interface map
	checkoutRequestID, _ := response["CheckoutRequestID"].(string)
	merchantRequestID, _ := response["MerchantRequestID"].(string)
	var err error

	// Handle voucher payment (no delivery involved)
	if strings.ToLower(req.Type) == "voucher" {
		_, err = db.Exec(`
		INSERT INTO stk_push_responses (
			order_id, amount, checkout_request_id,
			merchant_request_id, status, type
		)
		VALUES (?, ?, ?, ?, 'PROCESSING', "VOUCHER")
	`, req.OrderID, req.Amount, checkoutRequestID, merchantRequestID)

		if err != nil {
			return err
		}
		return nil
	} else {
		// Handle product order payment (includes delivery)
		_, err = db.Exec(`
		INSERT INTO stk_push_responses (
			order_id, delivery_id, amount, checkout_request_id,
			merchant_request_id, status, type
		)
		VALUES (?, ?, ?, ?, ?, 'PROCESSING', "PRODUCT")
	`, req.OrderID, req.DeliveryID, req.Amount, checkoutRequestID, merchantRequestID)

		if err != nil {
			return err
		}

		return nil
	}
}

// UpdateStkResponse updates STK Push payment status and retrieves order details.
//
// This function updates the payment status and returns order/delivery IDs for
// further processing (e.g., updating order status, sending notifications).
//
// Parameters:
//   - checkoutRequestID: string - M-Pesa checkout request ID to update
//   - status: string - New payment status ("COMPLETED", "FAILED", etc.)
//
// Returns:
//   - string: delivery_id (empty for voucher orders)
//   - string: order_id
//   - string: type ("PRODUCT" or "VOUCHER")
//   - error: Database error or nil on success
func UpdateStkResponse(db DBExecutor, checkoutRequestID string, status string) (string, string, string, error) {
	// 1. Update payment status in stk_push_responses
	_, err := db.Exec(`
		UPDATE stk_push_responses SET status = ?
		WHERE checkout_request_id = ?
	`, status, checkoutRequestID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to update status: %w", err)
	}

	// 2. Retrieve delivery_id, order_id, and type for further processing
	var deliveryID, orderID, orderType sql.NullString
	err = db.QueryRow(`
		SELECT delivery_id, order_id, type
		FROM stk_push_responses
		WHERE checkout_request_id = ? LIMIT 1
	`, checkoutRequestID).Scan(
		&deliveryID, &orderID, &orderType,
	)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to fetch delivery/order ids: %w", err)
	}

	// Convert NullString -> string (empty if NULL)
	return nullToString(deliveryID), nullToString(orderID), nullToString(orderType), nil
}

// nullToString converts sql.NullString to regular string.
//
// This is a helper function to safely handle nullable database fields.
//
// Parameters:
//   - ns: sql.NullString - Nullable string from database
//
// Returns:
//   - string: The string value if valid, empty string if NULL
func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// UpdateDeliveryOrderTables updates order status after payment completion.
//
// This function sets order status and payment status based on payment outcome.
// Used as part of payment webhook processing flow.
//
// Parameters:
//   - deliveryID: string - Delivery ID (currently not used but reserved for future)
//   - orderId: string - Order ID to update
//   - status: string - Payment status from M-Pesa ("COMPLETED" or "FAILED")
//
// Returns:
//   - error: Database error or nil on success
//
// Status Mapping:
//   - "COMPLETED" -> orderStatus = "PAID"
//   - Any other status -> orderStatus = "FAILED"
func UpdateDeliveryOrderTables(db DBExecutor, deliveryID, orderId string, status string) error {
	// status passed is either "COMPLETED" or "FAILED" based on M-Pesa response
	orderStatus := "PROCESSING"
	paymentStatus := "PAID"
	if status != "COMPLETED" {
		orderStatus = "FAILED"
		paymentStatus = "FAILED"
	}

	// Update order status and payment status
	_, err := db.Exec(`
		UPDATE orders SET status = ?, payment_status = ?
		WHERE order_id = ?
	`, orderStatus, paymentStatus, orderId)
	if err != nil {
		return fmt.Errorf("failed to update orders table: %w", err)
	}

	// Delivery status update disabled (commented out)
	// _, err = DB.Exec(`
	// 	UPDATE deliveries SET status = 'PAID'
	// 	WHERE delivery_id = ?
	// `, deliveryID)
	// if err != nil {
	// 	return fmt.Errorf("failed to update deliveries table: %w", err)
	// }

	return nil
}

// UpdateVoucherOrderTables updates voucher order and activates the voucher after payment.
//
// This function handles the post-payment workflow for voucher purchases by:
//  1. Updating the voucher order status
//  2. Activating the purchased voucher
//
// Parameters:
//   - orderId: string - The voucher_order_id to update
//   - status: string - New order status ("COMPLETED", "PAID", etc.)
//
// Returns:
//   - error: "order not found", database error, or nil on success
//
// Workflow:
//  1. Update voucher_orders.status
//  2. Fetch voucher_id from the order
//  3. Activate the voucher (set is_active = true)
func UpdateVoucherOrderTables(db DBExecutor, orderId string, status string) error {
	// Update voucher order status
	_, err := db.Exec(`
    UPDATE voucher_orders SET status = ?
    WHERE voucher_order_id = ?
`, status, orderId)
	if err != nil {
		return fmt.Errorf("failed to update voucher order status: %w", err)
	}

	// Fetch the voucher_id associated with this order
	var voucherID string
	err = db.QueryRow(`
	SELECT voucher_id FROM voucher_orders WHERE voucher_order_id = ?
`, orderId).Scan(&voucherID)
	if err != nil {
		return fmt.Errorf("failed to fetch voucher_id: %w", err)
	}

	// Activate the voucher
	isActive := true
	_, err = db.Exec(`
		UPDATE vouchers SET is_active = ?
		WHERE voucher_id = ?
	`, isActive, voucherID)
	if err != nil {
		return fmt.Errorf("failed to update deliveries table: %w", err)
	}

	return nil
}

// SetVoucherAsRedeemed marks a voucher as redeemed after use.
//
// This function sets the is_redeemed flag to prevent double-spending of vouchers.
//
// Parameters:
//   - code: string - The voucher code to mark as redeemed
//
// Returns:
//   - error: Database error or nil on success
func SetVoucherAsRedeemed(db DBExecutor, code string) error {
	// Mark voucher as redeemed
	_, err := db.Exec(`
	UPDATE vouchers SET is_redeemed = true WHERE code = ?`, code)
	if err != nil {
		return err
	}
	return nil
}

// CreatePayment records a new payment transaction in the database.
//
// This function creates a payment record linked to an order with transaction tracking.
// Validates order existence and transaction ID uniqueness before insertion.
//
// Parameters:
//   - p: dtos.Payment containing:
//   - OrderID: The order being paid for (validated for existence)
//   - Amount: Payment amount
//   - VoucherID: Optional voucher used in payment
//   - Status: Payment status (defaults to "PENDING" if empty)
//   - PaymentMethod: Method used ("card", "mpesa", "cash", etc.)
//   - TransactionID: Unique transaction identifier (validated for uniqueness)
//
// Returns:
