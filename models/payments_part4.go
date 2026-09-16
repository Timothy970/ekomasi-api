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
	"errors"
)

// isPaymentOptionThere validates if a payment option exists by ID.
//
// This is an internal helper function used before updates, retrievals, or deletions.
//
// Parameters:
//   - id: string - The payment option id to check
//
// Returns:
//   - error: "payment option not found" if missing, nil if exists, or database error
func isPaymentOptionThere(db DBExecutor, id string) error {
	// Check record existence using utility function
	exists, err := RecordExists(db, "payment_options", "id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("payment option not found")
	}
	return nil
}

// GetPaymentOptionByID retrieves a single payment option by its ID.
//
// This function fetches complete payment option details including configs.
//
// Parameters:
//   - id: string - The payment option id to retrieve
//
// Returns:
//   - *dtos.PaymentOption: Payment option details including:
//   - ID, Name, Type
//   - Configs (raw JSON string)
//   - IsActive, CreatedAt
//   - error: "payment option not found" or database error
func GetPaymentOptionByID(db DBExecutor, id string) (*dtos.PaymentOption, error) {
	// Validate payment option exists
	err := isPaymentOptionThere(db, id)
	if err != nil {
		return nil, err
	}

	// Query payment option details
	var p dtos.PaymentOption
	err = db.QueryRow(`SELECT id, name, type, config_json, is_active, created_at
		FROM payment_options WHERE id = ?`, id).Scan(&p.ID, &p.Name, &p.Type, &p.Configs, &p.IsActive, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePaymentOption updates a payment option with dynamic config handling.
//
// This function always updates name, type, and is_active fields.
// The config_json field is only updated if a non-nil value is provided.
//
// Parameters:
//   - id: string - The payment option id to update
//   - paymentOption: dtos.PaymentOptionUpdate containing:
//   - Name: New display name (required)
//   - Type: New payment type (required)
//   - IsActive: New active status (required)
//   - Configs: Optional JSON config (only updated if not nil)
//
// Returns:
//   - error: "payment option not found" or database error
func UpdatePaymentOption(db DBExecutor, id string, paymentOption dtos.PaymentOptionUpdate) error {
	// Validate payment option exists
	err := isPaymentOptionThere(db, id)
	if err != nil {
		return err
	}

	// Build dynamic update query (always update name, type, status)
	query := `
		UPDATE payment_options
		SET name = ?, type = ?, is_active = ?`
	var args []interface{}
	args = append(args, paymentOption.Name, paymentOption.Type, paymentOption.IsActive)

	// Conditionally update configs if provided
	if paymentOption.Configs != nil {
		query += `, config_json = ?`
		args = append(args, paymentOption.Configs)
	}

	// Add WHERE clause
	query += ` WHERE id = ?`
	args = append(args, id)

	// Execute update
	_, err = db.Exec(query, args...)
	return err
}

// DeletePaymentOption permanently removes a payment option from the database.
//
// Warning: This is a hard delete operation. Consider soft delete (is_active = false)
// for maintaining historical data and audit trails.
//
// Parameters:
//   - id: string - The payment option id to delete
//
// Returns:
//   - error: "payment option not found" or database error
func DeletePaymentOption(db DBExecutor, id string) error {
	// Validate payment option exists
	err := isPaymentOptionThere(db, id)
	if err != nil {
		return err
	}

	// Hard delete payment option
	query := `DELETE FROM payment_options WHERE id = ?`
	_, err = db.Exec(query, id)
	return err
}

// GetCheckoutRequestIDByOrderID retrieves the most recent M-Pesa checkout request ID for an order.
//
// This function is used to query M-Pesa payment status by finding the checkout_request_id
// associated with an order. Returns the most recent checkout request if multiple exist.
//
// Parameters:
//   - orderID: string - The order_id to lookup
//
// Returns:
//   - string: The checkout_request_id from M-Pesa STK Push
//   - error: "checkout request ID not found" if no STK Push record exists, or database error
func GetCheckoutRequestIDByOrderID(db DBExecutor, orderID string) (string, error) {
	// Query most recent checkout request for order
	var checkoutRequestID sql.NullString
	err := db.QueryRow(`
		SELECT checkout_request_id
		FROM stk_push_responses
		WHERE order_id = ? ORDER BY created_at DESC LIMIT 1`, orderID).Scan(&checkoutRequestID)
	if err != nil {
		return "", err
	}

	// Handle nullable checkout_request_id
	if !checkoutRequestID.Valid {
		return "", errors.New("checkout request ID not found")
	}

	return checkoutRequestID.String, nil
}
