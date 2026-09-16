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
	"math"

	"github.com/teris-io/shortid"
)

// - error: "order not found", "transaction ID already exists", or database error
func CreatePayment(db DBExecutor, p dtos.Payment) error {
	// Validate order exists
	err := IsOrderThere(db, p.OrderID)
	if err != nil {
		return err
	}

	// Validate transaction ID is unique
	err = isTransactionIDUnique(db, p.TransactionID)
	if err != nil {
		return err
	}

	// Generate unique payment ID
	paymentID, _ := shortid.Generate()

	// Set default status if not provided
	status := "PENDING"
	if p.Status != "" {
		status = p.Status
	}

	// Insert payment record
	query := `
		INSERT INTO payments (payment_id, order_id, amount, voucher_id, status, payment_method, transaction_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, paymentID, p.OrderID, p.Amount, p.VoucherID, status, p.PaymentMethod, p.TransactionID)
	return err
}

// isPaymentThere validates that a payment exists in the database.
//
// This is an internal helper function used before updates or retrievals.
//
// Parameters:
//   - id: string - The payment_id to validate
//
// Returns:
//   - error: "payment not found" if ID doesn't exist, database error, or nil if exists
func isPaymentThere(db DBExecutor, id string) error {
	exists, err := RecordExists(db, "payments", paymentid, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(nopayment)
	}
	return nil
}

// GetPaymentByID retrieves a single payment by its ID.
//
// This function fetches complete payment details including order, amount, status,
// payment method, and transaction information.
//
// Parameters:
//   - paymentID: string - The unique payment_id to retrieve
//
// Returns:
//   - *dtos.Payment: Payment details including:
//   - PaymentID, OrderID, Amount, VoucherID
//   - Status, PaymentMethod, TransactionID
//   - CreatedAt timestamp
//   - error: "payment not found", sql.ErrNoRows, or database error
func GetPaymentByID(db DBExecutor, paymentID string) (*dtos.Payment, error) {
	// Validate payment exists
	err := isPaymentThere(db, paymentID)
	if err != nil {
		return nil, err
	}

	// Query payment details
	query := `
		SELECT payment_id, order_id, amount, voucher_id, status, payment_method, transaction_id, created_at
		FROM payments
		WHERE payment_id = ?`

	var p dtos.Payment
	err = db.QueryRow(query, paymentID).Scan(
		&p.PaymentID, &p.OrderID, &p.Amount, &p.VoucherID,
		&p.Status, &p.PaymentMethod, &p.TransactionID, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// ListPayments retrieves paginated payments ordered by most recent first.
//
// This function returns all payment records with pagination metadata,
// useful for admin dashboards and payment history displays.
//
// Parameters:
//   - page: int - Page number (1-indexed)
//   - limit: int - Number of payments per page
//
// Returns:
//   - []dtos.Payment: Array of payments ordered by created_at DESC
//   - *dtos.PaginationMeta: Pagination metadata with:
//   - Page, Size, TotalItems, TotalPages
//   - HasPrev, HasNext flags
//   - error: Database error or nil on success
func ListPayments(db DBExecutor, page, limit int) ([]dtos.Payment, *dtos.PaginationMeta, error) {
	// Calculate pagination offset
	offset := (page - 1) * limit

	// Count total payments
	var totalItems int
	err := db.QueryRow("SELECT COUNT(*) FROM payments").Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Fetch paginated payments ordered by most recent
	query := `
		SELECT payment_id, order_id, amount, voucher_id, status, payment_method, transaction_id, created_at
		FROM payments
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan payment rows
	var payments []dtos.Payment
	for rows.Next() {
		var p dtos.Payment
		err := rows.Scan(&p.PaymentID, &p.OrderID, &p.Amount, &p.VoucherID, &p.Status, &p.PaymentMethod, &p.TransactionID, &p.CreatedAt)
		if err != nil {
			return nil, nil, err
		}
		payments = append(payments, p)
	}

	// Build pagination metadata
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: int(math.Ceil(float64(totalItems) / float64(limit))),
		HasPrev:    page > 1,
		HasNext:    page*limit < totalItems,
	}

	return payments, meta, nil
}

// UpdatePayment updates the status of an existing payment.
//
// This function modifies only the payment status, typically used to mark
// payments as "COMPLETED", "FAILED", "REFUNDED", etc.
//
// Parameters:
//   - p: dtos.PaymentUpdate containing:
//   - Status: New payment status
//   - paymentID: string - The payment_id to update
//
// Returns:
//   - error: "payment not found" if ID doesn't exist, or database error
func UpdatePayment(db DBExecutor, p dtos.PaymentUpdate, paymentID string) error {
	// Validate payment exists
	err := isPaymentThere(db, paymentID)
	if err != nil {
		return err
	}

	// Update payment status
	query := `
		UPDATE payments
		SET status = ?
		WHERE payment_id = ?`

	_, err = db.Exec(query, p.Status, paymentID)
	return err
}

// DeletePayment permanently removes a payment record from the database.
//
// This function validates payment existence before deletion.
// Use with caution as this is a permanent delete operation.
//
// Parameters:
//   - paymentID: string - The payment_id to delete
//
// Returns:
//   - error: "payment not found" if ID doesn't exist, or database error
//
// Warning: This is a hard delete with no recovery. Consider soft-delete for production.
func DeletePayment(db DBExecutor, paymentID string) error {
	// Validate payment exists
	err := isPaymentThere(db, paymentID)
	if err != nil {
		return err
	}

	// Permanently delete payment
	query := `DELETE FROM payments WHERE payment_id = ?`
	_, err = db.Exec(query, paymentID)
	return err
}

// AddRefundRequest creates a new refund request for an order.
//
// This function validates order existence and creates a refund record
// with the requested amount and reason.
//
// Parameters:
//   - refund: dtos.Refund containing:
//   - OrderID: Order to refund (validated for existence)
//   - Amount: Refund amount requested
//   - Reason: Reason for refund request
//   - Status: Initial refund status ("pending", "approved", etc.)
//   - userID: string - User requesting the refund
//
// Returns:
//   - error: "order not found" or database error
func AddRefundRequest(db DBExecutor, refund dtos.Refund, userID string) error {
	// Validate order exists
	exists, err := RecordExists(db, "orders", "order_id = ?", refund.OrderID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("order not found")
	}

	// Insert refund request
	query := `INSERT INTO refunds (user_id, order_id, amount, reason, status)
				  VALUES (?, ?, ?, ?, ?)
	`

	_, err = db.Exec(query, userID, refund.OrderID, refund.Amount, refund.Reason, refund.Status)
	if err != nil {
		return err
	}

	return nil
}

// ProcessRefund updates the status of a refund request.
//
// This function is typically used by admins to approve/reject refund requests.
