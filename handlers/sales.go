// Package handlers provides HTTP request handlers for sales analytics and reporting.
// This file contains handlers for sales trends analysis, customer segmentation, regional sales,
// revenue tracking, and various sales performance metrics. Supports multiple export formats
// including JSON, CSV, Excel, and PDF for comprehensive business intelligence reporting.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// Common constants used across sales reporting handlers
var (
	contentType        = "Content-Type"
	totalSales         = "Total Sales"
	contentDisposition = "Content-Disposition"
	avgOrderValue      = "Avg Order Value"
)

// GetSalesTrendsSummary provides a summary of sales trends comparing current and previous periods.
// Calculates period-over-period changes in sales metrics for business performance analysis.
// Useful for dashboards and executive summaries showing growth trends.
//
// @Summary      Get sales trends summary
// @Description  Retrieve sales trends summary with period-over-period comparison
// @Tags         Sales
// @Produce      json
// @Param        start_date  query     string                     false  "Start date (YYYY-MM-DD)"
// @Param        end_date    query     string                     false  "End date (YYYY-MM-DD)"
// @Success      200         {object}  map[string]interface{}    "Sales trends summary"
// @Failure      404         {object}  dtos.ErrorResponse         "Failed to generate report"
// @Router       /api/reports/sales/trends/summary [get]
func GetSalesTrendsSummary(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Parse date range from query parameters (start_date, end_date)
	startTime, endTime, err := ParseDateRange(r)
	// Calculate previous period range for comparison metrics
	// Previous period has the same duration as current period
	diff := endTime.Sub(startTime)
	prevEnd := startTime.Add(-24 * time.Hour)
	prevStart := prevEnd.Add(-diff)

	// Fetch sales trends summary from database with period comparison
	report, err := models.GetSalesTrendsSummary(startTime, endTime, prevStart, prevEnd)
	if err != nil {
		// Database query failed or no data available for the period
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales trends summary report :" + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Return summary report with period-over-period metrics

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales trend summary report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Sales trend summary report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// GetSalesTrendsOverTime provides detailed sales trend data over a specified time period.
// Supports multiple granularity levels (daily, weekly, monthly) for time-series analysis.
// Essential for visualizing sales performance charts and identifying seasonal patterns.
//
// @Summary      Get sales trends over time
// @Description  Retrieve detailed sales trends with configurable time granularity
// @Tags         Sales
// @Produce      json
// @Param        start_date  query     string                     false  "Start date (YYYY-MM-DD)"
// @Param        end_date    query     string                     false  "End date (YYYY-MM-DD)"
// @Param        period      query     string                     false  "Time granularity (daily, weekly, monthly) default: daily"
// @Success      200         {object}  map[string]interface{}    "Sales trends over time"
// @Failure      500         {object}  dtos.ErrorResponse         "Failed to generate report"
// @Router       /api/reports/sales/trends/overtime [get]
func GetSalesTrendsOverTime(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Parse date range from query parameters
	startTime, endTime, err := ParseDateRange(r)
	// Set default period granularity to daily
	period := "daily"
	// Override with query parameter if provided (daily, weekly, monthly)
	periodStr := r.URL.Query().Get("period")
	if periodStr != "" {
		period = periodStr
	}
	// Fetch time-series sales data with specified granularity
	report, err := models.GetSalesTrendsOverTime(period, startTime, endTime)
	if err != nil {
		// Database query failed or invalid period specified
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales trends over time report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Return time-series data for chart visualization

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales trend report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Sales trend report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// GetCustomerSegmentation analyzes customer distribution across different segments.
// Segments customers by purchase behavior, value, or frequency for targeted marketing.
// Helps identify high-value customers and opportunities for customer engagement.
//
// @Summary      Get customer segmentation analysis
// @Description  Retrieve customer segmentation data for the specified period
// @Tags         Sales
// @Produce      json
// @Param        start_date  query     string                              false  "Start date (YYYY-MM-DD)"
// @Param        end_date    query     string                              false  "End date (YYYY-MM-DD)"
// @Success      200         {object}  map[string]interface{}             "Customer segmentation data"
// @Failure      404         {object}  dtos.ErrorResponse                  "Failed to generate report"
// @Router       /api/reports/sales/customer-segmentation [get]
func GetCustomerSegmentation(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Parse date range from query parameters
	start, end, _ := ParseDateRange(r)

	// Fetch customer segmentation data from database
	data, err := models.GetCustomerSegmentation(start, end)
	if err != nil {
		// Database query failed or no customer data available
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get customer segmentation report",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Build response with period metadata and segment data
	resp := dtos.CustomerSegmentationResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
		}{Start: start, End: end},
		Segments: data,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Customer segmentation report",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Customer segmentation report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetSalesOverview provides a high-level summary of overall sales performance.
// Returns key metrics like total sales, order count, average order value, and growth indicators.
// Ideal for executive dashboards and quick performance snapshots.
//
// @Summary      Get sales overview
// @Description  Retrieve high-level sales performance summary with key metrics
// @Tags         Sales
// @Produce      json
// @Success      200  {object}  map[string]interface{}     "Sales overview data"
// @Failure      404  {object}  dtos.ErrorResponse     "Failed to retrieve overview"
// @Router       /api/reports/sales/overview [get]
func GetSalesOverview(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Fetch overall sales summary metrics from database
	data, err := models.GetSalesOverview()
	if err != nil {
		// Database query failed or insufficient data
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales overview",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return sales overview with key performance indicators
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales overview retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   data,
		Message:   "Sales overview retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetSalesByRegion analyzes sales performance broken down by geographic region.
// Supports multiple export formats (JSON, CSV, Excel, PDF) for reporting flexibility.
// Essential for regional sales managers and geographic performance analysis.
//
// @Summary      Get sales by region
// @Description  Retrieve sales performance data segmented by geographic region with export options
// @Tags         Sales
// @Produce      json
// @Param        start_date  query     string                        false  "Start date (YYYY-MM-DD)"
// @Param        end_date    query     string                        false  "End date (YYYY-MM-DD)"
// @Param        export      query     string                        false  "Export format (csv, xlsx, pdf)"
// @Success      200         {object}  map[string]interface{}        "Sales by region data"
// @Failure      404         {object}  dtos.ErrorResponse            "Failed to generate report"
// @Router       /api/reports/sales/by-region [get]
func GetSalesByRegion(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Check if user requested specific export format (csv, xlsx, pdf)
	export := r.URL.Query().Get("export")
	// Parse date range from query parameters
	start, end, _ := ParseDateRange(r)

	// Fetch regional sales data from database
	data, err := models.GetSalesByRegion(start, end)
	if err != nil {
		// Database query failed or no regional data available
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales by region report",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Build response with period metadata and regional breakdown
	resp := dtos.SalesByRegionResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
		}{Start: start, End: end},
		Regions: data,
	}

	// Handle export format requests - generate file and return early
	switch export {
	case "csv":
		// Export as CSV file for spreadsheet import
		exportCSV(w, data)
		return
	case "xlsx":
		// Export as Excel file for advanced analysis
		exportExcel(w, data)
		return
	case "pdf":
		// Export as PDF file for printing/sharing
		exportPDF(w, data)
		return
	}

	// Default: return JSON response

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales segmentation report",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Sales segmentation report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ---------------- CSV Export ----------------
// exportCSV generates a CSV file of regional sales data.
// Creates a downloadable CSV file with headers and data rows.
func exportCSV(w http.ResponseWriter, data []dtos.RegionSales) {
	// Set response headers for CSV file download
	w.Header().Set(contentType, "text/csv")
	w.Header().Set(contentDisposition, "attachment;filename=sales_by_region.csv")

	// Initialize CSV writer
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header row with column names
	writer.Write([]string{"Region", totalSales, avgOrderValue, "Transactions"})
	// Write data rows for each region
	for _, r := range data {
		writer.Write([]string{
			r.Region,
			strconv.FormatFloat(r.TotalSales, 'f', 2, 64),
			strconv.FormatFloat(r.AvgOrderValue, 'f', 2, 64),
			strconv.Itoa(r.Transactions),
		})
	}
}

// ---------------- Excel Export ----------------
// exportExcel generates an Excel (.xlsx) file of regional sales data.
// Creates a formatted Excel spreadsheet with headers and numeric data.
func exportExcel(w http.ResponseWriter, data []dtos.RegionSales) {
	// Create new Excel file
	f := excelize.NewFile()
	sheet := "Sheet1"

	// Write header row to first row
	headers := []string{"Region", totalSales, avgOrderValue, "Transactions"}
	for i, h := range headers {
		// Convert column index to letter (A, B, C, D)
		col := string(rune('A' + i))
		f.SetCellValue(sheet, col+"1", h)
	}

	// Write data rows starting from row 2
	for i, r := range data {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, r.Region)
		f.SetCellValue(sheet, "B"+row, r.TotalSales)
		f.SetCellValue(sheet, "C"+row, r.AvgOrderValue)
		f.SetCellValue(sheet, "D"+row, r.Transactions)
	}

	// Set response headers for Excel file download
	w.Header().Set(contentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set(contentDisposition, "attachment;filename=sales_by_region.xlsx")
	// Write Excel file to response
	_ = f.Write(w)
}

// ---------------- PDF Export ----------------
// exportPDF generates a PDF document of regional sales data.
// Creates a formatted PDF report with title and data table.
func exportPDF(w http.ResponseWriter, data []dtos.RegionSales) {
	// Initialize PDF with portrait orientation, millimeters, A4 size
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	// Add report title
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "Sales by Region Report")
	pdf.Ln(12)

	// Prepare table headers with bold font
	pdf.SetFont("Arial", "B", 10)
	headers := []string{"Region", totalSales, avgOrderValue, "Transactions"}
	colWidths := []float64{50, 40, 40, 40}

	// Draw table header row with borders and centered text
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// Draw table data rows with regular font
	pdf.SetFont("Arial", "", 10)
	for _, r := range data {
		pdf.CellFormat(colWidths[0], 8, r.Region, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[1], 8, strconv.FormatFloat(r.TotalSales, 'f', 2, 64), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[2], 8, strconv.FormatFloat(r.AvgOrderValue, 'f', 2, 64), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[3], 8, strconv.Itoa(r.Transactions), "1", 0, "C", false, 0, "")
		pdf.Ln(-1)
	}

	w.Header().Set(contentType, "application/pdf")
	w.Header().Set(contentDisposition, "attachment;filename=sales_by_region.pdf")
	_ = pdf.Output(w)
}

// GetSalesVsOrdersPerMonth provides monthly sales and order volume comparison for a year.
// Shows correlation between sales revenue and order count to identify seasonal patterns.
// Useful for capacity planning and understanding sales concentration vs distribution.
//
// @Summary      Get sales vs orders per month
// @Description  Retrieve monthly comparison of sales revenue and order count for a specific year
// @Tags         Sales
// @Produce      json
// @Param        year  query     int                          true   "Year (YYYY format)"
// @Success      200   {object}  map[string]interface{}        "Monthly sales and orders data"
// @Failure      400   {object}  map[string]interface{}         "Invalid year parameter"
// @Failure      404   {object}  dtos.ErrorResponse           "Failed to generate report"
// @Router       /api/reports/sales/monthly-comparison [get]
func GetSalesVsOrdersPerMonth(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract year parameter from query string
	yearStr := r.URL.Query().Get("year")
	// Convert year string to integer
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		// Year parameter is missing or not a valid integer
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid year parameter: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid Year Parameter",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Fetch monthly sales and order data for the specified year
	report, err := models.GetSalesVsOrdersPerMonth(year)
	if err != nil {
		// Database query failed or no data for specified year
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales vs orders per month report :" + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return monthly breakdown for trend analysis
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales vs orders per month report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Sales vs orders per month report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// GetRevenueVsExpenses compares revenue against expenses for profitability analysis.
// Supports multiple time filters (week, month, quarter, year) for different analysis periods.
// Critical for understanding profit margins and cost management effectiveness.
//
// @Summary      Get revenue vs expenses comparison
// @Description  Retrieve revenue and expenses comparison with configurable time filter
// @Tags         Sales
// @Produce      json
// @Param        filter  query     string                       false  "Time filter (week, month, quarter, year) default: week"
// @Success      200     {object}  map[string]interface{}       "Revenue and expenses data"
// @Failure      404     {object}  dtos.ErrorResponse           "Failed to generate report"
// @Router       /api/reports/sales/revenue-vs-expenses [get]
func GetRevenueVsExpenses(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract filter parameter from query string
	filterType := r.URL.Query().Get("filter")
	// Set default filter to week if not specified
	filter := "week"
	if filterType != "" {
		filter = filterType
	}

	// Fetch revenue and expense comparison data with specified time filter
	report, err := models.GetRevenueVsExpenses(filter)
	if err != nil {
		// Database query failed or insufficient financial data
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get revenue vs expenses report :" + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message: err.Error(),

			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return profitability analysis data
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Revenue vs expenses report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Revenue vs expenses report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// GetRevenueCustomersOrdersOverview provides a comprehensive business overview.
// Combines revenue, customer count, and order volume into a single dashboard view.
// Essential for executive dashboards and quick business health assessment.
//
// @Summary      Get revenue, customers, and orders overview
// @Description  Retrieve comprehensive overview of revenue, customer count, and order metrics
// @Tags         Sales
// @Produce      json
// @Success      200  {object}  map[string]interface{}     "Business overview data"
// @Failure      404  {object}  dtos.ErrorResponse        "Failed to generate overview"
// @Router       /api/reports/sales/business-overview [get]
func GetRevenueCustomersOrdersOverview(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Fetch comprehensive business metrics from database
	report, err := models.GetRevenueCustomersOrdersOverview()
	if err != nil {
		// Database query failed or insufficient data for comprehensive overview
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get revenue, customers and orders overview :" + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return combined business metrics for dashboard display
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Revenue, customers and orders overview generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Revenue, customers and orders overview generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
