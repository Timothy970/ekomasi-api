// Package models provides the authentication and user management layer for the Ekomasi e-commerce platform.
//
// This package handles core authentication operations including:
//   - User registration and account creation with role assignment
//   - User retrieval by email, phone number, and user ID
//   - User profile updates with validation
//   - One-Time Password (OTP) generation, storage, and verification
//   - Password reset token management
//   - Last login timestamp tracking
//   - User uniqueness validation (email, phone number)
//   - Database record existence validation for users, products, variants
//
// The authentication system supports:
//   - Role-based access control (RBAC) integration
//   - Multi-factor authentication via OTP
//   - Optional fields using sql.NullString and sql.NullTime
//   - User address management integration
//   - Dynamic field updates with validation
//
// Database Schema:
//   - users table: Stores user profiles with role references
//   - otps table: Manages one-time passwords with expiration
//   - reset_tokens table: Handles password reset workflows
//   - addresses table: User delivery addresses (referenced)
//   - roles table: RBAC role definitions (referenced)
package models

import (
	"database/sql"
	"errors"

	"ekomasi_backend/dtos"
)

// DB is now defined in database.go

// GetUserByEmail retrieves a user by their email address with role information.
//
// This function performs a JOIN with the roles table to fetch the user's role name.
// It handles nullable fields (first_name, last_name, email, phone_number) gracefully.
//
// Parameters:
//   - email: The email address to search for (case-sensitive)
//
// Returns:
//   - *dtos.User: Pointer to the user object containing ID, names, email, phone, and role
//   - error: sql.ErrNoRows if user not found (returns nil), or database error if query fails
func GetUserByEmail(db DBExecutor, email string, tenantID int) (*dtos.User, error) {
	// Execute query with JOIN to roles table
	row := db.QueryRow("SELECT u.user_id, u.first_name, u.last_name, u.email, r.name, u.role_id, u.phone_number, u.status FROM users u JOIN roles r ON u.role_id = r.role_id WHERE email = ? AND u.tenant_id = ?", email, tenantID)

	var user dtos.User
	// Handle nullable database fields
	var phone sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString
	var userRoleID string

	// Scan query result into user struct and nullable fields
	err := row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &userRoleID, &phone, &user.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, err
	}

	// Initialize fields to empty strings
	user.FirstName = ""
	user.LastName = ""
	user.Email = ""
	user.Phone = ""

	// Convert nullable fields to Go native types if valid
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if userEmail.Valid {
		user.Email = userEmail.String
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	user.Permissions, err = GetPermissionsByRoleID(db, userRoleID)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByPhone retrieves a user by their phone number with role information.
//
// This function performs a JOIN with the roles table to fetch the user's role name.
// It handles nullable fields (first_name, last_name, email) gracefully.
//
// Parameters:
//   - phone: The phone number to search for (exact match)
//
// Returns:
//   - *dtos.User: Pointer to the user object containing ID, names, email, phone, and role
//   - error: sql.ErrNoRows if user not found (returns nil), or database error if query fails
func GetUserByPhone(db DBExecutor, phone string, tenantID int) (*dtos.User, error) {
	// Execute query with JOIN to roles table
	row := db.QueryRow("SELECT u.user_id, u.first_name, u.last_name, u.email, r.name, u.role_id, u.phone_number, u.status FROM users u JOIN roles r ON u.role_id = r.role_id WHERE phone_number = ? AND u.tenant_id = ?", phone, tenantID)

	var user dtos.User
	// Handle nullable database fields
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString
	var userRoleID string

	// Scan query result into user struct and nullable fields
	err := row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &userRoleID, &user.Phone, &user.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, err
	}

	// Initialize fields to empty strings
	user.FirstName = ""
	user.LastName = ""
	user.Email = ""

	// Convert nullable fields to Go native types if valid
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if userEmail.Valid {
		user.Email = userEmail.String
	}
	user.Permissions, err = GetPermissionsByRoleID(db, userRoleID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// helper function to get permissions by role id
// params : roleID string
// returns : []string , error
func GetPermissionsByRoleID(db DBExecutor, roleID string) ([]string, error) {
	rows, err := db.Query(`SELECT pm.permission_key from permissions_master pm 
	JOIN role_permissions rp ON pm.permission_master_id = rp.permission_id
	JOIN roles r ON rp.role_id = r.role_id
	WHERE r.role_id = ?`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var permissions []string
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}
	return permissions, nil
}

// func GetUserByPhone(phone string) (*dtos.User, error) {
// 	row := DB.QueryRow("SELECT user_id, first_name, last_name, email, password_hash, role, phone_number FROM users WHERE phone_number = ?", phone)
// 	var user dtos.User

// 	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.PasswordHash, &user.Role, &user.Phone)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

//		return &user, nil
//	}

// isUserThere validates that a user exists in the database by user_id.
//
// This function uses the RecordExists helper to check for user existence.
//
// Parameters:
//   - id: The user_id to validate
//
// Returns:
//   - error: nil if user exists, "user not found" error if not found, or database error
func isUserThere(db DBExecutor, id string) error {
	// Check if user record exists in users table
	exists, err := RecordExists(db, "users", "user_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user not found")
	}
	return nil
}

// IsProductThere validates that a product exists in the database by product_id.
//
// This function uses the RecordExists helper to check for product existence.
//
// Parameters:
//   - id: The product_id to validate
//
// Returns:
//   - error: nil if product exists, "product with ID {id} not found" error if not found, or database error
func IsProductThere(db DBExecutor, id string) error {
	// Check if product record exists in products table
	exists, err := RecordExists(db, "products", "product_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("product with ID " + id + " not found")
	}
	return nil
}

// isRestockNotification checks if a user already has a restock notification for a specific product.
//
// This function prevents duplicate restock notification subscriptions by verifying
// that the combination of user_id and product_id doesn't already exist.
//
// Parameters:
//   - userID: The user_id to check
//   - productID: The product_id to check
//
// Returns:
//   - error: nil if no duplicate exists, "notification already exists" error if duplicate found, or database error
func isRestockNotification(db DBExecutor, userID, productID string) error {
	// Check if restock notification already exists for this user-product combination
	exists, err := RecordExists(db, "restock_notifications", "product_id = ? AND user_id = ?", productID, userID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("notification for this product already exists")
	}
	return nil
}

// isVariantThere validates that a product variant exists in the database by variant_id.
//
// This function uses the RecordExists helper to check for variant existence.
//
// Parameters:
//   - id: The variant_id to validate
//
// Returns:
//   - error: nil if variant exists, "variant not found" error if not found, or database error
func isVariantThere(db DBExecutor, id string) error {
	// Check if variant record exists in variants table
	exists, err := RecordExists(db, "variants", "variant_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("variant not found")
	}
	return nil
}

// GetUserByUserID retrieves comprehensive user information by user_id including addresses.
//
// This function validates user existence first, then fetches complete user profile data
// with role information via JOIN. It performs a separate query to retrieve all associated
// addresses and handles nullable fields gracefully.
//
// Parameters:
//   - id: The user_id to retrieve
//
// Returns:
//   - *dtos.Users: Pointer to complete user object including profile, role, timestamps, status, and addresses
