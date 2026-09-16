// Package handlers provides HTTP request handlers for sales analytics and reporting.
// This file contains handlers for sales trends analysis, customer segmentation, regional sales,
// revenue tracking, and various sales performance metrics. Supports multiple export formats
// including JSON, CSV, Excel, and PDF for comprehensive business intelligence reporting.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// @Description  Retrieve sales performance data segmented by geographic region with export options
// @Tags         Sales
// @Produce      json
// @Param        start_date  query     string                        false  "Start date (YYYY-MM-DD)"
// @Param        end_date    query     string                        false  "End date (YYYY-MM-DD)"
// @Param        export      query     string                        false  "Export format (csv, xlsx, pdf)"
// @Success      200         {object}  map[string]any        "Sales by region data"
// @Failure      404         {object}  dtos.ErrorResponse            "Failed to generate report"
// @Router       /api/reports/sales/by-region [get]
func GetSalesByRegion(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Check if user requested specific export format (csv, xlsx, pdf)
	export := c.Query("export")
	// Parse date range from query parameters
	startDate, endDate, _ := ParseDateRange(c.Request)

	// Fetch regional sales data from database
	data, err := models.GetSalesByRegion(startDate, endDate)
	if err != nil {
		// Database query failed or no regional data available
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales by region report",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Build response with period metadata and regional breakdown
	resp := dtos.SalesByRegionResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
		}{Start: startDate, End: endDate},
		Regions: data,
	}

	// Handle export format requests - generate file and return early
	switch export {
	case "csv":
		// Export as CSV file for spreadsheet import
		exportCSV(c, data)
		return
	case "xlsx":
		// Export as Excel file for advanced analysis
		exportExcel(c, data)
		return
	case "pdf":
		// Export as PDF file for printing/sharing
		exportPDF(c, data)
		return
	}

	// Default: return JSON response

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales segmentation report",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Sales segmentation report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ---------------- CSV Export ----------------
// exportCSV generates a CSV file of regional sales data.
// Creates a downloadable CSV file with headers and data rows.
func exportCSV(c *gin.Context, data []dtos.RegionSales) {
	// Set response headers for CSV file download
	c.Header(contentType, "text/csv")
	c.Header(contentDisposition, "attachment;filename=sales_by_region.csv")

	// Initialize CSV writer
	writer := csv.NewWriter(c.Writer)
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
func exportExcel(c *gin.Context, data []dtos.RegionSales) {
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
	c.Header(contentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header(contentDisposition, "attachment;filename=sales_by_region.xlsx")
	// Write Excel file to response
	_ = f.Write(c.Writer)
}

// ---------------- PDF Export ----------------
// exportPDF generates a PDF document of regional sales data.
// Creates a formatted PDF report with title and data table.
func exportPDF(c *gin.Context, data []dtos.RegionSales) {
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

	c.Header(contentType, "application/pdf")
	c.Header(contentDisposition, "attachment;filename=sales_by_region.pdf")
	_ = pdf.Output(c.Writer)
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
// @Success      200   {object}  map[string]any        "Monthly sales and orders data"
// @Failure      400   {object}  map[string]any         "Invalid year parameter"
// @Failure      404   {object}  dtos.ErrorResponse           "Failed to generate report"
// @Router       /api/reports/sales/monthly-comparison [get]
func GetSalesVsOrdersPerMonth(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract year parameter from query string
	yearStr := c.Query("year")
	// Convert year string to integer
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		// Year parameter is missing or not a valid integer
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid year parameter: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid Year Parameter",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	// Fetch monthly sales and order data for the specified year
	report, err := models.GetSalesVsOrdersPerMonth(year)
	if err != nil {
		// Database query failed or no data for specified year
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales vs orders per month report :" + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	// Return monthly breakdown for trend analysis
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales vs orders per month report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Sales vs orders per month report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetRevenueVsExpenses compares revenue against expenses for profitability analysis.
// Supports multiple time filters (week, month, quarter, year) for different analysis periods.
// Critical for understanding profit margins and cost management effectiveness.
//
// @Summary      Get revenue vs expenses comparison
