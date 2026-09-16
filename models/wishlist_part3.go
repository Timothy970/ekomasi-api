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
	"fmt"
)

// This function performs a batch update to migrate image URLs from
// the old storage location to the new one. This is a utility function
// for data migration.
//
// Returns:
//   - int64: Number of rows affected (categories updated)
//   - error: Database error or nil on success
func UpdateImageURLs(db DBExecutor) (int64, error) {
	// Batch update category image URLs
	query := `
        UPDATE categories
        SET image = REPLACE(
            image,
            'https://storage.googleapis.com/m_tickets',
            'https://bucket.emalify.com'
        )
        WHERE image LIKE 'https://storage.googleapis.com/m_tickets%';`

	res, err := db.Exec(query)
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
func GetMyWishlistItems(db DBExecutor, userID string) (dtos.AllWishlist, error) {
	var wishlistID string
	var Name string
	var IsPublic bool

	// Get user's first wishlist (users typically have one primary wishlist)
	query := `SELECT wishlist_id, name, is_public FROM wishlists WHERE user_id = ? LIMIT 1`
	err := db.QueryRow(query, userID).Scan(&wishlistID, &Name, &IsPublic)

	if err != nil {
		if err == sql.ErrNoRows {
			return dtos.AllWishlist{}, fmt.Errorf("no wishlist found for user")
		}
		return dtos.AllWishlist{}, err
	}

	// Fetch all products in the wishlist
	products, err := fetchProductsForWishlist(db, wishlistID)
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
