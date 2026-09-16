// Package models provides data access functions for the Ekomasi e-commerce platform.
//
// This file contains functions for managing users and related data:
//   - User CRUD operations (activate, deactivate, delete)
//   - User address management (create, retrieve, update, delete)
//   - User listing with pagination, search, and role filtering
//   - Token-based user retrieval
//   - Purchase history and wishlist category analysis
//   - Product recommendations based on user behavior
//   - Newsletter subscription management
package models

import (
	"ekomasi_backend/dtos"
	"fmt"
	"math"
	"strings"

	"github.com/teris-io/shortid"
)

// buildPagination creates pagination metadata from query results.
//
// This helper function calculates page numbers and navigation flags.
//
// Parameters:
//   - limit: int - Items per page
//   - offset: int - Current offset
//   - totalItems: int - Total matching items
//
// Returns:
//   - *dtos.PaginationMeta: Pagination metadata
func buildPagination(limit, offset, totalItems int) *dtos.PaginationMeta {
	// Calculate current page number
	page := (offset / limit) + 1
	// Calculate total pages (ceiling division)
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	return &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
}

// GetPurchasedCategories retrieves distinct product categories from user's purchase history.
//
// This function analyzes a user's orders to identify which product categories
// they have purchased from, useful for personalized recommendations.
//
// Parameters:
//   - userID: string - The user ID to analyze
//
// Returns:
//   - []string: Array of distinct category IDs from purchased products
//   - error: Database error or nil on success
func GetPurchasedCategories(db DBExecutor, userID string) ([]string, error) {
	// Query distinct categories from user's order history
	rows, err := db.Query(`
		SELECT DISTINCT p.category_id
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		JOIN products p ON oi.product_id = p.product_id
		WHERE o.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// GetWishlistCategories retrieves distinct product categories from user's wishlist.
//
// This function analyzes a user's wishlist to identify which product categories
// they're interested in, useful for personalized recommendations.
//
// Parameters:
//   - userID: string - The user ID to analyze
//
// Returns:
//   - []string: Array of distinct category IDs from wishlist products
//   - error: Database error or nil on success
func GetWishlistCategories(db DBExecutor, userID string) ([]string, error) {
	// Query distinct categories from user's wishlist
	rows, err := db.Query(`
		SELECT DISTINCT p.category_id
		FROM wishlists w
		JOIN wishlist_items wi ON w.wishlist_id = wi.wishlist_id
		JOIN products p ON wi.product_id = p.product_id
		WHERE w.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// GetProductsByCategories retrieves paginated products from specified categories.
//
// This function is useful for showing related products based on user's purchase
// history or wishlist categories. Returns enriched product data including
// images, warranties, features, variants, and tax information.
//
// Parameters:
//   - categories: []string - Array of category IDs to fetch products from
//   - page: int - Page number (1-based)
//   - size: int - Number of products per page
//
// Returns:
//   - []dtos.Product: Array of enriched products with images, warranties, features, variants, tax
//   - dtos.PaginationMeta: Pagination info (page, size, totals, navigation flags)
//   - error: Database error or nil on success (returns empty result if no categories)
func GetProductsByCategories(db DBExecutor, categories []string, page, size int) ([]dtos.Product, dtos.PaginationMeta, error) {
	// Return empty result if no categories provided
	if len(categories) == 0 {
		return []dtos.Product{}, dtos.PaginationMeta{}, nil
	}

	// Calculate offset for pagination
	offset := (page - 1) * size

	// Build dynamic IN clause with placeholders
	placeholders := strings.Repeat(",?", len(categories)-1)
	query := `
		SELECT SQL_CALC_FOUND_ROWS
		       p.product_id, p.name, p.description, p.sku, p.price, 
		       p.category_id, p.stock_quantity, p.search_vector, 
		       p.created_at, p.last_updated_at
		FROM products p
		WHERE p.category_id IN (?` + placeholders + `)
		LIMIT ? OFFSET ?`

	// Build arguments array for query
	args := make([]interface{}, len(categories)+2)
	for i, v := range categories {
		args[i] = v
	}
	args[len(categories)] = size
	args[len(categories)+1] = offset

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	// Process each product and enrich with related data
	var products []dtos.Product
	for rows.Next() {
		var p dtos.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price,
			&p.CategoryID, &p.StockQuantity, &p.SearchVector,
			&p.CreatedAt, &p.LastUpdated,
		); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}

		// Fetch product images (ignores errors)
		p.Images, _ = fetchProductImages(db, p.ID)

		// Fetch product warranties
		warranties, err := FetchProductWarranties(db, p.ID)
		if err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		p.Warranty = &warranties

		// Fetch product features
		features, err := fetchProductFeatures(db, p.ID)
		if err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		p.Features = features

		// Fetch product variants
		variants, err := getProductVariants(db, p.ID)
		if err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		p.ProductVariants = variants

		// Fetch product tax information
		tax, err := fetchProductTax(db, p.ID)
		if err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		p.Tax = &tax

		products = append(products, p)
	}

	// Get total count using FOUND_ROWS()
	var totalItems int
	if err := db.QueryRow(`SELECT FOUND_ROWS()`).Scan(&totalItems); err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Build pagination metadata
	totalPages := (totalItems + size - 1) / size // Ceiling division
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return products, meta, nil
}

// isSubscriberThere validates that an email is not already subscribed.
//
// This helper function prevents duplicate newsletter subscriptions.
//
// Parameters:
//   - email: string - The email address to check
//
// Returns:
//   - error: "email already subscribed" if exists, database error, or nil if unique
func isSubscriberThere(db DBExecutor, email string) error {
	// Check if email already subscribed
	exists, err := RecordExists(db, "subscribers", "email = ?", email)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("email already subscribed")
	}
	return nil
}

// CreateSubscribers creates a new newsletter subscriber.
//
// This function validates email uniqueness and creates a subscriber record.
//
// Parameters:
//   - email: string - The email address to subscribe
//
// Returns:
//   - error: "email already subscribed", database error, or nil on success
func CreateSubscribers(db DBExecutor, email string) error {
	// Validate email not already subscribed
	err := isSubscriberThere(db, email)
	if err != nil {
		return err
	}

	// Generate unique subscriber ID
	subscriberID, _ := shortid.Generate()

	// Insert new subscriber
	_, err = db.Exec(`
		INSERT INTO subscribers (subscriber_id, email)
		VALUES (?, ?)`,
		subscriberID, email,
	)
	if err != nil {
		return err
	}
	return nil
}
