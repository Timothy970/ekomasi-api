package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"time"
)

func GetVariationPrice(db DBExecutor, variationSKU string) (float64, string, error) {
	var additionalPrice float64
	var variationName string
	err := db.QueryRow(`SELECT additional_price, name FROM product_variant_combinations WHERE sku = ?`, variationSKU).Scan(&additionalPrice, &variationName)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", nil // No additional price for this variation - not an error
		}
		return 0, "", err
	}
	return additionalPrice, variationName, nil
}

// UpdateCartItem updates the quantity of a specific product in a cart.
//
// This function validates cart and product existence before updating the quantity.
// The cart timestamp is automatically updated after modification.
//
// Parameters:
//   - cartID: The cart_id containing the item
//   - productID: The product_id to update
//   - quantity: The new quantity (replaces existing quantity)
//
// Returns:
//   - error: Validation error or database error if update fails
func UpdateCartItem(db DBExecutor, cartID string, productID string, quantity int, tenantID int) error {
	// Validate cart exists
	err := isCartThere(db, cartID, tenantID)
	if err != nil {
		return err
	}
	// Validate product exists
	err = IsProductThere(db, productID)
	if err != nil {
		return err
	}
	// Update quantity for specific cart item
	_, err = db.Exec(`
		UPDATE cart_items SET quantity = ?
		WHERE cart_id = ? AND product_id = ?
	`, quantity, cartID, productID)
	// Update cart timestamp to track activity
	err = UpdateCartTimestamp(db, cartID)
	return err
}

// DeleteCartItem removes a specific product from a cart.
//
// This function validates cart and product existence before deleting the item.
// The cart timestamp is automatically updated after modification.
//
// Parameters:
//   - cartID: The cart_id containing the item
//   - productID: The product_id to remove
//
// Returns:
//   - error: Validation error or database error if deletion fails
func DeleteCartItem(db DBExecutor, cartID, productID string, tenantID int) error {
	// Validate cart exists
	err := isCartThere(db, cartID, tenantID)
	if err != nil {
		return err
	}
	// Validate product exists
	err = IsProductThere(db, productID)
	if err != nil {
		return err
	}
	// Delete cart item record
	_, err = db.Exec(`
		DELETE FROM cart_items WHERE cart_id = ? AND product_id = ?
	`, cartID, productID)
	// Update cart timestamp to track activity
	err = UpdateCartTimestamp(db, cartID)
	return err
}

// GetProductPromotionData retrieves active promotion information for a specific product.
//
// This function performs a three-way JOIN across promotion tables to fetch the promotion
// type (e.g., "percentage", "fixed_amount") and value for a product if an active promotion exists.
//
// Parameters:
//   - productID: The product_id to check for promotions
//
// Returns:
//   - dtos.PromotionData: Contains promotion Type and Value, empty struct if no promotion
//   - error: Database error if query fails, nil if no promotion (returns empty PromotionData)
//
// PromotionData Structure:
//   - Type: Promotion type name (e.g., "percentage", "buy_one_get_one")
//   - Value: Promotion value (percentage number or fixed amount)
func GetProductPromotionData(db DBExecutor, productID string) (dtos.PromotionData, error) {
	// Query with JOINs to get promotion type and value
	query := `
		SELECT 
			pt.name AS promotion_type, 
			pt.value AS promotion_value
		FROM promotion_products pp
		INNER JOIN promotions p ON p.promotion_id = pp.promotion_id
		INNER JOIN promotion_types pt ON pt.id = p.promotion_type_id
		WHERE pp.product_id = ?
		LIMIT 1
	`

	var promotionType string
	var promotionValue float64 // Promotion value (percentage or amount)

	// Execute query and scan result
	err := db.QueryRow(query, productID).Scan(&promotionType, &promotionValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return dtos.PromotionData{}, nil // No promotion for product - not an error
		}
		return dtos.PromotionData{}, err
	}

	// Return promotion details
	return dtos.PromotionData{
		Type:  promotionType,
		Value: promotionValue,
	}, nil
}

// GetCartAbandonmentRate calculates cart abandonment statistics for a date range.
//
// This function analyzes cart behavior within a specified period to calculate:
//   - Total carts created
//   - Completed purchases (active within 7 days or empty)
//   - Abandoned carts (with items, inactive for 7+ days)
//   - Abandonment rate percentage
//
// Parameters:
//   - start: Start date/time for analysis period
//   - end: End date/time for analysis period
//
// Returns:
//   - *AbandonmentStats: Pointer to statistics object with totals and rate
//   - error: Database error if queries fail
//
// Abandonment Definition:
//   - Cart has items AND hasn't been updated in 7+ days
//   - Completed carts: updated within 7 days OR empty (checkout completed)
//
// Rate Calculation: (abandoned_carts / total_carts) * 100
func GetCartAbandonmentRate(db DBExecutor, start, end time.Time) (*AbandonmentStats, error) {
	var cartsCreated, completedPurchases, abandonedCarts int

	// Count total carts created within the period
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM cart 
		WHERE created_at BETWEEN ? AND ?
	`, start, end).Scan(&cartsCreated)
	if err != nil {
		return nil, err
	}

	// Count completed purchases (carts updated within 7 days OR empty carts)
	err = db.QueryRow(`
    SELECT COUNT(*)
    FROM (
        SELECT c.cart_id
        FROM cart c
        LEFT JOIN cart_items ci ON c.cart_id = ci.cart_id
        WHERE c.updated_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
           OR c.cart_id NOT IN (SELECT cart_id FROM cart_items)
        GROUP BY c.cart_id
    ) AS completed
`).Scan(&completedPurchases)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Count abandoned carts (have items AND not updated in 7+ days)
	threshold := time.Now().AddDate(0, 0, -7) // 7 days ago
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT c.cart_id)
		FROM cart c
		JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE c.created_at BETWEEN ? AND ?
		  AND c.updated_at < ?
	`, start, end, threshold).Scan(&abandonedCarts)
	if err != nil {
		return nil, err
	}

	// Calculate abandonment rate percentage
	rate := 0.0
	if cartsCreated > 0 {
		rate = (float64(abandonedCarts) / float64(cartsCreated)) * 100
	}

	// Return statistics
	return &AbandonmentStats{
		TotalCarts: cartsCreated,
		// CompletedPurchases: completedPurchases,
		AbandonedCarts: abandonedCarts,
		Rate:           rate,
	}, nil
}

// AbandonmentTrend represents cart abandonment statistics for a specific time period.
//
// Used by GetCartAbandonmentTrend to provide time-series analytics.
type AbandonmentTrend struct {
	Date         string `json:"date"`          // Period identifier (date, week, month, quarter, year)
	CartsCreated int    `json:"carts_created"` // Total carts created in period
	// CompletedPurchases int     `json:"completed_purchases"`
	AbandonmentRate float64 `json:"abandonment_rate"` // Abandonment rate percentage for period
}

// AbandonmentStats represents overall cart abandonment statistics.
//
// Used by GetCartAbandonmentRate to provide summary analytics.
type AbandonmentStats struct {
	// CompletedPurchases int     `json:"active_carts"`
	Rate           float64 `json:"rate"`            // Abandonment rate percentage
	AbandonedCarts int     `json:"abandoned_carts"` // Total abandoned carts
	TotalCarts     int     `json:"total_carts"`     // Total carts created
}

// GetCartAbandonmentTrend calculates cart abandonment trends over time with flexible period grouping.
//
// This function analyzes cart abandonment patterns across a date range, grouping results by
// the specified time period (daily, weekly, monthly, quarterly, yearly).
//
// Parameters:
//   - start: Start date/time for analysis period
//   - end: End date/time for analysis period
//   - period: Time grouping ("daily", "weekly", "monthly", "quarterly", "yearly")
//
// Returns:
//   - []AbandonmentTrend: Array of trend data points, one per period
//   - error: Database error if queries fail
//
// Period Formats:
//   - daily: "2025-08-15" (DATE)
//   - weekly: "202533" (YEARWEEK)
//   - monthly: "2025-08" (YYYY-MM)
//   - quarterly: "2025-Q3" (YEAR-QX)
//   - yearly: "2025" (YEAR)
//
// Each trend point includes:
//   - Date: Period identifier in appropriate format
//   - CartsCreated: Total carts created in that period
//   - AbandonmentRate: Percentage of abandoned carts in that period
