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
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"strings"
)

// database error, or nil on success
func UpdateUserAddress(db DBExecutor, addressID, userID string, req *dtos.UserAdress) error {
	// Validate address exists
	err := isAddressThere(db, addressID)
	if err != nil {
		return err
	}

	// Verify address belongs to user
	exists, err := RecordExists(db, "user_addresses", "user_id = ? AND address_id = ?", userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to check variant existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("address provided does not belong to user")
	}

	// Update address details
	_, err = db.Exec(`
		UPDATE user_addresses
		SET address = ?, country = ? ,apartment = ?, city = ?, zip_code = ?
		WHERE user_id = ? AND address_id = ?
	`, req.Address, req.Country, req.Apartment, req.City, req.ZipCode, userID, addressID)

	return err
}

// DeleteUserAddress permanently removes a user address.
//
// This function validates address existence and ownership before deletion.
//
// Parameters:
//   - addressID: string - The address ID to delete
//   - userID: string - The user ID (for ownership verification)
//
// Returns:
//   - error: "address not found", "failed to delete user address",
//     database error, or nil on success
func DeleteUserAddress(db DBExecutor, addressID, userID string) error {
	// Check if address exists and belongs to user
	exists, err := RecordExists(db, "user_addresses", getAddress, userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to check variant existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("address not found")
	}

	// Delete address record
	_, err = db.Exec("DELETE FROM user_addresses WHERE user_id = ? AND address_id = ?", userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to delete user address: %w", err)
	}

	return nil
}

// GetAllUsersWithPagination retrieves users with pagination, search, and role filtering.
//
// This function orchestrates user retrieval by:
//  1. Counting total matching users
//  2. Fetching paginated user data
//  3. Building pagination metadata
//
// Parameters:
//   - limit: int - Number of items per page (defaults to 10 if <= 0)
//   - offset: int - Number of items to skip
//   - q: string - Search query (case-insensitive) across first name, last name, email, phone
//   - role: string - Filter by user role (case-insensitive)
//
// Returns:
//   - []dtos.Users: Array of users with addresses, roles, and status
//   - *dtos.PaginationMeta: Pagination info (page, size, totals, navigation flags)
//   - error: Database error or nil on success
func GetAllUsersWithPagination(db DBExecutor, limit, offset int, q, role string, isAdmin string, tenantID int) ([]dtos.Users, *dtos.PaginationMeta, error) {
	// Set default limit
	if limit <= 0 {
		limit = 10
	}

	// Step 1: Count total users matching filters
	totalItems, err := countUsers(db, q, role, isAdmin, tenantID)
	if err != nil {
		return nil, nil, err
	}

	// Step 2: Fetch paginated users
	users, err := fetchUsers(db, limit, offset, q, role, isAdmin, tenantID)
	if err != nil {
		return nil, nil, err
	}

	// Step 3: Build pagination metadata
	meta := buildPagination(limit, offset, totalItems)

	return users, meta, nil
}

// countUsers counts total users matching search and role filters.
//
// This helper function builds a dynamic count query with optional
// search and role filters.
//
// Parameters:
//   - q: string - Search term for name, email, or phone
//   - role: string - Role filter
//   - isAdmin: string - If not empty, excludes customer role
//
// Returns:
//   - int: Total count of matching users
//   - error: Database error or nil on success
func countUsers(db DBExecutor, q, role, isAdmin string, tenantID int) (int, error) {
	query := "SELECT COUNT(*) FROM users"
	var args []any
	var conditions []string

	// Scope by tenant ID
	conditions = append(conditions, "tenant_id = ?")
	args = append(args, tenantID)

	// Add search filter (multi-field case-insensitive partial match)
	if q != "" {
		q = "%" + strings.ToLower(q) + "%"
		conditions = append(conditions, `(LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR phone_number LIKE ? )`)
		args = append(args, q, q, q, q)
	}

	// Add role filter (case-insensitive exact match)
	if role != "" {
		conditions = append(conditions, `LOWER(role) = ?`)
		args = append(args, strings.ToLower(role))
	}

	// Exclude customer role when isAdmin is not empty
	if isAdmin != "" {
		conditions = append(conditions, `LOWER(role) != ?`)
		args = append(args, "customer")
	}

	// Combine filters with WHERE clause
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Execute count query
	var total int
	if err := db.QueryRow(query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return total, nil
}

// fetchUsers retrieves paginated users with search and role filters.
//
// This helper function builds a dynamic select query with filtering,
// ordering, and pagination.
//
// Parameters:
//   - limit: int - Number of users to retrieve
//   - offset: int - Number of users to skip
//   - q: string - Search term
//   - role: string - Role filter
//   - isAdmin: string - If not empty, excludes customer role
//
// Returns:
//   - []dtos.Users: Array of users (without addresses populated)
//   - error: Database error or nil on success
func fetchUsers(db DBExecutor, limit, offset int, q, role, isAdmin string, tenantID int) ([]dtos.Users, error) {
	query := `
		SELECT user_id, first_name, last_name, email, role, phone_number, last_login, created_at, status
		FROM users
	`

	var args []any
	var conditions []string

	// Scope by tenant ID
	conditions = append(conditions, "tenant_id = ?")
	args = append(args, tenantID)

	// Add search filter
	if q != "" {
		q = "%" + strings.ToLower(q) + "%"
		conditions = append(conditions, `(LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR phone_number LIKE ? )`)
		args = append(args, q, q, q, q)
	}

	// Add role filter
	if role != "" {
		conditions = append(conditions, `LOWER(role) = ?`)
		args = append(args, strings.ToLower(role))
	}

	// Exclude customer role when isAdmin is not empty
	if isAdmin != "" {
		conditions = append(conditions, `LOWER(role) != ?`)
		args = append(args, "customer")
	}

	// Combine conditions
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add sorting (newest first) and pagination
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	// Process each user row
	var users []dtos.Users
	for rows.Next() {
		user, err := scanUserRow(db, rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// scanUserRow scans a database row into a user struct.
//
// This helper function handles nullable fields and formats dates,
// and retrieves associated user addresses.
//
// Parameters:
//   - rows: *sql.Rows - The result set to scan from
//
// Returns:
//   - dtos.Users: User struct with all fields populated
//   - error: Database error or nil on success
func scanUserRow(db DBExecutor, rows *sql.Rows) (dtos.Users, error) {
	var user dtos.Users
	var phone, firstName, lastName, userEmail sql.NullString
	var dateJoined, lastLogin sql.NullTime

	// Scan user fields (handling NULL values)
	if err := rows.Scan(
		&user.ID, &firstName, &lastName, &userEmail,
		&user.Role, &phone, &lastLogin, &dateJoined, &user.Status,
	); err != nil {
		return dtos.Users{}, err
	}

	// Handle nullable string fields
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

	// Format datetime fields
	if dateJoined.Valid {
		user.DateJoined = dateJoined.Time.Format("2006-01-02 15:04:05")
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time.Format("2006-01-02 15:04:05")
	}

	// Fetch user addresses (ignores errors, returns empty array on failure)
	user.UserAddress, _ = GetUserAddresses(db, user.ID)

	return user, nil
}
