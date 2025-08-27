package dtos

import "time"

// Chart of Accounts
type ChartOfAccount struct {
	AccountID   string  `json:"account_id"`
	AccountCode string  `json:"account_code"`
	AccountName string  `json:"account_name"`
	AccountType string  `json:"account_type"`
	Balance     float64 `json:"balance"`
}

type CreateAccountRequest struct {
	AccountCode string `json:"account_code" validate:"required"`
	AccountName string `json:"account_name" validate:"required"`
	AccountType string `json:"account_type" validate:"required"`
}

type UpdateAccountRequest struct {
	AccountName string `json:"account_name" validate:"required"`
	AccountType string `json:"account_type" validate:"required"`
}

// Journal Entries
type JournalEntry struct {
	EntryID     string    `json:"entry_id"`
	OrderID     *string   `json:"order_id,omitempty"`
	PaymentID   *string   `json:"payment_id,omitempty"`
	PoID        *string   `json:"po_id,omitempty"`
	AccountID   string    `json:"account_id"`
	Debit       float64   `json:"debit"`
	Credit      float64   `json:"credit"`
	EntryDate   time.Time `json:"entry_date"`
	Description *string   `json:"description,omitempty"`
}

type CreateJournalEntryRequest struct {
	OrderID     *string `json:"order_id,omitempty"`
	PaymentID   *string `json:"payment_id,omitempty"`
	PoID        *string `json:"po_id,omitempty"`
	AccountID   string  `json:"account_id" validate:"required"`
	Debit       float64 `json:"debit" validate:"required"`
	Credit      float64 `json:"credit" validate:"required"`
	Description *string `json:"description,omitempty"`
}

type UpdateJournalEntryRequest struct {
	Debit  float64 `json:"debit" validate:"required"`
	Credit float64 `json:"credit" validate:"required"`
	// Description *string `json:"description,omitempty"`
}
