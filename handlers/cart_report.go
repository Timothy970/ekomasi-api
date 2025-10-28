package handlers

import (
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"net/http"
	"time"
)

func CartAbandonmentReport(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	startTime, endTime, err := ParseDateRange(r)
	// Call model
	report, err := models.GetCartAbandonmentRate(startTime, endTime)
	if err != nil {
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
func CartAbandonmentTrendReport(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	period := "daily"
	periodStr := r.URL.Query().Get("period")
	if periodStr != "" {
		period = periodStr
	}
	startTime, endTime, err := ParseDateRange(r)

	// Call model
	report, err := models.GetCartAbandonmentTrend(startTime, endTime, period)
	if err != nil {
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

// ParseDateRange extracts and validates start & end dates from a request.
// If no dates are provided, defaults to the last 30 days (end = today).
func ParseDateRange(r *http.Request) (time.Time, time.Time, error) {
	// default: today as end, last 30 days as start
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		parsed, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start date: %v", err)
		}
		startTime = parsed
	}

	if endStr := r.URL.Query().Get("end"); endStr != "" {
		parsed, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end date: %v", err)
		}
		endTime = parsed
	}

	return startTime, endTime, nil
}
