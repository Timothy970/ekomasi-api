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
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"time"
)

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
	var args []any

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
	var args []any

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
