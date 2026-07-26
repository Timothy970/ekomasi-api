// Package models provides data access functions for reminder and re-engagement operations.
//
// This file handles user re-engagement queries for:
//   - Abandoned cart reminders (users with stale cart items)
//   - Wishlist reminders (users with inactive wishlists)
//   - Time-based filtering (items not updated in X days)
//
// These functions support marketing automation and customer retention by identifying
// users who may benefit from reminder notifications about items they've saved.
package models

import (
	"ekomasi_backend/dtos"
)

// GetUsersWithStaleCart retrieves users with abandoned cart items.
//
// This function identifies users whose cart items haven't been updated in the
// specified number of days, indicating potential abandoned carts that may benefit
// from reminder notifications or re-engagement campaigns.
//
// Parameters:
//   - days: int - Number of days of inactivity to qualify as "stale"
//     (e.g., 7 for carts inactive for a week)
//
// Returns:
//   - []dtos.User: Array of users with stale carts containing:
//   - ID: User ID
//   - Email: User's email address for notifications
//   - Phone: User's phone number for SMS notifications
//   - error: Database error or nil on success
func GetUsersWithStaleCart(days int) ([]dtos.User, error) {
	// Query users with cart items older than specified days
	// Uses INTERVAL to calculate date threshold (NOW() - X days)
	rows, err := DB.Query(`
		SELECT DISTINCT u.user_id, u.email, u.phone_number
		FROM users u
		JOIN cart c ON u.user_id = c.user_id
		JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE ci.updated_at < NOW() - INTERVAL ? DAY`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect users with stale carts (DISTINCT ensures one entry per user)
	var users []dtos.User
	for rows.Next() {
		var u dtos.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Phone); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// GetUsersWithStaleWishlist retrieves users with inactive wishlist items.
//
// This function identifies users whose wishlists haven't been updated in the
// specified number of days, allowing for targeted re-engagement campaigns to
// remind users about saved items or notify them of price drops/promotions.
//
// Parameters:
//   - days: int - Number of days of inactivity to qualify as "stale"
//     (e.g., 30 for wishlists inactive for a month)
//
// Returns:
//   - []dtos.User: Array of users with stale wishlists containing:
//   - ID: User ID
//   - Email: User's email address for notifications
//   - Phone: User's phone number for SMS notifications
//   - error: Database error or nil on success
func GetUsersWithStaleWishlist(days int) ([]dtos.User, error) {
	// Query users with wishlists older than specified days
	// Uses INTERVAL to calculate date threshold (NOW() - X days)
	rows, err := DB.Query(`
		SELECT DISTINCT u.user_id, u.email, u.phone_number
		FROM users u
		JOIN wishlists w ON u.user_id = w.user_id
		JOIN wishlist_items wi ON w.wishlist_id = wi.wishlist_id
		WHERE w.last_updated_at < NOW() - INTERVAL ? DAY`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect users with stale wishlists (DISTINCT ensures one entry per user)
	var users []dtos.User
	for rows.Next() {
		var u dtos.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Phone); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
