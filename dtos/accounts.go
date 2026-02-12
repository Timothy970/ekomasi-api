package dtos

import "time"

// Chart of Accounts
type ChartOfAccount struct {
	AccountID     string  `json:"account_id"`
	AccountCode   string  `json:"account_code"`
	AccountName   string  `json:"account_name"`
	AccountType   string  `json:"account_type"`
	Balance       float64 `json:"balance"`
	StatementType *string `json:"statement_type"`
	Description   *string `json:"description"`
	Status        *string `json:"status"`
	Category      *string `json:"category"`
}

type CreateAccountRequest struct {
	AccountCode   *string `json:"account_code"` // Optional - if not provided, will be auto-generated
	AccountName   string  `json:"account_name" validate:"required" `
	AccountType   string  `json:"account_type" validate:"required"`
	StatementType string  `json:"statement_type" validate:"required"`
	Description   *string `json:"description"`
	Status        string  `json:"status" validate:"required,oneof=active archived"`
	Category      string  `json:"category" validate:"required"`
}

type NextAccountCodeResponse struct {
	AccountType string `json:"account_type"`
	NextCode    string `json:"next_code"`
	MinCode     int    `json:"min_code"`
	MaxCode     int    `json:"max_code"`
}

type UpdateAccountRequest struct {
	AccountName   string  `json:"account_name" validate:"required"`
	AccountType   string  `json:"account_type" validate:"required"`
	StatementType string  `json:"statement_type" validate:"required"`
	Description   *string `json:"description"`
	Status        string  `json:"status" validate:"required,oneof=active archived"`
	Category      string  `json:"category" validate:"required"`
}

// Journal Entries
type JournalEntryLine struct {
	LineID          string  `json:"line_id"`
	AccountID       string  `json:"account_id"`
	AccountCode     string  `json:"account_code,omitempty"`
	AccountName     string  `json:"account_name,omitempty"`
	Debit           float64 `json:"debit"`
	Credit          float64 `json:"credit"`
	LineDescription *string `json:"line_description,omitempty"`
}

type JournalEntry struct {
	EntryID     string             `json:"entry_id"`
	OrderID     *string            `json:"order_id,omitempty"`
	PaymentID   *string            `json:"payment_id,omitempty"`
	PoID        *string            `json:"po_id,omitempty"`
	EntryDate   time.Time          `json:"entry_date"`
	Description *string            `json:"description,omitempty"`
	Reference   *string            `json:"reference,omitempty"`
	Lines       []JournalEntryLine `json:"lines"`
	TotalDebit  float64            `json:"total_debit"`
	TotalCredit float64            `json:"total_credit"`
	IsBalanced  bool               `json:"is_balanced"`
}

type JournalEntryLineRequest struct {
	AccountID       string  `json:"account_id" validate:"required"`
	Debit           float64 `json:"debit" validate:"min=0"`
	Credit          float64 `json:"credit" validate:"min=0"`
	LineDescription *string `json:"line_description,omitempty"`
}

type CreateJournalEntryRequest struct {
	EntryDate   *string                   `json:"entry_date,omitempty"`
	Description *string                   `json:"description,omitempty"`
	Reference   *string                   `json:"reference,omitempty"`
	Lines       []JournalEntryLineRequest `json:"lines" validate:"required,min=2,dive"`
}

type UpdateJournalEntryRequest struct {
	EntryDate   *string                   `json:"entry_date,omitempty"`
	Description *string                   `json:"description,omitempty"`
	Reference   *string                   `json:"reference,omitempty"`
	Lines       []JournalEntryLineRequest `json:"lines" validate:"omitempty,min=2,dive"`
}

// AccountCodeRange defines the range of account codes for each account type
type AccountCodeRange struct {
	Min int
	Max int
}

type AccountTypeStats struct {
	AccountType  string  `json:"account_type"`
	TotalAmount  float64 `json:"total_amount"`
	AccountCount int     `json:"account_count"`
}

type AccountStatsResponse struct {
	TotalActive int                `json:"total_active"`
	Archived    int                `json:"archived"`
	TypeStats   []AccountTypeStats `json:"type_stats"`
}
