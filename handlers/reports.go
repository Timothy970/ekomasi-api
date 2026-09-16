// Package handlers provides HTTP request handlers for financial and business reporting.
// This file contains handlers for generating accounting reports (balance sheet, income statement,
// cash flow, general ledger), CSV exports, and business intelligence reports (top selling products,
// low stock alerts). These reports provide critical insights for business decision-making.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
func BalanceSheet(c *gin.Context) {
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

	// Extract 'as_of' date parameter from query string
	asOfStr := c.Query("as_of")
	var asOf time.Time
	var err error

	if asOfStr == "" {
		asOf = time.Now()
		asOfStr = asOf.Format(date)
	} else {
		// Parse date string into time.Time object
		asOf, err = time.Parse(date, asOfStr)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Reports",
					Description: "Invalid as of date for balance sheet report",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid as of date",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}

	// Extract 'compare_with' date parameter
	compareWithStr := c.Query("compare_with")
	var compareWith *time.Time

	if compareWithStr != "" {
		t, err := time.Parse(date, compareWithStr)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Reports",
					Description: "Invalid comparison date for balance sheet report",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid comparison date",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		compareWith = &t
	}

	// Fetch balance sheet data from database
	sections, err := models.BalanceSheet(asOf, compareWith)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate balance sheet report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Build complete balance sheet response structure
	resp := dtos.BalanceSheetResponse{
		AsOf:        asOf,
		CompareWith: compareWith,
		Sections:    sections,
	}

	// Calculate balance check
	var totalAssets, totalLiabilities, totalEquity float64
	for _, sec := range sections {
		switch sec.SectionName {
		case "Assets":
			totalAssets = sec.Total
		case "Liabilities":
			totalLiabilities = sec.Total
		case "Equity":
			totalEquity = sec.Total
		}
	}

	resp.BalanceCheck = totalAssets - (totalLiabilities + totalEquity)

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: fmt.Sprintf("Balance sheet as of %s generated successfully", asOfStr),
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   fmt.Sprintf("Balance sheet as of %s", asOfStr),
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func IncomeStatement(c *gin.Context) {
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
	// Extract date range parameters from query string
	fromStr := c.Query("from")
	toStr := c.Query("to")
	// Validate that both date parameters are provided
	if fromStr == "" || toStr == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "from and to are required and must be in YYYY-MM-DD format",
				Code:        http.StatusBadRequest,
			},
			Message:   "from and to are required and must be in YYYY-MM-DD format",
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
				Description: "Invalid from date for income statement report",
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
				Description: "Invalid to date for income statement report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidto,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch income statement data from database (revenue, expenses, totals)
	revenue, expenses, tr, te, err := models.IncomeStatement(from, to)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate income statement report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: fmt.Sprintf("Income statement from %s to %s generated successfully", fromStr, toStr),
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Income statement",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
