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

	"github.com/teris-io/shortid"
)

// getAddress is a constant query condition for address lookups
var getAddress = "user_id = ? AND address_id = ?"

// DeleteUserByID permanently removes a user from the system.
//
// This function validates user existence before deletion.
//
// Parameters:
//   - userID: string - The unique user ID to delete
//
// Returns:
//   - error: "user not found", database error, or nil on success
func DeleteUserByID(db DBExecutor, userID string) error {
	// Validate user exists
	err := isUserThere(db, userID)
	if err != nil {
		return err
	}

	// Delete user record
	_, err = db.Exec(`
		DELETE FROM users
		WHERE user_id = ?
	`, userID)

	return err
}

// getUserStatus retrieves the current status of a user.
//
// This helper function checks whether a user is active or inactive.
//
// Parameters:
//   - userID: string - The unique user ID to check
//
// Returns:
//   - string: User status ("active" or "inactive")
func getUserStatus(db DBExecutor, userID string) string {
	var status string
	// Query user status (ignores errors, returns empty string if not found)
	_ = db.QueryRow(`SELECT status FROM users WHERE user_id = ?`, userID).Scan(&status)
	return status
}

// ActivateUserByID activates an inactive user account.
//
// This function validates user existence and checks if already active
// before changing status to "active".
//
// Parameters:
//   - userID: string - The unique user ID to activate
//
// Returns:
//   - error: "user not found", "user is already active", database error, or nil on success
func ActivateUserByID(db DBExecutor, userID string) error {
	// Validate user exists
	err := isUserThere(db, userID)
	if err != nil {
		return err
	}

	// Check current status
	status := getUserStatus(db, userID)
	if status == "active" {
		return fmt.Errorf("user is already active")
	}

	// Activate user
	_, err = db.Exec(`
		UPDATE users
		SET status = "active"
		WHERE user_id = ?
	`, userID)

	return err
}

// DeactivateUserByID deactivates an active user account.
//
// This function validates user existence and checks if already inactive
// before changing status to "inactive".
//
// Parameters:
//   - userID: string - The unique user ID to deactivate
//
// Returns:
//   - error: "user not found", "user is already deactivated", database error, or nil on success
func DeactivateUserByID(db DBExecutor, userID string) error {
	// Validate user exists
	err := isUserThere(db, userID)
	if err != nil {
		return err
	}

	// Check current status
	status := getUserStatus(db, userID)
	if status == "inactive" {
		return fmt.Errorf("user is already deactivated")
	}

	// Deactivate user
	_, err = db.Exec(`
		UPDATE users
		SET status = "inactive"
		WHERE user_id = ?
	`, userID)

	return err
}

// GetUserIdByToken retrieves a user ID from an authentication token.
//
// This function looks up the user_tokens table to find the user associated
// with the provided authentication token.
//
// Parameters:
//   - token: string - The authentication token
//
// Returns:
//   - string: The user ID associated with the token
//   - error: Database error, "no rows" if token not found, or nil on success
func GetUserIdByToken(db DBExecutor, token string) (string, error) {
	var userID string

	// Query user ID by token
	err := db.QueryRow("SELECT user_id FROM user_tokens WHERE token = ?", token).Scan(&userID)
	if err != nil {
		return "", err
	}

	return userID, nil
}

// CreateUserAddress creates a new address for a user.
//
// This function generates a unique address ID and stores the complete
// address details including country, city, apartment, and zip code.
//
// Parameters:
//   - req: dtos.UserAdress containing:
//   - Address: Street address
//   - Country: Country name
//   - Apartment: Apartment/unit number
//   - City: City name
//   - ZipCode: Postal/ZIP code
//   - userID: string - The user ID this address belongs to
//
// Returns:
//   - error: Database error or nil on success
func CreateUserAddress(db DBExecutor, req dtos.UserAdress, userID string) error {
	// Generate unique address ID
	addressID, _ := shortid.Generate()

	// Insert new address record
	_, err := db.Exec(`
		INSERT INTO user_addresses (address_id, user_id, address, country, apartment,city,zip_code)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		addressID, userID, req.Address, req.Country, req.Apartment, req.City, req.ZipCode,
	)
	if err != nil {
		return err
	}
	return nil
}

// GetUserAddresses retrieves all addresses for a user.
//
// This function returns addresses ordered by creation date (newest first).
//
// Parameters:
//   - userID: string - The user ID to retrieve addresses for
//
// Returns:
//   - []dtos.UserAddress: Array of user addresses with:
//   - AddressID: Unique address identifier
//   - Address: Street address
//   - Country: Country name
//   - Apartment: Apartment/unit number
//   - City: City name
//   - ZipCode: Postal/ZIP code
//   - error: Database error or nil on success
func GetUserAddresses(db DBExecutor, userID string) ([]dtos.UserAddress, error) {
	// Query user addresses ordered by creation date
	rows, err := db.Query(`
		SELECT address_id, address, country,apartment,city,zip_code
		FROM user_addresses
		WHERE user_id = ?
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []dtos.UserAddress

	// Process each address
	for rows.Next() {
		var addr dtos.UserAddress
		if err := rows.Scan(&addr.AddressID, &addr.Address, &addr.Country, &addr.Apartment, &addr.City, &addr.ZipCode); err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return addresses, nil
}

// isAddressThere validates that an address exists in the database.
//
// This helper function is used before update or delete operations.
//
// Parameters:
//   - addressID: string - The unique address ID to check
//
// Returns:
//   - error: "address not found", database error, or nil if address exists
func isAddressThere(db DBExecutor, addressID string) error {
	// Check address existence
	exists, err := RecordExists(db, "user_addresses", "address_id = ?", addressID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("address not found")
	}
	return nil
}

// UpdateUserAddress updates an existing user address.
//
// This function validates address existence and ownership before updating
// all address fields.
//
// Parameters:
//   - addressID: string - The unique address ID to update
//   - userID: string - The user ID (for ownership verification)
//   - req: *dtos.UserAdress containing updated:
//   - Address: New street address
//   - Country: New country
//   - Apartment: New apartment/unit number
//   - City: New city
//   - ZipCode: New postal/ZIP code
//
// Returns:
//   - error: "address not found", "address provided does not belong to user",
