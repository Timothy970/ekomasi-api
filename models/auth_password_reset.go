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
	"log"
	"time"

	"ekomasi_backend/dtos"

	"github.com/google/uuid"
)

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
func GetUserByResetToken(db DBExecutor, token string) (*dtos.User, error) {
	var email string
	var tenantID int
	// Look up email and tenant_id associated with this reset token
	err := db.QueryRow(`
		SELECT r.email, u.tenant_id 
		FROM reset_tokens r 
		JOIN users u ON r.email = u.email 
		WHERE r.token = ?`, token).Scan(&email, &tenantID)
	if err != nil {
		return nil, err
	}
	// Retrieve full user object by email
	return GetUserByEmail(db, email, tenantID)
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
func StoreOTP(db DBExecutor, userID string, otp string, duration time.Duration) error {
	// Calculate expiration time
	expiry := time.Now().Add(duration)
	// Generate unique OTP record ID
	otpID := uuid.New().String()

	// Insert OTP record with expiration and unused status
	query := `
		INSERT INTO otps (id, user_id, code, expires_at, used)
		VALUES (?, ?, ?, ?, FALSE)
	`
	_, err := db.Exec(query, otpID, userID, otp, expiry)
	if err != nil {
		log.Printf("Failed to store OTP: %v", err) // Log error for debugging
	}
	return err
}
