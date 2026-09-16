// Package handlers provides HTTP request handlers for sales analytics and reporting.
// This file contains handlers for sales trends analysis, customer segmentation, regional sales,
// revenue tracking, and various sales performance metrics. Supports multiple export formats
// including JSON, CSV, Excel, and PDF for comprehensive business intelligence reporting.
package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// @Description  Retrieve revenue and expenses comparison with configurable time filter
// @Tags         Sales
// @Produce      json
// @Param        filter  query     string                       false  "Time filter (week, month, quarter, year) default: week"
// @Success      200     {object}  map[string]interface{}       "Revenue and expenses data"
// @Failure      404     {object}  dtos.ErrorResponse           "Failed to generate report"
// @Router       /api/reports/sales/revenue-vs-expenses [get]
func GetRevenueVsExpenses(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract filter parameter from query string
	filterType := c.Query("filter")
	// Set default filter to week if not specified
	filter := "week"
	if filterType != "" {
		filter = filterType
	}

	// Fetch revenue and expense comparison data with specified time filter
	report, err := models.GetRevenueVsExpenses(filter)
	if err != nil {
		// Database query failed or insufficient financial data
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get revenue vs expenses report :" + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message: err.Error(),

			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	// Return profitability analysis data
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Revenue vs expenses report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Revenue vs expenses report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func GetRevenueCustomersOrdersOverview(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Fetch comprehensive business metrics from database
	report, err := models.GetRevenueCustomersOrdersOverview()
	if err != nil {
		// Database query failed or insufficient data for comprehensive overview
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get revenue, customers and orders overview :" + err.Error(),
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
	// Return combined business metrics for dashboard display
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Revenue, customers and orders overview generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Revenue, customers and orders overview generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
