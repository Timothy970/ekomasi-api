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
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

// normalBalanceExpr generates SQL expression for calculating account normal balance.
//
// This helper function implements double-entry bookkeeping rules:
//   - Debit-normal accounts (Assets, Expenses): balance = debit - credit
//   - Credit-normal accounts (Liabilities, Equity, Income): balance = credit - debit
//
// Parameters:
//   - tableAlias: string - The SQL table alias for journal_entries (e.g., "je")
//
// Returns:
//   - string: SQL CASE expression that calculates balance based on account type
func normalBalanceExpr(tableAlias string) string {
	// Generate SQL CASE expression for account balance calculation
	// Assets/Expenses increase with debits (debit-normal)
	// Liabilities/Equity/Income increase with credits (credit-normal)
	return fmt.Sprintf(`
		CASE 
			WHEN ca.account_type IN ('Asset','Expense') THEN (%s.debit - %s.credit)
			ELSE (%s.credit - %s.debit)
		END
	`, tableAlias, tableAlias, tableAlias, tableAlias)
}

// ===== Balance Sheet =====

// BalanceSheet generates a balance sheet report as of a specific date.
//
// The balance sheet shows the financial position at a point in time by summing
// all journal entries from the beginning through the as-of date. It returns
// three separate arrays for Assets, Liabilities, and Equity accounts.
//
// Parameters:
//   - asOf: time.Time - The date for which to generate the balance sheet (inclusive)
//
// Returns:
//   - assets: []dtos.BsRow - Array of asset accounts with balances
//   - liabilities: []dtos.BsRow - Array of liability accounts with balances
//   - equity: []dtos.BsRow - Array of equity accounts with balances
//   - err: error - Database error or nil on success
//
// Each BsRow contains: AccountID, AccountCode, AccountName, AccountType, Balance
func BalanceSheet(asOf time.Time, compareWith *time.Time) (sections []dtos.BalanceSheetSection, err error) {
	// Helper to fetch balances for a specific date
	fetchBalances := func(date time.Time) (map[string]float64, error) {
		q := fmt.Sprintf(`
			SELECT 
				ca.account_id,
				COALESCE(SUM(%s), 0) AS balance
			FROM chart_of_accounts ca
			LEFT JOIN journal_entries je
				ON je.entry_date <= ?
			LEFT JOIN journal_entry_lines jel 
				ON jel.entry_id = je.entry_id
				AND jel.account_id = ca.account_id
			GROUP BY ca.account_id
		`, normalBalanceExpr("jel"))

		rows, err := DB.Query(q, date)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		balances := make(map[string]float64)
		for rows.Next() {
			var accountID string
			var balance float64
			if err := rows.Scan(&accountID, &balance); err != nil {
				return nil, err
			}
			balances[accountID] = balance
		}
		return balances, nil
	}

	// Fetch current balances
	currentBalances, err := fetchBalances(asOf)
	if err != nil {
		return nil, err
	}

	// Fetch comparison balances if date provided
	var compareBalances map[string]float64
	if compareWith != nil {
		compareBalances, err = fetchBalances(*compareWith)
		if err != nil {
			return nil, err
		}
	}

	// Fetch all accounts with details
	qAccounts := `
		SELECT 
			account_id, account_code, account_name, account_type, category
		FROM chart_of_accounts
		ORDER BY account_code
	`
	rows, err := DB.Query(qAccounts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Organize data structure
	// Struct structure: Section -> Category -> Account
	type accountData struct {
		ID       string
		Code     string
		Name     string
		Type     string
		Category string
	}

	// Map to hold sections
	sectionMap := make(map[string]map[string][]accountData)
	sectionOrder := []string{"Asset", "Liability", "Equity"} // Defined order

	for rows.Next() {
		var a accountData
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.Type, &a.Category); err != nil {
			return nil, err
		}

		// Initialize maps if nil
		if sectionMap[a.Type] == nil {
			sectionMap[a.Type] = make(map[string][]accountData)
		}
		sectionMap[a.Type][a.Category] = append(sectionMap[a.Type][a.Category], a)
	}

	// Build final result
	// Map section types to proper display names
	sectionNames := map[string]string{
		"Asset":     "Assets",
		"Liability": "Liabilities",
		"Equity":    "Equity",
	}

	for _, secType := range sectionOrder {
		if categories, ok := sectionMap[secType]; ok {
			section := dtos.BalanceSheetSection{
				SectionName: sectionNames[secType],
			}

			var secTotal, secPrevTotal float64

			// Process categories
			for catName, accounts := range categories {
				category := dtos.BalanceSheetCategory{
					Category: catName,
				}
				var catTotal, catPrevTotal float64

				for _, acc := range accounts {
					bal := currentBalances[acc.ID]
					bsAccount := dtos.BalanceSheetAccount{
						AccountID:   acc.ID,
						AccountCode: acc.Code,
						AccountName: acc.Name,
						Balance:     bal,
					}

					catTotal += bal

					// Handle comparison
					if compareWith != nil {
						prevBal := compareBalances[acc.ID]
						bsAccount.PreviousBalance = &prevBal

						diff := bal - prevBal
						bsAccount.Change = &diff

						if prevBal != 0 {
							pct := (diff / prevBal) * 100
							bsAccount.ChangePercent = &pct
						}

						catPrevTotal += prevBal
					}

					category.Accounts = append(category.Accounts, bsAccount)
				}

				category.Total = catTotal
				secTotal += catTotal

				if compareWith != nil {
					category.PreviousTotal = &catPrevTotal
					diff := catTotal - catPrevTotal
					category.Change = &diff
					if catPrevTotal != 0 {
						pct := (diff / catPrevTotal) * 100
						category.ChangePercent = &pct
					}
					secPrevTotal += catPrevTotal
				}

				section.Categories = append(section.Categories, category)
			}

			section.Total = secTotal
			if compareWith != nil {
				section.PreviousTotal = &secPrevTotal
				diff := secTotal - secPrevTotal
				section.Change = &diff
				if secPrevTotal != 0 {
					pct := (diff / secPrevTotal) * 100
					section.ChangePercent = &pct
				}
			}

			sections = append(sections, section)
		}
	}

	return sections, nil
}

// ===== Income Statement =====

// IncomeStatement generates an income statement (profit & loss) for a date range.
//
// The income statement shows financial performance over a period by calculating:
//   - Revenue (Income accounts): amount = credit - debit (credit-normal)
//   - Expenses: amount = debit - credit (debit-normal)
//   - Net income = total revenue - total expenses
//
// Parameters:
//   - from: time.Time - Start date of the period (inclusive)
//   - to: time.Time - End date of the period (inclusive, adds 1 day for proper range)
//
// Returns:
//   - revenue: []dtos.IsRow - Array of income account rows with amounts
//   - expenses: []dtos.IsRow - Array of expense account rows with amounts
//   - totalRevenue: float64 - Sum of all revenue
//   - totalExpenses: float64 - Sum of all expenses
//   - err: error - Database error or nil on success
//
// Each IsRow contains: AccountID, AccountCode, AccountName, Type, Amount
func IncomeStatement(from, to time.Time) (revenue, expenses []dtos.IsRow, totalRevenue, totalExpenses float64, err error) {
	// Query income and expense accounts with proper sign calculation
	q := `
		SELECT 
			ca.account_id,
			ca.account_code,
			ca.account_name,
			ca.account_type,
			COALESCE(SUM(
				CASE 
					WHEN ca.account_type = 'Income' THEN (je.credit - je.debit)
					WHEN ca.account_type = 'Expense' THEN (je.debit - je.credit)
					ELSE 0
				END
			), 0) AS amount
		FROM chart_of_accounts ca
		LEFT JOIN journal_entries je 
			ON je.account_id = ca.account_id
			AND je.entry_date >= ? AND je.entry_date < DATE_ADD(?, INTERVAL 1 DAY)
		WHERE ca.account_type IN ('Income','Expense')
		GROUP BY ca.account_id, ca.account_code, ca.account_name, ca.account_type
		ORDER BY ca.account_code
	`
	rows, err := DB.Query(q, from, to)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	defer rows.Close()

	// Scan and categorize into revenue/expenses with totals
	var rev, exp []dtos.IsRow
	var tr, te float64
	for rows.Next() {
		var row dtos.IsRow
		if err := rows.Scan(&row.AccountID, &row.AccountCode, &row.AccountName, &row.Type, &row.Amount); err != nil {
			return nil, nil, 0, 0, err
		}
		// Separate income and expenses (exclude zero balances)
		if row.Type == "Income" && row.Amount != 0 {
			rev = append(rev, row)
			tr += row.Amount // Accumulate total revenue
		} else if row.Type == "Expense" && row.Amount != 0 {
			exp = append(exp, row)
			te += row.Amount // Accumulate total expenses
		}
	}
	return rev, exp, tr, te, nil
}

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
	args := func(extra ...interface{}) []interface{} {
		a := make([]interface{}, 0, len(cashAccountIDs)+len(extra))
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
	if err = DB.QueryRow(qBegin, append([]interface{}{from}, toAny(cashAccountIDs)...)...).Scan(&beginCash); err != nil {
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
	if err = DB.QueryRow(qEnd, append([]interface{}{to}, toAny(cashAccountIDs)...)...).Scan(&endCash); err != nil {
		return
	}

	return
}

// toAny converts a string slice to an interface{} slice for SQL query arguments.
//
// Parameters:
//   - ss: []string - String slice to convert
//
// Returns:
//   - []interface{} - Interface slice suitable for DB.Query variadic args
func toAny(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
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

// ExportAccountsToCSV exports chart of accounts to CSV format with optional filtering.
//
// This function generates a CSV export of accounts from the chart of accounts with
// optional filters for account type and code prefix.
//
// Parameters:
//   - w: io.Writer - The writer to output CSV data (e.g., http.ResponseWriter, file)
//   - accountType: string - Optional filter for account type ("Asset", "Liability", "Equity", "Income", "Expense")
//     Empty string = all types
//   - codePrefix: string - Optional filter for account codes starting with prefix (e.g., "1000" for all 1000-series)
//     Empty string = all codes
//
// Returns:
//   - error: Database error or nil on success
//
// CSV Format:
//
//	Header: Account ID, Code, Name, Type, Balance
//	Data: One row per account with formatted balance (2 decimal places)
//	Data: One row per account with formatted balance (2 decimal places)
func ExportAccountsToCSV(w io.Writer, accountType, codePrefix, q string) error {
	// Build dynamic query with optional filters
	query := `
		SELECT account_id, account_code, account_name, account_type, balance
		FROM chart_of_accounts
		WHERE 1=1
	`
	var args []interface{}

	// Add account type filter if provided
	if accountType != "" {
		query += " AND account_type = ?"
		args = append(args, accountType)
	}

	// Add code prefix filter if provided (uses LIKE for pattern matching)
	if codePrefix != "" {
		query += " AND account_code LIKE ?"
		args = append(args, codePrefix+"%")
	}

	// Add q filter if provided
	if q != "" {
		query += " AND (account_name LIKE ? OR account_code LIKE ? OR account_type LIKE ?)"
		searchTerm := "%" + q + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Sort by account code for logical ordering
	query += " ORDER BY account_code"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Initialize CSV writer
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write CSV header row
	if err := csvWriter.Write([]string{"Account ID", "Code", "Name", "Type", "Balance"}); err != nil {
		return err
	}

	// Write data rows
	for rows.Next() {
		var id, code, name, accType string
		var balance float64
		if err := rows.Scan(&id, &code, &name, &accType, &balance); err != nil {
			return err
		}
		// Format balance with 2 decimal places
		record := []string{id, code, name, accType, fmt.Sprintf("%.2f", balance)}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	return rows.Err()
}

// ExportBalanceSheetToCSV exports the balance sheet to a CSV file.
func ExportBalanceSheetToCSV(w io.Writer, asOf time.Time, compareWith *time.Time) error {
	sections, err := BalanceSheet(asOf, compareWith)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Header: Section, Category, Account Code, Account Name, Balance
	header := []string{"Section", "Category", "Account Code", "Account Name", "Balance"}
	if compareWith != nil {
		header = append(header, "Previous Balance", "Change", "% Change")
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	for _, section := range sections {
		for _, category := range section.Categories {
			for _, acc := range category.Accounts {
				record := []string{
					section.SectionName,
					category.Category,
					acc.AccountCode,
					acc.AccountName,
					fmt.Sprintf("%.2f", acc.Balance),
				}

				if compareWith != nil {
					prev := 0.0
					if acc.PreviousBalance != nil {
						prev = *acc.PreviousBalance
					}
					change := 0.0
					if acc.Change != nil {
						change = *acc.Change
					}
					pct := 0.0
					if acc.ChangePercent != nil {
						pct = *acc.ChangePercent
					}

					record = append(record,
						fmt.Sprintf("%.2f", prev),
						fmt.Sprintf("%.2f", change),
						fmt.Sprintf("%.2f%%", pct),
					)
				}

				if err := csvWriter.Write(record); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ExportJournalEntriesToCSV exports journal entries to CSV format with optional filtering.
//
// This function generates a CSV export of journal entries with optional filters for
// date range and account ID.
//
// Parameters:
//   - w: io.Writer - The writer to output CSV data (e.g., http.ResponseWriter, file)
//   - startDate: string - Optional start date filter (format: "YYYY-MM-DD")
//     Empty string = no start date filter
//   - endDate: string - Optional end date filter (format: "YYYY-MM-DD")
//     Empty string = no end date filter
//   - accountID: string - Optional filter for specific account ID
//     Empty string = all accounts
//   - q: string - Optional search query for description or reference
//     Empty string = no search filter
//
// Returns:
//   - error: Database error or nil on success
//
// CSV Format:
//
//	Header: Entry ID, Order ID, Payment ID, PO ID, Account ID, Debit, Credit, Entry Date, Description
//	Data: One row per journal entry with formatted amounts (2 decimal places)
//	      Sorted by entry_date DESC (newest first)
func ExportJournalEntriesToCSV(w io.Writer, startDate, endDate, accountID, q string) error {
	// Build dynamic query with optional filters
	query := `
		SELECT 
			je.entry_id, 
			je.reference,
			jel.account_id, 
			jel.debit, 
			jel.credit, 
			je.entry_date, 
			je.description,
			ca.account_name,
			ca.account_type
		FROM journal_entries je
		JOIN journal_entry_lines jel ON je.entry_id = jel.entry_id
		JOIN chart_of_accounts ca ON jel.account_id = ca.account_id
		WHERE 1=1
	`
	var args []interface{}

	// Add start date filter if provided
	if startDate != "" {
		query += " AND je.entry_date >= ?"
		args = append(args, startDate)
	}

	// Add end date filter if provided
	if endDate != "" {
		query += " AND je.entry_date <= ?"
		args = append(args, endDate)
	}

	// Add account filter if provided
	if accountID != "" {
		query += " AND jel.account_id = ?"
		args = append(args, accountID)
	}

	// Add q filter if provided
	if q != "" {
		query += " AND (je.description LIKE ? OR je.reference LIKE ? OR jel.line_description LIKE ?)"
		searchTerm := "%" + q + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Sort by date descending (newest first)
	query += " ORDER BY je.entry_date DESC, jel.line_id ASC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Initialize CSV writer
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write CSV header row
	if err := csvWriter.Write([]string{
		"Entry ID",
		"Reference", "Account Name", "Account Type",
		"Debit", "Credit", "Entry Date", "Description",
	}); err != nil {
		return err
	}

	// Write data rows
	for rows.Next() {
		var (
			entryID, reference, accountName, accountType, accID string
			debit, credit                                       float64
			entryDate                                           time.Time
			description                                         sql.NullString
		)
		if err := rows.Scan(&entryID, &reference, &accID, &debit, &credit, &entryDate, &description, &accountName, &accountType); err != nil {
			return err
		}
		// Format record with 2 decimal places for amounts, formatted date
		record := []string{
			entryID,
			reference,
			accountName,
			accountType,
			fmt.Sprintf("%.2f", debit),
			fmt.Sprintf("%.2f", credit),
			entryDate.Format("2006-01-02 15:04:05"),
			description.String, // Empty string if NULL
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	return rows.Err()
}

// GetTopSellingProducts retrieves top-selling products with sales analytics.
//
// This function generates a sales report showing products ranked by quantity sold,
// with optional time-based filtering (daily, weekly, monthly, yearly).
//
// Parameters:
//   - timeRange: string - Time filter: "daily", "weekly", "monthly", "yearly", or "" (all time)
//   - page: int - Page number (1-based)
//   - size: int - Number of products per page
//
// Returns:
//   - []dtos.TopProduct: Array of top-selling products containing:
//   - ProductID: Unique product identifier
//   - ProductName: Product name
//   - TotalQuantity: Total units sold
//   - ProductImage: First product image URL (may be empty)
//   - TotalRevenue: Total revenue from product sales
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
//
// Time Ranges:
//   - "daily": Products sold today
//   - "weekly": Products sold in the last 7 days (including today)
//   - "monthly": Products sold in the current month
//   - "yearly": Products sold in the current year
//   - "": All-time top sellers (no date filter)
func GetTopSellingProducts(timeRange string, page, size int) ([]dtos.TopProduct, *dtos.PaginationMeta, error) {
	var totalCount int
	now := time.Now()
	normalizedTimeRange := normalizeTopSellingTimeRange(timeRange)
	const dateRangeFilter = "o.created_at >= ? AND o.created_at < ?"

	// Build date filter
	dateFilter := ""
	switch normalizedTimeRange {
	case "daily":
		dateFilter = dateRangeFilter
	case "weekly":
		dateFilter = dateRangeFilter
	case "monthly":
		dateFilter = dateRangeFilter
	case "yearly":
		dateFilter = dateRangeFilter
	}

	offset := (page - 1) * size

	// Base WHERE clause (only completed and paid orders)
	whereClause := "WHERE o.status IN ('completed', 'COMPLETED') AND o.payment_status IN ('PAID', 'paid')"
	if dateFilter != "" {
		whereClause += " AND " + dateFilter
	}

	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT oi.product_id
			FROM order_items oi
			JOIN orders o ON oi.order_id = o.order_id
			` + whereClause + `
			GROUP BY oi.product_id
		) AS grouped_products
	`

	err := DB.QueryRow(countQuery, getDateFilterArgs(normalizedTimeRange, now)...).Scan(&totalCount)
	if err != nil {
		return nil, nil, err
	}

	query := `
		SELECT 
			p.product_id,
			p.name AS product_name,
			SUM(oi.quantity) AS total_quantity,
			pi.url AS product_image,
			SUM(oi.quantity * oi.unit_price) AS total_revenue
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		JOIN products p ON oi.product_id = p.product_id
		LEFT JOIN (
			SELECT product_id, MIN(url) AS url
			FROM product_images
			GROUP BY product_id
		) pi ON pi.product_id = p.product_id
		` + whereClause + `
		GROUP BY p.product_id, p.name, pi.url
		ORDER BY total_quantity DESC
		LIMIT ? OFFSET ?
	`

	args := append(getDateFilterArgs(normalizedTimeRange, now), size, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var results []dtos.TopProduct
	for rows.Next() {
		var tp dtos.TopProduct
		var image sql.NullString

		if err := rows.Scan(
			&tp.ProductID,
			&tp.ProductName,
			&tp.TotalQuantity,
			&image,
			&tp.TotalRevenue,
		); err != nil {
			return nil, nil, err
		}

		if image.Valid {
			tp.ProductImage = image.String
		}

		results = append(results, tp)
	}

	meta := &dtos.PaginationMeta{
		TotalItems: totalCount,
		Page:       page,
		Size:       size,
		TotalPages: (totalCount + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    offset+size < totalCount,
	}

	return results, meta, nil
}

// normalizeTopSellingTimeRange normalizes time_range aliases used by clients.
func normalizeTopSellingTimeRange(timeRange string) string {
	switch strings.ToLower(strings.TrimSpace(timeRange)) {
	case "daily", "day", "today":
		return "daily"
	case "weekly", "week", "this_week", "last_7_days", "last7days":
		return "weekly"
	case "monthly", "month", "this_month", "montly":
		return "monthly"
	case "yearly", "year", "this_year", "annual":
		return "yearly"
	default:
		return ""
	}
}

// getDateFilterArgs builds SQL query arguments for date filtering.
//
// This is an internal helper function that generates the appropriate query arguments
// based on the time range filter.
//
// Parameters:
//   - timeRange: string - The time range filter ("daily", "weekly", "monthly", "yearly", or "")
//   - now: time.Time - The current timestamp for date calculations
//
// Returns:
//   - []interface{}: Array of arguments for SQL query:
//   - "daily": [startOfDay, startOfNextDay]
//   - "weekly": [startOfDay(6 days ago), startOfNextDay]
//   - "monthly": [startOfMonth, startOfNextMonth]
//   - "yearly": [startOfYear, startOfNextYear]
//   - "": [] - Empty array (no date filter)
func getDateFilterArgs(timeRange string, now time.Time) []interface{} {
	loc := now.Location()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	startOfNextDay := startOfDay.AddDate(0, 0, 1)

	switch timeRange {
	case "daily":
		return []interface{}{startOfDay, startOfNextDay}
	case "weekly":
		startOfLast7Days := startOfDay.AddDate(0, 0, -6)
		return []interface{}{startOfLast7Days, startOfNextDay}
	case "monthly":
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		startOfNextMonth := startOfMonth.AddDate(0, 1, 0)
		return []interface{}{startOfMonth, startOfNextMonth}
	case "yearly":
		startOfYear := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, loc)
		startOfNextYear := startOfYear.AddDate(1, 0, 0)
		return []interface{}{startOfYear, startOfNextYear}
	default:
		// No date filter - return empty array
		return []interface{}{}
	}
}
