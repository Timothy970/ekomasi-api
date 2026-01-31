// Package models provides data access functions for the Adenzo e-commerce platform.
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
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"
	"math"
	"strings"

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
//     database error, or nil on success
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
func GetAllUsersWithPagination(db DBExecutor, limit, offset int, q, role string) ([]dtos.Users, *dtos.PaginationMeta, error) {
	// Set default limit
	if limit <= 0 {
		limit = 10
	}

	// Step 1: Count total users matching filters
	totalItems, err := countUsers(db, q, role)
	if err != nil {
		return nil, nil, err
	}

	// Step 2: Fetch paginated users
	users, err := fetchUsers(db, limit, offset, q, role)
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
//
// Returns:
//   - int: Total count of matching users
//   - error: Database error or nil on success
func countUsers(db DBExecutor, q, role string) (int, error) {
	query := "SELECT COUNT(*) FROM users"
	var args []interface{}
	var conditions []string

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
//
// Returns:
//   - []dtos.Users: Array of users (without addresses populated)
//   - error: Database error or nil on success
func fetchUsers(db DBExecutor, limit, offset int, q, role string) ([]dtos.Users, error) {
	query := `
		SELECT user_id, first_name, last_name, email, role, phone_number, last_login, created_at, status
		FROM users
	`

	var args []interface{}
	var conditions []string

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

// buildPagination creates pagination metadata from query results.
//
// This helper function calculates page numbers and navigation flags.
//
// Parameters:
//   - limit: int - Items per page
//   - offset: int - Current offset
//   - totalItems: int - Total matching items
//
// Returns:
//   - *dtos.PaginationMeta: Pagination metadata
func buildPagination(limit, offset, totalItems int) *dtos.PaginationMeta {
	// Calculate current page number
	page := (offset / limit) + 1
	// Calculate total pages (ceiling division)
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	return &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
}

// GetPurchasedCategories retrieves distinct product categories from user's purchase history.
//
// This function analyzes a user's orders to identify which product categories
// they have purchased from, useful for personalized recommendations.
//
// Parameters:
//   - userID: string - The user ID to analyze
//
// Returns:
//   - []string: Array of distinct category IDs from purchased products
//   - error: Database error or nil on success
func GetPurchasedCategories(db DBExecutor, userID string) ([]string, error) {
	// Query distinct categories from user's order history
	rows, err := db.Query(`
		SELECT DISTINCT p.category_id
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		JOIN products p ON oi.product_id = p.product_id
		WHERE o.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// GetWishlistCategories retrieves distinct product categories from user's wishlist.
//
// This function analyzes a user's wishlist to identify which product categories
// they're interested in, useful for personalized recommendations.
//
// Parameters:
//   - userID: string - The user ID to analyze
//
// Returns:
//   - []string: Array of distinct category IDs from wishlist products
//   - error: Database error or nil on success
func GetWishlistCategories(db DBExecutor, userID string) ([]string, error) {
	// Query distinct categories from user's wishlist
	rows, err := db.Query(`
		SELECT DISTINCT p.category_id
		FROM wishlists w
		JOIN wishlist_items wi ON w.wishlist_id = wi.wishlist_id
		JOIN products p ON wi.product_id = p.product_id
		WHERE w.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// GetProductsByCategories retrieves paginated products from specified categories.
//
// This function is useful for showing related products based on user's purchase
// history or wishlist categories. Returns enriched product data including
// images, warranties, features, variants, and tax information.
//
// Parameters:
//   - categories: []string - Array of category IDs to fetch products from
//   - page: int - Page number (1-based)
//   - size: int - Number of products per page
//
// Returns:
//   - []dtos.Product: Array of enriched products with images, warranties, features, variants, tax
//   - dtos.PaginationMeta: Pagination info (page, size, totals, navigation flags)
//   - error: Database error or nil on success (returns empty result if no categories)
func GetProductsByCategories(db DBExecutor, categories []string, page, size int) ([]dtos.Product, dtos.PaginationMeta, error) {
	// Return empty result if no categories provided
	if len(categories) == 0 {
		return []dtos.Product{}, dtos.PaginationMeta{}, nil
	}

	// Calculate offset for pagination
	offset := (page - 1) * size

	// Build dynamic IN clause with placeholders
	placeholders := strings.Repeat(",?", len(categories)-1)
	query := `
		SELECT SQL_CALC_FOUND_ROWS
		       p.product_id, p.name, p.description, p.sku, p.price, 
		       p.category_id, p.stock_quantity, p.search_vector, 
		       p.created_at, p.last_updated_at
		FROM products p
		WHERE p.category_id IN (?` + placeholders + `)
		LIMIT ? OFFSET ?`

	// Build arguments array for query
	args := make([]interface{}, len(categories)+2)
	for i, v := range categories {
		args[i] = v
	}
	args[len(categories)] = size
	args[len(categories)+1] = offset

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	// Process each product and enrich with related data
	var products []dtos.Product
	for rows.Next() {
		var p dtos.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price,
			&p.CategoryID, &p.StockQuantity, &p.SearchVector,
			&p.CreatedAt, &p.LastUpdated,
		); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}

		// Fetch product images (ignores errors)
		p.Images, _ = fetchProductImages(db, p.ID)

		// Fetch product warranties
		warranties, err := FetchProductWarranties(db, p.ID)
		if err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		p.Warranty = &warranties

		// Fetch product features
		features, err := fetchProductFeatures(db, p.ID)
		if err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		p.Features = features

		// Fetch product variants
		variants, err := getProductVariants(db, p.ID)
		if err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		p.ProductVariants = variants

		// Fetch product tax information
		tax, err := fetchProductTax(db, p.ID)
		if err != nil {
			return nil, dtos.PaginationMeta{}, err
		}
		p.Tax = &tax

		products = append(products, p)
	}

	// Get total count using FOUND_ROWS()
	var totalItems int
	if err := db.QueryRow(`SELECT FOUND_ROWS()`).Scan(&totalItems); err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Build pagination metadata
	totalPages := (totalItems + size - 1) / size // Ceiling division
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return products, meta, nil
}

// isSubscriberThere validates that an email is not already subscribed.
//
// This helper function prevents duplicate newsletter subscriptions.
//
// Parameters:
//   - email: string - The email address to check
//
// Returns:
//   - error: "email already subscribed" if exists, database error, or nil if unique
func isSubscriberThere(db DBExecutor, email string) error {
	// Check if email already subscribed
	exists, err := RecordExists(db, "subscribers", "email = ?", email)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("email already subscribed")
	}
	return nil
}

// CreateSubscribers creates a new newsletter subscriber.
//
// This function validates email uniqueness and creates a subscriber record.
//
// Parameters:
//   - email: string - The email address to subscribe
//
// Returns:
//   - error: "email already subscribed", database error, or nil on success
func CreateSubscribers(db DBExecutor, email string) error {
	// Validate email not already subscribed
	err := isSubscriberThere(db, email)
	if err != nil {
		return err
	}

	// Generate unique subscriber ID
	subscriberID, _ := shortid.Generate()

	// Insert new subscriber
	_, err = db.Exec(`
		INSERT INTO subscribers (subscriber_id, email)
		VALUES (?, ?)`,
		subscriberID, email,
	)
	if err != nil {
		return err
	}
	return nil
}
