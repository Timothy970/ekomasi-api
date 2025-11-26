package models

import (
	"adenzo_backend/dtos"
	"errors"
	"strings"

	"github.com/teris-io/shortid"
)

func GetAllTransactions(page, limit int, status, q string) ([]dtos.TransactionsList, *dtos.PaginationMeta, error) {
	offset := (page - 1) * limit

	where := "WHERE 1=1"
	var args []interface{}
	//use lower case for status comparison
	if status != "" && status != "All" {
		where += " AND LOWER(status) LIKE ?"
		args = append(args, "%"+status+"%")
	}
	if q != "" {
		where += `
			AND (LOWER(mpesa_reference) LIKE ? OR LOWER(order_id) LIKE ? OR LOWER(transaction_reference) LIKE ? OR LOWER(phone_number) LIKE ? OR LOWER(account_number) LIKE ?)
		`
		args = append(args, "%"+strings.ToLower(q)+"%", "%"+strings.ToLower(q)+"%", "%"+strings.ToLower(q)+"%", "%"+strings.ToLower(q)+"%", "%"+strings.ToLower(q)+"%")
	}
	// Count total transactions
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
	var transactions []dtos.TransactionsList
	for rows.Next() {
		var t dtos.TransactionsList
		if err := rows.Scan(&t.TransactionID, &t.OrderID, &t.MpesaReference, &t.TransactionReference, &t.PhoneNumber, &t.Amount, &t.AccountNumber, &t.Status, &t.TransactionDate, &t.PaymentMethod); err != nil {
			return nil, nil, err
		}
		transactions = append(transactions, t)
	}
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: (totalItems + limit - 1) / limit,
		HasPrev:    page > 1,
		HasNext:    page*limit < totalItems,
	}
	return transactions, meta, nil
}

func GetTransactionByID(transactionID string) (*dtos.TransactionsList, error) {
	exists, err := RecordExists("transactions", "transaction_id = ?", transactionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("transaction with ID " + transactionID + " doesn't exist")
	}

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

func InsertTransaction(t *dtos.TransactionsList) error {
	transactionID, _ := shortid.Generate()
	query := `
		INSERT INTO transactions (transaction_id, order_id, mpesa_reference, transaction_reference, phone_number, amount, account_number, status, payment_method)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := DB.Exec(query, transactionID, t.OrderID, t.MpesaReference, t.TransactionReference, t.PhoneNumber, t.Amount, t.AccountNumber, t.Status, t.PaymentMethod)
	return err
}

func UpdateTransactionStatus(orderID, status string) error {
	query := `
		UPDATE transactions
		SET status = ?
		WHERE order_id = ?`
	_, err := DB.Exec(query, status, orderID)
	return err
}

func UpdateMpesaReceiptNumber(mpesaReceiptNumber, orderID string) error {
	query := `
		UPDATE transactions
		SET mpesa_reference = ?
		WHERE order_id = ?`
	_, err := DB.Exec(query, mpesaReceiptNumber, orderID)
	return err
}

func UpdateTransactionStatusByID(transactionID, status string) (string, error) {
	query := `
		UPDATE transactions
		SET status = ?
		WHERE transaction_id = ?`
	_, err := DB.Exec(query, status, transactionID)
	if err != nil {
		return "", err
	}
	//get order id associated with this transaction
	var orderID string
	err = DB.QueryRow(`SELECT order_id FROM transactions WHERE transaction_id = ?`, transactionID).Scan(&orderID)
	if err != nil {
		return "", err
	}
	return orderID, nil
}

func UpdateOrderPaymentStatus(orderID, status string) error {
	query := `
		UPDATE orders
		SET payment_status = ?
		WHERE order_id = ?`
	_, err := DB.Exec(query, status, orderID)
	return err
}
