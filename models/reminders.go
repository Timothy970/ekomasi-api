package models

import (
	"adenzo_backend/dtos"
)

// Get users with stale cart items
func GetUsersWithStaleCart(days int) ([]dtos.User, error) {
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

// Get users with stale wishlist items
func GetUsersWithStaleWishlist(days int) ([]dtos.User, error) {
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
