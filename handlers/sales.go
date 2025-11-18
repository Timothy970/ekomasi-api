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

var (
	contentType        = "Content-Type"
	totalSales         = "Total Sales"
	contentDisposition = "Content-Disposition"
	avgOrderValue      = "Avg Order Value"
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

func GetCustomerSegmentation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	start, end, _ := ParseDateRange(r)

	data, err := models.GetCustomerSegmentation(start, end)
	if err != nil {
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
func GetSalesOverview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	data, err := models.GetSalesOverview()
	if err != nil {
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
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales overview retrieved successfully",
			Code:        http.StatusCreated,
		},
		Payload:   data,
		Message:   "Sales overview retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Sales by region handler
func GetSalesByRegion(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	export := r.URL.Query().Get("export")
	start, end, _ := ParseDateRange(r)

	data, err := models.GetSalesByRegion(start, end)
	if err != nil {
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
func exportCSV(w http.ResponseWriter, data []dtos.RegionSales) {
	w.Header().Set(contentType, "text/csv")
	w.Header().Set(contentDisposition, "attachment;filename=sales_by_region.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"Region", totalSales, avgOrderValue, "Transactions"})
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

	headers := []string{"Region", totalSales, avgOrderValue, "Transactions"}
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

	w.Header().Set(contentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set(contentDisposition, "attachment;filename=sales_by_region.xlsx")
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
	headers := []string{"Region", totalSales, avgOrderValue, "Transactions"}
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

	w.Header().Set(contentType, "application/pdf")
	w.Header().Set(contentDisposition, "attachment;filename=sales_by_region.pdf")
	_ = pdf.Output(w)
}

func GetSalesVsOrdersPerMonth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
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
	report, err := models.GetSalesVsOrdersPerMonth(year)
	if err != nil {
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

func GetRevenueVsExpenses(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	filterType := r.URL.Query().Get("filter")
	filter := "week"
	if filterType != "" {
		filter = filterType
	}

	report, err := models.GetRevenueVsExpenses(filter)
	if err != nil {
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

func GetRevenueCustomersOrdersOverview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	report, err := models.GetRevenueCustomersOrdersOverview()
	if err != nil {
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
