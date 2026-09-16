// Package models provides data access functions for the Ekomasi e-commerce backend.
//
// sales.go handles sales analytics and reporting including:
//   - Sales trends and growth analysis
//   - Revenue vs expenses tracking
//   - Customer segmentation by region
//   - Sales overview dashboards
//   - Period-over-period comparisons
package models

import (
	"ekomasi_backend/dtos"
	"fmt"
	"time"
)

// nowTime defines the datetime format used throughout sales analytics
var nowTime = "2006-01-02 15:04:05"

// SalesSummaryOverTime represents sales metrics for a specific time period.
// Used for period-over-period comparison with growth analysis.
type SalesSummaryOverTime struct {
	Period struct {
		Start string  `json:"start"` // Period start timestamp
		End   string  `json:"end"`   // Period end timestamp
		Type  *string `json:"type"`  // Period type (day, week, month, etc.)
	} `json:"period"`
	Metrics struct {
		SalesVolume      int     `json:"sales_volume"`      // Total units sold
		Revenue          float64 `json:"revenue"`           // Total revenue
		GrowthPercentage float64 `json:"growth_percentage"` // Growth vs previous period
	} `json:"metrics"`
	Comparison struct {
		PreviousPeriod struct {
			SalesVolume int     `json:"sales_volume"` // Previous period units sold
			Revenue     float64 `json:"revenue"`      // Previous period revenue
		} `json:"previous_period"`
	} `json:"comparison"`
}

// SalesTrendsSummary represents sales comparison between two custom periods.
// Provides flexibility for comparing any two date ranges.
type SalesTrendsSummary struct {
	Period struct {
		StartPeriod struct {
			Start string `json:"start"` // First period start
			End   string `json:"end"`   // First period end
		} `json:"start_period"`
		EndPeriod struct {
			Start string `json:"start"` // Second period start
			End   string `json:"end"`   // Second period end
		} `json:"end_period"`
	} `json:"period"`
	Metrics struct {
		SalesVolume      int     `json:"sales_volume"`      // Current period units sold
		Revenue          float64 `json:"revenue"`           // Current period revenue
		GrowthPercentage float64 `json:"growth_percentage"` // Growth vs previous period
	} `json:"metrics"`
	Comparison struct {
		PreviousPeriod struct {
			SalesVolume int     `json:"sales_volume"` // Previous period units sold
			Revenue     float64 `json:"revenue"`      // Previous period revenue
		} `json:"previous_period"`
	} `json:"comparison"`
}

// GetSalesTrendsSummary compares sales between two custom date ranges.
//
// This function provides flexible period comparison for trend analysis,
// calculating sales volume, revenue, and growth percentage.
//
// Parameters:
//   - start: time.Time - Current period start date
//   - end: time.Time - Current period end date
//   - prevStart: time.Time - Comparison period start date
//   - prevEnd: time.Time - Comparison period end date
//
// Returns:
//   - SalesTrendsSummary: Comparison metrics containing:
//   - Period definitions (start/end for both periods)
//   - Current metrics (sales volume, revenue, growth %)
//   - Previous period metrics for comparison
//   - error: Database error or nil on success
func GetSalesTrendsSummary(start, end, prevStart, prevEnd time.Time) (SalesTrendsSummary, error) {

	// Query current period sales metrics
	var currentVolume int
	var currentRevenue float64
	err := DB.QueryRow(`
		SELECT COALESCE(SUM(oi.quantity), 0),
		       COALESCE(SUM(oi.quantity * oi.unit_price), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.created_at BETWEEN ? AND ?
	`, start, end).Scan(&currentVolume, &currentRevenue)
	if err != nil {
		return SalesTrendsSummary{}, err
	}

	// Query previous period sales metrics for comparison
	var prevVolume int
	var prevRevenue float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.quantity), 0),
		       COALESCE(SUM(oi.quantity * oi.unit_price), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.created_at BETWEEN ? AND ?
	`, prevStart, prevEnd).Scan(&prevVolume, &prevRevenue)
	if err != nil {
		return SalesTrendsSummary{}, err
	}

	// Calculate growth percentage (avoid division by zero)
	growth := 0.0
	if prevRevenue > 0 {
		growth = ((currentRevenue - prevRevenue) / prevRevenue) * 100
	}

	// Build response structure
	response := SalesTrendsSummary{}
	response.Period.StartPeriod.Start = fmt.Sprintf("%s", start.Format(nowTime))
	response.Period.StartPeriod.End = fmt.Sprintf("%s", end.Format(nowTime))
	response.Period.EndPeriod.Start = fmt.Sprintf("%s", prevStart.Format(nowTime))
	response.Period.EndPeriod.End = fmt.Sprintf("%s", prevEnd.Format(nowTime))
	response.Metrics.SalesVolume = currentVolume
	response.Metrics.Revenue = currentRevenue
	response.Metrics.GrowthPercentage = growth
	response.Comparison.PreviousPeriod.SalesVolume = prevVolume
	response.Comparison.PreviousPeriod.Revenue = prevRevenue

	return response, nil
}

// SalesTrendPoint represents sales metrics for a single time point.
// Used in time-series sales trend analysis.
type SalesTrendPoint struct {
	Date             string  `json:"date"`              // Date/period identifier
	SalesVolume      int     `json:"sales_volume"`      // Units sold in this period
	Revenue          float64 `json:"revenue"`           // Revenue for this period
	GrowthPercentage float64 `json:"growth_percentage"` // Growth vs previous point
}

// SalesTrendsOverTime represents a time-series of sales data.
// Provides trend analysis over multiple time periods.
type SalesTrendsOverTime struct {
	Period struct {
		Start string `json:"start"` // Overall start date
		End   string `json:"end"`   // Overall end date
		Type  string `json:"type"`  // Grouping type (day/week/month/quarter/year)
	} `json:"period"`
	Data []SalesTrendPoint `json:"data"` // Array of trend points
}

// GetSalesTrendsOverTime retrieves sales trends grouped by time period.
//
// This function provides time-series analysis of sales data with dynamic
// grouping (daily, weekly, monthly, quarterly, or yearly).
//
// Parameters:
//   - period: string - Grouping type:
//   - "day" (default): Daily breakdown
//   - "week": Weekly aggregation (YEARWEEK)
//   - "month": Monthly aggregation (YYYY-MM)
//   - "quarter": Quarterly aggregation (YYYY-Q#)
//   - "year": Yearly aggregation
//   - start: time.Time - Date range start
//   - end: time.Time - Date range end
//
// Returns:
//   - SalesTrendsOverTime: Time-series data containing:
//   - Period definition (start, end, type)
//   - Data array with sales volume, revenue, and growth % for each period
//   - error: Database error or nil on success
func GetSalesTrendsOverTime(period string, start, end time.Time) (SalesTrendsOverTime, error) {

	// Determine SQL grouping expression based on period type
	var groupBy string
	switch period {
	case "week":
		groupBy = "YEARWEEK(o.created_at)"
	case "month":
		groupBy = "DATE_FORMAT(o.created_at, '%Y-%m')"
	case "quarter":
		groupBy = "CONCAT(YEAR(o.created_at), '-Q', QUARTER(o.created_at))"
	case "year":
		groupBy = "YEAR(o.created_at)"
	default: // Default to daily
		groupBy = "DATE(o.created_at)"
	}

	// Build dynamic query with period-specific grouping
	query := fmt.Sprintf(`
		SELECT %s AS period_date,
		       COALESCE(SUM(oi.quantity), 0) AS sales_volume,
		       COALESCE(SUM(oi.quantity * oi.unit_price), 0) AS revenue
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.created_at BETWEEN ? AND ?
		GROUP BY period_date
		ORDER BY period_date ASC
	`, groupBy)

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return SalesTrendsOverTime{}, err
	}
	defer rows.Close()

	var data []SalesTrendPoint
	var prevRevenue float64 // Track previous period for growth calculation

	for rows.Next() {
		var periodDate string
		var salesVolume int
		var revenue float64

		if err := rows.Scan(&periodDate, &salesVolume, &revenue); err != nil {
			return SalesTrendsOverTime{}, err
		}

		// Calculate growth vs previous period (0 for first period)
		growth := 0.0
		if prevRevenue > 0 {
			growth = ((revenue - prevRevenue) / prevRevenue) * 100
		}
		prevRevenue = revenue // Update for next iteration

		data = append(data, SalesTrendPoint{
			Date:             periodDate,
			SalesVolume:      salesVolume,
			Revenue:          revenue,
			GrowthPercentage: growth,
		})
	}

	// Build response with period metadata
	response := SalesTrendsOverTime{}
	response.Period.Start = fmt.Sprintf("%s", start.Format(nowTime))
	response.Period.End = fmt.Sprintf("%s", end.Format(nowTime))
	response.Period.Type = period
	response.Data = data

	return response, nil
}

// GetCustomerSegmentation analyzes customers by delivery location.
//
// This function provides customer insights grouped by delivery address,
// calculating order count, total sales, and average values per segment.
//
// Parameters:
//   - start: time.Time - Date range start
//   - end: time.Time - Date range end
//
// Returns:
//   - []dtos.CustomerSegment: Array of segments containing:
//   - SegmentName: Delivery address
//   - OrderCount, TotalSales, AvgPurchaseValue, AvgOrderValue, Transactions
//   - error: Database error or nil on success
func GetCustomerSegmentation(start, end time.Time) ([]dtos.CustomerSegment, error) {
	query := `
        SELECT 
            d.delivery_address AS segment_name,
            COUNT(DISTINCT o.order_id) AS order_count,
            SUM(oi.unit_price * oi.quantity) AS total_sales,
            AVG(oi.unit_price * oi.quantity) AS avg_purchase_value,
            (SUM(oi.unit_price * oi.quantity) / COUNT(DISTINCT o.order_id)) AS avg_order_value,
            COUNT(DISTINCT o.order_id) AS transactions
        FROM orders o
        JOIN order_items oi ON o.order_id = oi.order_id
        JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.created_at BETWEEN ? AND ?
        GROUP BY d.delivery_address
    `

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []dtos.CustomerSegment
	for rows.Next() {
		var seg dtos.CustomerSegment
		if err := rows.Scan(&seg.SegmentName, &seg.OrderCount, &seg.TotalSales, &seg.AvgPurchaseValue, &seg.AvgOrderValue, &seg.Transactions); err != nil {
			return nil, err
		}
		segments = append(segments, seg)
	}
	return segments, nil
}
