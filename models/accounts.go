package models

import (
	"adenzo_backend/dtos"
	"errors"
	"math"

	"github.com/teris-io/shortid"
)

var whereAccountID = "account_id = ?"
var whereEntryID = "entry_id = ?"
var noAccount = "account not found"
var noEntry = "entry not found"

func isChartAccountCodeThere(code string) error {
	exists, err := RecordExists("chart_of_accounts", "account_code = ?", code)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("variant not found")
	}
	return nil
}

// Chart of Accounts
var CreateAccount = func(req dtos.CreateAccountRequest) (string, error) {
	err := isChartAccountCodeThere(req.AccountCode)
	if err != nil {
		return "", err
	}
	id, _ := shortid.Generate()

	_, err = DB.Exec(`
		INSERT INTO chart_of_accounts (account_id, account_code, account_name, account_type)
		VALUES (?, ?, ?, ?)`,
		id, req.AccountCode, req.AccountName, req.AccountType,
	)
	return id, err
}

var ListAccounts = func(page, size int) ([]dtos.ChartOfAccount, dtos.PaginationMeta, error) {
	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM chart_of_accounts`).Scan(&total)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	offset := (page - 1) * size
	rows, err := DB.Query(`
		SELECT account_id, account_code, account_name, account_type, balance
		FROM chart_of_accounts
		ORDER BY account_code
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	var accounts []dtos.ChartOfAccount
	for rows.Next() {
		var acc dtos.ChartOfAccount
		if err := rows.Scan(&acc.AccountID, &acc.AccountCode, &acc.AccountName, &acc.AccountType, &acc.Balance); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		accounts = append(accounts, acc)
	}

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

var GetAccount = func(id string) (dtos.ChartOfAccount, error) {
	exists, err := RecordExists("chart_of_accounts", whereAccountID, id)
	if err != nil {
		return dtos.ChartOfAccount{}, err
	}
	if !exists {
		return dtos.ChartOfAccount{}, errors.New(noAccount)
	}
	var acc dtos.ChartOfAccount
	err = DB.QueryRow(`
		SELECT account_id, account_code, account_name, account_type, balance
		FROM chart_of_accounts WHERE account_id = ?`, id).
		Scan(&acc.AccountID, &acc.AccountCode, &acc.AccountName, &acc.AccountType, &acc.Balance)
	return acc, err
}

var UpdateAccount = func(id string, req dtos.UpdateAccountRequest) error {
	exists, err := RecordExists("chart_of_accounts", whereAccountID, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noAccount)
	}
	_, err = DB.Exec(`
		UPDATE chart_of_accounts SET account_name = ?, account_type = ? WHERE account_id = ?`,
		req.AccountName, req.AccountType, id,
	)
	return err
}

var DeleteAccount = func(id string) error {
	exists, err := RecordExists("chart_of_accounts", whereAccountID, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noAccount)
	}
	_, err = DB.Exec(`DELETE FROM chart_of_accounts WHERE account_id = ?`, id)
	return err
}

// ===== Journal Entries =====

var CreateEntry = func(req dtos.CreateJournalEntryRequest) (string, error) {
	exists, err := RecordExists("chart_of_accounts", whereAccountID, req.AccountID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", errors.New(noAccount)
	}
	if req.OrderID != nil {
		err := IsOrderThere(*req.OrderID)
		if err != nil {
			return "", err
		}
	}
	if req.PaymentID != nil {
		err := isPaymentThere(*req.PaymentID)
		if err != nil {
			return "", err
		}
	}
	if req.PoID != nil {
		err := isPurchaseOrderThere(*req.PoID)
		if err != nil {
			return "", err
		}
	}
	id, _ := shortid.Generate()

	_, err = DB.Exec(`
		INSERT INTO journal_entries (entry_id, order_id, payment_id, po_id, account_id, debit, credit, description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.OrderID, req.PaymentID, req.PoID, req.AccountID, req.Debit, req.Credit, req.Description,
	)
	return id, err
}

var ListEntries = func(page, size int) ([]dtos.JournalEntry, dtos.PaginationMeta, error) {
	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM journal_entries`).Scan(&total)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	offset := (page - 1) * size
	rows, err := DB.Query(`
		SELECT entry_id, order_id, payment_id, po_id, account_id, debit, credit, entry_date, description
		FROM journal_entries
		ORDER BY entry_date DESC
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	var entries []dtos.JournalEntry
	for rows.Next() {
		var e dtos.JournalEntry
		if err := rows.Scan(&e.EntryID, &e.OrderID, &e.PaymentID, &e.PoID, &e.AccountID,
			&e.Debit, &e.Credit, &e.EntryDate, &e.Description); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		entries = append(entries, e)
	}

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

var GetEntry = func(id string) (dtos.JournalEntry, error) {
	exists, err := RecordExists("journal_entries", whereEntryID, id)
	if err != nil {
		return dtos.JournalEntry{}, err
	}
	if !exists {
		return dtos.JournalEntry{}, errors.New(noEntry)
	}
	var e dtos.JournalEntry
	err = DB.QueryRow(`
		SELECT entry_id, order_id, payment_id, po_id, account_id, debit, credit, entry_date, description
		FROM journal_entries WHERE entry_id = ?`, id).
		Scan(&e.EntryID, &e.OrderID, &e.PaymentID, &e.PoID, &e.AccountID, &e.Debit, &e.Credit, &e.EntryDate, &e.Description)
	return e, err
}

var UpdateEntry = func(id string, req dtos.UpdateJournalEntryRequest) error {
	exists, err := RecordExists("journal_entries", whereEntryID, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noEntry)
	}
	_, err = DB.Exec(`
		UPDATE journal_entries SET debit = ?, credit = ? WHERE entry_id = ?`,
		req.Debit, req.Credit, id,
	)
	return err
}

var DeleteEntry = func(id string) error {
	exists, err := RecordExists("journal_entries", whereEntryID, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noEntry)
	}
	_, err = DB.Exec(`DELETE FROM journal_entries WHERE entry_id = ?`, id)
	return err
}
