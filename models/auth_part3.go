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
	"errors"
	"fmt"
	"strings"
	"time"

	"ekomasi_backend/dtos"
)

// Conditionally inserts (if not empty): first_name, last_name, email, phone_number
func insertUser(db DBExecutor, userID, roleID string, input dtos.RegisterRequest, role string, tenantID int) error {
	// Initialize base columns that are always inserted
	columns := []string{"user_id", "role_id", "role", "tenant_id"}
	values := []interface{}{userID, roleID, role, tenantID}

	// Helper function to add optional fields only if not empty
	addIfNotEmpty := func(field string, value string) {
		if value != "" {
			columns = append(columns, field)
			values = append(values, value)
		}
	}

	// Add optional user fields if provided
	addIfNotEmpty("first_name", input.Firstname)
	addIfNotEmpty("last_name", input.Lastname)
	addIfNotEmpty("email", input.Email)
	addIfNotEmpty("phone_number", input.Phonenumber)

	// Build dynamic INSERT query with correct number of placeholders
	query := fmt.Sprintf(
		"INSERT INTO users (%s) VALUES (%s)",
		strings.Join(columns, ", "),
		strings.Repeat("?, ", len(columns)-1)+"?", // Generate ?, ?, ?
	)

	// Execute insert with dynamic values
	_, err := db.Exec(query, values...)
	return err
}

// UpdateLastLogin updates the last_login timestamp for a user to the current time.
//
// This function is typically called after successful user authentication to track
// when the user last accessed the system.
//
// Parameters:
//   - userID: The user_id to update
//
// Returns:
//   - error: Database error if update fails, nil on success
func UpdateLastLogin(db DBExecutor, userID string) error {
	// Update last_login to current timestamp
	_, err := db.Exec("UPDATE users SET last_login = ? WHERE user_id = ?", time.Now(), userID)
	return err
}

// isEmailAndPhoneThere validates email and phone uniqueness for user updates.
//
// This function is similar to validateUniqueUserIdentifiers but includes the current
// user's ID to exclude them from the uniqueness check. Used during profile updates
// to ensure the new email/phone doesn't belong to another user.
//
// Parameters:
//   - email: Email address to validate (empty string skips validation)
//   - phone: Phone number to validate (empty string skips validation)
//   - userID: The current user's ID to exclude from uniqueness check
//
// Returns:
//   - error: nil if unique, "email already exists" or "phone number already exists" error if duplicate found
func isEmailAndPhoneThere(db DBExecutor, email, phone, userID string, tenantID int) error {
	// Check email uniqueness if provided (excluding current user)
	if email != "" {
		if exists, _ := EmailExistsForOtherUser(db, userID, email, tenantID); exists {
			return errors.New("email already exists for another user")
		}
	}
	// Check phone number uniqueness if provided (excluding current user)
	if phone != "" {
		if exists, _ := PhoneExistsForOtherUser(db, userID, phone, tenantID); exists {
			return errors.New("phone number already exists for another user")
		}
	}
	return nil
}

// buildUpdateFields constructs the SET clause and values for the user update query.
//
// This helper function extracts the field-building logic to reduce complexity.
//
// Parameters:
//   - input: RegisterRequest DTO containing fields to update
//   - role: The role name string (if role is being updated)
//
// Returns:
//   - []string: Array of SET clauses for the UPDATE query
//   - []interface{}: Array of values corresponding to the SET clauses
func buildUpdateFields(input dtos.RegisterRequest, role string) ([]string, []interface{}) {
	setClauses := []string{}
	values := []interface{}{}

	// Add firstname to update if provided
	if input.Firstname != "" {
		setClauses = append(setClauses, "first_name = ?")
		values = append(values, input.Firstname)
	}
	// Add lastname to update if provided
	if input.Lastname != "" {
		setClauses = append(setClauses, "last_name = ?")
		values = append(values, input.Lastname)
	}
	// Add email to update if provided
	if input.Email != "" {
		setClauses = append(setClauses, "email = ?")
		values = append(values, input.Email)
	}
	// Add phone number to update if provided
	if input.Phonenumber != "" {
		setClauses = append(setClauses, "phone_number = ?")
		values = append(values, input.Phonenumber)
	}
	// Add role_id to update if provided
	if input.RoleID != "" {
		setClauses = append(setClauses, "role_id = ?")
		values = append(values, input.RoleID)
	}
	// Add role name to update if role_id provided
	if role != "" {
		setClauses = append(setClauses, "role = ?")
		values = append(values, role)
	}

	return setClauses, values
}

// FindByIdAndUpdate updates user profile fields with validation.
//
// This function performs comprehensive validation before updating:
//   - Validates user existence
//   - Validates role existence if role_id is being changed
//   - Validates email/phone uniqueness against other users
//   - Builds dynamic UPDATE query for only provided fields
//   - Retrieves and returns updated user object
//
// Parameters:
//   - input: RegisterRequest DTO containing fields to update (empty fields are skipped)
//   - userID: The user_id to update
//
// Returns:
//   - *dtos.Users: Pointer to updated user object with complete profile including addresses
//   - error: Validation error, "no fields to update" error, or database error if update fails
//
// Only non-empty fields in input will be updated. Empty strings are ignored.
func FindByIdAndUpdate(db DBExecutor, input dtos.RegisterRequest, userID string, tenantID int) (*dtos.Users, error) {
	// Validate that user exists before attempting update
	err := isUserThere(db, userID)
	if err != nil {
		return nil, err
	}
	role := ""
	// Validate role exists if role_id is being changed
	if input.RoleID != "" {
		err = isRoleThere(db, input.RoleID)
		if err != nil {
			return nil, err
		}
		// Fetch role name for update
		role, err = GetRoleNameByID(db, input.RoleID)
		if err != nil {
			return nil, err
		}
	}
	// Validate email/phone uniqueness against other users (excluding current user)
	err = isEmailAndPhoneThere(db, input.Email, input.Phonenumber, userID, tenantID)
	if err != nil {
		return nil, err
	}

	// Build SET clause dynamically - only update non-empty fields
	setClauses, values := buildUpdateFields(input, role)

	// Validate that at least one field is being updated
	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add userID for WHERE clause
	values = append(values, userID)

	// Build dynamic UPDATE query with all SET clauses
	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE user_id = ?`, strings.Join(setClauses, ", "))

	// Execute update query
	_, err = db.Exec(query, values...)
	if err != nil {
		return nil, err
	}
	// Retrieve and return updated user object with complete profile
	user, err := GetUserByUserID(db, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// EmailExistsForOtherUser checks if an email address is already used by another user.
//
// This function is used during user updates to ensure email uniqueness while allowing
// the current user to keep their existing email (by excluding their user_id from the check).
//
// Parameters:
//   - userID: The user_id to exclude from the check (current user)
//   - email: The email address to check for duplicates
//
// Returns:
//   - bool: true if email exists for another user, false if unique or only used by current user
//   - error: Database error if query fails
func EmailExistsForOtherUser(db DBExecutor, userID, email string, tenantID int) (bool, error) {
	var count int
	// Count users with this email excluding the current user
	query := `
		SELECT COUNT(*) 
		FROM users 
		WHERE email = ? 
		  AND user_id <> ?
		  AND tenant_id = ?`
	err := db.QueryRow(query, email, userID, tenantID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// PhoneExistsForOtherUser checks if a phone number is already used by another user.
//
// This function is used during user updates to ensure phone number uniqueness while allowing
// the current user to keep their existing phone number (by excluding their user_id from the check).
//
// Parameters:
//   - userID: The user_id to exclude from the check (current user)
//   - phone: The phone number to check for duplicates
//
// Returns:
//   - bool: true if phone exists for another user, false if unique or only used by current user
//   - error: Database error if query fails
func PhoneExistsForOtherUser(db DBExecutor, userID, phone string, tenantID int) (bool, error) {
	var count int
	// Count users with this phone number excluding the current user
	query := `
		SELECT COUNT(*) 
		FROM users 
		WHERE phone_number = ? 
		  AND user_id <> ?
		  AND tenant_id = ?`
	err := db.QueryRow(query, phone, userID, tenantID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
