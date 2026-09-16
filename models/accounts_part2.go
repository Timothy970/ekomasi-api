// Package models provides database models and operations for the accounting system.
// Implements double-entry bookkeeping with chart of accounts and journal entries.
// Supports linking entries to orders, payments, and purchase orders for full audit trail.
package models

import (
	"ekomasi_backend/dtos"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/teris-io/shortid"
)

// GetAccount retrieves a single account by its ID from the chart of accounts.
//
// Parameters:
//   - id: Account ID to retrieve
//
// Returns:
//   - dtos.ChartOfAccount: Account details including ID, code, name, type, and current balance
//   - error: Error if account not found or database operation fails
var GetAccount = func(db DBExecutor, id string) (dtos.ChartOfAccount, error) {
	// Check if account exists before querying
	exists, err := RecordExists(db, "chart_of_accounts", whereAccountID, id)
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
	err = db.QueryRow(`
		SELECT account_id, account_code, account_name, account_type, balance, statement_type, description, status, category
		FROM chart_of_accounts WHERE account_id = ?`, id).
		Scan(&acc.AccountID, &acc.AccountCode, &acc.AccountName, &acc.AccountType, &acc.Balance, &acc.StatementType, &acc.Description, &acc.Status, &acc.Category)
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
var UpdateAccount = func(db DBExecutor, id string, req dtos.UpdateAccountRequest) error {
	// Verify account exists before updating
	exists, err := RecordExists(db, "chart_of_accounts", whereAccountID, id)
	if err != nil {
		// Database error during existence check
		return err
	}
	if !exists {
		// Account not found
		return errors.New(noAccount)
	}
	// Update account name and type (code and balance remain unchanged)
	_, err = db.Exec(`
		UPDATE chart_of_accounts SET account_name = ?, account_type = ?, statement_type = ?, description = ?, status = ?, category = ? WHERE account_id = ?`,
		req.AccountName, req.AccountType, req.StatementType, req.Description, req.Status, req.Category, id,
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
var DeleteAccount = func(db DBExecutor, id string) error {
	// Verify account exists before deletion
	exists, err := RecordExists(db, "chart_of_accounts", whereAccountID, id)
	if err != nil {
		// Database error during existence check
		return err
	}
	if !exists {
		// Account not found
		return errors.New(noAccount)
	}
	// Delete account from chart of accounts
	_, err = db.Exec(`DELETE FROM chart_of_accounts WHERE account_id = ?`, id)
	return err
}

// ===== Journal Entries Management =====

// CreateEntry creates a new journal entry with multiple account lines for double-entry bookkeeping.
// Validates that debits equal credits and that all referenced accounts exist before creation.
// Implements standard accounting practice where each transaction affects at least two accounts.
//
// Parameters:
//   - req: CreateJournalEntryRequest containing entry details and multiple account lines
//
// Returns:
//   - string: Generated entry ID if successful
//   - error: Error if validation fails or database operation fails
var CreateEntry = func(db DBExecutor, req dtos.CreateJournalEntryRequest) (string, error) {
	// Validate that we have at least 2 lines (basic double-entry requirement)
	if len(req.Lines) < 2 {
		return "", errors.New("journal entry must have at least 2 lines")
	}

	// Calculate totals and validate debits equal credits
	var totalDebit, totalCredit float64
	for _, line := range req.Lines {
		totalDebit += line.Debit
		totalCredit += line.Credit

		// Validate each line has only debit OR credit, not both
		if line.Debit > 0 && line.Credit > 0 {
			return "", errors.New("each line must have either debit or credit, not both")
		}
		if line.Debit == 0 && line.Credit == 0 {
			return "", errors.New("each line must have either debit or credit amount")
		}

		// Validate that the account exists
		exists, err := RecordExists(db, "chart_of_accounts", whereAccountID, line.AccountID)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("account with ID %s not found", line.AccountID)
		}
		//validate that the account is active
		var status string
		err = db.QueryRow(`SELECT status FROM chart_of_accounts WHERE account_id = ?`, line.AccountID).Scan(&status)
		if err != nil {
			return "", err
		}
		if status != "active" {
			return "", fmt.Errorf("account with ID %s is not active", line.AccountID)
		}
	}

	// Validate that debits equal credits
	if fmt.Sprintf("%.2f", totalDebit) != fmt.Sprintf("%.2f", totalCredit) {
		return "", fmt.Errorf("debits (%.2f) must equal credits (%.2f)", totalDebit, totalCredit)
	}

	// Generate unique entry ID
	entryID, _ := shortid.Generate()

	// Set entry date to now if not provided
	entryDate := time.Now()
	if req.EntryDate != nil {
		entryDate = StringToTime(*req.EntryDate)
	}

	// Insert journal entry header
	_, err := db.Exec(`
		INSERT INTO journal_entries (entry_id, entry_date, description, reference)
		VALUES (?, ?, ?, ?)`,
		entryID, entryDate, req.Description, req.Reference,
	)
	if err != nil {
		return "", err
	}

	// Insert journal entry lines
	for _, line := range req.Lines {
		lineID, _ := shortid.Generate()
		_, err = db.Exec(`
			INSERT INTO journal_entry_lines (line_id, entry_id, account_id, debit, credit, line_description)
			VALUES (?, ?, ?, ?, ?, ?)`,
			lineID, entryID, line.AccountID, line.Debit, line.Credit, line.LineDescription,
		)
		if err != nil {
			return "", err
		}

		// Update account balance
		balanceChange := line.Debit - line.Credit
		_, err = db.Exec(`
			UPDATE chart_of_accounts 
			SET balance = balance + ? 
			WHERE account_id = ?`,
			balanceChange, line.AccountID,
		)
		if err != nil {
			return "", err
		}
	}

	return entryID, nil
}

// ListEntries retrieves a paginated list of journal entries with their account lines.
// Entries are ordered by date descending (most recent first) for chronological audit trail.
//
// Parameters:
//   - page: Current page number (1-indexed)
//   - size: Number of entries per page
//
// Returns:
//   - []dtos.JournalEntry: List of journal entries with all account lines and balances
//   - dtos.PaginationMeta: Pagination metadata (total items, total pages, has next/prev)
//   - error: Error if database operation fails
var ListEntries = func(db DBExecutor, page, size int, q string) ([]dtos.JournalEntry, dtos.PaginationMeta, error) {
	// Build WHERE clause for filtering
	whereClause := ""
	args := []any{}

	if q != "" {
		whereClause = " WHERE (description LIKE ? OR reference LIKE ?)"
		searchTerm := "%" + q + "%"
		args = append(args, searchTerm, searchTerm)
	}

	// Get total count of journal entries for pagination metadata
	var total int
	countQuery := `SELECT COUNT(*) FROM journal_entries` + whereClause
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Calculate offset for pagination
	offset := (page - 1) * size

	// Fetch paginated entries ordered by entry date descending
	selectQuery := `
		SELECT entry_id, entry_date, description, reference
		FROM journal_entries` + whereClause + `
		ORDER BY entry_date DESC
		LIMIT ? OFFSET ?`

	args = append(args, size, offset)
	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	// Scan entries into slice
	var entries []dtos.JournalEntry
	for rows.Next() {
		var e dtos.JournalEntry
		if err := rows.Scan(&e.EntryID, &e.EntryDate, &e.Description, &e.Reference); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}

		// Fetch lines for this entry
		lines, err := getEntryLines(db, e.EntryID)
		if err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		e.Lines = lines

		// Calculate totals
		for _, line := range lines {
			e.TotalDebit += line.Debit
			e.TotalCredit += line.Credit
		}
		e.IsBalanced = fmt.Sprintf("%.2f", e.TotalDebit) == fmt.Sprintf("%.2f", e.TotalCredit)

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
