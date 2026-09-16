// Package handlers provides HTTP request handlers for financial and business reporting.
// This file contains handlers for generating accounting reports (balance sheet, income statement,
// cash flow, general ledger), CSV exports, and business intelligence reports (top selling products,
// low stock alerts). These reports provide critical insights for business decision-making.
package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
func ExportAccountsCSVHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for exporting accounts)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Reports", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract optional filter parameters from query string
	accountType := c.Query("account_type")
	codePrefix := c.Query("code_prefix")
	q := c.Query("q")

	// Set response headers for CSV file download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=chart_of_accounts.csv")

	// Generate and stream CSV data directly to response writer
	if err := models.ExportAccountsToCSV(c.Writer, accountType, codePrefix, q); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to export chart of accounts to CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
	}
}

// ExportBalanceSheetCSVHandler exports the balance sheet to a CSV file.
//
// @Summary      Export balance sheet to CSV
// @Description  Download a CSV file containing the balance sheet report
// @Tags         Reports
// @Produce      text/csv
// @Param        as_of         query  string  false  "Report date (YYYY-MM-DD)"
// @Param        compare_with  query  string  false  "Comparison date (YYYY-MM-DD)"
// @Success      200           "CSV file download"
// @Failure      500           {object}  dtos.ErrorResponse  "Internal server error"
// @Security     BearerAuth
// @Router       /reports/export/balance-sheet [get]
func ExportBalanceSheetCSVHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Reports", "")
	if !ok {
		return
	}

	// Extract 'as_of' date parameter
	asOfStr := c.Query("as_of")
	var asOf time.Time
	var err error

	if asOfStr == "" {
		asOf = time.Now()
	} else {
		asOf, err = time.Parse(date, asOfStr)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Reports",
					Description: "Invalid as of date for balance sheet export",
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
					Description: "Invalid comparison date for balance sheet export",
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

	// Set response headers for CSV file download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=balance_sheet.csv")

	// Generate and stream CSV data directly to response writer
	if err := models.ExportBalanceSheetToCSV(c.Writer, asOf, compareWith); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to export balance sheet to CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
func ExportJournalEntriesCSVHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for exporting journal entries)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Reports", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract optional filter parameters from query string
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	accountID := c.Query("account_id")
	q := c.Query("q")

	// Set response headers for CSV file download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=journal_entries.csv")

	// Generate and stream CSV data directly to response writer
	if err := models.ExportJournalEntriesToCSV(c.Writer, startDate, endDate, accountID, q); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to export journal entries to CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
// @Param        time_range  query  string  false  "Time range (daily, weekly, monthly, yearly)"
// @Param        page        query  int     false  "Page number (default: 1)"
// @Param        size        query  int     false  "Page size (default: 10)"
// @Success      200         {object}  map[string]any  "Top selling products with pagination"
// @Failure      500         {object}  dtos.ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /reports/top-selling-products [get]
func TopSellingProductsReport(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for viewing sales reports)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Reports", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract time range filter (e.g., "last_7_days", "last_30_days", "this_month")
	timeRange := c.Query("time_range")
	// Parse pagination parameters from query string
	page, limit := parsePagination(c.Query("page"), c.Query("size"))

	// Fetch top selling products from database
	products, pagination, err := models.GetTopSellingProducts(timeRange, page, limit)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate top selling products report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
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
		Request:   c.Request,
		RawBody:   requestSummary})
}

//Report to get Low Stock Products
