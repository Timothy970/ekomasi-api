// Package models provides data access functions for the Adenzo backend.
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
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/teris-io/shortid"
)

var nopayment = "payment not found"
var paymentid = "payment_id = ?"

// StoreStkResponse stores M-Pesa STK Push response for tracking payment status.
//
// This function records the initial STK Push request details including checkout
// and merchant request IDs for both product orders and voucher purchases.
//
// Parameters:
//   - response: map[string]interface{} containing M-Pesa API response with:
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
func StoreStkResponse(response map[string]interface{}, req dtos.MpesaRequest) error {
	// Extract M-Pesa response IDs safely from interface map
	checkoutRequestID, _ := response["CheckoutRequestID"].(string)
	merchantRequestID, _ := response["MerchantRequestID"].(string)
	var err error

	// Handle voucher payment (no delivery involved)
	if strings.ToLower(req.Type) == "voucher" {
		_, err = DB.Exec(`
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
		_, err = DB.Exec(`
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
func UpdateStkResponse(checkoutRequestID string, status string) (string, string, string, error) {
	// 1. Update payment status in stk_push_responses
	_, err := DB.Exec(`
		UPDATE stk_push_responses SET status = ?
		WHERE checkout_request_id = ?
	`, status, checkoutRequestID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to update status: %w", err)
	}

	// 2. Retrieve delivery_id, order_id, and type for further processing
	var deliveryID, orderID, orderType sql.NullString
	err = DB.QueryRow(`
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
func UpdateDeliveryOrderTables(deliveryID, orderId string, status string) error {
	// Determine order status based on payment result
	orderStatus := "PAID"
	if status != "COMPLETED" {
		orderStatus = "FAILED"
	}

	// Update order status and payment status
	_, err := DB.Exec(`
		UPDATE orders SET status = ?, payment_status = ?
		WHERE order_id = ?
	`, orderStatus, orderStatus, orderId)
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
func UpdateVoucherOrderTables(orderId string, status string) error {
	// Update voucher order status
	_, err := DB.Exec(`
    UPDATE voucher_orders SET status = ?
    WHERE voucher_order_id = ?
`, status, orderId)
	if err != nil {
		return fmt.Errorf("failed to update voucher order status: %w", err)
	}

	// Fetch the voucher_id associated with this order
	var voucherID string
	err = DB.QueryRow(`
	SELECT voucher_id FROM voucher_orders WHERE voucher_order_id = ?
`, orderId).Scan(&voucherID)
	if err != nil {
		return fmt.Errorf("failed to fetch voucher_id: %w", err)
	}

	// Activate the voucher
	isActive := true
	_, err = DB.Exec(`
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
func SetVoucherAsRedeemed(code string) error {
	// Mark voucher as redeemed
	_, err := DB.Exec(`
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
//   - error: "order not found", "transaction ID already exists", or database error
func CreatePayment(p dtos.Payment) error {
	// Validate order exists
	err := IsOrderThere(p.OrderID)
	if err != nil {
		return err
	}

	// Validate transaction ID is unique
	err = isTransactionIDUnique(p.TransactionID)
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
	_, err = DB.Exec(query, paymentID, p.OrderID, p.Amount, p.VoucherID, status, p.PaymentMethod, p.TransactionID)
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
func isPaymentThere(id string) error {
	exists, err := RecordExists("payments", paymentid, id)
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
func GetPaymentByID(paymentID string) (*dtos.Payment, error) {
	// Validate payment exists
	err := isPaymentThere(paymentID)
	if err != nil {
		return nil, err
	}

	// Query payment details
	query := `
		SELECT payment_id, order_id, amount, voucher_id, status, payment_method, transaction_id, created_at
		FROM payments
		WHERE payment_id = ?`

	var p dtos.Payment
	err = DB.QueryRow(query, paymentID).Scan(
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
func ListPayments(page, limit int) ([]dtos.Payment, *dtos.PaginationMeta, error) {
	// Calculate pagination offset
	offset := (page - 1) * limit

	// Count total payments
	var totalItems int
	err := DB.QueryRow("SELECT COUNT(*) FROM payments").Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Fetch paginated payments ordered by most recent
	query := `
		SELECT payment_id, order_id, amount, voucher_id, status, payment_method, transaction_id, created_at
		FROM payments
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := DB.Query(query, limit, offset)
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
func UpdatePayment(p dtos.PaymentUpdate, paymentID string) error {
	// Validate payment exists
	err := isPaymentThere(paymentID)
	if err != nil {
		return err
	}

	// Update payment status
	query := `
		UPDATE payments
		SET status = ?
		WHERE payment_id = ?`

	_, err = DB.Exec(query, p.Status, paymentID)
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
func DeletePayment(paymentID string) error {
	// Validate payment exists
	err := isPaymentThere(paymentID)
	if err != nil {
		return err
	}

	// Permanently delete payment
	query := `DELETE FROM payments WHERE payment_id = ?`
	_, err = DB.Exec(query, paymentID)
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
func AddRefundRequest(refund dtos.Refund, userID string) error {
	// Validate order exists
	exists, err := RecordExists("orders", "order_id = ?", refund.OrderID)
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

	_, err = DB.Exec(query, userID, refund.OrderID, refund.Amount, refund.Reason, refund.Status)
	if err != nil {
		return err
	}

	return nil
}

// ProcessRefund updates the status of a refund request.
//
// This function is typically used by admins to approve/reject refund requests.
//
// Parameters:
//   - req: dtos.RefundPayload containing:
//   - Status: New refund status ("approved", "rejected", "processed", etc.)
//   - refundID: string - The refund ID to update
//
// Returns:
//   - error: "refund not found" or database error
func ProcessRefund(req dtos.RefundPayload, refundID string) error {
	// Validate refund exists
	exists, err := RecordExists("refunds", whereID, refundID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("refund not found")
	}

	// Update refund status
	query := `UPDATE refunds SET status = ? WHERE id = ?`
	_, err = DB.Exec(query, req.Status, refundID)
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
func ListRefunds(page, size int) ([]dtos.Refund, *dtos.PaginationMeta, error) {
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
	err := DB.QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	// Fetch paginated refunds ordered by most recent
	rows, err := DB.Query(`
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
func GetRefundByID(id int) (*dtos.Refund, error) {
	var refund dtos.Refund
	// Query by user_id (not refund id)
	err := DB.QueryRow(`
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
func AddVoucherHistory(code string, amount float64, itemsLog []dtos.CartItem) error {
	// Generate unique history ID
	historyID, _ := shortid.Generate()

	// Convert cart items to JSON for storage
	jsonData, err := json.Marshal(itemsLog)
	if err != nil {
		return err
	}

	// Validate voucher exists
	err = IsVoucherThereByCode(code)
	if err != nil {
		return err
	}

	// Fetch voucher details
	voucher, err := GetVoucherByCode(code)
	if err != nil {
		return err
	}

	// Insert history record with JSON items log
	query := `
		INSERT INTO vouchers_history (history_id, voucher_id, amount_redeemed, items_log)
		VALUES (?, ?, ?, ?)
	`

	_, err = DB.Exec(query, historyID, voucher.VoucherID, amount, jsonData)
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
func CreatePaymentOption(paymentOption dtos.PaymentOption) error {
	// Generate unique payment option ID
	paymentOptionID, _ := shortid.Generate()

	// Insert payment option with JSON config
	query := `
		INSERT INTO payment_options (id, name, type, config_json, is_active)
		VALUES (?, ?, ?, ?, ?)`
	_, err := DB.Exec(query, paymentOptionID, paymentOption.Name, paymentOption.Type, paymentOption.Configs, paymentOption.IsActive)
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
func ListPaymentOptions(searchParam, status string, page, size int) ([]dtos.PaymentOption, *dtos.PaginationMeta, error) {
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
	var countArgs []interface{}

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
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Build the main select query similarly
	query := `
		SELECT id, name, type, config_json, is_active, created_at
		FROM payment_options
	`
	var queryArgs []interface{}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
		queryArgs = append(queryArgs, countArgs...)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs = append(queryArgs, size, offset)

	// Query rows
	rows, err := DB.Query(query, queryArgs...)
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

// isPaymentOptionThere validates if a payment option exists by ID.
//
// This is an internal helper function used before updates, retrievals, or deletions.
//
// Parameters:
//   - id: string - The payment option id to check
//
// Returns:
//   - error: "payment option not found" if missing, nil if exists, or database error
func isPaymentOptionThere(id string) error {
	// Check record existence using utility function
	exists, err := RecordExists("payment_options", "id = ?", id)
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
func GetPaymentOptionByID(id string) (*dtos.PaymentOption, error) {
	// Validate payment option exists
	err := isPaymentOptionThere(id)
	if err != nil {
		return nil, err
	}

	// Query payment option details
	var p dtos.PaymentOption
	err = DB.QueryRow(`SELECT id, name, type, config_json, is_active, created_at
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
func UpdatePaymentOption(id string, paymentOption dtos.PaymentOptionUpdate) error {
	// Validate payment option exists
	err := isPaymentOptionThere(id)
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
	_, err = DB.Exec(query, args...)
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
func DeletePaymentOption(id string) error {
	// Validate payment option exists
	err := isPaymentOptionThere(id)
	if err != nil {
		return err
	}

	// Hard delete payment option
	query := `DELETE FROM payment_options WHERE id = ?`
	_, err = DB.Exec(query, id)
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
func GetCheckoutRequestIDByOrderID(orderID string) (string, error) {
	// Query most recent checkout request for order
	var checkoutRequestID sql.NullString
	err := DB.QueryRow(`
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
