// Package models provides data access functions for the Adenzo e-commerce backend.
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
	"adenzo_backend/dtos"
	"database/sql"
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
func CreateWishList(body dtos.CreateWishlist, userID string) (*dtos.Wishlist, error) {
	// Generate unique wishlist ID
	wishlistID, _ := shortid.Generate()

	// Insert wishlist record
	query := `INSERT INTO wishlists (wishlist_id, user_id, name, is_public) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, wishlistID, userID, body.Name, body.IsPublic)
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
func GetAllUserWishList(userID, wishlistID string, limit, page int) ([]dtos.AllWishlist, *dtos.PaginationMeta, error) {
	// Validate user ID is provided
	if userID == "" {
		return nil, nil, errors.New("userID is required")
	}

	// Build query based on whether specific wishlist or all wishlists requested
	query, args := buildWishlistQuery(userID, wishlistID, limit, page)

	rows, err := DB.Query(query, args...)
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
		products, err := fetchProductsForWishlist(wishlist.WishlistID)
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
	meta, err := buildPaginationMeta(userID, limit, page)
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
//   - []interface{}: Query arguments
func buildWishlistQuery(userID, wishlistID string, limit, page int) (string, []interface{}) {
	query := `SELECT wishlist_id, name, is_public FROM wishlists WHERE user_id = ?`
	args := []interface{}{userID}

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
func fetchProductsForWishlist(wishlistID string) ([]dtos.Product, error) {
	// Join wishlist_items with products to get product details
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id, 
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
		FROM wishlist_items wi
		JOIN products p ON wi.product_id = p.product_id
		WHERE wi.wishlist_id = ?
	`

	rows, err := DB.Query(query, wishlistID)
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
		images, err := fetchProductImages(p.ID)
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
func buildPaginationMeta(userID string, limit, page int) (*dtos.PaginationMeta, error) {
	// Count total wishlists for user
	var totalItems int
	err := DB.QueryRow(`SELECT COUNT(*) FROM wishlists WHERE user_id = ?`, userID).Scan(&totalItems)
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
func GetOrCreateWishlist(userID, wishlistName string) (string, error) {
	// Check if wishlist exists for user
	exists, err := RecordExists("wishlists", "user_id = ?", userID)
	if err != nil {
		return "", err
	}

	if exists {
		// Return existing wishlist ID
		return GetWishlistByUserID(userID)
	}

	// Create new wishlist
	return CreateNewWishList(wishlistName, userID)
}

// GetWishlistByUserID retrieves the first wishlist for a user.
//
// This function returns the first wishlist ID found for the user.
// Useful for systems where users typically have one primary wishlist.
//
// Parameters:
//   - userID: string - The user to get wishlist for
//
// Returns:
//   - string: Wishlist ID
//   - error: Database error, "no rows" if no wishlist, or nil on success
func GetWishlistByUserID(userID string) (string, error) {
	var wishlistID string
	// Get first wishlist for user
	query := `SELECT wishlist_id FROM wishlists WHERE user_id = ? LIMIT 1`
	err := DB.QueryRow(query, userID).Scan(&wishlistID)
	if err != nil {
		return "", err
	}
	return wishlistID, nil
}

// CreateNewWishList creates a new wishlist with the specified name.
//
// This function creates a public wishlist by default (is_public = 1).
//
// Parameters:
//   - Name: string - Wishlist name
//   - userID: string - The user creating the wishlist
//
// Returns:
//   - string: Generated wishlist ID
//   - error: Database error or nil on success
func CreateNewWishList(Name, userID string) (string, error) {
	// Generate unique wishlist ID
	wishlistID, _ := shortid.Generate()

	// Insert wishlist record (default public)
	query := `INSERT INTO wishlists (wishlist_id, user_id, name, is_public) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, wishlistID, userID, Name, 1)
	if err != nil {
		return "", err
	}

	return wishlistID, nil
}

// CreateWishListItem adds a product to a wishlist.
//
// This function validates:
// - Product exists
// - Wishlist exists
// - Product is not already in the wishlist (prevents duplicates)
//
// Parameters:
//   - wishlistID: string - The wishlist to add product to
//   - productID: string - The product to add
//   - userID: string - The user adding the item (for authorization)
//
// Returns:
//   - error: "product not found", "wishlist not found", "product already in wishlist",
//     database error, or nil on success
func CreateWishListItem(wishlistID, productID, userID string) error {
	// Validate product exists
	err := IsProductThere(productID)
	if err != nil {
		return err
	}

	// Validate wishlist exists
	exists, err := RecordExists("wishlists", fetchwishlist, wishlistID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nowishlist)
	}

	// Check if product already in wishlist (prevent duplicates)
	exists, err = RecordExists("wishlist_items", "wishlist_id = ? AND product_id = ?", wishlistID, productID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("product already in wishlist")
	}

	// Generate unique wishlist item ID
	itemID, _ := shortid.Generate()

	// Insert wishlist item
	query := `INSERT INTO wishlist_items (wishlist_item_id, wishlist_id, product_id) VALUES (?, ?, ?)`
	_, err = DB.Exec(query, itemID, wishlistID, productID)
	if err != nil {
		return err
	}
	return nil
}

// RemoveWishlistItem removes a product from a wishlist.
//
// This function validates:
// - Product exists
// - Wishlist exists
// - Product is in the wishlist
//
// Parameters:
//   - wishlistID: string - The wishlist to remove product from
//   - productID: string - The product to remove
//   - userID: string - The user removing the item (for authorization)
//
// Returns:
//   - error: "product not found", "wishlist not found", "product not in wishlist",
//     database error, or nil on success
func RemoveWishlistItem(wishlistID, productID, userID string) error {
	// Validate product exists
	exists, err := RecordExists("products", "product_id = ?", productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not found")
	}

	// Validate wishlist exists
	exists, err = RecordExists("wishlists", fetchwishlist, wishlistID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nowishlist)
	}

	// Validate product is in wishlist
	exists, err = RecordExists("wishlist_items", "wishlist_id = ? AND product_id = ?", wishlistID, productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not in wishlist")
	}

	// Remove wishlist item
	query := `DELETE FROM wishlist_items WHERE product_id = ?`
	_, err = DB.Exec(query, productID)
	if err != nil {
		return err
	}
	return nil
}

// GetWishlistByID retrieves a specific wishlist with its products.
//
// This function fetches the wishlist details and all associated products.
// Returns nil if wishlist not found.
//
// Parameters:
//   - wishlistID: string - The wishlist ID to retrieve
//
// Returns:
//   - []dtos.AllWishlist: Array containing the wishlist (single element), or nil if not found
//   - error: Database error or nil on success
func GetWishlistByID(wishlistID string) ([]dtos.AllWishlist, error) {
	// Retrieve wishlist by ID
	query := `SELECT wishlist_id, name, is_public FROM wishlists WHERE wishlist_id = ?`
	rows, err := DB.Query(query, wishlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []dtos.AllWishlist
	for rows.Next() {
		var w dtos.AllWishlist
		if err := rows.Scan(&w.WishlistID, &w.Name, &w.IsPublic); err != nil {
			return nil, err
		}

		// Fetch associated products for this wishlist
		productQuery := `
			SELECT 
				p.product_id, p.name, p.description, p.sku, p.price, p.category_id, 
				p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
			FROM wishlist_items wi
			JOIN products p ON wi.product_id = p.product_id
			WHERE wi.wishlist_id = ?
		`
		productRows, err := DB.Query(productQuery, w.WishlistID)
		if err != nil {
			return nil, err
		}

		// Process each product
		var products []dtos.Product
		for productRows.Next() {
			var p dtos.Product
			if err := productRows.Scan(
				&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
				&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
			); err != nil {
				productRows.Close()
				return nil, err
			}
			products = append(products, p)
		}
		productRows.Close()

		w.Products = products
		lists = append(lists, w)
	}

	// Return nil if wishlist not found
	if len(lists) == 0 {
		return nil, nil
	}
	return lists, nil
}

// DeleteWishList permanently removes a wishlist and its items.
//
// Parameters:
//   - wishlistID: string - The wishlist ID to delete
//   - userID: string - The user deleting the wishlist (for authorization)
//
// Returns:
//   - error: "wishlist not found", database error, or nil on success
func DeleteWishList(wishlistID, userID string) error {
	// Validate wishlist exists
	exists, err := RecordExists("wishlists", fetchwishlist, wishlistID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nowishlist)
	}

	// Delete wishlist (cascade should remove wishlist_items)
	_, err = DB.Exec("DELETE FROM wishlists WHERE wishlist_id = ? AND user_id = ?", wishlistID, userID)
	return err
}

// UpdateImageURLs updates category image URLs in bulk.
//
// This function performs a batch update to migrate image URLs from
// the old storage location to the new one. This is a utility function
// for data migration.
//
// Returns:
//   - int64: Number of rows affected (categories updated)
//   - error: Database error or nil on success
func UpdateImageURLs() (int64, error) {
	// Batch update category image URLs
	query := `
        UPDATE categories
        SET image = REPLACE(
            image,
            'https://storage.googleapis.com/m_tickets',
            'https://bucket.emalify.com'
        )
        WHERE image LIKE 'https://storage.googleapis.com/m_tickets%';`

	res, err := DB.Exec(query)
	if err != nil {
		return 0, err
	}

	// Return number of categories updated
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// GetMyWishlistItems retrieves a user's primary wishlist with products.
//
// This function returns the first wishlist for a user with all associated
// products and their images. Useful for systems where users have one main
// wishlist.
//
// Parameters:
//   - userID: string - The user whose wishlist to retrieve
//
// Returns:
//   - dtos.AllWishlist: Wishlist with products
//   - error: "no wishlist found for user", database error, or nil on success
func GetMyWishlistItems(userID string) (dtos.AllWishlist, error) {
	var wishlistID string
	var Name string
	var IsPublic bool

	// Get user's first wishlist (users typically have one primary wishlist)
	query := `SELECT wishlist_id, name, is_public FROM wishlists WHERE user_id = ? LIMIT 1`
	err := DB.QueryRow(query, userID).Scan(&wishlistID, &Name, &IsPublic)

	if err != nil {
		if err == sql.ErrNoRows {
			return dtos.AllWishlist{}, fmt.Errorf("no wishlist found for user")
		}
		return dtos.AllWishlist{}, err
	}

	// Fetch all products in the wishlist
	products, err := fetchProductsForWishlist(wishlistID)
	if err != nil {
		return dtos.AllWishlist{}, err
	}

	// Build complete wishlist data
	wishlist := dtos.AllWishlist{
		WishlistID: wishlistID,
		Name:       Name,
		IsPublic:   IsPublic,
		Products:   products,
	}
	return wishlist, nil

}
