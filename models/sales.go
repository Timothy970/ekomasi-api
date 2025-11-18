package models

import (
	"adenzo_backend/dtos"
	"fmt"
	"time"
)

var nowTime = "2006-01-02 15:04:05"

type SalesSummaryOverTime struct {
	Period struct {
		Start string  `json:"start"`
		End   string  `json:"end"`
		Type  *string `json:"type"`
	} `json:"period"`
	Metrics struct {
		SalesVolume      int     `json:"sales_volume"`
		Revenue          float64 `json:"revenue"`
		GrowthPercentage float64 `json:"growth_percentage"`
	} `json:"metrics"`
	Comparison struct {
		PreviousPeriod struct {
			SalesVolume int     `json:"sales_volume"`
			Revenue     float64 `json:"revenue"`
		} `json:"previous_period"`
	} `json:"comparison"`
}
type SalesTrendsSummary struct {
	Period struct {
		StartPeriod struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"start_period"`
		EndPeriod struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"end_period"`
	} `json:"period"`
	Metrics struct {
		SalesVolume      int     `json:"sales_volume"`
		Revenue          float64 `json:"revenue"`
		GrowthPercentage float64 `json:"growth_percentage"`
	} `json:"metrics"`
	Comparison struct {
		PreviousPeriod struct {
			SalesVolume int     `json:"sales_volume"`
			Revenue     float64 `json:"revenue"`
		} `json:"previous_period"`
	} `json:"comparison"`
}

func GetSalesTrendsSummary(start, end, prevStart, prevEnd time.Time) (SalesTrendsSummary, error) {

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

	growth := 0.0
	if prevRevenue > 0 {
		growth = ((currentRevenue - prevRevenue) / prevRevenue) * 100
	}

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

type SalesTrendPoint struct {
	Date             string  `json:"date"`
	SalesVolume      int     `json:"sales_volume"`
	Revenue          float64 `json:"revenue"`
	GrowthPercentage float64 `json:"growth_percentage"`
}

type SalesTrendsOverTime struct {
	Period struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Type  string `json:"type"`
	} `json:"period"`
	Data []SalesTrendPoint `json:"data"`
}

func GetSalesTrendsOverTime(period string, start, end time.Time) (SalesTrendsOverTime, error) {

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
	default:
		groupBy = "DATE(o.created_at)"
	}

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
	var prevRevenue float64

	for rows.Next() {
		var periodDate string
		var salesVolume int
		var revenue float64

		if err := rows.Scan(&periodDate, &salesVolume, &revenue); err != nil {
			return SalesTrendsOverTime{}, err
		}

		growth := 0.0
		if prevRevenue > 0 {
			growth = ((revenue - prevRevenue) / prevRevenue) * 100
		}
		prevRevenue = revenue

		data = append(data, SalesTrendPoint{
			Date:             periodDate,
			SalesVolume:      salesVolume,
			Revenue:          revenue,
			GrowthPercentage: growth,
		})
	}

	response := SalesTrendsOverTime{}
	response.Period.Start = fmt.Sprintf("%s", start.Format(nowTime))
	response.Period.End = fmt.Sprintf("%s", end.Format(nowTime))
	response.Period.Type = period
	response.Data = data

	return response, nil
}

// Customer segmentation by delivery location
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

// Sales by region
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
		if totalSales > 0 {
			rgn.Percentage = (rgn.TotalSales / totalSales) * 100
		}
		regions = append(regions, rgn)
	}
	return regions, nil
}

// get sales overview, return total sales, monthly sales, today's sales, percentage of increase/decrease from previous month
func GetSalesOverview() (map[string]any, error) {
	overview := make(map[string]any)
	// Total Sales
	var totalSales float64
	err := DB.QueryRow(`SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0) FROM order_items oi`).Scan(&totalSales)
	if err != nil {
		return nil, err
	}
	overview["total_sales"] = totalSales

	// Monthly Sales
	var monthlySales float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o	
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE MONTH(o.created_at) = MONTH(CURRENT_DATE())
		  AND YEAR(o.created_at) = YEAR(CURRENT_DATE())
	`).Scan(&monthlySales)
	if err != nil {
		return nil, err
	}
	overview["monthly_sales"] = monthlySales

	// Today's Sales
	var todaysSales float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o	
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE DATE(o.created_at) = CURRENT_DATE()
	`).Scan(&todaysSales)
	if err != nil {
		return nil, err
	}
	overview["todays_sales"] = todaysSales

	// Percentage Change from Previous Month
	var prevMonthlySales float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o	
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE MONTH(o.created_at) = MONTH(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND YEAR(o.created_at) = YEAR(CURRENT_DATE() - INTERVAL 1 MONTH)
	`).Scan(&prevMonthlySales)
	if err != nil {
		return nil, err
	}
	percentageChange := 0.0
	if prevMonthlySales > 0 {
		percentageChange = ((monthlySales - prevMonthlySales) / prevMonthlySales) * 100
	}
	overview["percentage_change"] = percentageChange
	return overview, nil
}

// get sales vs orders per month, filteres by range of year eg 2025, 2024 , 2026 etc.
// sales are orders with a status of Delivered/delivered/DELIVERED orders are not yet
// so will return orders and sales by month, january to december
func GetSalesVsOrdersPerMonth(year int) (map[string]any, error) {
	salesVsOrders := make(map[string]any)
	type MonthData struct {
		Month       string  `json:"month"`
		TotalOrders int     `json:"total_orders"`
		TotalSales  float64 `json:"total_sales"`
	}
	var data []MonthData
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

type RevenueExpense struct {
	Period   string  `json:"period"` // day (YYYY-MM-DD) or month name
	Revenue  float64 `json:"revenue"`
	Expenses float64 `json:"expenses"`
}

type RevenueExpenseFilter struct {
	Year int `json:"year"`
	Week int `json:"week"` // optional
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

func getISOWeek(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

//return total revenue, revenue for that month, percentage increase/decrease from previous month
//return total customers, new customers for that month, percentage increase/decrease from previous month
//return total orders, orders for that month, percentage increase/decrease from previous month

type RevenueCustomersOrdersOverview struct {
	Revenue   RevenueOverview   `json:"revenue"`
	Customers CustomersOverview `json:"customers"`
	Orders    OrdersOverview    `json:"orders"`
}
type RevenueOverview struct {
	TotalRevenue            float64 `json:"total_revenue"`
	MonthlyRevenue          float64 `json:"monthly_revenue"`
	RevenuePercentageChange float64 `json:"revenue_percentage_change"`
}
type CustomersOverview struct {
	TotalCustomers            int     `json:"total_customers"`
	MonthlyNewCustomers       int     `json:"monthly_new_customers"`
	CustomersPercentageChange float64 `json:"customers_percentage_change"`
}
type OrdersOverview struct {
	TotalOrders            int     `json:"total_orders"`
	MonthlyOrders          int     `json:"monthly_orders"`
	OrdersPercentageChange float64 `json:"orders_percentage_change"`
}

func GetRevenueCustomersOrdersOverview() (RevenueCustomersOrdersOverview, error) {
	overview := RevenueCustomersOrdersOverview{}
	//revenue is for orders with status Delivered/delivered/DELIVERED
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
	// Revenue for this month
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
	// Percentage change from previous month
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
	percentageChange := 0.0
	if prevMonthlyRevenue > 0 {
		percentageChange = ((monthlyRevenue - prevMonthlyRevenue) / prevMonthlyRevenue) * 100
	}
	overview.Revenue.RevenuePercentageChange = percentageChange

	// Total Customers
	var totalCustomers int
	err = DB.QueryRow(`SELECT COUNT(DISTINCT user_id) FROM users`).Scan(&totalCustomers)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Customers.TotalCustomers = totalCustomers
	// New Customers for this month
	var monthlyNewCustomers int
	err = DB.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM users
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE())
		  AND YEAR(created_at) = YEAR(CURRENT_DATE())
	`).Scan(&monthlyNewCustomers)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Customers.MonthlyNewCustomers = monthlyNewCustomers
	// Percentage change from previous month
	var prevMonthlyNewCustomers int
	err = DB.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM users
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND YEAR(created_at) = YEAR(CURRENT_DATE() - INTERVAL 1 MONTH)
	`).Scan(&prevMonthlyNewCustomers)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	percentageChange = 0.0
	if prevMonthlyNewCustomers > 0 {
		percentageChange = ((float64(monthlyNewCustomers) - float64(prevMonthlyNewCustomers)) / float64(prevMonthlyNewCustomers)) * 100
	}
	overview.Customers.CustomersPercentageChange = percentageChange

	// Total Orders
	var totalOrders int
	err = DB.QueryRow(`SELECT COUNT(order_id) FROM orders`).Scan(&totalOrders)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Orders.TotalOrders = totalOrders
	// Orders for this month
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
	// Percentage change from previous month
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
	percentageChange = 0.0
	if prevMonthlyOrders > 0 {
		percentageChange = ((float64(monthlyOrders) - float64(prevMonthlyOrders)) / float64(prevMonthlyOrders)) * 100
	}
	overview.Orders.OrdersPercentageChange = percentageChange
	return overview, nil
}
