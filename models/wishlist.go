// Package models provides data access functions for the Ekomasi e-commerce backend.
//
// This file contains wishlist management operations including:
//   - Wishlist CRUD (create, read, update, delete)
//   - Wishlist item management (add/remove products)
//   - User wishlist retrieval with pagination
//   - Public/private wishlist support
//   - Product association with wishlists
//
// Each user can have multiple wishlists, and each wishlist can contain
// multiple products. Wishlists can be public or private.
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"fmt"

	"github.com/teris-io/shortid"
)

var nowishlist = "Wishlist not found"
var fetchwishlist = "wishlist_id = ?"

// CreateWishList creates a new wishlist for a user.
//
// This function generates a unique wishlist ID and creates a wishlist
// with the specified name and visibility settings.
//
// Parameters:
//   - body: dtos.CreateWishlist containing:
//   - Name: Wishlist name (e.g., "Holiday Gifts", "Birthday Wishlist")
//   - IsPublic: Whether the wishlist is publicly visible
//   - userID: string - The user creating the wishlist
//
// Returns:
//   - *dtos.Wishlist: The created wishlist with ID, name, and visibility
//   - error: Database error or nil on success
func CreateWishList(db DBExecutor, body dtos.CreateWishlist, userID string) (*dtos.Wishlist, error) {
	// Generate unique wishlist ID
	wishlistID, _ := shortid.Generate()

	// Insert wishlist record
	query := `INSERT INTO wishlists (wishlist_id, user_id, name, is_public) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, wishlistID, userID, body.Name, body.IsPublic)
	if err != nil {
		return nil, err
	}

	return &dtos.Wishlist{WishlistID: wishlistID, Name: body.Name, IsPublic: body.IsPublic}, nil
}

// GetAllUserWishList retrieves wishlists for a user with optional filtering and pagination.
//
// This function can retrieve:
// - All wishlists for a user (with pagination)
// - A specific wishlist by ID
//
// Each wishlist includes its associated products with images.
//
// Parameters:
//   - userID: string - The user whose wishlists to retrieve (required)
//   - wishlistID: string - Optional specific wishlist ID (empty for all wishlists)
//   - limit: int - Number of wishlists per page (ignored if wishlistID provided)
//   - page: int - Page number for pagination (ignored if wishlistID provided)
//
// Returns:
//   - []dtos.AllWishlist: Array of wishlists with products
//   - *dtos.PaginationMeta: Pagination metadata (nil if specific wishlist requested)
//   - error: "userID is required", "no wishlist found", database error, or nil on success
func GetAllUserWishList(db DBExecutor, userID, wishlistID string, limit, page int) ([]dtos.AllWishlist, *dtos.PaginationMeta, error) {
	// Validate user ID is provided
	if userID == "" {
		return nil, nil, errors.New("userID is required")
	}

	// Build query based on whether specific wishlist or all wishlists requested
	query, args := buildWishlistQuery(userID, wishlistID, limit, page)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Process each wishlist
	var lists []dtos.AllWishlist
	for rows.Next() {
		wishlist, err := scanWishlistRow(rows)
		if err != nil {
			return nil, nil, err
		}

		// Fetch products for this wishlist
		products, err := fetchProductsForWishlist(db, wishlist.WishlistID)
		if err != nil {
			return nil, nil, err
		}

		wishlist.Products = products
		lists = append(lists, wishlist)
	}

	// Return error if specific wishlist was requested but not found
	if wishlistID != "" && len(lists) == 0 {
		return nil, nil, fmt.Errorf("no wishlist found with ID: %s", wishlistID)
	}

	// Skip pagination for specific wishlist requests
	if wishlistID != "" {
		return lists, nil, nil
	}

	// Build pagination metadata for all wishlists query
	meta, err := buildPaginationMeta(db, userID, limit, page)
	if err != nil {
		return nil, nil, err
	}

	return lists, meta, nil
}

// buildWishlistQuery constructs the SQL query for retrieving wishlists.
//
// This helper function builds different queries based on whether a specific
// wishlist is requested or all wishlists with pagination.
//
// Parameters:
//   - userID: string - User to filter wishlists by
//   - wishlistID: string - Optional specific wishlist ID
//   - limit: int - Number of items per page
//   - page: int - Page number
//
// Returns:
//   - string: SQL query
//   - []any: Query arguments
func buildWishlistQuery(userID, wishlistID string, limit, page int) (string, []any) {
	query := `SELECT wishlist_id, name, is_public FROM wishlists WHERE user_id = ?`
	args := []any{userID}

	// Add specific wishlist filter if provided
	if wishlistID != "" {
		query += ` AND wishlist_id = ?`
		args = append(args, wishlistID)
	} else {
		// Add pagination for all wishlists query
		offset := (page - 1) * limit
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	return query, args
}

// scanWishlistRow scans a wishlist row from database results.
//
// Parameters:
//   - rows: *sql.Rows - Database result rows
//
// Returns:
//   - dtos.AllWishlist: Scanned wishlist data
//   - error: Scan error or nil on success
func scanWishlistRow(rows *sql.Rows) (dtos.AllWishlist, error) {
	var w dtos.AllWishlist
	err := rows.Scan(&w.WishlistID, &w.Name, &w.IsPublic)
	return w, err
}

// fetchProductsForWishlist retrieves all products in a wishlist.
//
// This function fetches complete product information including images
// for all products associated with the specified wishlist.
//
// Parameters:
//   - wishlistID: string - The wishlist ID to fetch products for
//
// Returns:
//   - []dtos.Product: Array of products with images
//   - error: Database error or nil on success
func fetchProductsForWishlist(db DBExecutor, wishlistID string) ([]dtos.Product, error) {
	// Join wishlist_items with products to get product details
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id, 
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
		FROM wishlist_items wi
		JOIN products p ON wi.product_id = p.product_id
		WHERE wi.wishlist_id = ?
	`

	rows, err := db.Query(query, wishlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process each product
	var products []dtos.Product
	for rows.Next() {
		var p dtos.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		); err != nil {
			return nil, err
		}

		// Fetch product images
		images, err := fetchProductImages(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Images = images
		products = append(products, p)
	}
	return products, nil
}

// buildPaginationMeta constructs pagination metadata for wishlist listings.
//
// Parameters:
//   - userID: string - User to count wishlists for
//   - limit: int - Items per page
//   - page: int - Current page number
//
// Returns:
//   - *dtos.PaginationMeta: Pagination metadata (page, size, total, has prev/next)
//   - error: Database error or nil on success
func buildPaginationMeta(db DBExecutor, userID string, limit, page int) (*dtos.PaginationMeta, error) {
	// Count total wishlists for user
	var totalItems int
	err := db.QueryRow(`SELECT COUNT(*) FROM wishlists WHERE user_id = ?`, userID).Scan(&totalItems)
	if err != nil {
		return nil, err
	}

	// Calculate pagination values
	totalPages := (totalItems + limit - 1) / limit // Ceiling division
	return &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}, nil
}

// GetOrCreateWishlist retrieves existing wishlist or creates a new one.
//
// This function checks if the user has an existing wishlist and returns
// its ID, or creates a new wishlist if none exists.
//
// Parameters:
//   - userID: string - The user to get/create wishlist for
//   - wishlistName: string - Name for new wishlist (only used if creating)
//
// Returns:
//   - string: Wishlist ID (existing or newly created)
//   - error: Database error or nil on success
func GetOrCreateWishlist(db DBExecutor, userID, wishlistName string) (string, error) {
	// Check if wishlist exists for user
	exists, err := RecordExists(db, "wishlists", "user_id = ?", userID)
	if err != nil {
		return "", err
	}

	if exists {
		// Return existing wishlist ID
		return GetWishlistByUserID(db, userID)
	}

	// Create new wishlist
	return CreateNewWishList(db, wishlistName, userID)
}
