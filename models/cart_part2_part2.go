package models

import (
	"database/sql"
	"fmt"
	"time"
)

func GetCartAbandonmentTrend(db DBExecutor, start, end time.Time, period string) ([]AbandonmentTrend, error) {
	var groupBy, periodSelect string

	// Determine SQL grouping and date formatting based on period
	switch period {
	case "daily":
		groupBy = "DATE(c.created_at)"
		periodSelect = "DATE(c.created_at)"
	case "weekly":
		groupBy = "YEARWEEK(c.created_at)"
		periodSelect = "YEARWEEK(c.created_at)"
	case "monthly":
		groupBy = "DATE_FORMAT(c.created_at, '%Y-%m')"
		periodSelect = "DATE_FORMAT(c.created_at, '%Y-%m')"
	case "quarterly":
		groupBy = "CONCAT(YEAR(c.created_at), '-Q', QUARTER(c.created_at))"
		periodSelect = "CONCAT(YEAR(c.created_at), '-Q', QUARTER(c.created_at))"
	case "yearly":
		groupBy = "YEAR(c.created_at)"
		periodSelect = "YEAR(c.created_at)"
		// Default case removed - period must be specified
	}

	// Build dynamic query with period-specific grouping
	query := fmt.Sprintf(`
		SELECT %s AS period_date,
			   COUNT(DISTINCT c.cart_id) AS carts_created,
			   SUM(CASE 
					   WHEN c.updated_at >= DATE_SUB(NOW(), INTERVAL 7 DAY) 
							OR NOT EXISTS (SELECT 1 FROM cart_items ci WHERE ci.cart_id = c.cart_id) 
					   THEN 1 ELSE 0 
				   END) AS completed_purchases
		FROM cart c
		WHERE c.created_at BETWEEN ? AND ?
		GROUP BY %s
		ORDER BY %s ASC
	`, periodSelect, groupBy, groupBy)

	// Execute query to get cart counts per period
	rows, err := db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AbandonmentTrend
	threshold := time.Now().AddDate(0, 0, -7) // 7-day abandonment threshold

	// Process each period's results
	for rows.Next() {
		var periodDate string
		var cartsCreated, completedPurchases int

		if err := rows.Scan(&periodDate, &cartsCreated, &completedPurchases); err != nil {
			return nil, err
		}

		// Calculate abandoned carts for this specific period
		var abandonedCarts int
		// Query abandoned carts with items not updated in 7+ days for this period
		abandonedQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT c.cart_id)
			FROM cart c
			JOIN cart_items ci ON c.cart_id = ci.cart_id
			WHERE %s = ?
			  AND c.updated_at < ?
		`, periodSelect)

		err = db.QueryRow(abandonedQuery, periodDate, threshold).Scan(&abandonedCarts)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}

		// Calculate abandonment rate for this period
		rate := 0.0
		if cartsCreated > 0 {
			rate = (float64(abandonedCarts) / float64(cartsCreated)) * 100
		}

		// Append trend data point
		results = append(results, AbandonmentTrend{
			Date:         periodDate,
			CartsCreated: cartsCreated,
			// CompletedPurchases: completedPurchases,
			AbandonmentRate: rate,
		})
	}

	return results, nil
}
