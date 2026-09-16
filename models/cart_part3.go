// Package models provides the shopping cart management functionality for the Ekomasi e-commerce platform.
//
// This package handles core cart operations including:
//   - Cart creation and retrieval for authenticated and guest users
//   - Cart item management (add, update, delete)
//   - Stock availability validation before adding items
//   - Promotion data retrieval for cart pricing
//   - Cart abandonment analytics and tracking
//   - Order notification management
//   - Discount code type resolution (coupons, promo codes, vouchers)
//   - Tax estimation for cart totals
//
// Cart Features:
//   - User-specific carts for authenticated users
//   - Guest cart support with cart_id
//   - Automatic cart timestamp updates on modifications
//   - Real-time stock validation
//   - Duplicate key handling for cart item updates
//   - Cart abandonment detection (7-day inactivity threshold)
//
// Analytics Capabilities:
//   - Cart abandonment rate calculation
//   - Abandonment trend analysis (daily, weekly, monthly, quarterly, yearly)
//   - Paginated abandoned cart retrieval
//   - Date range filtering for analytics
//
// Database Schema:
//   - cart table: Stores cart metadata (cart_id, user_id, timestamps)
//   - cart_items table: Stores individual cart items with quantities
//   - products table: Referenced for stock validation and product details
//   - promotion_products, promotions, promotion_types: Referenced for pricing
//   - order_notifications table: Tracks pending order notifications
//   - charges table: Stores tax rates and other charges
package models

import (
	"database/sql"
	"fmt"
	"time"
)

// CartItem represents a single item in a cart for analytics.
//
// Used internally by analytics functions for cart item tracking.
type CartItem struct {
	ID        string    `json:"id"`         // Cart item ID
	ProductID string    `json:"product_id"` // Referenced product ID
	Quantity  int       `json:"quantity"`   // Number of units
	CreatedAt time.Time `json:"created_at"` // When item was added to cart
}

// Cart represents a shopping cart with its items for analytics.
//
// Used by AnalyticsRepository for abandoned cart retrieval.
type Cart struct {
	CartID    string     `json:"cart_id"`           // Unique cart identifier
	UserID    *string    `json:"user_id,omitempty"` // User ID (nil for guest carts)
	Items     []CartItem `json:"items"`             // Array of items in cart
	CreatedAt time.Time  `json:"created_at"`        // Cart creation timestamp
}

// PaginationMeta contains pagination metadata for paginated responses.
//
// Provides navigation information for paginated result sets.
type PaginationMeta struct {
	Page       int  `json:"page"`        // Current page number (1-based)
	Size       int  `json:"size"`        // Items per page
	TotalItems int  `json:"total_items"` // Total number of items across all pages
	TotalPages int  `json:"total_pages"` // Total number of pages
	HasPrev    bool `json:"has_prev"`    // True if previous page exists
	HasNext    bool `json:"has_next"`    // True if next page exists
}

// AbandonedCartsResponse represents a paginated response of abandoned carts.
//
// Used by GetAbandonedCarts to return carts with pagination metadata.
type AbandonedCartsResponse struct {
	Carts []Cart         `json:"carts"` // Array of abandoned cart objects
	Meta  PaginationMeta `json:"meta"`  // Pagination navigation metadata
}

// AnalyticsRepository provides methods for cart analytics operations.
//
// This repository pattern encapsulates database access for analytics queries.
type AnalyticsRepository struct {
	DB *sql.DB // Database connection
}

// GetAbandonedCarts retrieves paginated abandoned carts with items older than 7 days.
//
// This method performs a multi-step process:
//  1. Counts total abandoned carts for pagination metadata
//  2. Retrieves paginated cart records
//  3. For each cart, fetches associated abandoned items
//  4. Builds pagination metadata
//
// Parameters:
//   - page: Page number (1-based)
//   - size: Number of carts per page
//
// Returns:
//   - *AbandonedCartsResponse: Paginated carts with items and navigation metadata
//   - error: Database error if queries fail
//
// Abandonment Criteria:
//   - Cart has items with created_at older than 7 days
//   - Ordered by cart creation date (oldest first)
//
// Response Structure:
//   - Carts: Array of Cart objects with Items populated
//   - Meta: Pagination info (page, size, total, has_prev, has_next)
func (r *AnalyticsRepository) GetAbandonedCarts(page, size int) (*AbandonedCartsResponse, error) {
	// Calculate offset for pagination (0-based)
	offset := (page - 1) * size

	// Step 1: Count total abandoned carts for pagination
	var totalItems int
	countQuery := `
		SELECT COUNT(DISTINCT c.cart_id)
		FROM cart c
		JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE ci.created_at < NOW() - INTERVAL 7 DAY
	`
	if err := r.DB.QueryRow(countQuery).Scan(&totalItems); err != nil {
		return nil, fmt.Errorf("count abandoned carts: %w", err)
	}

	// Step 2: Get paginated abandoned carts
	query := `
		SELECT c.cart_id, c.user_id, c.created_at
		FROM cart c
		JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE ci.created_at < NOW() - INTERVAL 7 DAY
		GROUP BY c.cart_id
		ORDER BY c.created_at ASC
		LIMIT ? OFFSET ?
	`

	rows, err := r.DB.Query(query, size, offset)
	if err != nil {
		return nil, fmt.Errorf("query abandoned carts: %w", err)
	}
	defer rows.Close()

	var carts []Cart
	// Process each abandoned cart
	for rows.Next() {
		var cart Cart
		if err := rows.Scan(&cart.CartID, &cart.UserID, &cart.CreatedAt); err != nil {
			return nil, err
		}

		// Step 3: Get abandoned items for this cart (items older than 7 days)
		itemQuery := `
			SELECT id, product_id, quantity, created_at
			FROM cart_items
			WHERE cart_id = ? AND created_at < NOW() - INTERVAL 7 DAY
		`
		itemRows, err := r.DB.Query(itemQuery, cart.CartID)
		if err != nil {
			return nil, err
		}

		// Fetch all items for this cart
		for itemRows.Next() {
			var item CartItem
			if err := itemRows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.CreatedAt); err != nil {
				itemRows.Close()
				return nil, err
			}
			cart.Items = append(cart.Items, item)
		}
		itemRows.Close()

		carts = append(carts, cart)
	}

	// Step 4: Build pagination metadata
	totalPages := (totalItems + size - 1) / size // Ceiling division
	meta := PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	// Return paginated response
	return &AbandonedCartsResponse{
		Carts: carts,
		Meta:  meta,
	}, nil
}

// GetEstimatedTax retrieves the tax rate from the charges table.
//
// This function fetches the current tax rate configuration for cart total calculations.
// If no tax charge is configured, defaults to 16.0%.
//
// Parameters:
//   - None
//
// Returns:
//   - float64: Tax rate percentage (e.g., 16.0 for 16%)
//   - error: Database error if query fails (excluding ErrNoRows)
//
// Default Behavior:
//   - Returns 16.0 if no "tax" charge exists in database
//   - This is a fallback for unconfigured systems
func GetEstimatedTax(db DBExecutor) (float64, error) {
	var tax float64
	// Query tax charge value from charges table
	err := db.QueryRow(`
		SELECT charge_value
		FROM charges
		WHERE charge_name = "tax"
	`).Scan(&tax)
	if err != nil {
		if err == sql.ErrNoRows {
			// Default to 16% if no tax charge configured
			return 16.0, nil
		}
		return 0, err
	}

	return tax, nil
}

// StoreOrderNotification creates a pending notification record for an order.
//
// This function is called after order creation to queue a notification for sending.
// Notifications are processed asynchronously by background workers.
//
// Parameters:
//   - orderID: The order_id to create notification for
//
// Returns:
//   - error: Database error if insert fails, nil on success
func StoreOrderNotification(orderID string) error {
	// Insert notification record with pending status
	query := `
		INSERT INTO order_notifications (order_id, status, created_at)
		VALUES (?, 'pending', NOW())
	`
	_, err := DB.Exec(query, orderID)
	return err
}

// GetPendingOrderNotifications retrieves all order IDs with pending notifications.
//
// This function is used by background workers to fetch orders that need
// notification emails/SMS sent to customers.
//
// Parameters:
//   - None
//
// Returns:
//   - []string: Array of order_ids with pending notifications
//   - error: Database error if query fails, nil on success (empty array if no pending)
//
// Worker Usage:
//   - Call periodically to get pending notifications
//   - Process each order_id (send notification)
//   - Update status to 'sent' after processing
func GetPendingOrderNotifications() ([]string, error) {
	// Query all pending notification order IDs
	query := `
		SELECT order_id
		FROM order_notifications
		WHERE status = 'pending'
	`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orderIDs []string
	// Collect all pending order IDs
	for rows.Next() {
		var orderID string
		if err := rows.Scan(&orderID); err != nil {
			return nil, err
		}
		orderIDs = append(orderIDs, orderID)
	}

	return orderIDs, nil
}

// GetDiscountCodeType determines which type of discount code is being used.
//
// This function checks if a code exists in coupons, promo_codes, or vouchers tables
// and returns the appropriate type. Used for routing to correct validation logic.
//
// Parameters:
//   - code: The discount code string to identify
//
// Returns:
//   - string: Code type ("coupon", "promo_code", or "voucher"), defaults to "promo_code"
//
// Type Priority:
//  1. Check coupons table -> return "coupon"
//  2. Check promo_codes table -> return "promo_code"
//  3. Check vouchers table -> return "voucher"
//  4. Default: return "promo_code" (if not found or error)
func GetDiscountCodeType(db DBExecutor, code string) string {
	var discountType string
	// Check which table contains this code using CASE statement
	query := `
		SELECT CASE
			WHEN EXISTS (SELECT 1 FROM coupons WHERE code = ?) THEN 'coupon'
			WHEN EXISTS (SELECT 1 FROM promo_codes WHERE code = ?) THEN 'promo_code'
			WHEN EXISTS (SELECT 1 FROM vouchers WHERE code = ?) THEN 'voucher'
			ELSE 'promo_code'
		END
	`
	err := db.QueryRow(query, code, code, code).Scan(&discountType)
	// Default to promo_code if query fails
	if err != nil {
		return "promo_code"
	}
	return discountType
}
