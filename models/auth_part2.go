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

	"github.com/teris-io/shortid"
)

//   - error: "user not found" if user doesn't exist, or database error if query fails
//
// The Users struct includes:
//   - Basic info: ID, FirstName, LastName, Email, Phone, Role
//   - Timestamps: LastLogin (nullable), CreatedAt
//   - Status: Account status integer (active/inactive/suspended)
//   - Addresses: Array of user delivery addresses
func GetUserByUserID(db DBExecutor, id string) (*dtos.Users, error) {
	// Validate user exists before attempting retrieval
	err := isUserThere(db, id)
	if err != nil {
		return nil, err
	}

	// Execute query with JOIN to roles table for comprehensive user data
	row := db.QueryRow("SELECT u.user_id, u.first_name, u.last_name, u.email, r.name, u.role_id, u.phone_number, u.last_login, u.created_at, u.status FROM users u JOIN roles r ON u.role_id = r.role_id WHERE user_id = ?", id)

	var user dtos.Users
	// Handle nullable database fields
	var phone sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString
	var dateJoined, lastLogin sql.NullTime
	var roleID string

	// Scan query result into user struct and nullable fields
	err = row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &roleID, &phone, &lastLogin, &dateJoined, &user.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found (should not happen after validation)
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
	// Format timestamp fields for API response
	if dateJoined.Valid {
		user.DateJoined = dateJoined.Time.Format("2006-01-02 15:04:05")
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time.Format("2006-01-02 15:04:05")
	}
	// Fetch all associated addresses for this user
	user.UserAddress, _ = GetUserAddresses(db, user.ID)
	user.Permissions, err = GetPermissionsByRoleID(db, roleID)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUser creates a new user account with role assignment and validation.
//
// This function performs comprehensive validation before creating the user:
//   - Validates email and phone uniqueness
//   - Generates a unique user_id using shortid
//   - Resolves role_id (defaults to "customer" if not provided)
//   - Validates role existence
//   - Inserts user with dynamic field handling
//
// Parameters:
//   - input: RegisterRequest DTO containing user registration data (firstname, lastname, email, phonenumber, role_id)
//
// Returns:
//   - *dtos.User: Pointer to created user object with ID, names, email, role, and nil LastLogin
//   - error: Validation error, role error, or database error if creation fails
func CreateUser(db DBExecutor, input dtos.RegisterRequest, tenantID int) (*dtos.User, error) {
	// Validate unique email and phone to prevent duplicates
	if err := validateUniqueUserIdentifiers(db, input.Email, input.Phonenumber, tenantID); err != nil {
		return nil, err
	}

	// Generate unique user ID using shortid library
	userID, _ := shortid.Generate()

	// Resolve role ID (use provided or default to customer)
	roleID, err := resolveRoleID(db, input.RoleID)
	if err != nil {
		return nil, err
	}

	// Validate that the resolved role exists in database
	if err := isRoleThere(db, roleID); err != nil {
		return nil, err
	}
	// Fetch role name for response object
	role, err := GetRoleNameByID(db, roleID)
	if err != nil {
		return nil, err
	}

	// Insert user record with dynamic fields (only non-empty values)
	if err := insertUser(db, userID, roleID, input, role, tenantID); err != nil {
		return nil, err
	}

	// Return created user object for API response
	return &dtos.User{
		ID:        userID,
		FirstName: input.Firstname,
		LastName:  input.Lastname,
		Email:     input.Email,
		Role:      role,
		LastLogin: nil, // Nil for newly created user
	}, nil
}

// AddUser creates a new user account with simplified response (without LastLogin field).
//
// This function is similar to CreateUser but returns a User object without the LastLogin field.
// It performs the same validation and creation steps:
//   - Validates email and phone uniqueness
//   - Validates role existence
//   - Generates unique user_id
//   - Inserts user with dynamic field handling
//
// Parameters:
//   - input: RegisterRequest DTO containing user registration data (firstname, lastname, email, phonenumber, role_id)
//
// Returns:
//   - *dtos.User: Pointer to created user object with ID, names, email, and role (no LastLogin field)
//   - error: Validation error, role error, or database error if creation fails
func AddUser(db DBExecutor, input dtos.RegisterRequest, tenantID int) (*dtos.User, error) {
	// Validate unique email and phone to prevent duplicates
	if err := validateUniqueUserIdentifiers(db, input.Email, input.Phonenumber, tenantID); err != nil {
		return nil, err
	}
	// Validate that the role exists in database
	err := isRoleThere(db, input.RoleID)
	if err != nil {
		return nil, err
	}
	// Generate unique user ID using shortid library
	userID, _ := shortid.Generate()
	// Fetch role name for response object
	role, err := GetRoleNameByID(db, input.RoleID)
	if err != nil {
		return nil, err
	}

	// Insert user record with dynamic fields (only non-empty values)
	if err := insertUser(db, userID, input.RoleID, input, role, tenantID); err != nil {
		return nil, err
	}

	// Return created user object for API response (without LastLogin)
	return &dtos.User{
		ID:        userID,
		FirstName: input.Firstname,
		LastName:  input.Lastname,
		Email:     input.Email,
		Role:      role,
	}, nil
}

// validateUniqueUserIdentifiers checks if email and phone number are unique across users.
//
// This validation function prevents duplicate user accounts by checking if the provided
// email or phone number already exists for another user. Empty values are skipped.
//
// Parameters:
//   - email: Email address to validate (empty string skips validation)
//   - phone: Phone number to validate (empty string skips validation)
//
// Returns:
//   - error: nil if unique, "email already exists" or "phone number already exists" error if duplicate found
func validateUniqueUserIdentifiers(db DBExecutor, email, phone string, tenantID int) error {
	// Check email uniqueness if provided
	if email != "" {
		if exists, _ := EmailExistsForOtherUser(db, "userID", email, tenantID); exists {
			return errors.New("email already exists for another user")
		}
	}
	// Check phone number uniqueness if provided
	if phone != "" {
		if exists, _ := PhoneExistsForOtherUser(db, "userID", phone, tenantID); exists {
			return errors.New("phone number already exists for another user")
		}
	}
	return nil
}

// resolveRoleID determines the role_id to use for user creation.
//
// This function implements role defaulting logic:
//   - If a role_id is provided in input, use it as-is
//   - If not provided, default to the "customer" role
//   - Returns error if customer role not found when defaulting
//
// Parameters:
//   - inputRoleID: The role_id from user input (empty string triggers defaulting)
//
// Returns:
//   - string: The resolved role_id to use
//   - error: Error if customer role not found when defaulting, or database error
//
// This allows flexible user creation where role is optional and defaults to customer.
func resolveRoleID(db DBExecutor, inputRoleID string) (string, error) {
	// If role ID is provided, use it as-is (no validation here)
	if inputRoleID != "" {
		return inputRoleID, nil
	}

	// Otherwise, default to the "customer" role
	roleID, err := GetRoleIDForCustomerRole(db)
	if err != nil {
		return "", err
	}
	if roleID == "" {
		return "", errors.New("customer role not found")
	}

	return roleID, nil
}

// insertUser performs the database INSERT operation for creating a new user.
//
// This function builds a dynamic INSERT query that only includes non-empty fields,
// preventing NULL constraint violations and allowing flexible user creation.
//
// Parameters:
//   - userID: The generated unique user_id
//   - roleID: The validated role_id to assign
//   - input: RegisterRequest DTO containing optional user fields (firstname, lastname, email, phonenumber)
//   - role: The role name string for denormalization
//
// Returns:
//   - error: Database error if insert fails, nil on success
//
// Always inserts: user_id, role_id, role
