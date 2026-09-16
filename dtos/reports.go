package dtos

import "time"

type DateRange struct {
	From string `form:"from" json:"from"` // YYYY-MM-DD
	To   string `form:"to"   json:"to"`   // YYYY-MM-DD
}

type AsOf struct {
	AsOf string `form:"as_of" json:"as_of"` // YYYY-MM-DD
}

type BalanceSheetAccount struct {
	AccountID       string   `json:"account_id"`
	AccountCode     string   `json:"account_code"`
	AccountName     string   `json:"account_name"`
	Balance         float64  `json:"balance"`
	PreviousBalance *float64 `json:"previous_balance,omitempty"`
	Change          *float64 `json:"change,omitempty"`
	ChangePercent   *float64 `json:"change_percent,omitempty"`
}

type BalanceSheetCategory struct {
	Category      string                `json:"category"`
	Accounts      []BalanceSheetAccount `json:"accounts"`
	Total         float64               `json:"total"`
	PreviousTotal *float64              `json:"previous_total,omitempty"`
	Change        *float64              `json:"change,omitempty"`
	ChangePercent *float64              `json:"change_percent,omitempty"`
}

type BalanceSheetSection struct {
	SectionName   string                 `json:"section_name"` // "Assets", "Liabilities", "Equity"
	Categories    []BalanceSheetCategory `json:"categories"`
	Total         float64                `json:"total"`
	PreviousTotal *float64               `json:"previous_total,omitempty"`
	Change        *float64               `json:"change,omitempty"`
	ChangePercent *float64               `json:"change_percent,omitempty"`
}

type BalanceSheetResponse struct {
	AsOf         time.Time             `json:"as_of"`
	CompareWith  *time.Time            `json:"compare_with,omitempty"`
	Sections     []BalanceSheetSection `json:"sections"`
	BalanceCheck float64               `json:"balance_check"`
}

type BsRow struct {
	AccountID   string
	AccountCode string
	AccountName string
	AccountType string
	Category    string
	Balance     float64
}
type IsRow struct {
	AccountID   string
	AccountCode string
	AccountName string
	Amount      float64
	Type        string
}
type AcctInfo struct {
	ID   string
	Code string
	Name string
	Type string
}
type IncomeStatementLine struct {
	AccountID   string  `json:"account_id"`
	AccountCode string  `json:"account_code"`
	AccountName string  `json:"account_name"`
	Amount      float64 `json:"amount"`
}

type IncomeStatementResponse struct {
	From          time.Time             `json:"from"`
	To            time.Time             `json:"to"`
	Revenue       []IncomeStatementLine `json:"revenue"`
	Expenses      []IncomeStatementLine `json:"expenses"`
	TotalRevenue  float64               `json:"total_revenue"`
	TotalExpenses float64               `json:"total_expenses"`
	NetIncome     float64               `json:"net_income"`
}

type CashFlowRequest struct {
	From           string   `json:"from" validate:"required"`             // YYYY-MM-DD
	To             string   `json:"to" validate:"required"`               // YYYY-MM-DD
	CashAccountIDs []string `json:"cash_account_ids" validate:"required"` // which accounts are “cash”
}

type CashFlowResponse struct {
	From          time.Time `json:"from"`
	To            time.Time `json:"to"`
	TotalInflows  float64   `json:"total_inflows"`
	TotalOutflows float64   `json:"total_outflows"`
	NetChange     float64   `json:"net_change"`
	// Optional: Ending cash of those accounts over the period
	BeginningCash float64 `json:"beginning_cash"`
	EndingCash    float64 `json:"ending_cash"`
}

type LedgerEntry struct {
	EntryID     string    `json:"entry_id"`
	EntryDate   time.Time `json:"entry_date"`
	Description *string   `json:"description,omitempty"`
	Debit       float64   `json:"debit"`
	Credit      float64   `json:"credit"`
	// Running balance after this entry (sign based on account type)
	RunningBalance float64 `json:"running_balance"`
}

type LedgerResponse struct {
	AccountID      string         `json:"account_id"`
	AccountCode    string         `json:"account_code"`
	AccountName    string         `json:"account_name"`
	AccountType    string         `json:"account_type"`
	From           time.Time      `json:"from"`
	To             time.Time      `json:"to"`
	OpeningBalance float64        `json:"opening_balance"`
	Entries        []LedgerEntry  `json:"entries"`
	ClosingBalance float64        `json:"closing_balance"`
	Meta           PaginationMeta `json:"meta"`
}

type TopProduct struct {
	ProductID     string  `json:"product_id"`
	ProductName   string  `json:"product_name"`
	TotalQuantity int     `json:"total_quantity"`
	TotalRevenue  float64 `json:"total_revenue"`
	ProductImage  string  `json:"product_image"`
}
