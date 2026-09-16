package models

import (
	"time"
)

func GetCustomerRetentionSummary(start, end time.Time) (*CustomerRetentionSummary, error) {
	// Use CTE to find each customer's first order ever
	// Then categorize customers based on when their first order occurred
	query := `
		WITH first_orders AS (
			SELECT user_id, MIN(last_updated_at) AS first_order
			FROM orders
			GROUP BY user_id
		)
		SELECT 
			-- New customers: first order was in the period [start, end]
			COUNT(DISTINCT CASE WHEN fo.first_order BETWEEN ? AND ? THEN o.user_id END) AS new_customers,
			-- Returning customers: first order was before start, but ordered during [start, end]
			COUNT(DISTINCT CASE WHEN fo.first_order < ? AND o.last_updated_at BETWEEN ? AND ? THEN o.user_id END) AS returning_customers
		FROM orders o
		JOIN first_orders fo ON fo.user_id = o.user_id
		WHERE o.last_updated_at BETWEEN ? AND ?;
	`

	// Execute query with date parameters
	// Parameters: start, end (new customers), start, start, end (returning), start, end (WHERE filter)
	row := DB.QueryRow(query,
		start, end, // New customers check
		start, start, end, // Returning customers check
		start, end, // Overall WHERE filter
	)

	var summary CustomerRetentionSummary
	if err := row.Scan(&summary.NewCustomers, &summary.ReturningCustomers); err != nil {
		return nil, err
	}

	// Calculate retention rate as percentage of total customers
	total := summary.NewCustomers + summary.ReturningCustomers
	if total > 0 {
		summary.RetentionRate = float64(summary.ReturningCustomers) / float64(total) * 100
	}

	return &summary, nil
}
