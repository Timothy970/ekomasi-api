// Package handlers provides HTTP request handlers for financial and business reporting.
// This file contains handlers for generating accounting reports (balance sheet, income statement,
// cash flow, general ledger), CSV exports, and business intelligence reports (top selling products,
// low stock alerts). These reports provide critical insights for business decision-making.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

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
func CashFlow(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for viewing financial reports)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Reports", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.CashFlowRequest](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Reports") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Parse start date from string to time.Time
	from, err := time.Parse(date, req.From)
	if err != nil {
		// Invalid start date format, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid from date for cash flow report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidfrom,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Parse end date from string to time.Time
	to, err := time.Parse(date, req.To)
	if err != nil {
		// Invalid end date format, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid to date for cash flow report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidto,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Fetch cash flow data from database for specified cash accounts
	inflows, outflows, begin, end, err := models.CashFlow(from, to, req.CashAccountIDs)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate cash flow report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Cash flow report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Cash flow",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func Ledger(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for viewing ledger reports)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Reports", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract account ID from URL path parameters
	accountID := c.Param("account_id")

	// Extract date range parameters from query string
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "from and to are required and in YYYY-MM-DD format",
				Code:        http.StatusBadRequest,
			},
			Message:   "from and to are required YYYY-MM-DD",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Parse start date from string to time.Time
	from, err := time.Parse(date, fromStr)
	if err != nil {
		// Invalid start date format, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid from date for ledger report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidfrom,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Parse end date from string to time.Time
	to, err := time.Parse(date, toStr)
	if err != nil {
		// Invalid end date format, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid to date for ledger report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidto,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Parse pagination parameters from query string
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
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
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate ledger report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Ledger report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Ledger",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
