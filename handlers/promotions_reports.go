// Package handlers provides HTTP request handlers for promotion reporting and analytics.
// This file contains handlers for promotion effectiveness analysis, comparison reports,
// and summary statistics to help evaluate marketing campaign performance.
package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetEffectiveness retrieves effectiveness metrics for a specific promotion.
// It analyzes promotion performance including conversion rates, revenue impact,
// customer engagement, and ROI to evaluate campaign success.
//
// @Summary      Get promotion effectiveness report
// @Description  Retrieve detailed effectiveness metrics and KPIs for a specific promotion campaign
// @Tags         Reports
// @Produce      json
// @Param        promotion_id  path      string  true  "Promotion ID"
// @Success      200           {object}  map[string]any  "Promotion effectiveness metrics"
// @Failure      404           {object}  dtos.ErrorResponse      "Promotion not found"
// @Security     BearerAuth
// @Router       /reports/promotions/effectiveness/{promotion_id} [get]
func GetEffectiveness(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract promotion ID from URL path parameters
	promotionID := c.Param("promotion_id")

	// Fetch promotion effectiveness metrics from database
	effectiveness, err := models.GetEffectiveness(models.DB, promotionID)
	if err != nil {
		// Database query failed or promotion not found, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get promotion effectiveness report",
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
	// Return successful response with effectiveness metrics
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Promotion effectiveness report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   effectiveness,
		Message:   "Promotion effectiveness report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetComparison generates a comparison report for a promotion over a specific date range.
// It compares promotion performance metrics against baseline periods to measure
// incremental impact, sales lift, and campaign effectiveness.
//
// @Summary      Get promotion comparison report
// @Description  Compare promotion performance against baseline metrics for a specific date range
// @Tags         Reports
// @Produce      json
// @Param        promotion_id  path      string  true  "Promotion ID"
// @Param        start         query     string  true  "Start date (YYYY-MM-DD)"
// @Param        end           query     string  true  "End date (YYYY-MM-DD)"
// @Success      200           {object}  map[string]any  "Promotion comparison data"
// @Failure      400           {object}  dtos.ErrorResponse      "Invalid date range"
// @Failure      404           {object}  dtos.ErrorResponse      "Promotion not found"
// @Security     BearerAuth
// @Router       /reports/promotions/comparison/{promotion_id} [get]
func GetComparison(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract promotion ID from URL path parameters
	promotionID := c.Param("promotion_id")
	// Parse and validate start and end date from query parameters
	startTime, endTime, err := ParseDateRange(c.Request)
	// Fetch promotion comparison data for the specified date range
	comparison, err := models.GetComparison(models.DB, promotionID, startTime, endTime)
	if err != nil {
		// Date parsing failed or database query error, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get promotion with promotion ID " + promotionID + " comparison report",
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
	// Return successful response with comparison metrics
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Promotion for promotion ID " + promotionID + " comparison report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   comparison,
		Message:   "Promotion comparison report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetSummary generates an aggregate summary report for all promotions within a date range.
// It provides consolidated metrics including total revenue impact, customer participation,
// average discount rates, and overall campaign performance across all promotions.
//
// @Summary      Get promotion summary report
// @Description  Retrieve aggregate summary statistics for all promotions within a specific date range
// @Tags         Reports
// @Produce      json
// @Param        start  query     string  true  "Start date (YYYY-MM-DD)"
// @Param        end    query     string  true  "End date (YYYY-MM-DD)"
// @Success      200    {object}  map[string]any  "Promotion summary statistics"
// @Failure      400    {object}  dtos.ErrorResponse      "Invalid date range"
// @Failure      404    {object}  dtos.ErrorResponse      "No data found"
// @Security     BearerAuth
// @Router       /reports/promotions/summary [get]
func GetSummary(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Parse and validate start and end date from query parameters
	startTime, endTime, err := ParseDateRange(c.Request)
	// Fetch aggregate promotion summary data for the date range
	summary, err := models.GetPromotionSummary(models.DB, startTime, endTime)
	if err != nil {
		// Date parsing failed or database query error, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get promotion summary report",
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
	// Return successful response with summary statistics
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Promotion summary report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   summary,
		Message:   "Promotion summary report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
