package models

import (
	"adenzo_backend/dtos"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

// Helper: normal balance multiplier (Assets/Expenses are debit-normal = +1 for (debit-credit))
// Liabilities/Equity/Income are credit-normal = +1 for (credit-debit)
func normalBalanceExpr(tableAlias string) string {
	// MySQL CASE for sign
	// returns (debit - credit) for debit-normal accounts, else (credit - debit)
	// account_type is in chart_of_accounts
	return fmt.Sprintf(`
		CASE 
			WHEN ca.account_type IN ('Asset','Expense') THEN (%s.debit - %s.credit)
			ELSE (%s.credit - %s.debit)
		END
	`, tableAlias, tableAlias, tableAlias, tableAlias)
}

// ===== Balance Sheet =====

// Balance sheet as-of a date: sum all journal entries up to and including asOf.
func BalanceSheet(asOf time.Time) (assets, liabilities, equity []dtos.BsRow, err error) {
	q := fmt.Sprintf(`
		SELECT 
			ca.account_id,
			ca.account_code,
			ca.account_name,
			ca.account_type,
			COALESCE(SUM(%s), 0) AS balance
		FROM chart_of_accounts ca
		LEFT JOIN journal_entries je 
			ON je.account_id = ca.account_id
			AND je.entry_date <= ?
		GROUP BY ca.account_id, ca.account_code, ca.account_name, ca.account_type
		ORDER BY ca.account_code
	`, normalBalanceExpr("je"))

	rows, err := DB.Query(q, asOf)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	var a, l, e []dtos.BsRow
	for rows.Next() {
		var row dtos.BsRow
		if err := rows.Scan(&row.AccountID, &row.AccountCode, &row.AccountName, &row.AccountType, &row.Balance); err != nil {
			return nil, nil, nil, err
		}
		switch row.AccountType {
		case "Asset":
			a = append(a, row)
		case "Liability":
			l = append(l, row)
		case "Equity":
			e = append(e, row)
		}
	}
	return a, l, e, nil
}

// ===== Income Statement =====

// Income (credit-normal): amount = (credit - debit)
// Expense (debit-normal): amount = (debit - credit)

func IncomeStatement(from, to time.Time) (revenue, expenses []dtos.IsRow, totalRevenue, totalExpenses float64, err error) {
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

	var rev, exp []dtos.IsRow
	var tr, te float64
	for rows.Next() {
		var row dtos.IsRow
		if err := rows.Scan(&row.AccountID, &row.AccountCode, &row.AccountName, &row.Type, &row.Amount); err != nil {
			return nil, nil, 0, 0, err
		}
		if row.Type == "Income" && row.Amount != 0 {
			rev = append(rev, row)
			tr += row.Amount
		} else if row.Type == "Expense" && row.Amount != 0 {
			exp = append(exp, row)
			te += row.Amount
		}
	}
	return rev, exp, tr, te, nil
}

// ===== Cash Flow (Direct) =====

// Direct method driven by cash account IDs:
// inflows = SUM(credits to cash accounts), outflows = SUM(debits from cash accounts)
func CashFlow(from, to time.Time, cashAccountIDs []string) (inflows, outflows, beginCash, endCash float64, err error) {
	if len(cashAccountIDs) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("cashAccountIDs is required")
	}
	// Build IN clause
	placeholders := strings.Repeat("?,", len(cashAccountIDs))
	placeholders = strings.TrimRight(placeholders, ",")

	// args helper
	args := func(extra ...interface{}) []interface{} {
		a := make([]interface{}, 0, len(cashAccountIDs)+len(extra))
		for _, id := range cashAccountIDs {
			a = append(a, id)
		}
		a = append(a, extra...)
		return a
	}

	// Inflows and outflows within period
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

	// Beginning cash: sum normal balance of cash accounts prior to From
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

	// Ending cash: sum normal balance up to To
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

func toAny(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// ===== General Ledger =====

func getAccount(ctx context.Context, accountID string) (dtos.AcctInfo, error) {
	var a dtos.AcctInfo
	err := DB.QueryRowContext(ctx, `
		SELECT account_id, account_code, account_name, account_type
		FROM chart_of_accounts WHERE account_id = ?`, accountID).
		Scan(&a.ID, &a.Code, &a.Name, &a.Type)
	return a, err
}

// Opening balance up to (but NOT including) From
func openingBalance(ctx context.Context, accountID string, from time.Time, accountType string) (float64, error) {
	expr := normalBalanceExpr("je")
	var bal float64
	err := DB.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COALESCE(SUM(%s),0)
		FROM journal_entries je
		WHERE je.account_id = ? AND je.entry_date < ?
	`, expr), accountID, from).Scan(&bal)
	return bal, err
}

// Ledger entries with pagination and running balance
func Ledger(accountID string, from, to time.Time, page, size int) (acct dtos.AcctInfo, opening float64, entries []struct {
	EntryID string
	Date    time.Time
	Desc    *string
	Debit   float64
	Credit  float64
}, metaCount int, err error) {

	ctx := context.Background()
	acct, err = getAccount(ctx, accountID)
	if err != nil {
		return
	}

	// Opening balance before 'from'
	opening, err = openingBalance(ctx, accountID, from, acct.Type)
	if err != nil {
		return
	}

	// Count entries in range
	err = DB.QueryRow(`
		SELECT COUNT(*) FROM journal_entries
		WHERE account_id = ? 
		  AND entry_date >= ? AND entry_date < DATE_ADD(?, INTERVAL 1 DAY)
	`, accountID, from, to).Scan(&metaCount)
	if err != nil {
		return
	}

	offset := (page - 1) * size
	rows, err := DB.Query(`
		SELECT entry_id, entry_date, description, debit, credit
		FROM journal_entries
		WHERE account_id = ?
		  AND entry_date >= ? AND entry_date < DATE_ADD(?, INTERVAL 1 DAY)
		ORDER BY entry_date, entry_id
		LIMIT ? OFFSET ?
	`, accountID, from, to, size, offset)
	if err != nil {
		return
	}
	defer rows.Close()

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

func ExportAccountsToCSV(w io.Writer, accountType, codePrefix string) error {
	query := `
		SELECT account_id, account_code, account_name, account_type, balance
		FROM chart_of_accounts
		WHERE 1=1
	`
	var args []interface{}

	if accountType != "" {
		query += " AND account_type = ?"
		args = append(args, accountType)
	}
	if codePrefix != "" {
		query += " AND account_code LIKE ?"
		args = append(args, codePrefix+"%")
	}

	query += " ORDER BY account_code"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write([]string{"Account ID", "Code", "Name", "Type", "Balance"}); err != nil {
		return err
	}

	for rows.Next() {
		var id, code, name, accType string
		var balance float64
		if err := rows.Scan(&id, &code, &name, &accType, &balance); err != nil {
			return err
		}
		record := []string{id, code, name, accType, fmt.Sprintf("%.2f", balance)}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	return rows.Err()
}
func ExportJournalEntriesToCSV(w io.Writer, startDate, endDate, accountID string) error {
	query := `
		SELECT entry_id, order_id, payment_id, po_id, account_id, debit, credit, entry_date, description
		FROM journal_entries
		WHERE 1=1
	`
	var args []interface{}

	if startDate != "" {
		query += " AND entry_date >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND entry_date <= ?"
		args = append(args, endDate)
	}
	if accountID != "" {
		query += " AND account_id = ?"
		args = append(args, accountID)
	}

	query += " ORDER BY entry_date DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write([]string{
		"Entry ID", "Order ID", "Payment ID", "PO ID", "Account ID",
		"Debit", "Credit", "Entry Date", "Description",
	}); err != nil {
		return err
	}

	for rows.Next() {
		var (
			entryID, orderID, paymentID, poID, accID string
			debit, credit                            float64
			entryDate                                time.Time
			description                              sql.NullString
		)
		if err := rows.Scan(&entryID, &orderID, &paymentID, &poID, &accID, &debit, &credit, &entryDate, &description); err != nil {
			return err
		}
		record := []string{
			entryID,
			orderID,
			paymentID,
			poID,
			accID,
			fmt.Sprintf("%.2f", debit),
			fmt.Sprintf("%.2f", credit),
			entryDate.Format("2006-01-02 15:04:05"),
			description.String,
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	return rows.Err()
}
