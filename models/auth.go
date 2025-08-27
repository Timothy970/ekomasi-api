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

var DB *sql.DB // to be initialized in your db connection setup

// GetUserByEmail retrieves a user by their email address.
// It returns a User object or nil if no user is found.
func GetUserByEmail(email string) (*dtos.User, error) {
	row := DB.QueryRow("SELECT user_id, first_name, last_name, email, role, phone_number FROM users WHERE email = ?", email)

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
	row := DB.QueryRow("SELECT user_id, first_name, last_name, email, role, phone_number FROM users WHERE phone_number = ?", phone)

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
func isProductThere(id string) error {
	exists, err := RecordExists("products", "product_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("product not found")
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
func GetUserByUserID(id string) (*dtos.User, error) {
	err := isUserThere(id)
	if err != nil {
		return nil, err
	}

	row := DB.QueryRow("SELECT user_id, first_name, last_name, email, role, phone_number FROM users WHERE user_id = ?", id)

	var user dtos.User
	var phone sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var userEmail sql.NullString

	err = row.Scan(&user.ID, &firstName, &lastName, &userEmail, &user.Role, &phone)
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
func CreateUser(input dtos.RegisterRequest) (*dtos.User, error) {
	//check if email and phone number exists for other users
	if input.Email != "" {
		if exists, _ := EmailExistsForOtherUser("userID", input.Email); exists {
			return nil, errors.New("email already exists for another user")
		}
	}
	if input.Phonenumber != "" {
		if exists, _ := PhoneExistsForOtherUser("userID", input.Phonenumber); exists {
			return nil, errors.New("phone number already exists for another user")
		}
	}

	userID, _ := shortid.Generate()
	role := ""
	if input.Role == "" {
		role = "customer"
	} else {
		role = input.Role
	}
	// Start building columns and values
	columns := []string{"user_id", "role"}
	values := []interface{}{userID, role}

	if input.Firstname != "" {
		columns = append(columns, "first_name")
		values = append(values, input.Firstname)
	}
	if input.Lastname != "" {
		columns = append(columns, "last_name")
		values = append(values, input.Lastname)
	}
	if input.Email != "" {
		columns = append(columns, "email")
		values = append(values, input.Email)
	}

	if input.Phonenumber != "" {
		columns = append(columns, "phone_number")
		values = append(values, input.Phonenumber)
	}

	// Construct the dynamic SQL query
	query := fmt.Sprintf("INSERT INTO users (%s) VALUES (%s)",
		strings.Join(columns, ", "),
		strings.Repeat("?, ", len(columns)-1)+"?",
	)

	// Execute the query
	_, err := DB.Exec(query, values...)
	if err != nil {
		return nil, err
	}

	// Return created user object
	return &dtos.User{
		ID:        userID,
		FirstName: input.Firstname,
		LastName:  input.Lastname,
		Email:     input.Email,
		Role:      role,
	}, nil
}

func UpdateLastLogin(userID string) error {
	_, err := DB.Exec("UPDATE users SET last_login = ? WHERE user_id = ?", time.Now(), userID)
	return err
}

func FindByIdAndUpdate(input dtos.RegisterRequest, userID string) error {
	log.Printf("user id***%s", userID)
	// Check if user exists
	err := isUserThere(userID)
	if err != nil {
		return err
	}
	//check if email and phone number exists for other users
	if input.Email != "" {
		if exists, _ := EmailExistsForOtherUser(userID, input.Email); exists {
			return errors.New("email already exists for another user")
		}
	}
	if input.Phonenumber != "" {
		if exists, _ := PhoneExistsForOtherUser(userID, input.Phonenumber); exists {
			return errors.New("phone number already exists for another user")
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
	if input.Role != "" {
		setClauses = append(setClauses, "role = ?")
		values = append(values, input.Role)
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Add userID for WHERE clause
	values = append(values, userID)

	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE user_id = ?`, strings.Join(setClauses, ", "))

	_, err = DB.Exec(query, values...)
	if err != nil {
		return err
	}

	return nil
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
