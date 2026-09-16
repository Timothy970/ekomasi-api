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
	"fmt"
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
