package models

import (
	"context"
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GetOrCreateCustomerWallet retrieves or initializes a customer wallet for a specific tenant
func GetOrCreateCustomerWallet(db DBExecutor, userID, tenantID string) (*dtos.CustomerWalletDTO, error) {
	query := `
		SELECT id, user_id, tenant_id, balance, store_credit, currency, created_at, updated_at
		FROM customer_wallets
		WHERE user_id = ? AND tenant_id = ?
	`
	var wallet dtos.CustomerWalletDTO
	err := db.QueryRowContext(context.Background(), query, userID, tenantID).Scan(
		&wallet.ID, &wallet.UserID, &wallet.TenantID, &wallet.Balance, &wallet.StoreCredit,
		&wallet.Currency, &wallet.CreatedAt, &wallet.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Create new wallet
		walletID := uuid.New().String()
		insertQuery := `
			INSERT INTO customer_wallets (id, user_id, tenant_id, balance, store_credit, currency)
			VALUES (?, ?, ?, 0.00, 0.00, 'KES')
		`
		_, err := db.ExecContext(context.Background(), insertQuery, walletID, userID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to create customer wallet: %w", err)
		}
		wallet = dtos.CustomerWalletDTO{
			ID:          walletID,
			UserID:      userID,
			TenantID:    tenantID,
			Balance:     0.00,
			StoreCredit: 0.00,
			Currency:    "KES",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query customer wallet: %w", err)
	}

	wallet.Total = wallet.Balance + wallet.StoreCredit
	return &wallet, nil
}

// AddWalletBalance deposits funds or store credit into a customer's wallet and logs a ledger transaction
func AddWalletBalance(db DBExecutor, walletID, trxType string, amount float64, reference, gatewayName, description string) (*dtos.WalletTransactionDTO, error) {
	_, ok := db.(*sql.Tx)
	var localTx *sql.Tx
	var err error

	if !ok {
		localTx, err = DB.Begin()
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer localTx.Rollback()
		db = localTx
	}

	// Update balance or store credit
	var updateQuery string
	if trxType == "store_credit_refund" || trxType == "cashback" {
		updateQuery = `UPDATE customer_wallets SET store_credit = store_credit + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	} else {
		updateQuery = `UPDATE customer_wallets SET balance = balance + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	}

	if _, err := db.ExecContext(context.Background(), updateQuery, amount, walletID); err != nil {
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Get updated balance
	var currentBalance, currentStoreCredit float64
	err = db.QueryRowContext(context.Background(), "SELECT balance, store_credit FROM customer_wallets WHERE id = ?", walletID).Scan(&currentBalance, &currentStoreCredit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated balance: %w", err)
	}

	totalAfter := currentBalance + currentStoreCredit
	trxID := uuid.New().String()

	trxQuery := `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, balance_after, reference, gateway_name, status, description)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'COMPLETED', ?)
	`
	if _, err := db.ExecContext(context.Background(), trxQuery, trxID, walletID, trxType, amount, totalAfter, reference, gatewayName, description); err != nil {
		return nil, fmt.Errorf("failed to log wallet transaction: %w", err)
	}

	if localTx != nil {
		if err := localTx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit wallet transaction: %w", err)
		}
	}

	return &dtos.WalletTransactionDTO{
		ID:           trxID,
		WalletID:     walletID,
		Type:         trxType,
		Amount:       amount,
		BalanceAfter: totalAfter,
		Reference:    reference,
		GatewayName:  gatewayName,
		Status:       "COMPLETED",
		Description:  description,
		CreatedAt:    time.Now(),
	}, nil
}

// DeductWalletBalance deducts funds or store credit from customer wallet for order payment
func DeductWalletBalance(db DBExecutor, walletID string, useStoreCredit bool, amount float64, orderID string) error {
	var currentBalance, currentStoreCredit float64
	err := db.QueryRowContext(context.Background(), "SELECT balance, store_credit FROM customer_wallets WHERE id = ?", walletID).Scan(&currentBalance, &currentStoreCredit)
	if err != nil {
		return fmt.Errorf("failed to fetch wallet balance: %w", err)
	}

	if useStoreCredit {
		if currentStoreCredit < amount {
			return fmt.Errorf("insufficient store credit balance (available: %.2f, required: %.2f)", currentStoreCredit, amount)
		}
		_, err = db.ExecContext(context.Background(), "UPDATE customer_wallets SET store_credit = store_credit - ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", amount, walletID)
	} else {
		if currentBalance < amount {
			return fmt.Errorf("insufficient wallet balance (available: %.2f, required: %.2f)", currentBalance, amount)
		}
		_, err = db.ExecContext(context.Background(), "UPDATE customer_wallets SET balance = balance - ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", amount, walletID)
	}

	if err != nil {
		return fmt.Errorf("failed to deduct balance from wallet: %w", err)
	}

	var newBalance, newCredit float64
	_ = db.QueryRowContext(context.Background(), "SELECT balance, store_credit FROM customer_wallets WHERE id = ?", walletID).Scan(&newBalance, &newCredit)
	balanceAfter := newBalance + newCredit

	trxType := "purchase_payment"
	if useStoreCredit {
		trxType = "store_credit_payment"
	}

	trxID := uuid.New().String()
	trxQuery := `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, balance_after, reference, gateway_name, status, description)
		VALUES (?, ?, ?, ?, ?, ?, 'wallet', 'COMPLETED', ?)
	`
	desc := fmt.Sprintf("Payment for order #%s", orderID)
	_, err = db.ExecContext(context.Background(), trxQuery, trxID, walletID, trxType, amount, balanceAfter, orderID, desc)
	if err != nil {
		return fmt.Errorf("failed to log wallet payment transaction: %w", err)
	}

	return nil
}

// GetWalletTransactions retrieves ledger transactions for a wallet
func GetWalletTransactions(db DBExecutor, walletID string) ([]dtos.WalletTransactionDTO, error) {
	query := `
		SELECT id, wallet_id, type, amount, balance_after, reference, gateway_name, status, description, created_at
		FROM wallet_transactions
		WHERE wallet_id = ?
		ORDER BY created_at DESC
	`
	rows, err := db.QueryContext(context.Background(), query, walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to query wallet transactions: %w", err)
	}
	defer rows.Close()

	var transactions []dtos.WalletTransactionDTO
	for rows.Next() {
		var trx dtos.WalletTransactionDTO
		if err := rows.Scan(
			&trx.ID, &trx.WalletID, &trx.Type, &trx.Amount, &trx.BalanceAfter,
			&trx.Reference, &trx.GatewayName, &trx.Status, &trx.Description, &trx.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row: %w", err)
		}
		transactions = append(transactions, trx)
	}
	return transactions, nil
}
