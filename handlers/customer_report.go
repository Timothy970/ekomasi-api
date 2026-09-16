// Package handlers provides HTTP request handlers for the Ekomasi backend API.
// This file contains customer analytics and reporting handlers that track customer retention,
// behavior patterns, and lifecycle metrics to support customer relationship management.
package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetCustomerRetention generates a comprehensive customer retention report for a specified date range.
// This endpoint analyzes customer purchase behavior to distinguish between new and returning customers,
// providing insights into customer loyalty and business health.
//
// @Summary Generate customer retention report
// @Description Calculates customer retention metrics including new customers, returning customers, retention rate, and order details for each segment
// @Tags Reports
// @Produce json
// @Param start query string false "Start date for report (format: YYYY-MM-DD, default: 30 days ago)"
// @Param end query string false "End date for report (format: YYYY-MM-DD, default: today)"
// @Param duration query int false "Duration in months to consider a customer as returning (default: 0)"
// @Success 200 {object} map[string]interface{} "Customer retention report generated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid date format or parameters"
// @Failure 404 {object} map[string]interface{} "No data found for specified range"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/reports/customer-retention [get]
// @Security BearerAuth
func GetCustomerRetention(c *gin.Context) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse and validate date range from query parameters
	start, end, err := ParseDateRange(c.Request)

	// Parse optional duration parameter (in months) for retention calculation
	duration := 0
	months := c.Query("duration")
	if months != "" {
		// Convert duration string to integer (ignoring conversion errors, defaults to 0)
		duration, _ = strconv.Atoi(months)
	}

	// Retrieve customer retention data from the database
	ret, err := models.GetCustomerRetention(start, end, duration)
	if err != nil {
		// Return error response if retention data retrieval fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get customer retention report",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Calculate total customer base for retention rate computation
	total := ret.NewCustomers + ret.ReturningCustomers

	// Calculate retention rate as percentage of returning customers
	rate := 0.0
	if total > 0 {
		// Avoid division by zero, calculate percentage of returning customers
		rate = float64(ret.ReturningCustomers) / float64(total) * 100
	}

	// Return success response with comprehensive retention metrics
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Customer retention report generated successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"new_customers":              ret.NewCustomers,     // Count of first-time customers
			"returning_customers":        ret.NewCustomers,     // Count of repeat customers
			"retention_rate":             rate,                 // Percentage of returning customers
			"returning_customers_orders": ret.ReturningOrders,  // Total orders from returning customers
			"new_customers_orders":       ret.NewOrders,        // Total orders from new customers
			"returning_orders_details":   ret.ReturningDetails, // Detailed order breakdown for returning customers
			"new_orders_details":         ret.NewDetails,       // Detailed order breakdown for new customers
		},
		Message:   "Customer retention report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetCustomerRetentionTrends generates a time-series analysis of customer retention metrics.
// This endpoint provides trend data showing how retention rates change over time with
// configurable grouping by month, quarter, or year to identify patterns and seasonality.
//
// @Summary Generate customer retention trends report
// @Description Generates a time-series report showing customer retention trends over a specified period with configurable granularity (month, quarter, year)
// @Tags Reports
// @Produce json
// @Param start query string false "Start date for report (format: YYYY-MM-DD, default: 30 days ago)"
// @Param end query string false "End date for report (format: YYYY-MM-DD, default: today)"
// @Param period query string false "Time period grouping: month, quarter, or year (default: month)"
// @Success 200 {object} map[string]interface{} "Customer retention trend report generated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid date format or period parameter"
// @Failure 404 {object} map[string]interface{} "No data found for specified range"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/reports/customer-retention/trends [get]
// @Security BearerAuth
func GetCustomerRetentionTrends(c *gin.Context) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse and validate date range from query parameters
	start, end, err := ParseDateRange(c.Request)

	// Set default period to month, can be overridden by query parameter
	period := "month"
	periodStr := c.Query("period")
	if periodStr != "" {
		// Validate period parameter - must be month, quarter, or year
		if periodStr != "month" && periodStr != "quarter" && periodStr != "year" {
			// Return error if period parameter is invalid
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Reports",
					Description: "Invalid period specified for customer retention trends",
					Code:        http.StatusBadRequest,
				},
				Message:   "Period should be either month, quarter or year",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return
		}
		// Use validated custom period
		period = periodStr
	}

	// Retrieve customer retention trend data grouped by specified period
	ret, err := models.GetCustomerRetentionTrends(start, end, period)
	if err != nil {
		// Return error response if trend data retrieval fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get customer retention trends",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response with time-series retention trend data
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Customer retention trend report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   ret,
		Message:   "Customer retention trend report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetCustomerRetentionSummary generates a high-level summary of customer retention metrics.
// This endpoint provides aggregated retention statistics without detailed breakdowns,
// ideal for dashboards and quick overview displays.
//
// @Summary Generate customer retention summary
// @Description Generates an aggregated summary of customer retention metrics for a specified date range
// @Tags Reports
// @Produce json
// @Param start query string false "Start date for report (format: YYYY-MM-DD, default: 30 days ago)"
// @Param end query string false "End date for report (format: YYYY-MM-DD, default: today)"
// @Success 200 {object} map[string]interface{} "Customer retention summary generated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid date format"
// @Failure 404 {object} map[string]interface{} "No data found for specified range"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/reports/customer-retention/summary [get]
// @Security BearerAuth
func GetCustomerRetentionSummary(c *gin.Context) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse and validate date range from query parameters
	start, end, err := ParseDateRange(c.Request)

	// Retrieve aggregated customer retention summary from the database
	ret, err := models.GetCustomerRetentionSummary(start, end)
	if err != nil {
		// Return error response if summary data retrieval fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get customer retention summary",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response with aggregated retention summary metrics
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Customer retention summary report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   ret,
		Message:   "Customer retention summary report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
