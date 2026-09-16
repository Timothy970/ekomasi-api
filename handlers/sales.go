// Package handlers provides HTTP request handlers for sales analytics and reporting.
// This file contains handlers for sales trends analysis, customer segmentation, regional sales,
// revenue tracking, and various sales performance metrics. Supports multiple export formats
// including JSON, CSV, Excel, and PDF for comprehensive business intelligence reporting.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
// @Success      200         {object}  map[string]any    "Sales trends summary"
// @Failure      404         {object}  dtos.ErrorResponse         "Failed to generate report"
// @Router       /api/reports/sales/trends/summary [get]
func GetSalesTrendsSummary(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Parse date range from query parameters (start_date, end_date)
	startTime, endTime, err := ParseDateRange(c.Request)
	// Calculate previous period range for comparison metrics
	// Previous period has the same duration as current period
	diff := endTime.Sub(startTime)
	prevEnd := startTime.Add(-24 * time.Hour)
	prevStart := prevEnd.Add(-diff)

	// Fetch sales trends summary from database with period comparison
	report, err := models.GetSalesTrendsSummary(startTime, endTime, prevStart, prevEnd)
	if err != nil {
		// Database query failed or no data available for the period
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales trends summary report :" + err.Error(),
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

	// Return summary report with period-over-period metrics

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales trend summary report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Sales trend summary report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200         {object}  map[string]any    "Sales trends over time"
// @Failure      500         {object}  dtos.ErrorResponse         "Failed to generate report"
// @Router       /api/reports/sales/trends/overtime [get]
func GetSalesTrendsOverTime(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Parse date range from query parameters
	startTime, endTime, err := ParseDateRange(c.Request)
	// Set default period granularity to daily
	period := "daily"
	// Override with query parameter if provided (daily, weekly, monthly)
	periodStr := c.Query("period")
	if periodStr != "" {
		period = periodStr
	}
	// Fetch time-series sales data with specified granularity
	report, err := models.GetSalesTrendsOverTime(period, startTime, endTime)
	if err != nil {
		// Database query failed or invalid period specified
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales trends over time report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Return time-series data for chart visualization

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales trend report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Sales trend report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200         {object}  map[string]any             "Customer segmentation data"
// @Failure      404         {object}  dtos.ErrorResponse                  "Failed to generate report"
// @Router       /api/reports/sales/customer-segmentation [get]
func GetCustomerSegmentation(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Parse date range from query parameters
	startDate, endDate, _ := ParseDateRange(c.Request)

	// Fetch customer segmentation data from database
	data, err := models.GetCustomerSegmentation(startDate, endDate)
	if err != nil {
		// Database query failed or no customer data available
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get customer segmentation report",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Build response with period metadata and segment data
	resp := dtos.CustomerSegmentationResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
		}{Start: startDate, End: endDate},
		Segments: data,
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Customer segmentation report",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Customer segmentation report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200  {object}  map[string]any     "Sales overview data"
// @Failure      404  {object}  dtos.ErrorResponse     "Failed to retrieve overview"
// @Router       /api/reports/sales/overview [get]
func GetSalesOverview(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Fetch overall sales summary metrics from database
	data, err := models.GetSalesOverview()
	if err != nil {
		// Database query failed or insufficient data
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get sales overview",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Return sales overview with key performance indicators
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Sales overview retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   data,
		Message:   "Sales overview retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetSalesByRegion analyzes sales performance broken down by geographic region.
// Supports multiple export formats (JSON, CSV, Excel, PDF) for reporting flexibility.
// Essential for regional sales managers and geographic performance analysis.
//
// @Summary      Get sales by region
