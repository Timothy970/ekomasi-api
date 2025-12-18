package handlers

import (
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

//Return user logs info,

func GetUserLogs(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Get filters with validation
	filters := models.UserLogFilters{
		Module:    sanitizeString(r.URL.Query().Get("module")),
		Status:    sanitizeString(r.URL.Query().Get("status")),
		Role:      sanitizeString(r.URL.Query().Get("role")),
		StartDate: validateDate(r.URL.Query().Get("start_date")),
		EndDate:   validateDate(r.URL.Query().Get("end_date")),
		Search:    sanitizeString(r.URL.Query().Get("q")),
		Page:      page,
		Limit:     limit,
	}

	// Validate date range
	if filters.StartDate != "" && filters.EndDate != "" {
		if !isValidDateRange(filters.StartDate, filters.EndDate) {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Invalid date range",
					Code:        http.StatusBadRequest,
				},
				Message:   "End date must be after start date",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return
		}
	}
	logs, meta, err := models.GetUserLogsOptimized(filters)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch user logs",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "All user logs fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"logs": logs, "pagination": meta},
		Message:   "User logs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
func sanitizeString(s string) string {
	return strings.TrimSpace(s)
}

func validateDate(dateStr string) string {
	dateStr = sanitizeString(dateStr)
	if dateStr == "" {
		return ""
	}

	// Try parsing to ensure it's valid
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}

	return dateStr
}

func isValidDateRange(start, end string) bool {
	startTime, err1 := time.Parse("2006-01-02", start)
	endTime, err2 := time.Parse("2006-01-02", end)

	if err1 != nil || err2 != nil {
		return false
	}

	return endTime.After(startTime) || endTime.Equal(startTime)
}

// Get user logs by user ID
func GetUserLogsByUserID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	userID := mux.Vars(r)["user_id"]
	logs, meta, err := models.GetUserLogsByUserID(userID, limit, page)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch user logs for user ID " + userID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "User logs for user ID " + userID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"logs": logs, "pagination": meta},
		Message:   "User logs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
