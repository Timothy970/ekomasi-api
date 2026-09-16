// Package models provides data access functions for financial and sales reporting.
//
// This file handles accounting and analytics reports including:
//   - Balance Sheet (assets, liabilities, equity as of a date)
//   - Income Statement (revenue and expenses for a period)
//   - Cash Flow Statement (direct method with inflows/outflows)
//   - General Ledger (detailed account transactions with running balance)
//   - CSV exports for accounts and journal entries
//   - Top-selling products analytics (daily/weekly/monthly/yearly)
//
// The accounting system uses double-entry bookkeeping with:
//   - Debit-normal accounts: Assets, Expenses (balance = debit - credit)
//   - Credit-normal accounts: Liabilities, Equity, Income (balance = credit - debit)
//   - Normal balance calculations adjust for account type
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"strings"
	"time"
)

// GetTopSellingProducts retrieves top-selling products with sales analytics.
//
// This function generates a sales report showing products ranked by quantity sold,
// with optional time-based filtering (daily, weekly, monthly, yearly).
//
// Parameters:
//   - timeRange: string - Time filter: "daily", "weekly", "monthly", "yearly", or "" (all time)
//   - page: int - Page number (1-based)
//   - size: int - Number of products per page
//
// Returns:
//   - []dtos.TopProduct: Array of top-selling products containing:
//   - ProductID: Unique product identifier
//   - ProductName: Product name
//   - TotalQuantity: Total units sold
//   - ProductImage: First product image URL (may be empty)
//   - TotalRevenue: Total revenue from product sales
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
//
// Time Ranges:
//   - "daily": Products sold today
//   - "weekly": Products sold in the last 7 days (including today)
//   - "monthly": Products sold in the current month
//   - "yearly": Products sold in the current year
//   - "": All-time top sellers (no date filter)
func GetTopSellingProducts(timeRange string, page, size int) ([]dtos.TopProduct, *dtos.PaginationMeta, error) {
	var totalCount int
	now := time.Now()
	normalizedTimeRange := normalizeTopSellingTimeRange(timeRange)
	const dateRangeFilter = "o.created_at >= ? AND o.created_at < ?"

	// Build date filter
	dateFilter := ""
	switch normalizedTimeRange {
	case "daily":
		dateFilter = dateRangeFilter
	case "weekly":
		dateFilter = dateRangeFilter
	case "monthly":
		dateFilter = dateRangeFilter
	case "yearly":
		dateFilter = dateRangeFilter
	}

	offset := (page - 1) * size

	// Base WHERE clause (only completed and paid orders)
	whereClause := "WHERE o.status IN ('completed', 'COMPLETED') AND o.payment_status IN ('PAID', 'paid')"
	if dateFilter != "" {
		whereClause += " AND " + dateFilter
	}

	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT oi.product_id
			FROM order_items oi
			JOIN orders o ON oi.order_id = o.order_id
			` + whereClause + `
			GROUP BY oi.product_id
		) AS grouped_products
	`

	err := DB.QueryRow(countQuery, getDateFilterArgs(normalizedTimeRange, now)...).Scan(&totalCount)
	if err != nil {
		return nil, nil, err
	}

	query := `
		SELECT 
			p.product_id,
			p.name AS product_name,
			SUM(oi.quantity) AS total_quantity,
			pi.url AS product_image,
			SUM(oi.quantity * oi.unit_price) AS total_revenue
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		JOIN products p ON oi.product_id = p.product_id
		LEFT JOIN (
			SELECT product_id, MIN(url) AS url
			FROM product_images
			GROUP BY product_id
		) pi ON pi.product_id = p.product_id
		` + whereClause + `
		GROUP BY p.product_id, p.name, pi.url
		ORDER BY total_quantity DESC
		LIMIT ? OFFSET ?
	`

	args := append(getDateFilterArgs(normalizedTimeRange, now), size, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var results []dtos.TopProduct
	for rows.Next() {
		var tp dtos.TopProduct
		var image sql.NullString

		if err := rows.Scan(
			&tp.ProductID,
			&tp.ProductName,
			&tp.TotalQuantity,
			&image,
			&tp.TotalRevenue,
		); err != nil {
			return nil, nil, err
		}

		if image.Valid {
			tp.ProductImage = image.String
		}

		results = append(results, tp)
	}

	meta := &dtos.PaginationMeta{
		TotalItems: totalCount,
		Page:       page,
		Size:       size,
		TotalPages: (totalCount + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    offset+size < totalCount,
	}

	return results, meta, nil
}

// normalizeTopSellingTimeRange normalizes time_range aliases used by clients.
func normalizeTopSellingTimeRange(timeRange string) string {
	switch strings.ToLower(strings.TrimSpace(timeRange)) {
	case "daily", "day", "today":
		return "daily"
	case "weekly", "week", "this_week", "last_7_days", "last7days":
		return "weekly"
	case "monthly", "month", "this_month", "montly":
		return "monthly"
	case "yearly", "year", "this_year", "annual":
		return "yearly"
	default:
		return ""
	}
}

// getDateFilterArgs builds SQL query arguments for date filtering.
//
// This is an internal helper function that generates the appropriate query arguments
// based on the time range filter.
//
// Parameters:
//   - timeRange: string - The time range filter ("daily", "weekly", "monthly", "yearly", or "")
//   - now: time.Time - The current timestamp for date calculations
//
// Returns:
//   - []any: Array of arguments for SQL query:
//   - "daily": [startOfDay, startOfNextDay]
//   - "weekly": [startOfDay(6 days ago), startOfNextDay]
//   - "monthly": [startOfMonth, startOfNextMonth]
//   - "yearly": [startOfYear, startOfNextYear]
//   - "": [] - Empty array (no date filter)
func getDateFilterArgs(timeRange string, now time.Time) []any {
	loc := now.Location()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	startOfNextDay := startOfDay.AddDate(0, 0, 1)

	switch timeRange {
	case "daily":
		return []any{startOfDay, startOfNextDay}
	case "weekly":
		startOfLast7Days := startOfDay.AddDate(0, 0, -6)
		return []any{startOfLast7Days, startOfNextDay}
	case "monthly":
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		startOfNextMonth := startOfMonth.AddDate(0, 1, 0)
		return []any{startOfMonth, startOfNextMonth}
	case "yearly":
		startOfYear := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, loc)
		startOfNextYear := startOfYear.AddDate(1, 0, 0)
		return []any{startOfYear, startOfNextYear}
	default:
		// No date filter - return empty array
		return []any{}
	}
}
