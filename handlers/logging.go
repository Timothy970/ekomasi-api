// Package handlers provides HTTP request handlers for the Ekomasi backend API.
// This file contains user activity logging and audit trail handlers that track
// system usage, user actions, and provide comprehensive audit capabilities.
package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var dateFormat = "2006-01-02"

// GetUserLogs retrieves a paginated and filtered list of user activity logs.
// This endpoint provides comprehensive audit trail functionality with support for
// multiple filters including module, status, role, date range, and search queries.
// Admin access is required to view system logs.
//
// @Summary Get all user logs with filtering
// @Description Retrieves paginated user activity logs with optional filters for module, status, role, date range, and search. Admin access required.
// @Tags Admin, Logging
// @Produce json
// @Param page query int false "Page number for pagination (default: 1)"
// @Param size query int false "Number of items per page (default: 10)"
// @Param module query string false "Filter by module name (e.g., Products, Orders, Users)"
// @Param status query string false "Filter by log status (e.g., success, error)"
// @Param role query string false "Filter by user role"
// @Param start_date query string false "Start date for date range filter (format: YYYY-MM-DD)"
// @Param end_date query string false "End date for date range filter (format: YYYY-MM-DD)"
// @Param q query string false "Search query across log fields"
// @Success 200 {object} map[string]any "User logs retrieved successfully with pagination"
// @Failure 400 {object} map[string]any "Invalid date range or parameters"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/logs [get]
// @Security BearerAuth
func GetUserLogs(c *gin.Context) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", ""); !ok {
		return
	}

	// Parse pagination parameters from query string
	page, limit := parsePagination(c.Query("page"), c.Query("size"))

	// Build filter criteria with sanitized inputs to prevent injection attacks
	filters := models.UserLogFilters{
		Module:    sanitizeString(c.Query("module")),   // Module name filter
		Status:    sanitizeString(c.Query("status")),   // Status filter (success/error)
		Role:      sanitizeString(c.Query("role")),     // User role filter
		StartDate: validateDate(c.Query("start_date")), // Validated start date
		EndDate:   validateDate(c.Query("end_date")),   // Validated end date
		Search:    sanitizeString(c.Query("q")),        // General search query
		Page:      page,                                // Current page number
		Limit:     limit,                               // Items per page
	}

	// Validate that end date is after or equal to start date if both are provided
	if filters.StartDate != "" && filters.EndDate != "" {
		if !isValidDateRange(filters.StartDate, filters.EndDate) {
			// Return error if date range is invalid
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Invalid date range",
					Code:        http.StatusBadRequest,
				},
				Message:   "End date must be after start date",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return
		}
	}

	// Retrieve filtered and paginated logs from the database using optimized query
	logs, meta, err := models.GetUserLogsOptimized(models.DB, filters)
	if err != nil {
		// Return error response if log retrieval fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch user logs",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Return success response with logs and pagination metadata
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "All user logs fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"logs": logs, "pagination": meta},
		Message:   "User logs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// sanitizeString removes leading and trailing whitespace from a string input.
// This utility function helps prevent injection attacks and ensures clean data
// by normalizing user input before processing.
//
// Parameters:
//   - s: The string to sanitize
//
// Returns:
//   - Trimmed string with leading/trailing whitespace removed
func sanitizeString(s string) string {
	return strings.TrimSpace(s)
}

// validateDate validates and sanitizes a date string in YYYY-MM-DD format.
// This function ensures that date inputs are properly formatted before being
// used in database queries, preventing invalid date errors and SQL injection.
//
// Parameters:
//   - dateStr: Date string to validate (expected format: YYYY-MM-DD)
//
// Returns:
//   - Valid date string in YYYY-MM-DD format, or empty string if invalid
func validateDate(dateStr string) string {
	// Remove whitespace from date string
	dateStr = sanitizeString(dateStr)
	if dateStr == "" {
		// Return empty string if no date provided
		return ""
	}

	// Attempt to parse date using standard format to ensure validity
	_, err := time.Parse(dateFormat, dateStr)
	if err != nil {
		// Return empty string if date format is invalid
		return ""
	}

	// Return validated date string
	return dateStr
}

// isValidDateRange validates that a date range is logical and properly ordered.
// This function ensures that the end date is after or equal to the start date,
// preventing invalid date range queries.
//
// Parameters:
//   - start: Start date string in YYYY-MM-DD format
//   - end: End date string in YYYY-MM-DD format
//
// Returns:
//   - true if end date is after or equal to start date
//   - false if dates are invalid or end date is before start date
func isValidDateRange(start, end string) bool {
	// Parse start date
	startTime, err1 := time.Parse(dateFormat, start)
	// Parse end date
	endTime, err2 := time.Parse(dateFormat, end)

	// Return false if either date parsing failed
	if err1 != nil || err2 != nil {
		return false
	}

	// Validate that end date is after or equal to start date
	return endTime.After(startTime) || endTime.Equal(startTime)
}

// GetUserLogsByUserID retrieves activity logs for a specific user.
// This endpoint provides a filtered view of logs for a single user, useful for
// investigating specific user actions or troubleshooting user-reported issues.
// Admin access is required.
//
// @Summary Get user logs by user ID
// @Description Retrieves paginated activity logs for a specific user identified by their user ID. Admin access required.
// @Tags Admin, Logging
// @Produce json
// @Param user_id path string true "User ID to retrieve logs for"
// @Param page query int false "Page number for pagination (default: 1)"
// @Param size query int false "Number of items per page (default: 10)"
// @Success 200 {object} map[string]any "User logs retrieved successfully with pagination"
// @Failure 400 {object} map[string]any "Invalid user ID or parameters"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 404 {object} map[string]any "User not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/logs/user/{user_id} [get]
// @Security BearerAuth
func GetUserLogsByUserID(c *gin.Context) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", ""); !ok {
		return
	}

	// Parse pagination parameters from query string
	page, limit := parsePagination(c.Query("page"), c.Query("size"))

	// Extract user ID from URL path parameters
	userID := c.Param("user_id")

	// Retrieve logs for the specified user from the database
	logs, meta, err := models.GetUserLogsByUserID(models.DB, userID, limit, page)
	if err != nil {
		// Return error response if log retrieval fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch user logs for user ID " + userID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Return success response with user-specific logs and pagination metadata
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "User logs for user ID " + userID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"logs": logs, "pagination": meta},
		Message:   "User logs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
