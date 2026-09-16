// Package models provides database models and operations for the accounting system.
// Implements double-entry bookkeeping with chart of accounts and journal entries.
// Supports linking entries to orders, payments, and purchase orders for full audit trail.
package models

import (
	"ekomasi_backend/dtos"
	"errors"
	"fmt"
	"strings"

	"github.com/teris-io/shortid"
)

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
