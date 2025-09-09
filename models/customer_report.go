package models

import (
	"fmt"
	"time"
)

type CustomerOrderDetail struct {
	UserID     string  `json:"user_id"`
	OrderCount int64   `json:"order_count"`
	Revenue    float64 `json:"revenue"`
}

type CustomerRetention struct {
	NewCustomers       int64                 `json:"new_customers"`
	ReturningCustomers int64                 `json:"returning_customers"`
	ReturningOrders    int64                 `json:"returning_orders"`
	NewOrders          int64                 `json:"new_orders"`
	ReturningDetails   []CustomerOrderDetail `json:"returning_details"`
	NewDetails         []CustomerOrderDetail `json:"new_details"`
}

func GetCustomerRetention(start, end time.Time, months int) (CustomerRetention, error) {
	// If months is provided, adjust the start date
	if months > 0 {
		start = end.AddDate(0, -months, 0) // go back N months
	}

	// Summary
	summaryQuery := `
		WITH user_orders AS (
			SELECT 
				o.user_id,
				COUNT(*) AS total_orders_in_period,
				SUM(o.total_amount) AS total_revenue
			FROM orders o
			WHERE o.user_id IS NOT NULL
			  AND o.last_updated_at BETWEEN ? AND ?
			GROUP BY o.user_id
		)
		SELECT
			COUNT(CASE WHEN total_orders_in_period = 1 THEN 1 END) AS new_customers,
			COUNT(CASE WHEN total_orders_in_period > 1 THEN 1 END) AS returning_customers,
			COALESCE(SUM(CASE WHEN total_orders_in_period > 1 THEN total_orders_in_period ELSE 0 END), 0) AS returning_orders,
			COALESCE(SUM(CASE WHEN total_orders_in_period = 1 THEN total_orders_in_period ELSE 0 END), 0) AS new_orders
		FROM user_orders;
	`

	var result CustomerRetention
	err := DB.QueryRow(summaryQuery, start, end).
		Scan(&result.NewCustomers, &result.ReturningCustomers, &result.ReturningOrders, &result.NewOrders)
	if err != nil {
		return CustomerRetention{}, err
	}

	// Returning customers detail
	returningQuery := `
		SELECT o.user_id, COUNT(*) AS order_count, SUM(o.total_amount) AS revenue
		FROM orders o
		WHERE o.user_id IS NOT NULL
		  AND o.last_updated_at BETWEEN ? AND ?
		GROUP BY o.user_id
		HAVING COUNT(*) > 1
		ORDER BY order_count DESC;
	`

	rows, err := DB.Query(returningQuery, start, end)
	if err != nil {
		return CustomerRetention{}, err
	}
	defer rows.Close()

	var returning []CustomerOrderDetail
	for rows.Next() {
		var d CustomerOrderDetail
		if err := rows.Scan(&d.UserID, &d.OrderCount, &d.Revenue); err != nil {
			return CustomerRetention{}, err
		}
		returning = append(returning, d)
	}
	result.ReturningDetails = returning

	// New customers detail
	newQuery := `
		SELECT o.user_id, COUNT(*) AS order_count, SUM(o.total_amount) AS revenue
		FROM orders o
		WHERE o.user_id IS NOT NULL
		  AND o.last_updated_at BETWEEN ? AND ?
		GROUP BY o.user_id
		HAVING COUNT(*) = 1
		ORDER BY revenue DESC;
	`

	newRows, err := DB.Query(newQuery, start, end)
	if err != nil {
		return CustomerRetention{}, err
	}
	defer newRows.Close()

	var newDetails []CustomerOrderDetail
	for newRows.Next() {
		var d CustomerOrderDetail
		if err := newRows.Scan(&d.UserID, &d.OrderCount, &d.Revenue); err != nil {
			return CustomerRetention{}, err
		}
		newDetails = append(newDetails, d)
	}
	result.NewDetails = newDetails

	return result, nil
}

type CustomerRetentionTrend struct {
	Period             string
	NewCustomers       int64
	ReturningCustomers int64
	RetentionRate      float64
}

// groupBy = "month" | "quarter" | "year"
func GetCustomerRetentionTrends(start, end time.Time, groupBy string) ([]CustomerRetentionTrend, error) {
	var periodExpr string
	switch groupBy {
	case "month":
		periodExpr = "DATE_FORMAT(o.last_updated_at, '%Y-%m')" // e.g., 2025-09
	case "quarter":
		periodExpr = "CONCAT(YEAR(o.last_updated_at), '-Q', QUARTER(o.last_updated_at))"
	case "year":
		periodExpr = "YEAR(o.last_updated_at)"
	default:
		return nil, fmt.Errorf("invalid groupBy value: %s", groupBy)
	}

	// Each user aggregated by period
	query := fmt.Sprintf(`
		WITH user_orders AS (
			SELECT 
				%s AS period,
				o.user_id,
				COUNT(*) AS total_orders
			FROM orders o
			WHERE o.user_id IS NOT NULL
			  AND o.last_updated_at BETWEEN ? AND ?
			GROUP BY period, o.user_id
		)
		SELECT 
			period,
			COUNT(CASE WHEN total_orders = 1 THEN 1 END) AS new_customers,
			COUNT(CASE WHEN total_orders > 1 THEN 1 END) AS returning_customers
		FROM user_orders
		GROUP BY period
		ORDER BY period;
	`, periodExpr)

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []CustomerRetentionTrend
	for rows.Next() {
		var t CustomerRetentionTrend
		if err := rows.Scan(&t.Period, &t.NewCustomers, &t.ReturningCustomers); err != nil {
			return nil, err
		}

		total := t.NewCustomers + t.ReturningCustomers
		if total > 0 {
			t.RetentionRate = float64(t.ReturningCustomers) / float64(total) * 100
		} else {
			t.RetentionRate = 0
		}

		trends = append(trends, t)
	}

	return trends, nil
}

type CustomerRetentionSummary struct {
	NewCustomers       int64
	ReturningCustomers int64
	RetentionRate      float64
}

func GetCustomerRetentionSummary(start, end time.Time) (*CustomerRetentionSummary, error) {
	query := `
		WITH first_orders AS (
			SELECT user_id, MIN(last_updated_at) AS first_order
			FROM orders
			GROUP BY user_id
		)
		SELECT 
			COUNT(DISTINCT CASE WHEN fo.first_order BETWEEN ? AND ? THEN o.user_id END) AS new_customers,
			COUNT(DISTINCT CASE WHEN fo.first_order < ? AND o.last_updated_at BETWEEN ? AND ? THEN o.user_id END) AS returning_customers
		FROM orders o
		JOIN first_orders fo ON fo.user_id = o.user_id
		WHERE o.last_updated_at BETWEEN ? AND ?;
	`

	row := DB.QueryRow(query,
		start, end, // new customers
		start, start, end, // returning customers
		start, end, // WHERE filter
	)

	var summary CustomerRetentionSummary
	if err := row.Scan(&summary.NewCustomers, &summary.ReturningCustomers); err != nil {
		return nil, err
	}

	total := summary.NewCustomers + summary.ReturningCustomers
	if total > 0 {
		summary.RetentionRate = float64(summary.ReturningCustomers) / float64(total) * 100
	}

	return &summary, nil
}
