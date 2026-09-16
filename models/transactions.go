// Package models provides data access functions for the Ekomasi e-commerce platform.
//
// This file contains functions for managing financial transactions:
//   - Retrieve transactions with pagination, filtering, and search
//   - Insert new transaction records
//   - Update transaction and payment status
//   - M-Pesa receipt number management
//   - Order payment status synchronization
package models

import (
	"ekomasi_backend/dtos"
	"errors"
	"strings"

	"github.com/teris-io/shortid"
)

// GetAllTransactions retrieves all transactions with pagination, filtering, and search.
//
// This function supports:
//   - Pagination with page and limit parameters
//   - Status filtering (case-insensitive)
//   - Multi-field search across M-Pesa reference, order ID, transaction reference,
//     phone number, and account number
//
// Parameters:
//   - page: int - Page number (1-based)
//   - limit: int - Number of items per page
//   - status: string - Filter by transaction status (case-insensitive, "All" returns all statuses)
//   - q: string - Search query across multiple fields (case-insensitive partial match)
//
// Returns:
//   - []dtos.TransactionsList: Array of transactions with:
//   - TransactionID, OrderID, MpesaReference, TransactionReference
//   - PhoneNumber, Amount, AccountNumber, Status
//   - TransactionDate, PaymentMethod
//   - *dtos.PaginationMeta: Pagination info (page, size, totals, navigation flags)
//   - error: Database error or nil on success
func GetAllTransactions(page, limit int, status, q string) ([]dtos.TransactionsList, *dtos.PaginationMeta, error) {
	// Calculate offset for pagination
	offset := (page - 1) * limit

	// Build WHERE clause with dynamic filters
	where := "WHERE 1=1"
	var args []any

	// Add status filter (case-insensitive)
	if status != "" && status != "All" {
		where += " AND LOWER(status) LIKE ?"
		args = append(args, "%"+status+"%")
	}

	// Add multi-field search query (case-insensitive)
	if q != "" {
		where += `
			AND (LOWER(mpesa_reference) LIKE ? OR LOWER(order_id) LIKE ? OR LOWER(transaction_reference) LIKE ? OR LOWER(phone_number) LIKE ? OR LOWER(account_number) LIKE ?)
		`
		args = append(args, "%"+strings.ToLower(q)+"%", "%"+strings.ToLower(q)+"%", "%"+strings.ToLower(q)+"%", "%"+strings.ToLower(q)+"%", "%"+strings.ToLower(q)+"%")
	}

	// Get total count for pagination
	var totalItems int
	countQuery := `SELECT COUNT(*) FROM transactions ` + where
	err := DB.QueryRow(countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Fetch paginated transactions
	query := `
		SELECT transaction_id, order_id, mpesa_reference, transaction_reference, phone_number, amount, account_number, status, created_at, payment_method
		FROM transactions
	` + where + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Process results
	var transactions []dtos.TransactionsList
	for rows.Next() {
		var t dtos.TransactionsList
		if err := rows.Scan(&t.TransactionID, &t.OrderID, &t.MpesaReference, &t.TransactionReference, &t.PhoneNumber, &t.Amount, &t.AccountNumber, &t.Status, &t.TransactionDate, &t.PaymentMethod); err != nil {
			return nil, nil, err
		}
		transactions = append(transactions, t)
	}

	// Build pagination metadata
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: (totalItems + limit - 1) / limit, // Ceiling division
		HasPrev:    page > 1,
		HasNext:    page*limit < totalItems,
	}

	return transactions, meta, nil
}

// GetTransactionByID retrieves a single transaction by its ID.
//
// This function validates transaction existence before retrieval.
//
// Parameters:
//   - transactionID: string - The unique transaction ID to retrieve
//
// Returns:
//   - *dtos.TransactionsList: Transaction details including:
//   - TransactionID, OrderID, MpesaReference, TransactionReference
//   - PhoneNumber, Amount, AccountNumber, Status
//   - TransactionDate, PaymentMethod
//   - error: "transaction with ID <id> doesn't exist", database error, or nil on success
func GetTransactionByID(transactionID string) (*dtos.TransactionsList, error) {
	// Validate transaction exists
	exists, err := RecordExists(DB, "transactions", "transaction_id = ?", transactionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("transaction with ID " + transactionID + " doesn't exist")
	}

	// Retrieve transaction details
	var t dtos.TransactionsList
	query := `
		SELECT transaction_id, order_id, mpesa_reference, transaction_reference, phone_number, amount, account_number, status, created_at, payment_method
		FROM transactions
		WHERE transaction_id = ?
		`
	err = DB.QueryRow(query, transactionID).Scan(&t.TransactionID, &t.OrderID, &t.MpesaReference, &t.TransactionReference, &t.PhoneNumber, &t.Amount, &t.AccountNumber, &t.Status, &t.TransactionDate, &t.PaymentMethod)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// InsertTransaction creates a new transaction record.
//
// This function generates a unique transaction ID and inserts the transaction
// with all payment details including M-Pesa references.
//
// Parameters:
//   - t: *dtos.TransactionsList containing:
//   - OrderID: Associated order ID
//   - MpesaReference: M-Pesa receipt number
//   - TransactionReference: Internal transaction reference
//   - PhoneNumber: Customer phone number
//   - Amount: Transaction amount
//   - AccountNumber: Payment account number
//   - Status: Transaction status (e.g., "pending", "completed", "failed")
//   - PaymentMethod: Payment method used (e.g., "mpesa", "card")
//
// Returns:
//   - error: Database error or nil on success
func InsertTransaction(db DBExecutor, t *dtos.TransactionsList) error {
	// Generate unique transaction ID
	transactionID, _ := shortid.Generate()

	// Insert new transaction record
	query := `
		INSERT INTO transactions (transaction_id, order_id, mpesa_reference, transaction_reference, phone_number, amount, account_number, status, payment_method)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query, transactionID, t.OrderID, t.MpesaReference, t.TransactionReference, t.PhoneNumber, t.Amount, t.AccountNumber, t.Status, t.PaymentMethod)
	return err
}

// UpdateTransactionStatus updates the status of a transaction by order ID.
//
// This function updates the transaction status for all transactions
// associated with the specified order.
//
// Parameters:
//   - orderID: string - The order ID to update transactions for
//   - status: string - New transaction status (e.g., "completed", "failed", "refunded")
//
// Returns:
//   - error: Database error or nil on success
func UpdateTransactionStatus(db DBExecutor, orderID, status string) error {
	// Update transaction status for order
	query := `
		UPDATE transactions
		SET status = ?
		WHERE order_id = ?`
	_, err := db.Exec(query, status, orderID)
	return err
}

// UpdateMpesaReceiptNumber updates the M-Pesa receipt number for a transaction.
//
// This function updates the M-Pesa reference for all transactions
// associated with the specified order, typically after receiving
// payment confirmation from M-Pesa.
//
// Parameters:
//   - mpesaReceiptNumber: string - The M-Pesa receipt/reference number
//   - orderID: string - The order ID to update transactions for
//
// Returns:
//   - error: Database error or nil on success
func UpdateMpesaReceiptNumber(db DBExecutor, mpesaReceiptNumber, orderID string) error {
	// Update M-Pesa reference for order transactions
	query := `
		UPDATE transactions
		SET mpesa_reference = ?
		WHERE order_id = ?`
	_, err := db.Exec(query, mpesaReceiptNumber, orderID)
	return err
}

// UpdateTransactionStatusByID updates transaction status and returns the associated order ID.
//
// This function updates a specific transaction's status by its transaction ID
// and retrieves the associated order ID for further order processing.
//
// Parameters:
//   - transactionID: string - The unique transaction ID to update
//   - status: string - New transaction status (e.g., "completed", "failed", "refunded")
//
// Returns:
//   - string: The order ID associated with this transaction
//   - error: Database error or nil on success
func UpdateTransactionStatusByID(db DBExecutor, transactionID, status string) (string, error) {
	// Update transaction status
	query := `
		UPDATE transactions
		SET status = ?
		WHERE transaction_id = ?`
	_, err := db.Exec(query, status, transactionID)
	if err != nil {
		return "", err
	}

	// Retrieve associated order ID for order status synchronization
	var orderID string
	err = db.QueryRow(`SELECT order_id FROM transactions WHERE transaction_id = ?`, transactionID).Scan(&orderID)
	if err != nil {
		return "", err
	}

	return orderID, nil
}

// UpdateOrderPaymentStatus updates the payment status of an order.
//
// This function synchronizes the order's payment status with transaction status,
// typically called after a transaction status change.
//
// Parameters:
//   - orderID: string - The order ID to update
//   - status: string - New payment status (e.g., "paid", "pending", "failed")
//
// Returns:
//   - error: Database error or nil on success
func UpdateOrderPaymentStatus(db DBExecutor, orderID, status string) error {
	// Update order payment status
	query := `
		UPDATE orders
		SET payment_status = ?
		WHERE order_id = ?`
	_, err := db.Exec(query, status, orderID)
	return err
}
