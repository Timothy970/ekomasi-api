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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/teris-io/shortid"
)

// Parameters:
//   - req: dtos.RefundPayload containing:
//   - Status: New refund status ("approved", "rejected", "processed", etc.)
//   - refundID: string - The refund ID to update
//
// Returns:
//   - error: "refund not found" or database error
func ProcessRefund(db DBExecutor, req dtos.RefundPayload, refundID string) error {
	// Validate refund exists
	exists, err := RecordExists(db, "refunds", whereID, refundID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("refund not found")
	}

	// Update refund status
	query := `UPDATE refunds SET status = ? WHERE id = ?`
	_, err = db.Exec(query, req.Status, refundID)
	return err
}

// ListRefunds retrieves paginated refund requests with automatic input validation.
//
// This function returns all refund records ordered by most recent first,
// with automatic correction of invalid page/size values.
//
// Parameters:
//   - page: int - Page number (automatically adjusted to minimum 1 if invalid)
//   - size: int - Refunds per page (automatically adjusted to minimum 10 if invalid)
//
// Returns:
//   - []dtos.Refund: Array of refunds containing:
//   - ID, UserID, OrderID, Amount
//   - Status, CreatedAt
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func ListRefunds(db DBExecutor, page, size int) ([]dtos.Refund, *dtos.PaginationMeta, error) {
	// Validate and correct page number (minimum 1)
	if page < 1 {
		page = 1
	}
	// Validate and correct size (minimum 10)
	if size < 1 {
		size = 10
	}
	// Calculate pagination offset
	offset := (page - 1) * size

	// Count total refunds
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	// Fetch paginated refunds ordered by most recent
	rows, err := db.Query(`
		SELECT id, user_id, order_id, amount, status, created_at
		FROM refunds
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan refund rows
	var refunds []dtos.Refund
	for rows.Next() {
		var refund dtos.Refund
		if err := rows.Scan(&refund.ID, &refund.UserID, &refund.OrderID, &refund.Amount, &refund.Status, &refund.CreatedAt); err != nil {
			return nil, nil, err
		}
		refunds = append(refunds, refund)
	}

	// Build pagination metadata
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return refunds, meta, nil
}

// GetRefundByID retrieves a single refund by user ID.
//
// may be misleading - it returns the first refund for the given user.
//
// Parameters:
//   - id: int - The user_id whose refund to retrieve
//
// Returns:
//   - *dtos.Refund: Refund details or nil if not found
//   - error: "refund not found" if no refund exists for user, or database error
//
// Warning: This retrieves by user_id, not refund id. Consider renaming or
//
//	adding a separate function for refund id lookup.
func GetRefundByID(db DBExecutor, id int) (*dtos.Refund, error) {
	var refund dtos.Refund
	// Query by user_id (not refund id)
	err := db.QueryRow(`
		SELECT id, user_id, order_id, amount, status, created_at
		FROM refunds
		WHERE user_id = ?`, id).Scan(
		&refund.ID, &refund.UserID, &refund.OrderID, &refund.Amount, &refund.Status, &refund.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("refund not found")
	} else if err != nil {
		return nil, err
	}
	return &refund, nil
}

// AddVoucherHistory records voucher usage history with cart items log.
//
// This function creates a history record when a voucher is redeemed,
// storing the amount used and the items purchased.
//
// Parameters:
//   - code: string - The voucher code being redeemed (validated for existence)
//   - amount: float64 - Amount redeemed from the voucher
//   - itemsLog: []dtos.CartItem - Cart items purchased using the voucher (stored as JSON)
//
// Returns:
//   - error: Voucher not found, JSON marshal error, or database error
func AddVoucherHistory(db DBExecutor, code string, amount float64, itemsLog []dtos.CartItem) error {
	// Generate unique history ID
	historyID, _ := shortid.Generate()

	// Convert cart items to JSON for storage
	jsonData, err := json.Marshal(itemsLog)
	if err != nil {
		return err
	}

	// Validate voucher exists
	err = IsVoucherThereByCode(db, code)
	if err != nil {
		return err
	}

	// Fetch voucher details
	voucher, err := GetVoucherByCode(db, code)
	if err != nil {
		return err
	}

	// Insert history record with JSON items log
	query := `
		INSERT INTO vouchers_history (history_id, voucher_id, amount_redeemed, items_log)
		VALUES (?, ?, ?, ?)
	`

	_, err = db.Exec(query, historyID, voucher.VoucherID, amount, jsonData)
	if err != nil {
		return fmt.Errorf("failed to insert voucher history: %v", err)
	}

	return nil
}

// CreatePaymentOption creates a new configurable payment method.
//
// This function adds a payment option to the system with a unique ID and
// JSON configuration (e.g., API keys, credentials, settings).
//
// Parameters:
//   - paymentOption: dtos.PaymentOption - Payment option details:
//   - Name: Display name (e.g., "M-Pesa", "Credit Card")
//   - Type: Payment type (e.g., "mobile_money", "card")
//   - Configs: JSON configuration string (not unmarshaled)
//   - IsActive: Active status (true/false)
//
// Returns:
//   - error: Database error or nil on success
func CreatePaymentOption(db DBExecutor, paymentOption dtos.PaymentOption) error {
	// Generate unique payment option ID
	paymentOptionID, _ := shortid.Generate()

	// Insert payment option with JSON config
	query := `
		INSERT INTO payment_options (id, name, type, config_json, is_active)
		VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, paymentOptionID, paymentOption.Name, paymentOption.Type, paymentOption.Configs, paymentOption.IsActive)
	return err
}

// ListPaymentOptions retrieves paginated payment options with dynamic filtering.
//
// This function supports filtering by name (partial match) and active status,
// with dynamic query building based on provided filter values.
//
// Parameters:
//   - searchParam: string - Optional name filter (partial match via LIKE)
//   - status: string - Optional status filter ("active", "inactive", or "" for all)
//   - page: int - Page number for pagination
//   - size: int - Number of options per page
//
// Returns:
//   - []dtos.PaymentOption: Array of payment options with raw JSON configs
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func ListPaymentOptions(db DBExecutor, searchParam, status string, page, size int) ([]dtos.PaymentOption, *dtos.PaginationMeta, error) {
	// Calculate pagination offset
	offset := (page - 1) * size

	// Convert status string to boolean
	isActive := true
	if strings.ToLower(status) == "inactive" {
		isActive = false
	}

	// Build conditions and args for count query
	countQuery := `SELECT COUNT(*) FROM payment_options`
	var conditions []string
	var countArgs []any

	// Add name filter (partial match)
	if searchParam != "" {
		conditions = append(conditions, "name LIKE ?")
		countArgs = append(countArgs, "%"+searchParam+"%")
	}
	// Add active status filter
	if status != "" {
		conditions = append(conditions, "is_active = ?")
		countArgs = append(countArgs, isActive)
	}
	// Construct WHERE clause
	if len(conditions) > 0 {
		countQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Execute count query
	var totalItems int
	if err := db.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Build the main select query similarly
	query := `
		SELECT id, name, type, config_json, is_active, created_at
		FROM payment_options
	`
	var queryArgs []any
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
		queryArgs = append(queryArgs, countArgs...)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs = append(queryArgs, size, offset)

	// Query rows
	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan payment options
	var paymentOptions []dtos.PaymentOption
	for rows.Next() {
		var p dtos.PaymentOption
		if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.Configs, &p.IsActive, &p.CreatedAt); err != nil {
			return nil, nil, err
		}
		paymentOptions = append(paymentOptions, p)
	}

	// Build pagination metadata
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: int(math.Ceil(float64(totalItems) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < totalItems,
	}

	return paymentOptions, &meta, nil
}
