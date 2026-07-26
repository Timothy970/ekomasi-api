// Package models provides database models and operations for the accounting system.
// Implements double-entry bookkeeping with chart of accounts and journal entries.
// Supports linking entries to orders, payments, and purchase orders for full audit trail.
package models

import (
	"ekomasi_backend/dtos"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

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
func isChartAccountCodeThere(db DBExecutor, code string) error {
	// Check if account code exists in chart_of_accounts table
	exists, err := RecordExists(db, "chart_of_accounts", "account_code = ?", code)
	if err != nil {
		// Database query failed
		return err
	}
	if exists {
		// Account code found
		return errors.New("account code already exists")
	}
	return nil
}

// GetNextAccountCode retrieves the next available account code for a given account type.
// Each account type has a dedicated range:
// - Asset: 1000-1999
// - Liability: 2000-2999
// - Equity: 3000-3999
// - Revenue: 4000-4999
// - Expense: 5000-5999
//
// Parameters:
//   - accountType: The type of account (Asset/Liability/Equity/Revenue/Expense)
//
// Returns:
//   - string: The next available account code
//   - error: Error if account type is invalid or database operation fails
var GetNextAccountCode = func(db DBExecutor, accountType string, codeRange dtos.AccountCodeRange) (string, error) {
	// Find the highest existing code within this account type's range
	var maxCode *int
	err := db.QueryRow(`
		SELECT MAX(CAST(account_code AS UNSIGNED))
		FROM chart_of_accounts
		WHERE CAST(account_code AS UNSIGNED) BETWEEN ? AND ?`,
		codeRange.Min, codeRange.Max).Scan(&maxCode)

	if err != nil {
		return "", err
	}

	// If no accounts exist in this range, start at the minimum
	var nextCode int
	if maxCode == nil {
		nextCode = codeRange.Min
	} else {
		nextCode = *maxCode + 1
	}

	// Check if we've exceeded the maximum code for this account type
	if nextCode > codeRange.Max {
		return "", fmt.Errorf("no available account codes for %s accounts (range %d-%d is full)",
			accountType, codeRange.Min, codeRange.Max)
	}

	return fmt.Sprintf("%d", nextCode), nil
}

// ===== Chart of Accounts Management =====

// CreateAccount creates a new account in the chart of accounts.
// Validates account code exists before creation and generates unique account ID.
// If account code is not provided, it will be auto-generated based on account type.
//
// Parameters:
//   - req: CreateAccountRequest containing account code (optional), name, and type (Asset/Liability/Equity/Revenue/Expense)
//   - accountCode: The account code to use for the new account (auto-generated or provided)
//
// Returns:
//   - string: Generated account ID if successful
//   - error: Error if account code invalid or database operation fails
var CreateAccount = func(db DBExecutor, req dtos.CreateAccountRequest, accountCode string) (string, error) {
	var err error
	// Validate that the account code doesn't already exist
	err = isChartAccountCodeThere(db, accountCode)
	if err != nil {
		return "", err
	}
	//check if account name and description already exist
	err = isChartAccountNameAndDescriptionThere(db, req.AccountName, *req.Description)
	if err != nil {
		return "", err
	}

	// Generate unique account ID
	id, _ := shortid.Generate()

	// Insert new account into chart of accounts
	_, err = db.Exec(`
		INSERT INTO chart_of_accounts (account_id, account_code, account_name, account_type, statement_type, description, status, category)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, accountCode, req.AccountName, req.AccountType, req.StatementType, req.Description, req.Status, req.Category,
	)
	return id, err
}

// helper function to check if account name and description already exist in chart of accounts
// parameters: account name and description
// returns: error if account name and description already exist, nil if they don't
func isChartAccountNameAndDescriptionThere(db DBExecutor, name, description string) error {
	exists, err := RecordExists(db, "chart_of_accounts", "account_name = ?", name)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("account name already exists")
	}
	exists, err = RecordExists(db, "chart_of_accounts", "description = ?", description)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("account description already exists")
	}
	return nil
}

// ListAccounts retrieves a paginated list of accounts from the chart of accounts.
// Accounts are ordered by account code for consistent presentation.
//
// Parameters:
//   - page: Current page number (1-indexed)
//   - size: Number of accounts per page
//   - accountType: Optional filter by account type
//   - q: Optional search query for account name or code
//
// Returns:
//   - []dtos.ChartOfAccount: List of accounts with ID, code, name, type, and current balance
//   - dtos.PaginationMeta: Pagination metadata (total items, total pages, has next/prev)
//   - error: Error if database operation fails
var ListAccounts = func(db DBExecutor, page, size int, accountType, q string) ([]dtos.ChartOfAccount, dtos.PaginationMeta, error) {
	// Build WHERE clause for filtering
	whereClause := ""
	args := []interface{}{}

	if accountType != "" || q != "" {
		whereClause = " WHERE"
		conditions := []string{}

		if accountType != "" {
			conditions = append(conditions, " account_type = ?")
			args = append(args, accountType)
		}

		if q != "" {
			conditions = append(conditions, " (account_name LIKE ? OR account_code LIKE ? OR account_type LIKE ?)")
			searchTerm := "%" + q + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		for i, condition := range conditions {
			if i > 0 {
				whereClause += " AND"
			}
			whereClause += condition
		}
	}

	// Get total count of accounts for pagination metadata
	var total int
	countQuery := `SELECT COUNT(*) FROM chart_of_accounts` + whereClause
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		// Failed to get count
		return nil, dtos.PaginationMeta{}, err
	}

	// Calculate offset for pagination
	offset := (page - 1) * size
	// Fetch paginated accounts ordered by account code
	selectQuery := `
		SELECT account_id, account_code, account_name, account_type, balance, statement_type, description, status, category
		FROM chart_of_accounts` + whereClause + `
		ORDER BY account_code
		LIMIT ? OFFSET ?`

	args = append(args, size, offset)
	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		// Query failed
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	// Scan accounts into slice
	var accounts []dtos.ChartOfAccount
	for rows.Next() {
		var acc dtos.ChartOfAccount
		if err := rows.Scan(&acc.AccountID, &acc.AccountCode, &acc.AccountName, &acc.AccountType, &acc.Balance, &acc.StatementType, &acc.Description, &acc.Status, &acc.Category); err != nil {
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

// GetAccountStats retrieves statistics about the chart of accounts.
// Includes total active/archived counts and grouped balance/count by account type.
//
// Returns:
//   - dtos.AccountStatsResponse: Statistics overview
//   - error: Error if database operation fails
var GetAccountStats = func(db DBExecutor) (dtos.AccountStatsResponse, error) {
	var stats dtos.AccountStatsResponse

	// Get total active accounts
	err := db.QueryRow(`SELECT COUNT(*) FROM chart_of_accounts WHERE status = 'active'`).Scan(&stats.TotalActive)
	if err != nil {
		return stats, err
	}

	// Get total archived accounts
	err = db.QueryRow(`SELECT COUNT(*) FROM chart_of_accounts WHERE status = 'archived'`).Scan(&stats.Archived)
	if err != nil {
		return stats, err
	}

	// Get stats grouped by account type
	// Initialize all 5 account types with zero values
	accountTypes := []string{"asset", "liability", "equity", "revenue", "expense"}
	typeStatsMap := make(map[string]dtos.AccountTypeStats)
	for _, accType := range accountTypes {
		typeStatsMap[accType] = dtos.AccountTypeStats{
			AccountType:  accType,
			TotalAmount:  0,
			AccountCount: 0,
		}
	}

	rows, err := db.Query(`
		SELECT LOWER(account_type), COALESCE(SUM(balance), 0), COUNT(*)
		FROM chart_of_accounts
		GROUP BY LOWER(account_type)`)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var typeStat dtos.AccountTypeStats
		if err := rows.Scan(&typeStat.AccountType, &typeStat.TotalAmount, &typeStat.AccountCount); err != nil {
			return stats, err
		}
		typeStatsMap[typeStat.AccountType] = typeStat
	}

	// Convert map to slice in consistent order
	for _, accType := range accountTypes {
		stats.TypeStats = append(stats.TypeStats, typeStatsMap[accType])
	}

	return stats, nil
}

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
	args := []interface{}{}

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

// getEntryLines retrieves all lines for a journal entry with account details
func getEntryLines(db DBExecutor, entryID string) ([]dtos.JournalEntryLine, error) {
	rows, err := db.Query(`
		SELECT 
			jel.line_id, 
			jel.account_id, 
			ca.account_code,
			ca.account_name,
			jel.debit, 
			jel.credit, 
			jel.line_description
		FROM journal_entry_lines jel
		JOIN chart_of_accounts ca ON jel.account_id = ca.account_id
		WHERE jel.entry_id = ?
		ORDER BY jel.line_id`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []dtos.JournalEntryLine
	for rows.Next() {
		var line dtos.JournalEntryLine
		if err := rows.Scan(&line.LineID, &line.AccountID, &line.AccountCode, &line.AccountName,
			&line.Debit, &line.Credit, &line.LineDescription); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// GetEntry retrieves a single journal entry by its ID with all account lines.
//
// Parameters:
//   - id: Journal entry ID to retrieve
//
// Returns:
//   - dtos.JournalEntry: Entry details including all account lines with debits/credits
//   - error: Error if entry not found or database operation fails
var GetEntry = func(db DBExecutor, id string) (dtos.JournalEntry, error) {
	// Check if journal entry exists before querying
	exists, err := RecordExists(db, "journal_entries", whereEntryID, id)
	if err != nil {
		return dtos.JournalEntry{}, err
	}
	if !exists {
		return dtos.JournalEntry{}, errors.New(noEntry)
	}

	// Fetch journal entry header
	var e dtos.JournalEntry
	err = db.QueryRow(`
		SELECT entry_id, order_id, payment_id, po_id, entry_date, description, reference
		FROM journal_entries WHERE entry_id = ?`, id).
		Scan(&e.EntryID, &e.OrderID, &e.PaymentID, &e.PoID, &e.EntryDate, &e.Description, &e.Reference)
	if err != nil {
		return dtos.JournalEntry{}, err
	}

	// Fetch entry lines
	lines, err := getEntryLines(db, id)
	if err != nil {
		return dtos.JournalEntry{}, err
	}
	e.Lines = lines

	// Calculate totals
	for _, line := range lines {
		e.TotalDebit += line.Debit
		e.TotalCredit += line.Credit
	}
	e.IsBalanced = fmt.Sprintf("%.2f", e.TotalDebit) == fmt.Sprintf("%.2f", e.TotalCredit)

	return e, nil
}

// UpdateEntry updates an existing journal entry header and optionally its lines.
// If lines are provided, all existing lines are deleted and replaced with new ones.
//
// Parameters:
//   - id: Journal entry ID to update
//   - req: UpdateJournalEntryRequest containing updated entry details
//
// Returns:
//   - error: Error if entry not found, validation fails, or database operation fails
var UpdateEntry = func(db DBExecutor, id string, req dtos.UpdateJournalEntryRequest) error {
	// Verify entry exists before updating
	exists, err := RecordExists(db, "journal_entries", whereEntryID, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noEntry)
	}

	// Update entry header if fields are provided
	if req.EntryDate != nil || req.Description != nil || req.Reference != nil {
		query := "UPDATE journal_entries SET "
		args := []interface{}{}
		updates := []string{}

		if req.EntryDate != nil {
			updates = append(updates, "entry_date = ?")
			args = append(args, req.EntryDate)
		}
		if req.Description != nil {
			updates = append(updates, "description = ?")
			args = append(args, req.Description)
		}
		if req.Reference != nil {
			updates = append(updates, "reference = ?")
			args = append(args, req.Reference)
		}

		if len(updates) > 0 {
			query += strings.Join(updates, ", ") + " WHERE entry_id = ?"
			args = append(args, id)
			_, err = db.Exec(query, args...)
			if err != nil {
				return err
			}
		}
	}

	// Update lines if provided
	if req.Lines != nil && len(req.Lines) > 0 {
		// Validate at least 2 lines
		if len(req.Lines) < 2 {
			return errors.New("journal entry must have at least 2 lines")
		}

		// Validate debits equal credits
		var totalDebit, totalCredit float64
		for _, line := range req.Lines {
			totalDebit += line.Debit
			totalCredit += line.Credit

			if line.Debit > 0 && line.Credit > 0 {
				return errors.New("each line must have either debit or credit, not both")
			}
			if line.Debit == 0 && line.Credit == 0 {
				return errors.New("each line must have either debit or credit amount")
			}

			// Validate account exists
			exists, err := RecordExists(db, "chart_of_accounts", whereAccountID, line.AccountID)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("account with ID %s not found", line.AccountID)
			}
		}

		if fmt.Sprintf("%.2f", totalDebit) != fmt.Sprintf("%.2f", totalCredit) {
			return fmt.Errorf("debits (%.2f) must equal credits (%.2f)", totalDebit, totalCredit)
		}

		// Get old lines to reverse account balances
		oldLines, err := getEntryLines(db, id)
		if err != nil {
			return err
		}

		// Reverse old balances
		for _, line := range oldLines {
			balanceChange := -(line.Debit - line.Credit)
			_, err = db.Exec(`UPDATE chart_of_accounts SET balance = balance + ? WHERE account_id = ?`,
				balanceChange, line.AccountID)
			if err != nil {
				return err
			}
		}

		// Delete old lines
		_, err = db.Exec(`DELETE FROM journal_entry_lines WHERE entry_id = ?`, id)
		if err != nil {
			return err
		}

		// Insert new lines and update balances
		for _, line := range req.Lines {
			lineID, _ := shortid.Generate()
			_, err = db.Exec(`
				INSERT INTO journal_entry_lines (line_id, entry_id, account_id, debit, credit, line_description)
				VALUES (?, ?, ?, ?, ?, ?)`,
				lineID, id, line.AccountID, line.Debit, line.Credit, line.LineDescription)
			if err != nil {
				return err
			}

			// Apply new balance
			balanceChange := line.Debit - line.Credit
			_, err = db.Exec(`UPDATE chart_of_accounts SET balance = balance + ? WHERE account_id = ?`,
				balanceChange, line.AccountID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DeleteEntry removes a journal entry and all its lines from the system.
// Reverses all account balance changes before deletion.
//
// Parameters:
//   - id: Journal entry ID to delete
//
// Returns:
//   - error: Error if entry not found or database operation fails
var DeleteEntry = func(db DBExecutor, id string) error {
	// Verify entry exists before deletion
	exists, err := RecordExists(db, "journal_entries", whereEntryID, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noEntry)
	}

	// Get entry lines to reverse account balances
	lines, err := getEntryLines(db, id)
	if err != nil {
		return err
	}

	// Reverse account balances
	for _, line := range lines {
		balanceChange := -(line.Debit - line.Credit)
		_, err = db.Exec(`UPDATE chart_of_accounts SET balance = balance + ? WHERE account_id = ?`,
			balanceChange, line.AccountID)
		if err != nil {
			return err
		}
	}

	// Delete journal entry (lines will cascade delete)
	_, err = db.Exec(`DELETE FROM journal_entries WHERE entry_id = ?`, id)
	return err
}
