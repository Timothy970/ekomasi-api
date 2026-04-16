// Package models provides the shopping cart management functionality for the Adenzo e-commerce platform.
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
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/teris-io/shortid"
)

// isCartThere validates that a cart exists in the database by cart_id.
//
// This function uses the RecordExists helper to check for cart existence.
//
// Parameters:
//   - id: The cart_id to validate
//
// Returns:
//   - error: nil if cart exists, "cart not found" error if not found, or database error
func isCartThere(db DBExecutor, id string) error {
	// Check if cart record exists in cart table
	exists, err := RecordExists(db, "cart", "cart_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("cart not found")
	}
	return nil
}

// UpdateCartTimestamp updates the cart's updated_at timestamp to current time.
//
// This function is called automatically whenever cart items are modified to track
// cart activity and help identify abandoned carts.
//
// Parameters:
//   - cartID: The cart_id to update
//
// Returns:
//   - error: Database error if update fails, nil on success
func UpdateCartTimestamp(db DBExecutor, cartID string) error {
	// Update cart timestamp to NOW()
	_, err := db.Exec(`
        UPDATE cart 
        SET updated_at = NOW() 
        WHERE cart_id = ?`, cartID)
	return err
}

// InsertCartItem adds a product to a cart or updates quantity if it already exists.
//
// This function performs comprehensive validation before inserting:
//   - Validates cart existence
//   - Validates product existence
//   - Checks stock availability for requested quantity
//   - Uses ON DUPLICATE KEY UPDATE for upsert behavior
//   - Updates cart timestamp after modification
//
// Parameters:
//   - cartID: The cart_id to add the item to
//   - productID: The product_id to add
//   - quantity: The quantity to add (must not exceed available stock)
//
// Returns:
//   - error: Validation error, stock error, or database error if insert fails
func InsertCartItem(db DBExecutor, cartID string, productID string, quantity int, variationSKU *string) error {
	// Validate cart exists
	err := isCartThere(db, cartID)
	if err != nil {
		return err
	}
	// Validate product exists
	err = IsProductThere(db, productID)
	if err != nil {
		return err
	}
	// Check stock availability before adding to cart
	if err := isStockAvailable(db, productID, quantity); err != nil {
		return err
	}
	// Generate unique cart item ID
	ID, _ := shortid.Generate()

	// Insert or update cart item using ON DUPLICATE KEY UPDATE
	_, err = db.Exec(`
        INSERT INTO cart_items(id, cart_id, product_id, quantity, variation_sku)
        VALUES (?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE quantity = VALUES(quantity)
    `, ID, cartID, productID, quantity, variationSKU)
	if err != nil {
		return fmt.Errorf("failed to insert cart item: %w", err)
	}
	// Update cart timestamp to track activity
	err = UpdateCartTimestamp(db, cartID)
	if err != nil {
		return err
	}
	return nil
}

// isStockAvailable checks if enough stock is available for a given product and quantity.
//
// This function validates that the requested quantity does not exceed the product's
// current stock_quantity in the database. It's called before adding items to cart
// to prevent overselling.
//
// Parameters:
//   - productID: The product_id to check stock for
//   - quantity: The requested quantity to validate
//
// Returns:
//   - error: nil if stock is sufficient, error with details if insufficient or product not found
//
// Error Messages:
//   - "product not found" if productID doesn't exist
//   - "requested quantity (X) exceeds available stock (Y)" if insufficient stock
func isStockAvailable(db DBExecutor, productID string, quantity int) error {
	var available int
	// Query current stock quantity for product
	err := db.QueryRow(`
		SELECT stock_quantity
		FROM products
		WHERE product_id = ?
	`, productID).Scan(&available)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("product not found")
		}
		return fmt.Errorf("error checking stock: %w", err)
	}

	// Validate requested quantity against available stock
	if quantity > available {
		return fmt.Errorf("requested quantity (%d) exceeds available stock (%d)", quantity, available)
	}
	return nil
}

// CreateCart creates a new cart for a user or returns existing cart for authenticated users.
//
// This function handles both authenticated and guest cart creation:
//   - For authenticated users (UserID provided): Checks for existing cart and returns it if found
//   - For guest users (UserID nil): Creates new cart with generated ID
//   - Prevents duplicate carts per user
//
// Parameters:
//   - req: CreateCartRequest DTO containing optional UserID pointer
//
// Returns:
//   - string: The cart_id (existing or newly created)
//   - error: Database error if creation fails, nil on success
func CreateCart(db DBExecutor, req dtos.CreateCartRequest) (string, error) {
	// Generate unique cart ID
	cartID, _ := shortid.Generate()

	// For authenticated users, check if cart already exists
	if req.UserID != nil {
		cartID, err := GetUserCart(db, *req.UserID)
		if err != nil {
			return "", err
		}
		// Return existing cart if found
		if cartID != "" {
			return cartID, nil
		}
	}
	// Create new cart record
	_, err := db.Exec(`
        INSERT INTO cart(cart_id, user_id)
        VALUES (?, ?)
    `, cartID, req.UserID)
	return cartID, err
}

// GetUserCart retrieves the cart_id for a specific user.
//
// This function checks if an authenticated user already has a cart.
// Used before creating a new cart to prevent duplicates.
//
// Parameters:
//   - userID: The user_id to search for
//
// Returns:
//   - string: The cart_id if found, empty string if no cart exists
//   - error: Database error if query fails (sql.ErrNoRows returns empty string, not error)
func GetUserCart(db DBExecutor, userID string) (string, error) {
	var cartID string
	// Query for existing cart by user_id
	err := db.QueryRow(`
        SELECT cart_id FROM cart WHERE user_id = ?
    `, userID).Scan(&cartID)

	if err == sql.ErrNoRows {
		// No cart yet for this user - not an error
		return "", nil
	}

	return cartID, err
}

// GetCartItems retrieves all items in a cart with full product details.
//
// This function validates cart existence, then fetches all cart items and enriches
// them with complete product information by calling GetProductByID for each item.
//
// Parameters:
//   - cartID: The cart_id to retrieve items for
//
// Returns:
//   - []dtos.CartItem: Array of cart items with product details and quantities
//   - error: "cart not found" if cart doesn't exist, or database error
//
// CartItem Structure:
//   - Product: Full product object with details, pricing, images
//   - Quantity: Number of units in cart
func GetCartItems(db DBExecutor, cartID string) ([]dtos.CartItem, error) {
	// Validate cart exists before retrieving items
	err := isCartThere(db, cartID)
	if err != nil {
		return nil, err
	}

	// Query all items in cart
	rows, err := db.Query(`
		SELECT c.product_id, c.quantity, c.variation_sku
		FROM cart_items c
		WHERE c.cart_id = ?
	`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dtos.CartItem
	// Iterate through cart items and enrich with product details
	for rows.Next() {
		var productID string
		var quantity int
		var variationSKU *string
		if err := rows.Scan(&productID, &quantity, &variationSKU); err != nil {
			return nil, err
		}

		// Fetch complete product details for this item
		product, err := GetProductByID(db, productID)
		if err != nil {
			return nil, err
		}
		isVariation := false
		// if variationSKU != nil  get additional price
		if variationSKU != nil {
			// Fetch additional price for the variation
			additionalPrice, variationName, err := GetVariationPrice(db, *variationSKU)
			if err != nil {
				return nil, err
			}
			product.Price += additionalPrice
			// add - plus variation name to product name
			product.Name += " - " + variationName
			isVariation = true

		}

		// Build enriched CartItem DTO with full product details
		item := dtos.CartItem{
			Product:      *product, // Complete product object
			Quantity:     quantity,
			IsVariant:    isVariation,
			VariationSKU: variationSKU,
		}
		items = append(items, item)
	}

	return items, nil
}

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
func UpdateCartItem(db DBExecutor, cartID string, productID string, quantity int) error {
	// Validate cart exists
	err := isCartThere(db, cartID)
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
func DeleteCartItem(db DBExecutor, cartID, productID string) error {
	// Validate cart exists
	err := isCartThere(db, cartID)
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
