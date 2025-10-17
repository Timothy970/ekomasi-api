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

// GetUserByEmail retrieves a user by their email address.
// It returns a User object or nil if no user is found.
func GetUserByEmail(email string) (*dtos.User, error) {
	row := DB.QueryRow("SELECT u.user_id, u.first_name, u.last_name, u.email, r.name, u.phone_number FROM users u JOIN roles r ON u.role_id = r.role_id WHERE email = ?", email)

	var user dtos.User
	var phone sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString

	err := row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &phone)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
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

	return &user, nil
}
func GetUserByPhone(phone string) (*dtos.User, error) {
	row := DB.QueryRow("SELECT u.user_id, u.first_name, u.last_name, u.email, r.name, u.phone_number FROM users u JOIN roles r ON u.role_id = r.role_id WHERE phone_number = ?", phone)

	var user dtos.User
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString

	err := row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &user.Phone)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	user.FirstName = ""
	user.LastName = ""
	user.Email = ""

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
func isUserThere(id string) error {
	exists, err := RecordExists("users", "user_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user not found")
	}
	return nil
}
func IsProductThere(id string) error {
	exists, err := RecordExists("products", "product_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("product not found")
	}
	return nil
}
func isBundleThere(id string) error {
	exists, err := RecordExists("product_bundles", "bundle_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("bundle not found")
	}
	return nil
}
func isRestockNotification(userID, productID string) error {
	exists, err := RecordExists("restock_notifications", "product_id = ? AND user_id = ?", productID, userID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("notification for this product already exists")
	}
	return nil
}
func isVariantThere(id string) error {
	exists, err := RecordExists("variants", "variant_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("variant not found")
	}
	return nil
}
func GetUserByUserID(id string) (*dtos.Users, error) {
	err := isUserThere(id)
	if err != nil {
		return nil, err
	}

	row := DB.QueryRow("SELECT u.user_id, u.first_name, u.last_name, u.email, r.name, u.phone_number, u.last_login, u.created_at, u.status FROM users u JOIN roles r ON u.role_id = r.role_id WHERE user_id = ?", id)

	var user dtos.Users
	var phone sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString
	var dateJoined, lastLogin sql.NullTime

	err = row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &phone, &lastLogin, &dateJoined, &user.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
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
	if dateJoined.Valid {
		user.DateJoined = dateJoined.Time.Format("2006-01-02 15:04:05")
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time.Format("2006-01-02 15:04:05")
	}
	user.UserAddress, _ = GetUserAddresses(user.ID)

	return &user, nil
}
func CreateUser(input dtos.RegisterRequest) (*dtos.User, error) {
	// Validate unique email and phone
	if err := validateUniqueUserIdentifiers(input.Email, input.Phonenumber); err != nil {
		return nil, err
	}

	// Generate user ID
	userID, _ := shortid.Generate()

	// Resolve role
	roleID, err := resolveRoleID(input.RoleID)
	if err != nil {
		return nil, err
	}

	// Validate role existence
	if err := isRoleThere(roleID); err != nil {
		return nil, err
	}

	// Build dynamic insert query
	if err := insertUser(userID, roleID, input); err != nil {
		return nil, err
	}

	// Fetch role name for response
	role, err := GetRoleNameByID(roleID)
	if err != nil {
		return nil, err
	}

	// Return created user
	return &dtos.User{
		ID:        userID,
		FirstName: input.Firstname,
		LastName:  input.Lastname,
		Email:     input.Email,
		Role:      role,
	}, nil
}
func validateUniqueUserIdentifiers(email, phone string) error {
	if email != "" {
		if exists, _ := EmailExistsForOtherUser("userID", email); exists {
			return errors.New("email already exists for another user")
		}
	}
	if phone != "" {
		if exists, _ := PhoneExistsForOtherUser("userID", phone); exists {
			return errors.New("phone number already exists for another user")
		}
	}
	return nil
}
func resolveRoleID(inputRoleID string) (string, error) {
	// If role ID is provided, use it as-is
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
func insertUser(userID, roleID string, input dtos.RegisterRequest) error {
	columns := []string{"user_id", "role_id"}
	values := []interface{}{userID, roleID}

	addIfNotEmpty := func(field string, value string) {
		if value != "" {
			columns = append(columns, field)
			values = append(values, value)
		}
	}

	addIfNotEmpty("first_name", input.Firstname)
	addIfNotEmpty("last_name", input.Lastname)
	addIfNotEmpty("email", input.Email)
	addIfNotEmpty("phone_number", input.Phonenumber)

	query := fmt.Sprintf(
		"INSERT INTO users (%s) VALUES (%s)",
		strings.Join(columns, ", "),
		strings.Repeat("?, ", len(columns)-1)+"?",
	)

	_, err := DB.Exec(query, values...)
	return err
}

func UpdateLastLogin(userID string) error {
	_, err := DB.Exec("UPDATE users SET last_login = ? WHERE user_id = ?", time.Now(), userID)
	return err
}

func FindByIdAndUpdate(input dtos.RegisterRequest, userID string) (*dtos.Users, error) {
	log.Printf("user id***%s", userID)
	// Check if user exists
	err := isUserThere(userID)
	if err != nil {
		return nil, err
	}
	//check if email and phone number exists for other users
	if input.Email != "" {
		if exists, _ := EmailExistsForOtherUser(userID, input.Email); exists {
			return nil, errors.New("email already exists for another user")
		}
	}
	if input.Phonenumber != "" {
		if exists, _ := PhoneExistsForOtherUser(userID, input.Phonenumber); exists {
			return nil, errors.New("phone number already exists for another user")
		}
	}
	// Build SET clause dynamically
	setClauses := []string{}
	values := []interface{}{}

	if input.Firstname != "" {
		setClauses = append(setClauses, "first_name = ?")
		values = append(values, input.Firstname)
	}
	if input.Lastname != "" {
		setClauses = append(setClauses, "last_name = ?")
		values = append(values, input.Lastname)
	}
	if input.Email != "" {
		setClauses = append(setClauses, "email = ?")
		values = append(values, input.Email)
	}
	if input.Phonenumber != "" {
		setClauses = append(setClauses, "phone_number = ?")
		values = append(values, input.Phonenumber)
	}
	if input.RoleID != "" {
		setClauses = append(setClauses, "role_id = ?")
		values = append(values, input.RoleID)
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add userID for WHERE clause
	values = append(values, userID)

	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE user_id = ?`, strings.Join(setClauses, ", "))

	_, err = DB.Exec(query, values...)
	if err != nil {
		return nil, err
	}
	user, err := GetUserByUserID(userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}
func EmailExistsForOtherUser(userID, email string) (bool, error) {
	var count int
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

func PhoneExistsForOtherUser(userID, phone string) (bool, error) {
	var count int
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
func GetUserByResetToken(token string) (*dtos.User, error) {
	var email string
	err := DB.QueryRow("SELECT email FROM reset_tokens WHERE token = ?", token).Scan(&email)
	if err != nil {
		return nil, err
	}
	return GetUserByEmail(email)
}

// StoreOTP stores a new OTP for a user with an expiration time.
func StoreOTP(userID string, otp string, duration time.Duration) error {
	expiry := time.Now().Add(duration)
	otpID := uuid.New().String()

	query := `
		INSERT INTO otps (id, user_id, code, expires_at, used)
		VALUES (?, ?, ?, ?, FALSE)
	`
	_, err := DB.Exec(query, otpID, userID, otp, expiry)
	if err != nil {
		log.Printf("Failed to store OTP: %v", err)
	}
	return err
}

// VerifyOTP checks if the provided OTP is valid for the user and not expired.
func VerifyOTP(userID string, otp string) (bool, error) {
	var id string
	var expiresAt time.Time
	var used bool

	query := `
		SELECT id, expires_at, used
		FROM otps
		WHERE user_id = ? AND code = ?
		ORDER BY expires_at DESC
		LIMIT 1
	`

	err := DB.QueryRow(query, userID, otp).Scan(&id, &expiresAt, &used)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // OTP not found
		}
		log.Printf("Error querying OTP: %v", err)
		return false, err
	}

	if used || time.Now().After(expiresAt) {
		return false, nil
	}

	// Mark the OTP as used
	_, err = DB.Exec(`UPDATE otps SET used = TRUE WHERE id = ?`, id)
	if err != nil {
		log.Printf("Error updating OTP to used: %v", err)
		return false, err
	}

	return true, nil
}
