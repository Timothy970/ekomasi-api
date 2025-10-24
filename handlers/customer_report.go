package handlers

import (
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"strconv"
	"time"
)

func GetCustomerRetention(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	start, end, err := ParseDateRange(r)
	duration := 0
	months := r.URL.Query().Get("duration")
	if months != "" {
		duration, _ = strconv.Atoi(months)
	}
	ret, err := models.GetCustomerRetention(start, end, duration)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get customer retention report",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	total := ret.NewCustomers + ret.ReturningCustomers
	rate := 0.0
	if total > 0 {
		rate = float64(ret.ReturningCustomers) / float64(total) * 100
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Customer retention report generated successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"new_customers":              ret.NewCustomers,
			"returning_customers":        ret.NewCustomers,
			"retention_rate":             rate,
			"returning_customers_orders": ret.ReturningOrders,
			"new_customers_orders":       ret.NewOrders,
			"returning_orders_details":   ret.ReturningDetails,
			"new_orders_details":         ret.NewDetails,
		},
		Message:   "Customer retention report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// groupBy = "month" | "quarter" | "year"
func GetCustomerRetentionTrends(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	start, end, err := ParseDateRange(r)
	period := "month"
	periodStr := r.URL.Query().Get("period")
	if periodStr != "" {
		if periodStr != "month" && periodStr != "quarter" && periodStr != "year" {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Reports",
					Description: "Invalid period specified for customer retention trends",
					Code:        http.StatusBadRequest,
				},
				Message:   "Period should be either month, quarter or year",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
		period = periodStr
	}
	ret, err := models.GetCustomerRetentionTrends(start, end, period)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get customer retention trends",
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
			Description: "Customer retention trend report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   ret,
		Message:   "Customer retention trend report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func GetCustomerRetentionSummary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	start, end, err := ParseDateRange(r)
	ret, err := models.GetCustomerRetentionSummary(start, end)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get customer retention summary",
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
			Description: "Customer retention summary report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   ret,
		Message:   "Customer retention summary report generated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
