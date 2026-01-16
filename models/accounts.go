// Package models provides database models and operations for the accounting system.
// Implements double-entry bookkeeping with chart of accounts and journal entries.
// Supports linking entries to orders, payments, and purchase orders for full audit trail.
package models

import (
	"adenzo_backend/dtos"
	"errors"
	"math"

	"github.com/teris-io/shortid"
)

// SQL query fragments for common WHERE clauses
var whereAccountID = "account_id = ?"
var whereEntryID = "entry_id = ?"

// Error messages for consistent error responses
var noAccount = "account not found"
var noEntry = "entry not found"

// isChartAccountCodeThere validates that an account code exists in the chart of accounts.
// Used for validation before creating or updating accounts.
//
// Parameters:
//   - code: The account code to validate (e.g., "1000", "2000")
//
// Returns:
//   - error: nil if account code exists, error if not found or database error
func isChartAccountCodeThere(code string) error {
	// Check if account code exists in chart_of_accounts table
	exists, err := RecordExists("chart_of_accounts", "account_code = ?", code)
	if err != nil {
		// Database query failed
		return err
	}
	if !exists {
		// Account code not found in chart of accounts
		return errors.New("variant not found")
	}
	return nil
}

// ===== Chart of Accounts Management =====

// CreateAccount creates a new account in the chart of accounts.
// Validates account code exists before creation and generates unique account ID.
//
// Parameters:
//   - req: CreateAccountRequest containing account code, name, and type (Asset/Liability/Equity/Revenue/Expense)
//
// Returns:
//   - string: Generated account ID if successful
//   - error: Error if account code invalid or database operation fails
var CreateAccount = func(req dtos.CreateAccountRequest) (string, error) {
	// Validate that the account code exists in the system
	err := isChartAccountCodeThere(req.AccountCode)
	if err != nil {
		// Account code validation failed
		return "", err
	}
	// Generate unique account ID
	id, _ := shortid.Generate()

	// Insert new account into chart of accounts
	_, err = DB.Exec(`
		INSERT INTO chart_of_accounts (account_id, account_code, account_name, account_type)
		VALUES (?, ?, ?, ?)`,
		id, req.AccountCode, req.AccountName, req.AccountType,
	)
	return id, err
}

// ListAccounts retrieves a paginated list of accounts from the chart of accounts.
// Accounts are ordered by account code for consistent presentation.
//
// Parameters:
//   - page: Current page number (1-indexed)
//   - size: Number of accounts per page
//
// Returns:
//   - []dtos.ChartOfAccount: List of accounts with ID, code, name, type, and current balance
//   - dtos.PaginationMeta: Pagination metadata (total items, total pages, has next/prev)
//   - error: Error if database operation fails
var ListAccounts = func(page, size int) ([]dtos.ChartOfAccount, dtos.PaginationMeta, error) {
	// Get total count of accounts for pagination metadata
	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM chart_of_accounts`).Scan(&total)
	if err != nil {
		// Failed to get count
		return nil, dtos.PaginationMeta{}, err
	}

	// Calculate offset for pagination
	offset := (page - 1) * size
	// Fetch paginated accounts ordered by account code
	rows, err := DB.Query(`
		SELECT account_id, account_code, account_name, account_type, balance
		FROM chart_of_accounts
		ORDER BY account_code
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		// Query failed
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	// Scan accounts into slice
	var accounts []dtos.ChartOfAccount
	for rows.Next() {
		var acc dtos.ChartOfAccount
		if err := rows.Scan(&acc.AccountID, &acc.AccountCode, &acc.AccountName, &acc.AccountType, &acc.Balance); err != nil {
			// Row scan failed
			return nil, dtos.PaginationMeta{}, err
		}
		accounts = append(accounts, acc)
	}

	// Build pagination metadata
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return accounts, meta, nil
}

// GetAccount retrieves a single account by its ID from the chart of accounts.
//
// Parameters:
//   - id: Account ID to retrieve
//
// Returns:
//   - dtos.ChartOfAccount: Account details including ID, code, name, type, and current balance
//   - error: Error if account not found or database operation fails
var GetAccount = func(id string) (dtos.ChartOfAccount, error) {
	// Check if account exists before querying
	exists, err := RecordExists("chart_of_accounts", whereAccountID, id)
	if err != nil {
		// Database error during existence check
		return dtos.ChartOfAccount{}, err
	}
	if !exists {
		// Account not found
		return dtos.ChartOfAccount{}, errors.New(noAccount)
	}
	// Fetch account details
	var acc dtos.ChartOfAccount
	err = DB.QueryRow(`
		SELECT account_id, account_code, account_name, account_type, balance
		FROM chart_of_accounts WHERE account_id = ?`, id).
		Scan(&acc.AccountID, &acc.AccountCode, &acc.AccountName, &acc.AccountType, &acc.Balance)
	return acc, err
}

// UpdateAccount updates an existing account's name and type.
// Account code and balance cannot be changed through this operation.
//
// Parameters:
//   - id: Account ID to update
//   - req: UpdateAccountRequest containing new account name and type
//
// Returns:
//   - error: Error if account not found or database operation fails
var UpdateAccount = func(id string, req dtos.UpdateAccountRequest) error {
	// Verify account exists before updating
	exists, err := RecordExists("chart_of_accounts", whereAccountID, id)
	if err != nil {
		// Database error during existence check
		return err
	}
	if !exists {
		// Account not found
		return errors.New(noAccount)
	}
	// Update account name and type (code and balance remain unchanged)
	_, err = DB.Exec(`
		UPDATE chart_of_accounts SET account_name = ?, account_type = ? WHERE account_id = ?`,
		req.AccountName, req.AccountType, id,
	)
	return err
}

// DeleteAccount removes an account from the chart of accounts.
//
// Parameters:
//   - id: Account ID to delete
//
// Returns:
//   - error: Error if account not found or database operation fails
var DeleteAccount = func(id string) error {
	// Verify account exists before deletion
	exists, err := RecordExists("chart_of_accounts", whereAccountID, id)
	if err != nil {
		// Database error during existence check
		return err
	}
	if !exists {
		// Account not found
		return errors.New(noAccount)
	}
	// Delete account from chart of accounts
	_, err = DB.Exec(`DELETE FROM chart_of_accounts WHERE account_id = ?`, id)
	return err
}

// ===== Journal Entries Management =====

// CreateEntry creates a new journal entry for double-entry bookkeeping.
// Validates referenced account, order, payment, and purchase order before creation.
// Implements standard accounting practice where each transaction has debit and credit sides.
//
// Parameters:
//   - req: CreateJournalEntryRequest containing account ID, amounts, optional links to orders/payments/POs, and description
//
// Returns:
//   - string: Generated entry ID if successful
//   - error: Error if referenced entities don't exist or database operation fails
var CreateEntry = func(req dtos.CreateJournalEntryRequest) (string, error) {
	// Validate that the account exists in chart of accounts
	exists, err := RecordExists("chart_of_accounts", whereAccountID, req.AccountID)
	if err != nil {
		// Database error during account check
		return "", err
	}
	if !exists {
		// Account not found
		return "", errors.New(noAccount)
	}
	// Validate optional order reference if provided
	if req.OrderID != nil {
		err := IsOrderThere(*req.OrderID)
		if err != nil {
			// Order not found
			return "", err
		}
	}
	// Validate optional payment reference if provided
	if req.PaymentID != nil {
		err := isPaymentThere(*req.PaymentID)
		if err != nil {
			// Payment not found
			return "", err
		}
	}
	// Validate optional purchase order reference if provided
	if req.PoID != nil {
		err := isPurchaseOrderThere(*req.PoID)
		if err != nil {
			// Purchase order not found
			return "", err
		}
	}
	// Generate unique entry ID
	id, _ := shortid.Generate()

	// Insert journal entry with debit/credit amounts and optional references
	_, err = DB.Exec(`
		INSERT INTO journal_entries (entry_id, order_id, payment_id, po_id, account_id, debit, credit, description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.OrderID, req.PaymentID, req.PoID, req.AccountID, req.Debit, req.Credit, req.Description,
	)
	return id, err
}

// ListEntries retrieves a paginated list of journal entries.
// Entries are ordered by date descending (most recent first) for chronological audit trail.
//
// Parameters:
//   - page: Current page number (1-indexed)
//   - size: Number of entries per page
//
// Returns:
//   - []dtos.JournalEntry: List of journal entries with debit/credit amounts and references
//   - dtos.PaginationMeta: Pagination metadata (total items, total pages, has next/prev)
//   - error: Error if database operation fails
var ListEntries = func(page, size int) ([]dtos.JournalEntry, dtos.PaginationMeta, error) {
	// Get total count of journal entries for pagination metadata
	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM journal_entries`).Scan(&total)
	if err != nil {
		// Failed to get count
		return nil, dtos.PaginationMeta{}, err
	}

	// Calculate offset for pagination
	offset := (page - 1) * size
	// Fetch paginated entries ordered by entry date descending
	rows, err := DB.Query(`
		SELECT entry_id, order_id, payment_id, po_id, account_id, debit, credit, entry_date, description
		FROM journal_entries
		ORDER BY entry_date DESC
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		// Query failed
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	// Scan entries into slice
	var entries []dtos.JournalEntry
	for rows.Next() {
		var e dtos.JournalEntry
		if err := rows.Scan(&e.EntryID, &e.OrderID, &e.PaymentID, &e.PoID, &e.AccountID,
			&e.Debit, &e.Credit, &e.EntryDate, &e.Description); err != nil {
			// Row scan failed
			return nil, dtos.PaginationMeta{}, err
		}
		entries = append(entries, e)
	}

	// Build pagination metadata
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return entries, meta, nil
}

// GetEntry retrieves a single journal entry by its ID.
//
// Parameters:
//   - id: Journal entry ID to retrieve
//
// Returns:
//   - dtos.JournalEntry: Entry details including debit/credit amounts, account, and optional references
//   - error: Error if entry not found or database operation fails
var GetEntry = func(id string) (dtos.JournalEntry, error) {
	// Check if journal entry exists before querying
	exists, err := RecordExists("journal_entries", whereEntryID, id)
	if err != nil {
		// Database error during existence check
		return dtos.JournalEntry{}, err
	}
	if !exists {
		// Entry not found
		return dtos.JournalEntry{}, errors.New(noEntry)
	}
	// Fetch journal entry details
	var e dtos.JournalEntry
	err = DB.QueryRow(`
		SELECT entry_id, order_id, payment_id, po_id, account_id, debit, credit, entry_date, description
		FROM journal_entries WHERE entry_id = ?`, id).
		Scan(&e.EntryID, &e.OrderID, &e.PaymentID, &e.PoID, &e.AccountID, &e.Debit, &e.Credit, &e.EntryDate, &e.Description)
	return e, err
}

// UpdateEntry updates the debit and credit amounts of an existing journal entry.
//
// Parameters:
//   - id: Journal entry ID to update
//   - req: UpdateJournalEntryRequest containing new debit and credit amounts
//
// Returns:
//   - error: Error if entry not found or database operation fails
var UpdateEntry = func(id string, req dtos.UpdateJournalEntryRequest) error {
	// Verify entry exists before updating
	exists, err := RecordExists("journal_entries", whereEntryID, id)
	if err != nil {
		// Database error during existence check
		return err
	}
	if !exists {
		// Entry not found
		return errors.New(noEntry)
	}
	// Update debit and credit amounts (other fields remain unchanged)
	_, err = DB.Exec(`
		UPDATE journal_entries SET debit = ?, credit = ? WHERE entry_id = ?`,
		req.Debit, req.Credit, id,
	)
	return err
}

// DeleteEntry removes a journal entry from the system.
//
// Parameters:
//   - id: Journal entry ID to delete
//
// Returns:
//   - error: Error if entry not found or database operation fails
var DeleteEntry = func(id string) error {
	// Verify entry exists before deletion
	exists, err := RecordExists("journal_entries", whereEntryID, id)
	if err != nil {
		// Database error during existence check
		return err
	}
	if !exists {
		// Entry not found
		return errors.New(noEntry)
	}
	// Delete journal entry
	_, err = DB.Exec(`DELETE FROM journal_entries WHERE entry_id = ?`, id)
	return err
}
