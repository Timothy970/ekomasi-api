package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/teris-io/shortid"
)

var getAddress = "user_id = ? AND address = ?"

func DeleteUserByID(userID string) error {
	err := isUserThere(userID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
		DELETE FROM users
		WHERE user_id = ?
	`, userID)

	return err
}
func getUserStatus(userID string) string {
	var status string
	_ = DB.QueryRow(`SELECT status FROM users WHERE user_id = ?`, userID).Scan(&status)
	return status
}

// Activate user
func ActivateUserByID(userID string) error {
	err := isUserThere(userID)
	if err != nil {
		return err
	}
	status := getUserStatus(userID)
	if status == "active" {
		return fmt.Errorf("user is already active")
	}
	_, err = DB.Exec(`
		UPDATE users
		SET status = "active"
		WHERE user_id = ?
	`, userID)

	return err
}
func DeactivateUserByID(userID string) error {
	err := isUserThere(userID)
	if err != nil {
		return err
	}
	status := getUserStatus(userID)
	if status == "inactive" {
		return fmt.Errorf("user is already deactivated")
	}
	_, err = DB.Exec(`
		UPDATE users
		SET status = "inactive"
		WHERE user_id = ?
	`, userID)

	return err
}

// get user by token
func GetUserIdByToken(token string) (string, error) {
	var userID string
	err := DB.QueryRow("SELECT user_id FROM user_tokens WHERE token = ?", token).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

// insert user addresses
func CreateUserAddress(req dtos.UserAdress, userID string) error {
	//check if address exists
	exists, err := RecordExists("user_addresses", getAddress, userID, req.Address)
	if err != nil {
		return fmt.Errorf("failed : %w", err)
	}
	if exists {
		return fmt.Errorf("address already exists")
	}
	addressID, _ := shortid.Generate()
	//Safe to insert to DB
	_, err = DB.Exec(`
		INSERT INTO user_addresses (address_id, user_id, address)
		VALUES (?, ?, ?)`,
		addressID, userID, req.Address,
	)
	if err != nil {
		return err
	}
	return nil
}
func GetUserAddresses(userID string) ([]dtos.UserAddress, error) {
	rows, err := DB.Query(`
		SELECT address_id, address
		FROM user_addresses
		WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []dtos.UserAddress

	for rows.Next() {
		var addr dtos.UserAddress
		if err := rows.Scan(&addr.AddressID, &addr.Address); err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return addresses, nil
}

// update user address
func UpdateUserAddress(addressID, userID, address string) error {
	//check if address exists
	exists, err := RecordExists("user_addresses", "user_id = ? AND address_id = ?", userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to check variant existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("address not found")
	}
	exists, err = RecordExists("user_addresses", "user_id = ? AND address = ?", userID, address)
	if err != nil {
		return fmt.Errorf("failed to existence: %w", err)
	}
	if exists {
		return fmt.Errorf("address name already exists")
	}
	// Proceed with update
	_, err = DB.Exec(`
		UPDATE user_addresses
		SET address = ?
		WHERE user_id = ? AND address_id = ?
	`, address, userID, addressID)

	return err
}

// Delete user address
func DeleteUserAddress(addressID, userID string) error {
	//check if address exists
	exists, err := RecordExists("user_addresses", getAddress, userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to check variant existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("address not found")
	}
	_, err = DB.Exec("DELETE FROM user_addresses WHERE user_id = ? AND address = ?", userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to delete user address: %w", err)
	}

	return nil
}

// Get all users
func GetAllUsersWithPagination(limit, offset int) ([]dtos.User, *dtos.PaginationMeta, error) {
	// Fetch total count
	var totalItems int
	countQuery := "SELECT COUNT(*) FROM users"
	err := DB.QueryRow(countQuery).Scan(&totalItems)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Fetch paginated users
	query := `
		SELECT user_id, first_name, last_name, email, role, phone_number
		FROM users
		ORDER BY user_id DESC
		LIMIT ? OFFSET ?
	`
	rows, err := DB.Query(query, limit, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []dtos.User
	for rows.Next() {
		var user dtos.User
		var phone sql.NullString
		var firstName sql.NullString
		var lastName sql.NullString
		var userEmail sql.NullString

		if err := rows.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &phone); err != nil {
			return nil, nil, err
		}

		user.FirstName = ""
		user.LastName = ""
		user.Email = ""
		user.Phone = ""

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

		users = append(users, user)
	}

	// Calculate pagination meta
	page := (offset / limit) + 1
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return users, meta, nil
}

// Get purchased product categories for the user
func GetPurchasedCategories(userID string) ([]string, error) {
	rows, err := DB.Query(`
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

// Get wishlist product categories for the user
func GetWishlistCategories(userID string) ([]string, error) {
	rows, err := DB.Query(`
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

// Get related products by category
// GetProductsByCategories with pagination
func GetProductsByCategories(categories []string, page, size int) ([]dtos.Product, dtos.PaginationMeta, error) {
	if len(categories) == 0 {
		return []dtos.Product{}, dtos.PaginationMeta{}, nil
	}

	offset := (page - 1) * size

	// Build placeholders for IN clause
	placeholders := strings.Repeat(",?", len(categories)-1)
	query := `
		SELECT SQL_CALC_FOUND_ROWS
		       p.product_id, p.name, p.description, p.sku, p.price, 
		       p.category_id, p.stock_quantity, p.search_vector, 
		       p.created_at, p.last_updated_at
		FROM products p
		WHERE p.category_id IN (?` + placeholders + `)
		LIMIT ? OFFSET ?`

	args := make([]interface{}, len(categories)+2)
	for i, v := range categories {
		args[i] = v
	}
	args[len(categories)] = size
	args[len(categories)+1] = offset

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

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
		p.Images, _ = fetchProductImages(p.ID)
		products = append(products, p)
	}

	// Get total count
	var totalItems int
	if err := DB.QueryRow(`SELECT FOUND_ROWS()`).Scan(&totalItems); err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	totalPages := (totalItems + size - 1) / size
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
