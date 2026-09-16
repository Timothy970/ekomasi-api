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
	"ekomasi_backend/dtos"
	"errors"
	"fmt"

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
func isCartThere(db DBExecutor, id string, tenantID int) error {
	// Check if cart record exists in cart table
	exists, err := RecordExists(db, "cart", "cart_id = ? AND tenant_id = ?", id, tenantID)
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
func InsertCartItem(db DBExecutor, cartID string, productID string, quantity int, variationSKU *string, tenantID int) error {
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
func CreateCart(db DBExecutor, req dtos.CreateCartRequest, tenantID int) (string, error) {
	// Generate unique cart ID
	cartID, _ := shortid.Generate()

	// For authenticated users, check if cart already exists
	if req.UserID != nil {
		cartID, err := GetUserCart(db, *req.UserID, tenantID)
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
        INSERT INTO cart(cart_id, user_id, tenant_id)
        VALUES (?, ?, ?)
    `, cartID, req.UserID, tenantID)
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
func GetUserCart(db DBExecutor, userID string, tenantID int) (string, error) {
	var cartID string
	// Query for existing cart by user_id
	err := db.QueryRow(`
        SELECT cart_id FROM cart WHERE user_id = ? AND tenant_id = ?
    `, userID, tenantID).Scan(&cartID)

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
func GetCartItems(db DBExecutor, cartID string, tenantID int) ([]dtos.CartItem, error) {
	// Validate cart exists before retrieving items
	err := isCartThere(db, cartID, tenantID)
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
