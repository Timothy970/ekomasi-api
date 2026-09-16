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
