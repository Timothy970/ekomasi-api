// Package models provides data access functions for the Ekomasi backend application.
//
// This file contains promotion and promo code management functionality including:
//   - Promotion effectiveness tracking and analytics
//   - Promo code creation, validation, and redemption
//   - Promotion-to-product associations
//   - Promotion comparison with baseline periods
//   - Sales and revenue aggregation for promotional periods
//   - Active promo code lookup and validation
package models

import (
	"ekomasi_backend/dtos"
	"errors"
	"strings"
	"time"
)

// isPromotionThere validates if a promotion exists in the database.
//
// This is an internal helper function used before promotion operations.
//
// Parameters:
//   - id: string - The promotion_id to validate
//
// Returns:
//   - error: "promotion not found" if ID doesn't exist, database error, or nil if exists
func isPromotionThere(db DBExecutor, id string) error {
	// Check promotion existence
	exists, err := RecordExists(db, "promotions", "promotion_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("promotion not found")
	}
	return nil
}

// GetPromotionDetails retrieves promotion date range and linked product IDs.
//
// This function is used to fetch promotion metadata for effectiveness analysis.
//
// Parameters:
//   - promotionID: string - The promotion_id to retrieve
//
// Returns:
//   - time.Time: Promotion start date
//   - time.Time: Promotion end date
//   - []string: Array of product IDs linked to this promotion
//   - error: "promotion not found", database error, or nil on success
func GetPromotionDetails(db DBExecutor, promotionID string) (time.Time, time.Time, []string, error) {
	// Validate promotion exists
	err := isPromotionThere(db, promotionID)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}

	var startDate, endDate time.Time

	// Get promotion date range
	err = db.QueryRow(`
		SELECT start_date, end_date
		FROM promotions
		WHERE promotion_id = ?`, promotionID).Scan(&startDate, &endDate)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}

	// Get linked products
	rows, err := db.Query(`
		SELECT product_id
		FROM promotion_products
		WHERE promotion_id = ?`, promotionID)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}
	defer rows.Close()

	// Scan product IDs
	var products []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return time.Time{}, time.Time{}, nil, err
		}
		products = append(products, pid)
	}

	return startDate, endDate, products, nil
}

// GetPromotionEffectiveness calculates sales performance for promoted products during a time period.
//
// This function aggregates quantity sold and revenue for specific products
// within the given date range, typically used for promotion ROI analysis.
//
// Parameters:
//   - productIDs: []string - Product IDs to analyze (must not be empty)
//   - start: time.Time - Analysis period start date
//   - end: time.Time - Analysis period end date
//
// Returns:
//   - []dtos.PromotionEffectiveness: Array of effectiveness data per product:
//   - ProductID: Product identifier
//   - TotalSold: Total quantity sold
//   - Revenue: Total revenue generated
//   - error: Database error or nil on success
func GetPromotionEffectiveness(db DBExecutor, productIDs []string, start, end time.Time) ([]dtos.PromotionEffectiveness, error) {
	var query string
	var args []interface{}

	// Build dynamic IN clause with placeholders
	placeholders := "?" + strings.Repeat(",?", len(productIDs)-1)
	query = `
			SELECT oi.product_id, SUM(oi.quantity) AS total_sold, SUM(oi.quantity * oi.unit_price) AS revenue
			FROM order_items oi
			JOIN orders o ON oi.order_id = o.order_id
			WHERE oi.product_id IN (` + placeholders + `)
			  AND o.last_updated_at BETWEEN ? AND ?
			GROUP BY oi.product_id;
		`

	// Add product IDs to args
	for _, id := range productIDs {
		args = append(args, id)
	}
	// Add date range to args
	args = append(args, start, end)

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan results
	var results []dtos.PromotionEffectiveness
	for rows.Next() {
		var pe dtos.PromotionEffectiveness
		if err := rows.Scan(&pe.ProductID, &pe.TotalSold, &pe.Revenue); err != nil {
			return nil, err
		}
		results = append(results, pe)
	}

	return results, nil
}

// GetPromotionAggregate calculates total sales and revenue for specific products during a time period.
//
// This function returns aggregated metrics (not per-product breakdown) for promotional analysis.
//
// Parameters:
//   - productIDs: []string - Product IDs to aggregate (must not be empty)
//   - start: time.Time - Period start date
//   - end: time.Time - Period end date
//
// Returns:
//   - int64: Total quantity sold across all products
//   - float64: Total revenue across all products
//   - error: Database error or nil on success
func GetPromotionAggregate(db DBExecutor, productIDs []string, start, end time.Time) (int64, float64, error) {
	// Build dynamic IN clause with placeholders
	query := `
		SELECT COALESCE(SUM(oi.quantity),0), COALESCE(SUM(oi.quantity * oi.unit_price),0)
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE oi.product_id IN (?` + strings.Repeat(",?", len(productIDs)-1) + `)
		  AND o.last_updated_at BETWEEN ? AND ?
	`

	// Build args array with product IDs and date range
	args := make([]interface{}, len(productIDs)+2)
	for i, id := range productIDs {
		args[i] = id
	}
	args[len(productIDs)] = start
	args[len(productIDs)+1] = end

	// Execute aggregation query
	var qty int64
	var rev float64
	err := db.QueryRow(query, args...).Scan(&qty, &rev)
	return qty, rev, err
}

// GetNonPromotionAggregate calculates total sales and revenue for ALL products during a time period.
//
// This function is used as a baseline comparison for promotion effectiveness,
// showing overall store performance regardless of promotions.
//
// Parameters:
//   - start: time.Time - Period start date
//   - end: time.Time - Period end date
//
// Returns:
//   - int64: Total quantity sold (all products)
//   - float64: Total revenue (all products)
//   - error: Database error or nil on success
func GetNonPromotionAggregate(db DBExecutor, start, end time.Time) (int64, float64, error) {
	// Aggregate all orders in date range (no product filter)
	query := `
		SELECT 
			COALESCE(SUM(oi.quantity), 0) AS total_qty,
			COALESCE(SUM(oi.quantity * oi.unit_price), 0) AS total_rev
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE o.last_updated_at BETWEEN ? AND ?;
	`

	var qty int64
	var rev float64
	err := db.QueryRow(query, start, end).Scan(&qty, &rev)
	if err != nil {
		return 0, 0, err
	}
	return qty, rev, nil
}

// GetPromotionSummary retrieves performance metrics for all promotions within a date range.
//
// This function provides a comprehensive overview of promotion effectiveness,
// including promotions with no sales (NULL order handling).
//
// Parameters:
//   - start: time.Time - Report period start date
//   - end: time.Time - Report period end date
//
// Returns:
//
//   - []dtos.PromotionSummary: Array of promotion summaries containing:
//
//   - PromotionID: Unique promotion identifier
//
//   - PromotionType: Type of promotion (e.g., "Flash Sale", "BOGO")
//
//   - StartDate: Promotion start date
//
//   - EndDate: Promotion end date
//
//   - TotalSold: Total quantity sold (0 if no orders)
//
//   - Revenue: Total revenue generated (0 if no orders)
//
//   - error: Database error or nil on success
//
//     the analysis period (intersects start or end dates).
func GetPromotionSummary(db DBExecutor, start, end time.Time) ([]dtos.PromotionSummary, error) {
	// Query promotions with LEFT JOINs to include promotions with no sales
	query := `
SELECT 
    p.promotion_id, 
    pt.name, 
    p.start_date, 
    p.end_date,
    COALESCE(SUM(oi.quantity),0) AS total_sold, 
    COALESCE(SUM(oi.quantity * oi.unit_price),0) AS revenue
FROM promotions p
JOIN promotion_types pt ON p.promotion_type_id = pt.id
LEFT JOIN promotion_products pp ON p.promotion_id = pp.promotion_id
LEFT JOIN order_items oi ON pp.product_id = oi.product_id
LEFT JOIN orders o ON oi.order_id = o.order_id
WHERE (o.last_updated_at BETWEEN GREATEST(p.start_date, ?) 
                             AND LEAST(p.end_date, ?)
       OR o.order_id IS NULL)   -- Include promotions with no orders
GROUP BY p.promotion_id, pt.name, p.start_date, p.end_date;

	`
	rows, err := db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan promotion summary rows
	var summaries []dtos.PromotionSummary
	for rows.Next() {
		var ps dtos.PromotionSummary
		if err := rows.Scan(&ps.PromotionID, &ps.PromotionType, &ps.StartDate, &ps.EndDate, &ps.TotalSold, &ps.Revenue); err != nil {
			return nil, err
		}
		summaries = append(summaries, ps)
	}

	return summaries, nil
}
