// Package models provides data access functions for financial and sales reporting.
//
// This file handles accounting and analytics reports including:
//   - Balance Sheet (assets, liabilities, equity as of a date)
//   - Income Statement (revenue and expenses for a period)
//   - Cash Flow Statement (direct method with inflows/outflows)
//   - General Ledger (detailed account transactions with running balance)
//   - CSV exports for accounts and journal entries
//   - Top-selling products analytics (daily/weekly/monthly/yearly)
//
// The accounting system uses double-entry bookkeeping with:
//   - Debit-normal accounts: Assets, Expenses (balance = debit - credit)
//   - Credit-normal accounts: Liabilities, Equity, Income (balance = credit - debit)
//   - Normal balance calculations adjust for account type
package models

import (
	"ekomasi_backend/dtos"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// ===== Cash Flow (Direct) =====

// CashFlow generates a cash flow statement using the direct method.
//
// The direct method tracks actual cash movements:
//   - Inflows: Credits to cash accounts (cash received)
//   - Outflows: Debits to cash accounts (cash paid out)
//   - Beginning cash: Sum of cash account balances before period start
//   - Ending cash: Sum of cash account balances through period end
//
// Parameters:
//   - from: time.Time - Start date of the period
//   - to: time.Time - End date of the period (inclusive)
//   - cashAccountIDs: []string - Array of account IDs representing cash accounts
//     (e.g., "Cash on Hand", "Bank Account")
//
// Returns:
//   - inflows: float64 - Total cash inflows (credits to cash accounts)
//   - outflows: float64 - Total cash outflows (debits to cash accounts)
//   - beginCash: float64 - Cash balance at start of period
//   - endCash: float64 - Cash balance at end of period
//   - err: error - "cashAccountIDs is required", database error, or nil on success
//
// Formula: endCash = beginCash + inflows - outflows
func CashFlow(from, to time.Time, cashAccountIDs []string) (inflows, outflows, beginCash, endCash float64, err error) {
	// Validate cash account IDs are provided
	if len(cashAccountIDs) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("cashAccountIDs is required")
	}

	// Build IN clause placeholders for dynamic account list
	placeholders := strings.Repeat("?,", len(cashAccountIDs))
	placeholders = strings.TrimRight(placeholders, ",")

	// Helper function to build query arguments
	args := func(extra ...any) []any {
		a := make([]any, 0, len(cashAccountIDs)+len(extra))
		for _, id := range cashAccountIDs {
			a = append(a, id)
		}
		a = append(a, extra...)
		return a
	}

	// Calculate inflows (credits) and outflows (debits) within period
	qFlows := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(je.credit),0) AS inflows,
			COALESCE(SUM(je.debit),0)  AS outflows
		FROM journal_entries je
		WHERE je.account_id IN (%s)
		  AND je.entry_date >= ? AND je.entry_date < DATE_ADD(?, INTERVAL 1 DAY)
	`, placeholders)

	if err = DB.QueryRow(qFlows, args(from, to)...).Scan(&inflows, &outflows); err != nil {
		return
	}

	// Calculate beginning cash: sum normal balance before period start
	qBegin := fmt.Sprintf(`
		SELECT COALESCE(SUM(%s),0)
		FROM chart_of_accounts ca
		LEFT JOIN journal_entries je ON je.account_id = ca.account_id
			AND je.entry_date < ?
		WHERE ca.account_id IN (%s)
	`, normalBalanceExpr("je"), placeholders)
	if err = DB.QueryRow(qBegin, append([]any{from}, toAny(cashAccountIDs)...)...).Scan(&beginCash); err != nil {
		return
	}

	// Calculate ending cash: sum normal balance through period end
	qEnd := fmt.Sprintf(`
		SELECT COALESCE(SUM(%s),0)
		FROM chart_of_accounts ca
		LEFT JOIN journal_entries je ON je.account_id = ca.account_id
			AND je.entry_date <= ?
		WHERE ca.account_id IN (%s)
	`, normalBalanceExpr("je"), placeholders)
	if err = DB.QueryRow(qEnd, append([]any{to}, toAny(cashAccountIDs)...)...).Scan(&endCash); err != nil {
		return
	}

	return
}

// toAny converts a string slice to an any slice for SQL query arguments.
//
// Parameters:
//   - ss: []string - String slice to convert
//
// Returns:
//   - []any - Interface slice suitable for DB.Query variadic args
func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// ===== General Ledger =====

// getAccount retrieves account details from the chart of accounts.
//
// This is an internal helper function used by the Ledger function.
//
// Parameters:
//   - accountID: string - The account_id to retrieve
//
// Returns:
//   - dtos.AcctInfo: Account information (ID, Code, Name, Type)
//   - error: sql.ErrNoRows if not found, database error, or nil on success
func getAccount(accountID string) (dtos.AcctInfo, error) {
	var a dtos.AcctInfo
	err := isAccountThere(accountID)
	if err != nil {
		return a, err
	}
	// Query account details from chart of accounts
	err = DB.QueryRow(`
		SELECT account_id, account_code, account_name, account_type
		FROM chart_of_accounts WHERE account_id = ?`, accountID).
		Scan(&a.ID, &a.Code, &a.Name, &a.Type)
	return a, err
}

// helper function to check if account is there
func isAccountThere(accountID string) error {
	// Check if account code exists in chart_of_accounts table
	exists, err := RecordExists(DB, "chart_of_accounts", "account_id = ?", accountID)
	if err != nil {
		// Database query failed
		return err
	}
	if !exists {
		// Account not found in chart of accounts
		return errors.New("account not found")
	}
	return nil
}

// openingBalance calculates the account balance before a specified date.
//
// This is used to show the starting balance for a ledger period. It sums all
// journal entries from the beginning of time up to (but not including) the from date.
//
// Parameters:
//   - accountID: string - The account to calculate opening balance for
//   - from: time.Time - The date from which the ledger period begins (exclusive)
//
// Returns:
//   - float64: Opening balance using normal balance calculation (debit-credit or credit-debit)
//   - error: Database error or nil on success
func openingBalance(accountID string, from time.Time) (float64, error) {
	// Use normal balance expression for proper sign calculation
	expr := normalBalanceExpr("je")
	var bal float64

	// Sum all entries before the from date
	err := DB.QueryRow(fmt.Sprintf(`
		SELECT COALESCE(SUM(%s), 0)
		FROM journal_entries je
		INNER JOIN chart_of_accounts ca
			ON ca.account_id = je.account_id
		WHERE je.account_id = ?
		  AND je.entry_date < ?
	`, expr), accountID, from).Scan(&bal)
	return bal, err
}

// Ledger generates a general ledger report for a specific account with pagination.
//
// The general ledger shows detailed transaction history for an account including:
//   - Opening balance (balance before the period)
//   - All journal entries in the date range (entry_id, date, description, debit, credit)
//   - Pagination support for large ledgers
//
// Parameters:
//   - accountID: string - The account_id to generate ledger for
//   - from: time.Time - Start date of the period (inclusive)
//   - to: time.Time - End date of the period (inclusive, adds 1 day for proper range)
//   - page: int - Page number (1-based)
//   - size: int - Number of entries per page
//
// Returns:
//   - acct: dtos.AcctInfo - Account information (ID, Code, Name, Type)
//   - opening: float64 - Opening balance before the period start
//   - entries: []struct - Array of journal entries with:
//   - EntryID: Unique entry identifier
//   - Date: Transaction date
//   - Desc: Optional description (*string)
//   - Debit: Debit amount
//   - Credit: Credit amount
//   - metaCount: int - Total number of entries in the date range (for pagination)
//   - err: error - Database error or nil on success
func Ledger(accountID string, from, to time.Time, page, size int) (acct dtos.AcctInfo, opening float64, entries []struct {
	EntryID string
	Date    time.Time
	Desc    *string
	Debit   float64
	Credit  float64
}, metaCount int, err error) {

	// Retrieve account details
	acct, err = getAccount(accountID)
	if err != nil {
		log.Printf("get account error : %s", err)
		return
	}

	// Calculate opening balance before period start
	opening, err = openingBalance(accountID, from)
	if err != nil {
		log.Printf("opening balance error : %s", err)
		return
	}

	// Count total entries in date range for pagination metadata
	err = DB.QueryRow(`
		SELECT COUNT(*) FROM journal_entries
		WHERE account_id = ? 
		  AND entry_date >= ? AND entry_date < DATE_ADD(?, INTERVAL 1 DAY)
	`, accountID, from, to).Scan(&metaCount)
	if err != nil {
		log.Printf("count entries error : %s", err)
		return
	}

	// Calculate pagination offset
	offset := (page - 1) * size

	// Query paginated journal entries
	rows, err := DB.Query(`
		SELECT entry_id, entry_date, description, debit, credit
		FROM journal_entries
		WHERE account_id = ?
		  AND entry_date >= ? AND entry_date < DATE_ADD(?, INTERVAL 1 DAY)
		ORDER BY entry_date, entry_id
		LIMIT ? OFFSET ?
	`, accountID, from, to, size, offset)
	if err != nil {
		log.Printf("query entries error : %s", err)
		return
	}
	defer rows.Close()

	// Scan journal entries
	for rows.Next() {
		var e struct {
			EntryID string
			Date    time.Time
			Desc    *string
			Debit   float64
			Credit  float64
		}
		if err = rows.Scan(&e.EntryID, &e.Date, &e.Desc, &e.Debit, &e.Credit); err != nil {
			return
		}
		entries = append(entries, e)
	}

	return
}
