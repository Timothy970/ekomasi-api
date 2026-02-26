// Package models provides data access functions for the Adenzo e-commerce backend.
//
// sales.go handles sales analytics and reporting including:
//   - Sales trends and growth analysis
//   - Revenue vs expenses tracking
//   - Customer segmentation by region
//   - Sales overview dashboards
//   - Period-over-period comparisons
package models

import (
	"adenzo_backend/dtos"
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

// GetSalesByRegion analyzes sales performance by delivery region.
//
// This function provides regional sales breakdown with percentage contribution
// to total sales.
//
// Parameters:
//   - start: time.Time - Date range start
//   - end: time.Time - Date range end
//
// Returns:
//   - []dtos.RegionSales: Array of regions containing:
//   - Region: Delivery address
//   - TotalSales, AvgOrderValue, Transactions
//   - Percentage: Region's % of total sales
//   - error: Database error or nil on success
func GetSalesByRegion(start, end time.Time) ([]dtos.RegionSales, error) {
	query := `
        SELECT 
            d.delivery_address AS region,
            SUM(oi.unit_price * oi.quantity) AS total_sales,
            (SUM(oi.unit_price * oi.quantity) / COUNT(DISTINCT o.order_id)) AS avg_order_value,
            COUNT(DISTINCT o.order_id) AS transactions
        FROM orders o
        JOIN order_items oi ON o.order_id = oi.order_id
        JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.created_at BETWEEN ? AND ?
        GROUP BY d.delivery_address
    `

	// Calculate total sales across all regions for percentage calculation
	var totalSales float64
	err := DB.QueryRow(`
		SELECT 
			SUM(oi.unit_price * oi.quantity) AS total_sales
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.created_at BETWEEN ? AND ?
	`, start, end).Scan(&totalSales)

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []dtos.RegionSales
	for rows.Next() {
		var rgn dtos.RegionSales
		if err := rows.Scan(&rgn.Region, &rgn.TotalSales, &rgn.AvgOrderValue, &rgn.Transactions); err != nil {
			return nil, err
		}

		// Calculate region's percentage of total sales
		if totalSales > 0 {
			rgn.Percentage = (rgn.TotalSales / totalSales) * 100
		}
		regions = append(regions, rgn)
	}
	return regions, nil
}

// GetSalesOverview provides a quick sales dashboard summary.
//
// This function calculates key sales metrics:
//   - Total sales (all time)
//   - Monthly sales (current month)
//   - Today's sales
//   - Month-over-month percentage change
//
// Returns:
//   - map[string]any: Dashboard metrics containing:
//   - "total_sales": float64 - All-time sales
//   - "monthly_sales": float64 - Current month sales
//   - "todays_sales": float64 - Today's sales
//   - "percentage_change": float64 - Current vs previous month % change
//   - error: Database error or nil on success
func GetSalesOverview() (map[string]any, error) {
	overview := make(map[string]any)

	// Total Sales (all time) - filtering by paid status
	var totalSales float64
	var totalDiscount float64
	err := DB.QueryRow(`
		SELECT 
			COALESCE(SUM(oi.unit_price * oi.quantity), 0),
			COALESCE(SUM(o.total_discount), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.payment_status IN ('PAID', 'paid')
	`).Scan(&totalSales, &totalDiscount)
	if err != nil {
		return nil, err
	}
	overview["total_sales"] = totalSales
	overview["total_discount_amount"] = totalDiscount

	// Monthly Sales (current month) - filtering by paid status
	var monthlySales float64
	var monthlyDiscount float64
	err = DB.QueryRow(`
		SELECT 
			COALESCE(SUM(oi.unit_price * oi.quantity), 0),
			COALESCE(SUM(o.total_discount), 0)
		FROM orders o	
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE MONTH(o.created_at) = MONTH(CURRENT_DATE())
		  AND YEAR(o.created_at) = YEAR(CURRENT_DATE())
		  AND o.payment_status IN ('PAID', 'paid')
	`).Scan(&monthlySales, &monthlyDiscount)
	if err != nil {
		return nil, err
	}
	overview["monthly_sales"] = monthlySales
	overview["monthly_discount_amount"] = monthlyDiscount

	// Today's Sales and Discount - filtering by paid status
	var todaysSales float64
	var todaysDiscount float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0), COALESCE(SUM(o.total_discount), 0)
		FROM orders o    
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE DATE(o.created_at) = CURRENT_DATE()
		  AND o.payment_status IN ('PAID', 'paid')
	`).Scan(&todaysSales, &todaysDiscount)
	if err != nil {
		return nil, err
	}
	overview["todays_sales"] = todaysSales
	overview["todays_discount_amount"] = todaysDiscount

	// Percentage Change from Previous Month - filtering by paid status
	var prevMonthlySales float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o	
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE MONTH(o.created_at) = MONTH(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND YEAR(o.created_at) = YEAR(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND o.payment_status IN ('PAID', 'paid')
	`).Scan(&prevMonthlySales)
	if err != nil {
		return nil, err
	}

	// Calculate month-over-month percentage change
	percentageChange := 0.0
	if prevMonthlySales > 0 {
		percentageChange = ((monthlySales - prevMonthlySales) / prevMonthlySales) * 100
	}
	overview["percentage_change"] = percentageChange
	return overview, nil
}

// GetSalesVsOrdersPerMonth compares total orders vs delivered sales by month.
//
// This function distinguishes between:
//   - Orders: All orders (any status)
//   - Sales: Only orders with "Delivered" status (case-insensitive)
//
// Parameters:
//   - year: int - The year to analyze (e.g., 2024, 2025)
//
// Returns:
//   - map[string]any: Contains:
//   - "year": int - The queried year
//   - "data": []MonthData - Array of 12 months with:
//   - Month: Month name (January-December)
//   - TotalOrders: Count of all orders
//   - TotalSales: Revenue from delivered orders only
//   - error: Database error or nil on success
func GetSalesVsOrdersPerMonth(year int) (map[string]any, error) {
	salesVsOrders := make(map[string]any)

	// MonthData represents order and sales data for a single month
	type MonthData struct {
		Month       string  `json:"month"`        // Month name
		TotalOrders int     `json:"total_orders"` // All orders (any status)
		TotalSales  float64 `json:"total_sales"`  // Revenue from delivered orders only
	}

	var data []MonthData

	// Query orders and sales by month for the specified year
	query := `
	SELECT 
	    MONTHNAME(o.created_at) AS month,
	    COUNT(o.order_id) AS total_orders,
	    COALESCE(SUM(
	         CASE WHEN o.status IN ('Delivered', 'delivered', 'DELIVERED') 
	         THEN oi.unit_price * oi.quantity 
	         ELSE 0 END
	    ), 0) AS total_sales
	FROM orders o
	JOIN order_items oi ON o.order_id = oi.order_id
	WHERE YEAR(o.created_at) = ?
	GROUP BY MONTH(o.created_at), MONTHNAME(o.created_at)
	ORDER BY MONTH(o.created_at)
`

	rows, err := DB.Query(query, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build monthly data array
	for rows.Next() {
		var md MonthData
		if err := rows.Scan(&md.Month, &md.TotalOrders, &md.TotalSales); err != nil {
			return nil, err
		}
		data = append(data, md)
	}

	salesVsOrders["year"] = year
	salesVsOrders["data"] = data
	return salesVsOrders, nil
}

// RevenueExpense represents revenue and expense data for a time period.
type RevenueExpense struct {
	Period   string  `json:"period"`   // Day (YYYY-MM-DD) or month name
	Revenue  float64 `json:"revenue"`  // Sales revenue from orders
	Expenses float64 `json:"expenses"` // Inventory costs (buying price * quantity)
}

// RevenueExpenseFilter defines filtering options for revenue/expense queries.
// Currently unused but provided for future expansion.
type RevenueExpenseFilter struct {
	Year int `json:"year"` // Year to filter
	Week int `json:"week"` // Optional week number
}

func GetRevenueVsExpenses(filterType string) ([]RevenueExpense, error) {
	var (
		query string
		args  []interface{}
	)

	now := time.Now()
	currentYear := now.Year()
	currentWeek := getISOWeek(now)
	yearWeek := fmt.Sprintf("%d%02d", currentYear, currentWeek)

	if filterType == "week" {
		query = `
        SELECT 
            DAYNAME(date_day) AS period,
            COALESCE((
                SELECT SUM(oi.quantity * oi.unit_price)
                FROM order_items oi
                JOIN orders o ON oi.order_id = o.order_id
                WHERE DATE(o.created_at) = date_day
            ), 0) AS revenue,
            COALESCE((
                SELECT SUM(i.quantity * p.buying_price)
                FROM inventory i
                JOIN products p ON i.product_id = p.product_id
                WHERE DATE(i.created_at) = date_day
            ), 0) AS expenses
        FROM (
            SELECT DATE_ADD(
                STR_TO_DATE(CONCAT(?, ' Monday'), '%x%v %W'),
                INTERVAL n DAY
            ) AS date_day
            FROM (SELECT 0 n UNION SELECT 1 UNION SELECT 2 UNION 
                  SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6) days
        ) AS week_dates
        ORDER BY date_day;
        `

		// CURRENT YEAR + CURRENT WEEK
		args = []interface{}{yearWeek}

	} else {

		query = `
            SELECT 
                MONTHNAME(date_month) AS period,
                COALESCE((
                    SELECT SUM(oi.quantity * oi.unit_price)
                    FROM order_items oi
                    JOIN orders o ON oi.order_id = o.order_id
                    WHERE YEAR(o.created_at) = ?
                      AND MONTH(o.created_at) = MONTH(date_month)
                ), 0) AS revenue,
                COALESCE((
                    SELECT SUM(i.quantity * p.buying_price)
                    FROM inventory i
                    JOIN products p ON i.product_id = p.product_id
                    WHERE YEAR(i.created_at) = ?
                      AND MONTH(i.created_at) = MONTH(date_month)
                ), 0) AS expenses
            FROM (
                SELECT STR_TO_DATE(CONCAT(?, '-', m, '-01'), '%Y-%m-%d') AS date_month
                FROM (
                    SELECT 1 m UNION SELECT 2 UNION SELECT 3 UNION 
                    SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION
                    SELECT 7 UNION SELECT 8 UNION SELECT 9 UNION 
                    SELECT 10 UNION SELECT 11 UNION SELECT 12
                ) months
            ) AS year_months
            ORDER BY MONTH(date_month);
        `

		// CURRENT YEAR for all parameters
		args = []interface{}{currentYear, currentYear, currentYear}
	}

	// Execute query and collect results
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RevenueExpense
	for rows.Next() {
		var r RevenueExpense
		if err := rows.Scan(&r.Period, &r.Revenue, &r.Expenses); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

// getISOWeek returns the ISO week number for a given date.
//
// Parameters:
//   - t: time.Time - Date to get week number for
//
// Returns:
//   - int: ISO week number (1-53)
func getISOWeek(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

// RevenueCustomersOrdersOverview provides comprehensive business metrics dashboard.
// Aggregates key performance indicators for revenue, customers, and orders.
type RevenueCustomersOrdersOverview struct {
	Revenue   RevenueOverview   `json:"revenue"`   // Revenue metrics with growth
	Customers CustomersOverview `json:"customers"` // Customer acquisition metrics
	Orders    OrdersOverview    `json:"orders"`    // Order volume metrics
}

// RevenueOverview tracks revenue performance and growth.
type RevenueOverview struct {
	TotalRevenue            float64 `json:"total_revenue"`             // All-time revenue from delivered orders
	MonthlyRevenue          float64 `json:"monthly_revenue"`           // Current month revenue
	RevenuePercentageChange float64 `json:"revenue_percentage_change"` // Month-over-month % change
}

// CustomersOverview tracks customer acquisition and growth.
type CustomersOverview struct {
	TotalCustomers            int     `json:"total_customers"`             // Total registered users
	MonthlyNewCustomers       int     `json:"monthly_new_customers"`       // New users this month
	CustomersPercentageChange float64 `json:"customers_percentage_change"` // Month-over-month % change
}

// OrdersOverview tracks order volume and growth.
type OrdersOverview struct {
	TotalOrders            int     `json:"total_orders"`             // All-time order count
	MonthlyOrders          int     `json:"monthly_orders"`           // Orders this month
	OrdersPercentageChange float64 `json:"orders_percentage_change"` // Month-over-month % change
}

// GetRevenueCustomersOrdersOverview provides comprehensive business metrics.
//
// This dashboard function aggregates:
//   - Revenue metrics (total, monthly, growth %)
//   - Customer metrics (total, new this month, growth %)
//   - Order metrics (total, monthly, growth %)
//
// All metrics include month-over-month comparison.
//
// Returns:
//   - RevenueCustomersOrdersOverview: Complete dashboard metrics
//   - error: Database error or nil on success
func GetRevenueCustomersOrdersOverview() (RevenueCustomersOrdersOverview, error) {
	overview := RevenueCustomersOrdersOverview{}

	// ----- REVENUE METRICS -----
	// Total revenue from all delivered orders (all time)
	var totalRevenue float64
	err := DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.status IN ('Delivered', 'delivered', 'DELIVERED')
	`).Scan(&totalRevenue)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Revenue.TotalRevenue = totalRevenue

	// Revenue for current month from delivered orders
	var monthlyRevenue float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.status IN ('Delivered', 'delivered', 'DELIVERED')
		  AND MONTH(o.created_at) = MONTH(CURRENT_DATE())
		  AND YEAR(o.created_at) = YEAR(CURRENT_DATE())
	`).Scan(&monthlyRevenue)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Revenue.MonthlyRevenue = monthlyRevenue

	// Previous month revenue for comparison
	var prevMonthlyRevenue float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.status IN ('Delivered', 'delivered', 'DELIVERED')
		  AND MONTH(o.created_at) = MONTH(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND YEAR(o.created_at) = YEAR(CURRENT_DATE() - INTERVAL 1 MONTH)
	`).Scan(&prevMonthlyRevenue)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}

	// Calculate revenue growth percentage
	percentageChange := 0.0
	if prevMonthlyRevenue > 0 {
		percentageChange = ((monthlyRevenue - prevMonthlyRevenue) / prevMonthlyRevenue) * 100
	}
	overview.Revenue.RevenuePercentageChange = percentageChange

	// ----- CUSTOMERS METRICS -----
	// Total registered customers (all time)
	var totalCustomers int
	err = DB.QueryRow(`SELECT COUNT(DISTINCT user_id) FROM users WHERE status = 'active' AND role = 'customer'`).Scan(&totalCustomers)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Customers.TotalCustomers = totalCustomers

	// New customers registered this month
	var monthlyNewCustomers int
	err = DB.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM users
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE())
		  AND YEAR(created_at) = YEAR(CURRENT_DATE())
		  AND status = 'active'
		  AND role = 'customer'
	`).Scan(&monthlyNewCustomers)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Customers.MonthlyNewCustomers = monthlyNewCustomers

	// Previous month new customers for comparison
	var prevMonthlyNewCustomers int
	err = DB.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM users
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND YEAR(created_at) = YEAR(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND status = 'active'
		  AND role = 'customer'
	`).Scan(&prevMonthlyNewCustomers)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}

	// Calculate customer growth percentage
	percentageChange = 0.0
	if prevMonthlyNewCustomers > 0 {
		percentageChange = ((float64(monthlyNewCustomers) - float64(prevMonthlyNewCustomers)) / float64(prevMonthlyNewCustomers)) * 100
	}
	overview.Customers.CustomersPercentageChange = percentageChange

	// ----- ORDERS METRICS -----
	// Total orders (all time, any status)
	var totalOrders int
	err = DB.QueryRow(`SELECT COUNT(order_id) FROM orders`).Scan(&totalOrders)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Orders.TotalOrders = totalOrders

	// Orders created this month
	var monthlyOrders int
	err = DB.QueryRow(`
		SELECT COUNT(order_id)
		FROM orders
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE())
		  AND YEAR(created_at) = YEAR(CURRENT_DATE())
	`).Scan(&monthlyOrders)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Orders.MonthlyOrders = monthlyOrders

	// Previous month orders for comparison
	var prevMonthlyOrders int
	err = DB.QueryRow(`
		SELECT COUNT(order_id)
		FROM orders
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND YEAR(created_at) = YEAR(CURRENT_DATE() - INTERVAL 1 MONTH)
	`).Scan(&prevMonthlyOrders)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}

	// Calculate order growth percentage
	percentageChange = 0.0
	if prevMonthlyOrders > 0 {
		percentageChange = ((float64(monthlyOrders) - float64(prevMonthlyOrders)) / float64(prevMonthlyOrders)) * 100
	}
	overview.Orders.OrdersPercentageChange = percentageChange

	return overview, nil
}
