// Package models provides database models and operations for the accounting system.
// Implements double-entry bookkeeping with chart of accounts and journal entries.
// Supports linking entries to orders, payments, and purchase orders for full audit trail.
package models

import (
	"ekomasi_backend/dtos"
	"errors"
	"fmt"
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
	args := []any{}

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
