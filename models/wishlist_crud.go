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
	"ekomasi_backend/dtos"
	"fmt"

	"github.com/teris-io/shortid"
)

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
func GetWishlistByUserID(db DBExecutor, userID string) (string, error) {
	var wishlistID string
	// Get first wishlist for user
	query := `SELECT wishlist_id FROM wishlists WHERE user_id = ? LIMIT 1`
	err := db.QueryRow(query, userID).Scan(&wishlistID)
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
func CreateNewWishList(db DBExecutor, Name, userID string) (string, error) {
	// Generate unique wishlist ID
	wishlistID, _ := shortid.Generate()

	// Insert wishlist record (default public)
	query := `INSERT INTO wishlists (wishlist_id, user_id, name, is_public) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, wishlistID, userID, Name, 1)
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
func CreateWishListItem(db DBExecutor, wishlistID, productID, userID string) error {
	// Validate product exists
	err := IsProductThere(db, productID)
	if err != nil {
		return err
	}

	// Validate wishlist exists
	exists, err := RecordExists(db, "wishlists", fetchwishlist, wishlistID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nowishlist)
	}

	// Check if product already in wishlist (prevent duplicates)
	exists, err = RecordExists(db, "wishlist_items", "wishlist_id = ? AND product_id = ?", wishlistID, productID)
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
	_, err = db.Exec(query, itemID, wishlistID, productID)
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
func RemoveWishlistItem(db DBExecutor, wishlistID, productID, userID string) error {
	// Validate product exists
	exists, err := RecordExists(db, "products", "product_id = ?", productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not found")
	}

	// Validate wishlist exists
	exists, err = RecordExists(db, "wishlists", fetchwishlist, wishlistID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nowishlist)
	}

	// Validate product is in wishlist
	exists, err = RecordExists(db, "wishlist_items", "wishlist_id = ? AND product_id = ?", wishlistID, productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not in wishlist")
	}

	// Remove wishlist item
	query := `DELETE FROM wishlist_items WHERE product_id = ?`
	_, err = db.Exec(query, productID)
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
func GetWishlistByID(db DBExecutor, wishlistID string) ([]dtos.AllWishlist, error) {
	// Retrieve wishlist by ID
	query := `SELECT wishlist_id, name, is_public FROM wishlists WHERE wishlist_id = ?`
	rows, err := db.Query(query, wishlistID)
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
		productRows, err := db.Query(productQuery, w.WishlistID)
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
			p.Images, err = fetchProductImages(db, p.ID)
			if err != nil {
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
func DeleteWishList(db DBExecutor, wishlistID, userID string) error {
	// Validate wishlist exists
	exists, err := RecordExists(db, "wishlists", fetchwishlist, wishlistID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nowishlist)
	}

	// Delete wishlist (cascade should remove wishlist_items)
	_, err = db.Exec("DELETE FROM wishlists WHERE wishlist_id = ? AND user_id = ?", wishlistID, userID)
	return err
}

// UpdateImageURLs updates category image URLs in bulk.
//
