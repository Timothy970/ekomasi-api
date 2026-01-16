// Package models provides the authentication and user management layer for the Adenzo e-commerce platform.
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
	"fmt"
	"log"
	"strings"
	"time"

	"adenzo_backend/dtos"

	"github.com/google/uuid"
	"github.com/teris-io/shortid"
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
func GetUserByEmail(email string) (*dtos.User, error) {
	// Execute query with JOIN to roles table
	row := DB.QueryRow("SELECT u.user_id, u.first_name, u.last_name, u.email, r.name, u.phone_number FROM users u JOIN roles r ON u.role_id = r.role_id WHERE email = ?", email)

	var user dtos.User
	// Handle nullable database fields
	var phone sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString

	// Scan query result into user struct and nullable fields
	err := row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &phone)
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
func GetUserByPhone(phone string) (*dtos.User, error) {
	// Execute query with JOIN to roles table
	row := DB.QueryRow("SELECT u.user_id, u.first_name, u.last_name, u.email, r.name, u.phone_number FROM users u JOIN roles r ON u.role_id = r.role_id WHERE phone_number = ?", phone)

	var user dtos.User
	// Handle nullable database fields
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString

	// Scan query result into user struct and nullable fields
	err := row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &user.Phone)
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

	return &user, nil
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
func isUserThere(id string) error {
	// Check if user record exists in users table
	exists, err := RecordExists("users", "user_id = ?", id)
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
func IsProductThere(id string) error {
	// Check if product record exists in products table
	exists, err := RecordExists("products", "product_id = ?", id)
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
func isRestockNotification(userID, productID string) error {
	// Check if restock notification already exists for this user-product combination
	exists, err := RecordExists("restock_notifications", "product_id = ? AND user_id = ?", productID, userID)
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
func isVariantThere(id string) error {
	// Check if variant record exists in variants table
	exists, err := RecordExists("variants", "variant_id = ?", id)
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
//   - error: "user not found" if user doesn't exist, or database error if query fails
//
// The Users struct includes:
//   - Basic info: ID, FirstName, LastName, Email, Phone, Role
//   - Timestamps: LastLogin (nullable), CreatedAt
//   - Status: Account status integer (active/inactive/suspended)
//   - Addresses: Array of user delivery addresses
func GetUserByUserID(id string) (*dtos.Users, error) {
	// Validate user exists before attempting retrieval
	err := isUserThere(id)
	if err != nil {
		return nil, err
	}

	// Execute query with JOIN to roles table for comprehensive user data
	row := DB.QueryRow("SELECT u.user_id, u.first_name, u.last_name, u.email, r.name, u.phone_number, u.last_login, u.created_at, u.status FROM users u JOIN roles r ON u.role_id = r.role_id WHERE user_id = ?", id)

	var user dtos.Users
	// Handle nullable database fields
	var phone sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString
	var dateJoined, lastLogin sql.NullTime

	// Scan query result into user struct and nullable fields
	err = row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &phone, &lastLogin, &dateJoined, &user.Status)
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
	user.UserAddress, _ = GetUserAddresses(user.ID)

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
func CreateUser(input dtos.RegisterRequest) (*dtos.User, error) {
	// Validate unique email and phone to prevent duplicates
	if err := validateUniqueUserIdentifiers(input.Email, input.Phonenumber); err != nil {
		return nil, err
	}

	// Generate unique user ID using shortid library
	userID, _ := shortid.Generate()

	// Resolve role ID (use provided or default to customer)
	roleID, err := resolveRoleID(input.RoleID)
	if err != nil {
		return nil, err
	}

	// Validate that the resolved role exists in database
	if err := isRoleThere(roleID); err != nil {
		return nil, err
	}
	// Fetch role name for response object
	role, err := GetRoleNameByID(roleID)
	if err != nil {
		return nil, err
	}

	// Insert user record with dynamic fields (only non-empty values)
	if err := insertUser(userID, roleID, input, role); err != nil {
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
func AddUser(input dtos.RegisterRequest) (*dtos.User, error) {
	// Validate unique email and phone to prevent duplicates
	if err := validateUniqueUserIdentifiers(input.Email, input.Phonenumber); err != nil {
		return nil, err
	}
	// Validate that the role exists in database
	err := isRoleThere(input.RoleID)
	if err != nil {
		return nil, err
	}
	// Generate unique user ID using shortid library
	userID, _ := shortid.Generate()
	// Fetch role name for response object
	role, err := GetRoleNameByID(input.RoleID)
	if err != nil {
		return nil, err
	}

	// Insert user record with dynamic fields (only non-empty values)
	if err := insertUser(userID, input.RoleID, input, role); err != nil {
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
func validateUniqueUserIdentifiers(email, phone string) error {
	// Check email uniqueness if provided
	if email != "" {
		if exists, _ := EmailExistsForOtherUser("userID", email); exists {
			return errors.New("email already exists for another user")
		}
	}
	// Check phone number uniqueness if provided
	if phone != "" {
		if exists, _ := PhoneExistsForOtherUser("userID", phone); exists {
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
func resolveRoleID(inputRoleID string) (string, error) {
	// If role ID is provided, use it as-is (no validation here)
	if inputRoleID != "" {
		return inputRoleID, nil
	}

	// Otherwise, default to the "customer" role
	roleID, err := GetRoleIDForCustomerRole()
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
// Conditionally inserts (if not empty): first_name, last_name, email, phone_number
func insertUser(userID, roleID string, input dtos.RegisterRequest, role string) error {
	// Initialize base columns that are always inserted
	columns := []string{"user_id", "role_id", "role"}
	values := []interface{}{userID, roleID, role}

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
	_, err := DB.Exec(query, values...)
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
func UpdateLastLogin(userID string) error {
	// Update last_login to current timestamp
	_, err := DB.Exec("UPDATE users SET last_login = ? WHERE user_id = ?", time.Now(), userID)
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
func isEmailAndPhoneThere(email, phone, userID string) error {
	// Check email uniqueness if provided (excluding current user)
	if email != "" {
		if exists, _ := EmailExistsForOtherUser(userID, email); exists {
			return errors.New("email already exists for another user")
		}
	}
	// Check phone number uniqueness if provided (excluding current user)
	if phone != "" {
		if exists, _ := PhoneExistsForOtherUser(userID, phone); exists {
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
func FindByIdAndUpdate(input dtos.RegisterRequest, userID string) (*dtos.Users, error) {
	// Validate that user exists before attempting update
	err := isUserThere(userID)
	if err != nil {
		return nil, err
	}
	role := ""
	// Validate role exists if role_id is being changed
	if input.RoleID != "" {
		err = isRoleThere(input.RoleID)
		if err != nil {
			return nil, err
		}
		// Fetch role name for update
		role, err = GetRoleNameByID(input.RoleID)
		if err != nil {
			return nil, err
		}
	}
	// Validate email/phone uniqueness against other users (excluding current user)
	err = isEmailAndPhoneThere(input.Email, input.Phonenumber, userID)
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
	_, err = DB.Exec(query, values...)
	if err != nil {
		return nil, err
	}
	// Retrieve and return updated user object with complete profile
	user, err := GetUserByUserID(userID)
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
func EmailExistsForOtherUser(userID, email string) (bool, error) {
	var count int
	// Count users with this email excluding the current user
	query := `
		SELECT COUNT(*) 
		FROM users 
		WHERE email = ? 
		  AND user_id <> ?`
	err := DB.QueryRow(query, email, userID).Scan(&count)
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
func PhoneExistsForOtherUser(userID, phone string) (bool, error) {
	var count int
	// Count users with this phone number excluding the current user
	query := `
		SELECT COUNT(*) 
		FROM users 
		WHERE phone_number = ? 
		  AND user_id <> ?`
	err := DB.QueryRow(query, phone, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserByResetToken retrieves a user by their password reset token.
//
// This function is used in the password reset workflow. It looks up the email
// associated with the reset token and then retrieves the full user object.
//
// Parameters:
//   - token: The password reset token string
//
// Returns:
//   - *dtos.User: Pointer to the user object if token is valid
//   - error: sql.ErrNoRows if token not found, or database error if query fails
//
// Flow: token -> email lookup in reset_tokens table -> GetUserByEmail
func GetUserByResetToken(token string) (*dtos.User, error) {
	var email string
	// Look up email associated with this reset token
	err := DB.QueryRow("SELECT email FROM reset_tokens WHERE token = ?", token).Scan(&email)
	if err != nil {
		return nil, err
	}
	// Retrieve full user object by email
	return GetUserByEmail(email)
}

// StoreOTP stores a new one-time password (OTP) for a user with an expiration time.
//
// This function generates a unique OTP record in the database for multi-factor authentication.
// The OTP is marked as unused initially and will expire after the specified duration.
//
// Parameters:
//   - userID: The user_id for whom the OTP is being generated
//   - otp: The OTP code string (typically 6 digits)
//   - duration: Time duration after which the OTP expires (e.g., 5*time.Minute)
//
// Returns:
//   - error: Database error if insert fails, nil on success
//
// The generated OTP includes:
//   - Unique UUID id
//   - Expiration timestamp (current time + duration)
//   - used flag set to FALSE initially
func StoreOTP(userID string, otp string, duration time.Duration) error {
	// Calculate expiration time
	expiry := time.Now().Add(duration)
	// Generate unique OTP record ID
	otpID := uuid.New().String()

	// Insert OTP record with expiration and unused status
	query := `
		INSERT INTO otps (id, user_id, code, expires_at, used)
		VALUES (?, ?, ?, ?, FALSE)
	`
	_, err := DB.Exec(query, otpID, userID, otp, expiry)
	if err != nil {
		log.Printf("Failed to store OTP: %v", err) // Log error for debugging
	}
	return err
}

// VerifyOTP checks if the provided OTP is valid for the user and not expired.
//
// This function validates an OTP for multi-factor authentication by:
//   - Finding the most recent OTP record matching user_id and code
//   - Checking if OTP has been used already
//   - Verifying OTP hasn't expired
//   - Marking OTP as used if valid (prevents reuse)
//
// Parameters:
//   - userID: The user_id for whom the OTP is being verified
//   - otp: The OTP code string to verify
//
// Returns:
//   - bool: true if OTP is valid (unused and not expired), false otherwise
//   - error: Database error if query fails, nil for invalid/missing OTP
func VerifyOTP(userID string, otp string) (bool, error) {
	var id string
	var expiresAt time.Time
	var used bool

	// Find the most recent OTP record for this user and code
	query := `
		SELECT id, expires_at, used
		FROM otps
		WHERE user_id = ? AND code = ?
		ORDER BY expires_at DESC
		LIMIT 1
	`

	// Retrieve OTP record
	err := DB.QueryRow(query, userID, otp).Scan(&id, &expiresAt, &used)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // OTP not found - not an error
		}
		log.Printf("Error querying OTP: %v", err)
		return false, err
	}

	// Check if OTP has been used or has expired
	if used || time.Now().After(expiresAt) {
		return false, nil // Invalid OTP - already used or expired
	}

	// Mark the OTP as used to prevent reuse
	_, err = DB.Exec(`UPDATE otps SET used = TRUE WHERE id = ?`, id)
	if err != nil {
		log.Printf("Error updating OTP to used: %v", err)
		return false, err
	}

	// OTP is valid
	return true, nil
}
