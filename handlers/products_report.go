// Package handlers provides HTTP request handlers for product performance reporting.
// This file contains handlers for generating analytics and performance reports about products,
// including category-level and individual product metrics, sales trends, and performance data.
package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetProductPerformanceSummary generates a performance report for products within a specific category.
// This endpoint provides analytics including sales data, revenue, and performance metrics
// for all products in the specified category over a given date range.
//
// @Summary Get product performance summary by category
// @Description Generates a performance report for all products in a category within a date range
// @Tags Reports
// @Produce json
// @Param category_id query string true "Category ID for filtering products"
// @Param start_date query string false "Start date (YYYY-MM-DD) for report period"
// @Param end_date query string false "End date (YYYY-MM-DD) for report period"
// @Success 200 {object} map[string]any "Product performance report generated successfully"
// @Failure 400 {object} map[string]any "Category ID missing or invalid date range"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/reports/products/performance [get]
// @Security BearerAuth
func GetProductPerformanceSummary(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse date range from query parameters (start_date and end_date)
	startTime, endTime, err := ParseDateRange(c.Request)

	// Extract category ID from query parameters (required)
	categoryID := c.Query("category_id")
	if categoryID == "" {
		// Return error if category ID is not provided
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Category ID is required for product performance report",
				Code:        http.StatusBadRequest,
			},
			Message:   "Category ID is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Generate product performance report from database
	report, err := models.GetProductPerformance(models.DB, startTime, endTime, categoryID)
	if err != nil {
		// Return error if report generation fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate product performance report",
				Code:        http.StatusInternalServerError,
			},
			Message:   "failed to generate report: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Return success response with product performance data
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Product performance report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   report,
		Message:   "product performance report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetIndividualProductPerformanceSummary generates a detailed performance report for a single product.
// This endpoint provides comprehensive analytics including sales trends, revenue metrics,
// and performance data for a specific product over a given date range.
//
// @Summary Get individual product performance summary
// @Description Generates a detailed performance report for a single product within a date range
// @Tags Reports
// @Produce json
// @Param product_id path string true "Product ID"
// @Param start_date query string false "Start date (YYYY-MM-DD) for report period"
// @Param end_date query string false "End date (YYYY-MM-DD) for report period"
// @Success 200 {object} map[string]any "Individual product performance report generated successfully"
// @Failure 400 {object} map[string]any "Invalid date range"
// @Failure 404 {object} map[string]any "Product not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/reports/products/{product_id}/performance [get]
// @Security BearerAuth
func GetIndividualProductPerformanceSummary(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse date range from query parameters
	startTime, endTime, err := ParseDateRange(c.Request)

	// Extract product ID from URL path parameters
	productID := c.Param("product_id")

	// Generate individual product performance report from database
	performance, err := models.GetSingleProductPerformance(models.DB, productID, startTime, endTime)
	if err != nil {
		// Return error if report generation fails or product not found
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate individual product performance report",
				Code:        http.StatusInternalServerError,
			},
			Message:   "failed to generate report: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Return success response with individual product performance data
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Individual product performance report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   performance,
		Message:   "individual product performance report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
