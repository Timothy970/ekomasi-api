// Package handlers provides HTTP request handlers for financial and business reporting.
// This file contains handlers for generating accounting reports (balance sheet, income statement,
// cash flow, general ledger), CSV exports, and business intelligence reports (top selling products,
// low stock alerts). These reports provide critical insights for business decision-making.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Date format constants for error messages
var invalidfrom = "invalid from date"
var invalidto = "invalid to date"

// Standard date format for parsing (YYYY-MM-DD)
var date = "2006-01-02"

// BalanceSheet generates a financial balance sheet report as of a specific date.
// The balance sheet shows the company's financial position including assets, liabilities,
// and equity. It verifies the fundamental accounting equation: Assets = Liabilities + Equity.
//
// @Summary      Generate balance sheet report
// @Description  Retrieve a balance sheet showing assets, liabilities, and equity as of a specific date
// @Tags         Reports
// @Produce      json
// @Param        as_of  query     string                        true  "Report date (YYYY-MM-DD)"
// @Success      200    {object}  dtos.BalanceSheetResponse     "Balance sheet report"
// @Failure      400    {object}  dtos.ErrorResponse            "Invalid date format"
// @Failure      500    {object}  dtos.ErrorResponse            "Internal server error"
// @Security     BearerAuth
// @Router       /reports/balance-sheet [get]
func BalanceSheet(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for viewing financial reports)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Reports", "reports.view")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract 'as_of' date parameter from query string
	asOfStr := r.URL.Query().Get("as_of")
	// Parse date string into time.Time object
	asOf, err := time.Parse(date, asOfStr)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid as of date for balance sheet report",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid as of date",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch balance sheet data from database (assets, liabilities, equity)
	assets, liabs, equity, err := models.BalanceSheet(asOf)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate balance sheet report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Helper function to convert database rows to balance sheet sections
	toSection := func(rows []dtos.BsRow) dtos.BalanceSheetSection {
		out := dtos.BalanceSheetSection{}
		var total float64
		// Iterate through all accounts in this section
		for _, r := range rows {
			// Add account details to the section
			out.Accounts = append(out.Accounts, dtos.BalanceSheetAccount{
				AccountID:   r.AccountID,
				AccountCode: r.AccountCode,
				AccountName: r.AccountName,
				Balance:     r.Balance,
			})
			// Accumulate total balance for this section
			total += r.Balance
		}
		out.Total = total
		return out
	}

	// Build complete balance sheet response structure
	resp := dtos.BalanceSheetResponse{
		AsOf:        asOf,              // Report date
		Assets:      toSection(assets), // All asset accounts
		Liabilities: toSection(liabs),  // All liability accounts
		Equity:      toSection(equity), // All equity accounts
	}
	// Calculate balance check to verify accounting equation (should be 0)
	resp.BalanceCheck = resp.Assets.Total - (resp.Liabilities.Total + resp.Equity.Total)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: fmt.Sprintf("Balance sheet as of %s generated successfully", asOfStr),
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   fmt.Sprintf("Balance sheet as of %s", asOfStr),
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// IncomeStatement generates a profit and loss statement for a specified date range.
// It shows revenue, expenses, and net income (profit or loss) to measure business performance
// over a specific period.
//
// @Summary      Generate income statement report
// @Description  Retrieve an income statement showing revenue, expenses, and net income for a date range
// @Tags         Reports
// @Produce      json
// @Param        from  query     string                          true  "Start date (YYYY-MM-DD)"
// @Param        to    query     string                          true  "End date (YYYY-MM-DD)"
// @Success      200   {object}  dtos.IncomeStatementResponse    "Income statement report"
// @Failure      400   {object}  dtos.ErrorResponse              "Invalid date format or missing parameters"
// @Failure      500   {object}  dtos.ErrorResponse              "Internal server error"
// @Security     BearerAuth
// @Router       /reports/income-statement [get]
func IncomeStatement(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for viewing financial reports)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Reports", "reports.view")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract date range parameters from query string
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	// Validate that both date parameters are provided
	if fromStr == "" || toStr == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "from and to are required and must be in YYYY-MM-DD format",
				Code:        http.StatusBadRequest,
			},
			Message:   "from and to are required and must be in YYYY-MM-DD format",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Parse start date from string to time.Time
	from, err := time.Parse(date, fromStr)
	if err != nil {
		// Invalid start date format, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid from date for income statement report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidfrom,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Parse end date from string to time.Time
	to, err := time.Parse(date, toStr)
	if err != nil {
		// Invalid end date format, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid to date for income statement report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidto,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch income statement data from database (revenue, expenses, totals)
	revenue, expenses, tr, te, err := models.IncomeStatement(from, to)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate income statement report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Helper function to convert database rows to income statement lines
	toLines := func(rows []dtos.IsRow) []dtos.IncomeStatementLine {
		out := make([]dtos.IncomeStatementLine, 0, len(rows))
		// Transform each row into a statement line
		for _, r := range rows {
			out = append(out, dtos.IncomeStatementLine{
				AccountID:   r.AccountID,
				AccountCode: r.AccountCode,
				AccountName: r.AccountName,
				Amount:      r.Amount,
			})
		}
		return out
	}

	// Build complete income statement response structure
	resp := dtos.IncomeStatementResponse{
		From:          from,              // Report start date
		To:            to,                // Report end date
		Revenue:       toLines(revenue),  // All revenue accounts
		Expenses:      toLines(expenses), // All expense accounts
		TotalRevenue:  tr,                // Sum of all revenue
		TotalExpenses: te,                // Sum of all expenses
		NetIncome:     tr - te,           // Profit or loss (revenue - expenses)
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: fmt.Sprintf("Income statement from %s to %s generated successfully", fromStr, toStr),
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Income statement",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CashFlow generates a direct method cash flow statement for specified cash accounts.
// It tracks cash inflows and outflows to show the net change in cash position
// over a specific period.
//
// @Summary      Generate cash flow report
// @Description  Retrieve a cash flow statement showing inflows, outflows, and net cash change for specified accounts
// @Tags         Reports
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.CashFlowRequest      true  "Cash flow request with date range and account IDs"
// @Success      200      {object}  dtos.CashFlowResponse     "Cash flow report"
// @Failure      400      {object}  dtos.ErrorResponse        "Invalid request data"
// @Failure      500      {object}  dtos.ErrorResponse        "Internal server error"
// @Security     BearerAuth
// @Router       /reports/cash-flow [post]
func CashFlow(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for viewing financial reports)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Reports", "reports.view")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.CashFlowRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Reports") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Parse start date from string to time.Time
	from, err := time.Parse(date, req.From)
	if err != nil {
		// Invalid start date format, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid from date for cash flow report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidfrom,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Parse end date from string to time.Time
	to, err := time.Parse(date, req.To)
	if err != nil {
		// Invalid end date format, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid to date for cash flow report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidto,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Fetch cash flow data from database for specified cash accounts
	inflows, outflows, begin, end, err := models.CashFlow(from, to, req.CashAccountIDs)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate cash flow report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Build cash flow response structure
	resp := dtos.CashFlowResponse{
		From:          from,               // Report start date
		To:            to,                 // Report end date
		TotalInflows:  inflows,            // Sum of all cash received
		TotalOutflows: outflows,           // Sum of all cash paid
		NetChange:     inflows - outflows, // Net cash increase/decrease
		BeginningCash: begin,              // Cash balance at start
		EndingCash:    end,                // Cash balance at end
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Cash flow report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Cash flow",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Ledger generates a general ledger report for a specific account.
// It shows all transactions (journal entries) affecting an account with a running balance,
// providing a complete transaction history with pagination support.
//
// @Summary      Generate general ledger report
// @Description  Retrieve a paginated general ledger showing all transactions for a specific account with running balance
// @Tags         Reports
// @Produce      json
// @Param        account_id  path      string                true  "Account ID"
// @Param        from        query     string                true  "Start date (YYYY-MM-DD)"
// @Param        to          query     string                true  "End date (YYYY-MM-DD)"
// @Param        page        query     int                   false "Page number (default: 1)"
// @Param        size        query     int                   false "Page size (default: 10)"
// @Success      200         {object}  dtos.LedgerResponse   "General ledger report"
// @Failure      400         {object}  dtos.ErrorResponse    "Invalid parameters"
// @Failure      500         {object}  dtos.ErrorResponse    "Internal server error"
// @Security     BearerAuth
// @Router       /reports/ledger/{account_id} [get]
func Ledger(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for viewing ledger reports)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Reports", "reports.view")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract account ID from URL path parameters
	accountID := mux.Vars(r)["account_id"]

	// Extract date range parameters from query string
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "from and to are required and in YYYY-MM-DD format",
				Code:        http.StatusBadRequest,
			},
			Message:   "from and to are required YYYY-MM-DD",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Parse start date from string to time.Time
	from, err := time.Parse(date, fromStr)
	if err != nil {
		// Invalid start date format, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid from date for ledger report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidfrom,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Parse end date from string to time.Time
	to, err := time.Parse(date, toStr)
	if err != nil {
		// Invalid end date format, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid to date for ledger report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidto,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Parse pagination parameters from query string
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	// Apply default pagination values if not provided or invalid
	if page < 1 {
		page = 1 // Default to first page
	}
	if size <= 0 {
		size = 10 // Default page size
	}

	// Fetch ledger data from database (account info, opening balance, entries, total count)
	acct, opening, rows, total, err := models.Ledger(accountID, from, to, page, size)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate ledger report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Compute running balance across returned page
	running := opening
	entries := make([]dtos.LedgerEntry, 0, len(rows))
	// Determine sign convention based on account type:
	// Debit-normal accounts (Assets/Expenses): balance increases with debits
	// Credit-normal accounts (Liabilities/Equity/Revenue): balance increases with credits
	debitNormal := acct.Type == "Asset" || acct.Type == "Expense"

	// Process each transaction and calculate running balance
	for _, r := range rows {
		var delta float64
		// Calculate balance change based on account type
		if debitNormal {
			// For debit-normal accounts: debit increases, credit decreases
			delta = r.Debit - r.Credit
		} else {
			// For credit-normal accounts: credit increases, debit decreases
			delta = r.Credit - r.Debit
		}
		// Apply change to running balance
		running += delta
		// Add entry with calculated running balance
		entries = append(entries, dtos.LedgerEntry{
			EntryID:        r.EntryID,
			EntryDate:      r.Date,
			Description:    r.Desc,
			Debit:          r.Debit,
			Credit:         r.Credit,
			RunningBalance: running,
		})
	}

	// Build pagination metadata for navigation
	meta := dtos.PaginationMeta{
		Page:       page,                                           // Current page number
		Size:       size,                                           // Items per page
		TotalItems: total,                                          // Total number of transactions
		TotalPages: int(math.Ceil(float64(total) / float64(size))), // Total pages
		HasPrev:    page > 1,                                       // Has previous page
		HasNext:    page*size < total,                              // Has next page
	}

	// Build complete ledger response structure
	resp := dtos.LedgerResponse{
		AccountID:      acct.ID,   // Account identifier
		AccountCode:    acct.Code, // Account code
		AccountName:    acct.Name, // Account name
		AccountType:    acct.Type, // Account type (Asset, Liability, etc.)
		From:           from,      // Report start date
		To:             to,        // Report end date
		OpeningBalance: opening,   // Balance at start of period
		Entries:        entries,   // Paginated transaction entries
		ClosingBalance: running,   // Balance at end of entries
		Meta:           meta,      // Pagination metadata
	}
	// Return successful response with ledger data
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Ledger report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Ledger",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ExportAccountsCSVHandler exports the chart of accounts to a CSV file.
// It allows filtering by account type and code prefix for customized exports.
//
// @Summary      Export chart of accounts to CSV
// @Description  Download a CSV file containing the chart of accounts with optional filters
// @Tags         Reports
// @Produce      text/csv
// @Param        account_type  query  string  false  "Filter by account type (Asset, Liability, etc.)"
// @Param        code_prefix   query  string  false  "Filter by account code prefix"
// @Success      200           "CSV file download"
// @Failure      500           {object}  dtos.ErrorResponse  "Internal server error"
// @Security     BearerAuth
// @Router       /reports/export/accounts [get]
func ExportAccountsCSVHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for exporting accounts)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Reports", "reports.view")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract optional filter parameters from query string
	accountType := r.URL.Query().Get("account_type")
	codePrefix := r.URL.Query().Get("code_prefix")

	// Set response headers for CSV file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=chart_of_accounts.csv")

	// Generate and stream CSV data directly to response writer
	if err := models.ExportAccountsToCSV(w, accountType, codePrefix); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to export chart of accounts to CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
	}
}

// ExportJournalEntriesCSVHandler exports journal entries to a CSV file.
// It allows filtering by date range and specific account for customized exports.
//
// @Summary      Export journal entries to CSV
// @Description  Download a CSV file containing journal entries with optional date and account filters
// @Tags         Reports
// @Produce      text/csv
// @Param        start_date  query  string  false  "Start date filter (YYYY-MM-DD)"
// @Param        end_date    query  string  false  "End date filter (YYYY-MM-DD)"
// @Param        account_id  query  string  false  "Filter by specific account ID"
// @Success      200         "CSV file download"
// @Failure      500         {object}  dtos.ErrorResponse  "Internal server error"
// @Security     BearerAuth
// @Router       /reports/export/journal-entries [get]
func ExportJournalEntriesCSVHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for exporting journal entries)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Reports", "reports.view")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract optional filter parameters from query string
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	accountID := r.URL.Query().Get("account_id")

	// Set response headers for CSV file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=journal_entries.csv")

	// Generate and stream CSV data directly to response writer
	if err := models.ExportJournalEntriesToCSV(w, startDate, endDate, accountID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to export journal entries to CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
	}
}

// TopSellingProductsReport generates a report of best-selling products.
// It ranks products by sales volume or revenue over a specified time period,
// providing insights for inventory and marketing decisions.
//
// @Summary      Get top selling products report
// @Description  Retrieve a paginated list of best-selling products ranked by sales performance
// @Tags         Reports
// @Produce      json
// @Param        time_range  query  string  false  "Time range (last_7_days, last_30_days, this_month, etc.)"
// @Param        page        query  int     false  "Page number (default: 1)"
// @Param        size        query  int     false  "Page size (default: 10)"
// @Success      200         {object}  map[string]interface{}  "Top selling products with pagination"
// @Failure      500         {object}  dtos.ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /reports/top-selling-products [get]
func TopSellingProductsReport(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for viewing sales reports)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Reports", "reports.view")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract time range filter (e.g., "last_7_days", "last_30_days", "this_month")
	timeRange := r.URL.Query().Get("time_range")
	// Parse pagination parameters from query string
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	// Fetch top selling products from database
	products, pagination, err := models.GetTopSellingProducts(timeRange, page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate top selling products report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Top selling products report generated successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"products":   products,
			"pagination": pagination,
		},
		Message:   "Top selling products report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

//Report to get Low Stock Products
