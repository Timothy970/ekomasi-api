// Package handlers provides HTTP request handlers for the Adenzo backend API.
// This file contains cart analytics and reporting handlers that track cart abandonment
// metrics and trends to help understand customer shopping behavior.
package handlers

import (
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"net/http"
	"time"
)

// CartAbandonmentReport generates a comprehensive cart abandonment report for a specified date range.
// This endpoint calculates the percentage of shopping carts that were created but not converted
// to orders, providing valuable insights into potential checkout issues and customer drop-off.
//
// @Summary Generate cart abandonment rate report
// @Description Calculates the cart abandonment rate for a specified date range. Returns total carts, converted carts, abandoned carts, and abandonment percentage.
// @Tags Reports
// @Produce json
// @Param start query string false "Start date for report (format: YYYY-MM-DD, default: 30 days ago)"
// @Param end query string false "End date for report (format: YYYY-MM-DD, default: today)"
// @Success 200 {object} map[string]interface{} "Cart abandonment report generated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid date format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/reports/cart-abandonment [get]
// @Security BearerAuth
func CartAbandonmentReport(w http.ResponseWriter, r *http.Request) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(r)

	// Parse and validate date range from query parameters (defaults to last 30 days)
	startTime, endTime, err := ParseDateRange(r)

	// Retrieve cart abandonment statistics from the database
	report, err := models.GetCartAbandonmentRate(startTime, endTime)
	if err != nil {
		// Return error response if report generation fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate cart abandonment report",
				Code:        http.StatusInternalServerError,
			},
			Message:   "failed to generate report: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Return success response with cart abandonment metrics
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Cart abandonment report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Cart abandonment report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// CartAbandonmentTrendReport generates a time-series analysis of cart abandonment rates.
// This endpoint provides trend data over time, allowing analysis of abandonment patterns
// by day, week, or month to identify seasonal trends or issues.
//
// @Summary Generate cart abandonment trend report
// @Description Generates a time-series report showing cart abandonment rates over a specified period with configurable granularity (daily, weekly, monthly).
// @Tags Reports
// @Produce json
// @Param start query string false "Start date for report (format: YYYY-MM-DD, default: 30 days ago)"
// @Param end query string false "End date for report (format: YYYY-MM-DD, default: today)"
// @Param period query string false "Time period granularity: daily, weekly, or monthly (default: daily)"
// @Success 200 {object} map[string]interface{} "Cart abandonment trend report generated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid date format or period"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/reports/cart-abandonment/trend [get]
// @Security BearerAuth
func CartAbandonmentTrendReport(w http.ResponseWriter, r *http.Request) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(r)

	// Set default period to daily, can be overridden by query parameter
	period := "daily"
	periodStr := r.URL.Query().Get("period")
	if periodStr != "" {
		// Use custom period if provided (daily, weekly, or monthly)
		period = periodStr
	}

	// Parse and validate date range from query parameters
	startTime, endTime, err := ParseDateRange(r)

	// Retrieve cart abandonment trend data grouped by specified period
	report, err := models.GetCartAbandonmentTrend(startTime, endTime, period)
	if err != nil {
		// Return error response if trend report generation fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate cart abandonment trend report",
				Code:        http.StatusInternalServerError,
			},
			Message:   "failed to generate report: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Return success response with time-series abandonment trend data
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Cart abandonment trend report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "Cart abandonment trend report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// ParseDateRange extracts and validates start and end dates from HTTP request query parameters.
// This utility function is used by report handlers to standardize date range parsing.
//
// Date Format: YYYY-MM-DD (e.g., 2026-01-12)
// Query Parameters:
//   - start: Starting date for the report range (optional)
//   - end: Ending date for the report range (optional)
//
// Default Behavior:
// If no dates are provided, returns the last 30 days with end date as today.
//
// Returns:
//   - startTime: The beginning of the date range
//   - endTime: The end of the date range
//   - error: Validation error if date format is invalid
func ParseDateRange(r *http.Request) (time.Time, time.Time, error) {
	// Set default date range: last 30 days ending today
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30) // Subtract 30 days from end time

	// Parse custom start date if provided in query parameters
	if startStr := r.URL.Query().Get("start"); startStr != "" {
		// Attempt to parse start date in YYYY-MM-DD format
		parsed, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			// Return error if date format is invalid
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start date: %v", err)
		}
		startTime = parsed
	}

	// Parse custom end date if provided in query parameters
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		// Attempt to parse end date in YYYY-MM-DD format
		parsed, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			// Return error if date format is invalid
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end date: %v", err)
		}
		endTime = parsed
	}

	// Return validated date range
	return startTime, endTime, nil
}
