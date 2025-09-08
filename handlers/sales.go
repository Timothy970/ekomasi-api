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

func GetSalesTrendsSummary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	startTime, endTime, err := ParseDateRange(r)
	// Calculate previous period range
	diff := endTime.Sub(startTime)
	prevEnd := startTime.Add(-24 * time.Hour)
	prevStart := prevEnd.Add(-diff)

	report, err := models.GetSalesTrendsSummary(startTime, endTime, prevStart, prevEnd)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "failed to generate report: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   report,
		Message:   "Sales trend report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func GetSalesTrendsOverTime(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	startTime, endTime, err := ParseDateRange(r)
	period := "daily"
	periodStr := r.URL.Query().Get("period")
	if periodStr != "" {
		period = periodStr
	}
	report, err := models.GetSalesTrendsOverTime(period, startTime, endTime)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "failed to generate report: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   report,
		Message:   "Sales trend report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func GetCustomerSegmentation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	start, end, _ := ParseDateRange(r)

	data, err := models.GetCustomerSegmentation(start, end)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.CustomerSegmentationResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
		}{Start: start, End: end},
		Segments: data,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   resp,
		Message:   "Customer segmentation report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func GetSalesByRegion(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	export := r.URL.Query().Get("export")
	start, end, _ := ParseDateRange(r)

	data, err := models.GetSalesByRegion(start, end)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.SalesByRegionResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
		}{Start: start, End: end},
		Regions: data,
	}

	// Export support
	switch export {
	case "csv":
		exportCSV(w, data)
		return
	case "xlsx":
		exportExcel(w, data)
		return
	case "pdf":
		exportPDF(w, data)
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   resp,
		Message:   "Sales segmentation report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ---------------- CSV Export ----------------
func exportCSV(w http.ResponseWriter, data []dtos.RegionSales) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=sales_by_region.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"Region", "Total Sales", "Avg Order Value", "Transactions"})
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
func exportExcel(w http.ResponseWriter, data []dtos.RegionSales) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	headers := []string{"Region", "Total Sales", "Avg Order Value", "Transactions"}
	for i, h := range headers {
		col := string(rune('A' + i))
		f.SetCellValue(sheet, col+"1", h)
	}

	for i, r := range data {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, r.Region)
		f.SetCellValue(sheet, "B"+row, r.TotalSales)
		f.SetCellValue(sheet, "C"+row, r.AvgOrderValue)
		f.SetCellValue(sheet, "D"+row, r.Transactions)
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment;filename=sales_by_region.xlsx")
	_ = f.Write(w)
}

// ---------------- PDF Export ----------------
func exportPDF(w http.ResponseWriter, data []dtos.RegionSales) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "Sales by Region Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 10)
	headers := []string{"Region", "Total Sales", "Avg Order Value", "Transactions"}
	colWidths := []float64{50, 40, 40, 40}

	// table headers
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// table rows
	pdf.SetFont("Arial", "", 10)
	for _, r := range data {
		pdf.CellFormat(colWidths[0], 8, r.Region, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[1], 8, strconv.FormatFloat(r.TotalSales, 'f', 2, 64), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[2], 8, strconv.FormatFloat(r.AvgOrderValue, 'f', 2, 64), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[3], 8, strconv.Itoa(r.Transactions), "1", 0, "C", false, 0, "")
		pdf.Ln(-1)
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment;filename=sales_by_region.pdf")
	_ = pdf.Output(w)
}
