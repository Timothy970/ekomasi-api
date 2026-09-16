// Package models provides data access functions for the Ekomasi e-commerce platform.
//
// This file contains functions for managing vouchers (gift certificates):
//   - Voucher CRUD operations (create, retrieve, update, delete)
//   - Voucher code generation and validation
//   - Voucher purchase and redemption workflow
//   - Voucher design management
//   - Voucher history tracking and usage analytics
//   - Email scheduling for voucher delivery
//   - Pagination support for voucher listings
package models

import (
	"crypto/rand"
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

// StringToTime parses a datetime string into a time.Time object.
//
// This utility function attempts multiple common datetime formats:
//   - "2006-01-02 15:04:05" (datetime with seconds)
//   - "2006-01-02" (date only)
//   - "2006-01-02T15:04:05Z07:00" (RFC3339)
//   - "2006-01-02T15:04:05Z" (RFC3339 without timezone)
//
// Parameters:
//   - str: string - The datetime string to parse
//
// Returns:
//   - time.Time: Parsed time, or zero time if parsing fails
func StringToTime(str string) time.Time {
	// Define supported datetime formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}

	// Try each format until one succeeds
	for _, layout := range formats {
		t, err := time.Parse(layout, str)
		if err == nil {
			return t
		}
	}

	// Log error and return zero time if no format matched
	fmt.Println("Error parsing time:", str)
	return time.Time{}
}

// StringToBool converts a string to a boolean value.
//
// This utility function treats "true" and "1" as true,
// all other values as false.
//
// Parameters:
//   - str: string - The string to convert ("true", "1", etc.)
//
// Returns:
//   - bool: true if str is "true" or "1", false otherwise
func StringToBool(str string) bool {
	return str == "true" || str == "1"
}

// StringToFloat64 converts a string to a float64 value.
//
// This utility function parses numeric strings, returning 0 on error.
//
// Parameters:
//   - str: string - The numeric string to convert
//
// Returns:
//   - float64: Parsed float value, or 0.0 if parsing fails
func StringToFloat64(str string) float64 {
	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}
	return value
}

// IsVoucherThere validates that a voucher exists in the database.
//
// This helper function is used before voucher operations to ensure
// the voucher ID is valid.
//
// Parameters:
//   - voucherID: string - The unique voucher ID to check
//
// Returns:
//   - error: "voucher not found", database error, or nil if voucher exists
func IsVoucherThere(db DBExecutor, voucherID string) error {
	// Check voucher existence
	exists, err := RecordExists(db, "vouchers", "voucher_id = ?", voucherID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("voucher not found")
	}
	return nil
}

// secureRandomString generates a cryptographically secure random string.
//
// This helper function uses crypto/rand for secure random character selection
// from the alphabet (a-z, A-Z).
//
// Parameters:
//   - length: int - Number of characters to generate
//
// Returns:
//   - string: Random string of specified length
//   - error: Cryptographic error or nil on success
func secureRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, length)

	// Generate each character using crypto/rand
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

// GenerateVoucherCode creates a unique, secure voucher code.
//
// The generated code format is: "AbCD-1a2b3c4d5e"
//   - First 4 characters: Random letters (a-z, A-Z)
//   - Hyphen separator
//   - Last 12 characters: Hexadecimal (from 6 random bytes)
//
// Returns:
//   - string: Generated voucher code in format "XXXX-XXXXXXXXXXXX"
//   - error: Cryptographic error or nil on success
func GenerateVoucherCode() (string, error) {
	// Generate 4 random letters for prefix
	prefix, err := secureRandomString(4)
	if err != nil {
		return "", err
	}

	// Generate 6 random bytes (12 hex characters) for suffix
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	suffix := hex.EncodeToString(bytes)

	// Combine with hyphen separator
	return fmt.Sprintf("%s-%s", prefix, suffix), nil
}

// CreateVoucherOrder creates a new voucher purchase order.
//
// This function creates a pending order record for voucher purchase,
// tracking the amount and payment method.
//
// Parameters:
//   - amount: float64 - Voucher purchase amount
//   - voucherID: string - The voucher ID being purchased
//   - paymentMethod: string - Payment method (e.g., "MPESA", "CARD")
//
// Returns:
//   - string: Generated voucher order ID
//   - error: Database error or nil on success
func CreateVoucherOrder(db DBExecutor, amount float64, voucherID string, paymentMethod string) (string, error) {
	// Generate unique voucher order ID
	voucherOrderID, _ := shortid.Generate()

	// Insert voucher order with PENDING status
	query := `
		INSERT INTO voucher_orders (voucher_order_id, voucher_id, amount, status, payment_method)
		VALUES (?, ?, ?, ?,?)
	`
	_, err := db.Exec(query, voucherOrderID, voucherID, amount, "PENDING", strings.ToUpper(paymentMethod))
	if err != nil {
		return "", err
	}
	return voucherOrderID, nil
}

// UpdateVoucherPurchaseAmount increments the amount of an existing voucher order.
//
// This function adds to the existing voucher order amount and returns
// the voucher order ID.
//
// Parameters:
//   - amount: float64 - Amount to add to the existing order
//   - voucherID: string - The voucher ID to update
//
// Returns:
//   - string: The voucher order ID
//   - error: Database error or nil on success
func UpdateVoucherPurchaseAmount(db DBExecutor, amount float64, voucherID string) (string, error) {
	// Increment voucher order amount
	query := `
		UPDATE voucher_orders SET amount = amount + ? WHERE voucher_id = ?
	`
	_, err := db.Exec(query, amount, voucherID)
	if err != nil {
		return "", err
	}

	// Retrieve the voucher order ID
	selectQuery := `
		SELECT voucher_order_id FROM voucher_orders WHERE voucher_id = ?
	`
	var voucherOrderID string
	err = db.QueryRow(selectQuery, voucherID).Scan(&voucherOrderID)
	if err != nil {
		return "", err
	}
	return voucherOrderID, nil
}

// AddNewVoucher creates a new voucher with design validation.
//
// This function validates the design ID, generates a unique voucher code,
// and creates a voucher with the specified amount and expiry date.
//
// Parameters:
//   - v: dtos.Voucher containing:
//   - Amount: Voucher value
//   - ExpiryDate: Expiration date
//   - DesignID: Pointer to design ID (must be valid and active)
//   - Status: Pointer to status (defaults to "inactive" if nil)
//   - userID: string - ID of the user creating the voucher
//
// Returns:
//   - string: Generated voucher ID
//   - error: "design not found", "design not active", database error, or nil on success
func AddNewVoucher(db DBExecutor, v dtos.Voucher, userID string) (string, error) {
	// Validate design exists and is active
	err := ValidateDesignID(db, *v.DesignID)
	if err != nil {
		return "", err
	}

	// Generate unique voucher ID
	voucherID, _ := shortid.Generate()

	// Set default status
	status := "inactive"
	if v.Status != nil {
		status = *v.Status
	}

	// Generate unique voucher code
	code, _ := GenerateVoucherCode()

	// Insert new voucher (balance initially equals original value)
	query := `
		INSERT INTO vouchers (voucher_id, code, original_value, status,user_id, expiry_date, balance, design_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(query, voucherID, code, v.Amount, status, userID, v.ExpiryDate, v.Amount, v.DesignID)
	if err != nil {
		return "", err
	}
	return voucherID, nil
}

// InsertIntoVoucherPurchases creates a voucher purchase record.
//
// This function tracks the voucher purchase details including sender, recipient,
// personalized message, and delivery scheduling.
//
// Parameters:
//   - v: dtos.BuyVoucherData containing:
//   - ToName: Recipient name
//   - ToEmail: Recipient email address
//   - Message: Personalized message
//   - DeliveryTime: Scheduled delivery time
//   - FromName: Sender name
//   - userID: string - ID of the purchasing user
//   - voucherID: string - The voucher ID being purchased
//
// Returns:
//   - error: Database error or nil on success
func InsertIntoVoucherPurchases(db DBExecutor, v dtos.BuyVoucherData, userID, voucherID string) error {
	// Generate unique purchase ID
	purchaseID, _ := shortid.Generate()

	// Insert voucher purchase record with PENDING status
	query := `
		INSERT INTO voucher_purchases (purchase_id, voucher_id, from_user_id, to_name, to_email,personalized_msg, delivery_time, status, from_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query, purchaseID, voucherID, userID, v.ToName, v.ToEmail, v.Message, v.DeliveryTime, "PENDING", v.FromName)
	if err != nil {
		return err
	}
	return nil
}

// buildVoucherFilters constructs the WHERE clause and arguments for voucher queries.
func buildVoucherFilters(isRedeemed, status, code, customer string) (string, []interface{}) {
	query := ""
	args := []interface{}{}

	if isRedeemed != "" {
		query += " AND v.is_redeemed = ?"
		args = append(args, isRedeemed == "true")
	}

	if status != "" {
		query += " AND v.status = ?"
		args = append(args, status)
	}

	if code != "" {
		query += " AND v.code LIKE ?"
		args = append(args, "%"+code+"%")
	}

	if customer != "" {
		query += `
			AND (
				vp.to_email LIKE ?
				OR vp.to_name LIKE ?
				OR u_from.email LIKE ?
				OR u_from.phone_number LIKE ?
			)
		`
		args = append(args, "%"+customer+"%", "%"+customer+"%", "%"+customer+"%", "%"+customer+"%")
	}

	return query, args
}

// resolveVoucherParticipant resolves the From/To participant from multiple nullable fields.
func resolveVoucherParticipant(primary1, primary2, fallback1, fallback2 sql.NullString) *string {
	if primary1.Valid {
		return &primary1.String
	}
	if primary2.Valid {
		return &primary2.String
	}
	if fallback1.Valid {
		return &fallback1.String
	}
	if fallback2.Valid {
		return &fallback2.String
	}
	return nil
}

// scanVoucherRow scans a single voucher row and resolves participants.
func scanVoucherRow(rows *sql.Rows) (dtos.VoucherData, error) {
	var v dtos.VoucherData
	var (
		userID                 string
		fromEmail, fromPhone   sql.NullString
		toEmail, toName        sql.NullString
		ownerEmail, ownerPhone sql.NullString
	)

	if err := rows.Scan(
		&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.Status,
		&v.CreatedAt, &v.ExpiryDate, &userID, &v.IsReedemed,
		&fromEmail, &fromPhone, &toEmail, &toName, &ownerEmail, &ownerPhone,
	); err != nil {
		return dtos.VoucherData{}, fmt.Errorf("scan voucher failed: %w", err)
	}

	v.From = resolveVoucherParticipant(fromEmail, fromPhone, ownerEmail, ownerPhone)
	v.To = resolveVoucherParticipant(toEmail, toName, sql.NullString{}, sql.NullString{})

	return v, nil
}

// ListVouchers retrieves vouchers with pagination and dynamic filtering.
//
// This function supports filtering by redemption status, voucher status,
// code search, and customer search (name or email).
//
// Parameters:
//   - page: int - Page number (minimum 1)
//   - size: int - Items per page (minimum 1, defaults to 10)
//   - isRedeemed: string - Filter by redemption status ("true"/"false", empty for all)
//   - status: string - Filter by voucher status (e.g., "active", "inactive")
//   - code: string - Search by voucher code (partial match)
//   - customer: string - Search by customer name or email (partial match)
//
// Returns:
//   - []dtos.VoucherData: Array of vouchers with participant info
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func ListVouchers(
	db DBExecutor,
	page, size int,
	isRedeemed, status, code, customer string,
) ([]dtos.VoucherData, *dtos.PaginationMeta, error) {

	offset := (page - 1) * size

	// Base query with joins
	baseQuery := `
		FROM vouchers v
		LEFT JOIN voucher_purchases vp
			ON v.voucher_id = vp.voucher_id
		LEFT JOIN users u_from
			ON vp.from_user_id = u_from.user_id
		LEFT JOIN users u_owner
			ON v.user_id = u_owner.user_id
		WHERE 1=1
	`

	// Build filters
	filters, args := buildVoucherFilters(isRedeemed, status, code, customer)
	baseQuery += filters

	// Count total results
	countQuery := "SELECT COUNT(DISTINCT v.voucher_id) " + baseQuery
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count vouchers failed: %w", err)
	}

	// Build select query
	selectQuery := `
		SELECT DISTINCT
			v.voucher_id, v.code, v.balance, v.original_value, v.status,
			v.created_at, v.expiry_date, v.user_id, v.is_redeemed,
			u_from.email, u_from.phone_number, vp.to_email, vp.to_name,
			u_owner.email, u_owner.phone_number
	` + baseQuery + `
		ORDER BY v.created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, size, offset)
	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query vouchers failed: %w", err)
	}
	defer rows.Close()

	var vouchers []dtos.VoucherData
	for rows.Next() {
		v, err := scanVoucherRow(rows)
		if err != nil {
			return nil, nil, err
		}
		vouchers = append(vouchers, v)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: (total + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return vouchers, meta, nil
}

// getVoucherParticipants retrieves sender and recipient contact information.
//
// This helper function fetches the sender's contact info (email/phone) and
// recipient's email from voucher_purchases. Falls back to user table if
// voucher purchase record is not found.
//
// Parameters:
//   - voucherID: string - The voucher ID to get participants for
//   - userID: string - Fallback user ID if purchase record not found
//
// Returns:
//   - *string: Sender contact (email or phone)
//   - *string: Recipient email
//   - error: Database error or nil on success
func getVoucherParticipants(db DBExecutor, voucherID, userID string) (*string, *string, error) {
	log.Printf("getting voucher participants****")
	var phone, email, toEmail sql.NullString

	// Fetch sender details and recipient email from purchase record
	err := db.QueryRow(`
		SELECT u.phone_number, u.email, v.to_email
		FROM voucher_purchases v
		LEFT JOIN users u ON v.from_user_id = u.user_id
		WHERE v.voucher_id = ?
	`, voucherID).Scan(&phone, &email, &toEmail)

	if err != nil {
		if err == sql.ErrNoRows {
			// Fallback: Get phone/email for provided userID
			var fallbackPhone, fallbackEmail sql.NullString
			fallbackErr := db.QueryRow(`
				SELECT phone_number, email FROM users WHERE user_id = ?
			`, userID).Scan(&fallbackPhone, &fallbackEmail)

			if fallbackErr != nil {
				return nil, nil, fallbackErr
			}

			// Return email if available, otherwise phone
			if fallbackEmail.Valid {
				return &fallbackEmail.String, &fallbackEmail.String, nil
			}
			if fallbackPhone.Valid {
				return &fallbackPhone.String, &fallbackPhone.String, nil
			}
			return nil, nil, fmt.Errorf("no contact info found for user %s", userID)
		}
		return nil, nil, fmt.Errorf("failed to fetch voucher participants: %w", err)
	}

	// Return email if available, otherwise phone
	if email.Valid {
		return &email.String, &toEmail.String, nil
	}
	return &phone.String, &toEmail.String, nil
}

// GetVoucherByID retrieves detailed voucher information by ID.
//
// This function fetches complete voucher details including participant info
// and usage history.
//
// Parameters:
//   - voucherID: string - The unique voucher ID
//
// Returns:
//   - dtos.SingleVoucherData: Complete voucher data with history
//   - error: "voucher not found", database error, or nil on success
func GetVoucherByID(db DBExecutor, voucherID string) (dtos.SingleVoucherData, error) {
	// Validate voucher exists
	err := IsVoucherThere(db, voucherID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}

	var v dtos.SingleVoucherData
	var userID string
	query := `SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date, is_redeemed, user_id FROM vouchers WHERE voucher_id = ?`

	// Retrieve voucher basic info
	err = db.QueryRow(query, voucherID).Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.Status, &v.CreatedAt, &v.ExpiryDate, &v.IsReedemed, &userID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}

	// Get sender and recipient information
	v.From, v.To, err = getVoucherParticipants(db, v.VoucherID, userID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}

	// Get voucher usage history
	v.VoucherHistory, err = GetVoucherHistoryByVoucherID(db, voucherID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}

	return v, nil
}

// GetUserVoucherByID retrieves a specific voucher for a user.
//
// This function is similar to GetVoucherByID but validates user ownership.
//
// Parameters:
//   - voucherID: string - The unique voucher ID
//   - userID: string - The user ID to validate ownership
//
// Returns:
//   - dtos.SingleVoucherData: Complete voucher data with history
//   - error: "voucher not found", database error, or nil on success
func GetUserVoucherByID(db DBExecutor, voucherID, userID string) (dtos.SingleVoucherData, error) {
	// Validate voucher exists
	err := IsVoucherThere(db, voucherID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}

	var v dtos.SingleVoucherData
	// Query voucher with user ownership validation
	query := `SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date, is_redeemed FROM vouchers WHERE voucher_id = ? AND user_id = ?`

	err = db.QueryRow(query, voucherID, userID).Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.Status, &v.CreatedAt, &v.ExpiryDate, &v.IsReedemed)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}

	// Get participant information
	v.From, v.To, err = getVoucherParticipants(db, v.VoucherID, userID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}

	// Get voucher usage history
	v.VoucherHistory, err = GetVoucherHistoryByVoucherID(db, voucherID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}

	return v, nil
}

// GetUserVouchers retrieves all vouchers belonging to a user with pagination.
//
// Parameters:
//   - userID: string - The user ID to retrieve vouchers for
//   - page: int - Page number (1-based)
//   - limit: int - Items per page
//
// Returns:
//   - []dtos.VoucherData: Array of user's vouchers
//   - *PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func GetUserVouchers(db DBExecutor, userID string, page, limit int) ([]dtos.VoucherData, *PaginationMeta, error) {
	// Calculate offset for pagination
	offset := (page - 1) * limit

	// Count total vouchers for user
	var total int
	countQuery := `SELECT COUNT(*) FROM vouchers WHERE user_id = ?`
	if err := db.QueryRow(countQuery, userID).Scan(&total); err != nil {
		return nil, nil, err
	}

	// Fetch paginated vouchers
	query := `
		SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date
		FROM vouchers
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Process results
	var vouchers []dtos.VoucherData
	for rows.Next() {
		var v dtos.VoucherData
		if err := rows.Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.Status, &v.CreatedAt, &v.ExpiryDate); err != nil {
			return nil, nil, err
		}
		vouchers = append(vouchers, v)
	}

	// Build pagination metadata
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	pagination := &PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return vouchers, pagination, nil
}

// DeleteVoucher permanently removes a voucher from the system.
//
// Parameters:
//   - voucherID: string - The voucher ID to delete
//
// Returns:
//   - error: "voucher not found", database error, or nil on success
func DeleteVoucher(db DBExecutor, voucherID string) error {
	// Validate voucher exists
	err := IsVoucherThere(db, voucherID)
	if err != nil {
		return err
	}

	// Delete voucher record
	_, err = db.Exec(`DELETE FROM vouchers WHERE voucher_id = ?`, voucherID)
	if err != nil {
		return err
	}
	return nil
}

// VoucherUpdate updates voucher and purchase details.
//
// This function updates both the voucher table (amount, expiry, status)
// and the voucher_purchases table (recipient info, message, delivery time).
//
// Parameters:
//   - input: dtos.VoucherDataUpdate containing updated voucher data
//   - voucherID: string - The voucher ID to update
//
// Returns:
//   - error: "voucher not found", database error, or nil on success
func VoucherUpdate(db DBExecutor, input dtos.VoucherDataUpdate, voucherID string) error {
	// Validate voucher exists
	err := IsVoucherThere(db, voucherID)
	if err != nil {
		return err
	}
	status := "active"
	if input.Status != nil {
		status = *input.Status
	}

	query := `
		UPDATE vouchers
		SET original_value = ?, expiry_date = ?, status = ?
		WHERE voucher_id = ?
	`
	_, err = db.Exec(query, input.Amount, StringToTime(input.ExpiryDate), status, voucherID)
	if err != nil {
		return err
	}
	secondQuery := `
		UPDATE voucher_purchases
		SET to_name = ?, to_email = ?, personalized_msg = ?, delivery_time = ?, from_name = ?, notes = ?
		WHERE voucher_id = ?
	`
	_, err = db.Exec(secondQuery, input.ToName, input.ToEmail, input.Message, StringToTime(input.ExpiryDate), input.FromName, input.InternalNotes, voucherID)

	if err != nil {
		return err
	}
	return nil
}

// isTransactionIDUnique checks if a transaction ID is already in use.
//
// This validation prevents duplicate payment processing by ensuring
// each transaction ID is used only once.
//
// Parameters:
//   - id: string - The transaction ID to check
//
// Returns:
//   - error: "duplicate transaction id" if exists, database error, or nil if unique
func isTransactionIDUnique(db DBExecutor, id string) error {
	// Check if transaction ID exists in payments table
	exists, err := RecordExists(db, "payments", "transaction_id = ?", id)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("duplicate transaction id")
	}
	return nil
}

// IsVoucherThereByCode validates if a voucher code exists in the system.
//
// Parameters:
//   - code: string - The voucher code to validate
//
// Returns:
//   - error: "voucher with this code not found" or database error, nil if exists
func IsVoucherThereByCode(db DBExecutor, code string) error {
	// Check if voucher code exists
	exists, err := RecordExists(db, "vouchers", "code = ?", code)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("voucher with this code not found")
	}
	return nil
}

// RedeemVoucher processes voucher redemption by a user.
//
// This function validates the voucher (exists, active, not expired) and
// assigns it to the redeeming user. The voucher balance remains unchanged
// until used in a transaction.
//
// Parameters:
//   - code: string - The voucher code to redeem
//   - userID: string - The user redeeming the voucher
//
// Returns:
//   - *dtos.VoucherData: The redeemed voucher data
//   - error: Validation error (not found, inactive, expired) or database error
func RedeemVoucher(db DBExecutor, code, userID string) (*dtos.VoucherData, error) {
	// Validate voucher exists
	err := IsVoucherThereByCode(db, code)
	if err != nil {
		return nil, err
	}

	// Retrieve voucher details
	voucher, err := GetVoucherByCode(db, code)
	if err != nil {
		return nil, err
	}

	// Validate status is active
	if voucher.Status != "active" {
		return nil, fmt.Errorf("voucher is not active")
	}

	// Check voucher has not expired
	if voucher.ExpiryDate.Before(time.Now()) {
		return nil, fmt.Errorf("voucher has expired")
	}

	// Assign voucher to redeeming user
	_, err = db.Exec(`UPDATE vouchers SET user_id = ? WHERE code = ?`, userID, code)
	return &voucher, err
}

// GetVoucherByCode retrieves voucher details using the code.
//
// Parameters:
//   - code: string - The voucher code to retrieve
//
// Returns:
//   - dtos.VoucherData: Voucher data
//   - error: "voucher not found", database error, or nil on success
func GetVoucherByCode(db DBExecutor, code string) (dtos.VoucherData, error) {
	var v dtos.VoucherData
	// Retrieve voucher by code
	query := `SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date FROM vouchers WHERE code = ?`

	err := db.QueryRow(query, code).Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.Status, &v.CreatedAt, &v.ExpiryDate)
	if err != nil {
		return dtos.VoucherData{}, err
	}
	return v, nil
}

// GetUsersWithUnsentVoucherEmails retrieves vouchers ready for email delivery.
//
// This function finds all voucher purchases where:
// - Delivery time has arrived (delivery_time <= current time)
// - Email status is still PENDING
//
// Returns:
//   - []dtos.VoucherEmailInfo: Array of vouchers ready to send with sender/recipient details
//   - error: Database error or nil on success
func GetUsersWithUnsentVoucherEmails(db DBExecutor) ([]dtos.VoucherEmailInfo, error) {
	// Fetch vouchers ready for delivery
	rows, err := db.Query(`
		SELECT vp.voucher_id, vp.from_name, vp.to_name, vp.to_email, vp.personalized_msg, vp.delivery_time, v.original_value, v.expiry_date, v.code
		FROM voucher_purchases vp
		JOIN vouchers v ON vp.voucher_id = v.voucher_id
		WHERE vp.status = 'PENDING' AND vp.delivery_time <= ?`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process each pending voucher
	var infos []dtos.VoucherEmailInfo
	for rows.Next() {
		var info dtos.VoucherEmailInfo
		err := rows.Scan(&info.VoucherID, &info.FromName, &info.ToName, &info.ToEmail, &info.PersonalizedMsg, &info.DeliveryTime, &info.Amount, &info.ExpiryDate, &info.Code)
		if err != nil {
			return nil, err
		}
		log.Printf("message::::%s", info.PersonalizedMsg)
		infos = append(infos, info)
	}
	return infos, nil
}

// MarkVoucherEmailAsSent updates email delivery status to SENT.
//
// Call this after successfully sending the voucher email to prevent
// duplicate deliveries.
//
// Parameters:
//   - voucherID: string - The voucher ID that was emailed
//
// Returns:
//   - error: Database error or nil on success
func MarkVoucherEmailAsSent(db DBExecutor, voucherID string) error {
	// Update email status to SENT
	_, err := db.Exec(`UPDATE voucher_purchases SET status = 'SENT' WHERE voucher_id = ?`, voucherID)
	return err
}

// CreateVoucherDesign creates a new voucher design template.
//
// Design templates define the visual appearance of vouchers and can be
// reused for multiple vouchers. Designs can be active or inactive.
//
// Parameters:
//   - url: string - URL to the design image/template
//   - name: string - Name of the design template
//   - status: string - Status ("active" or "inactive")
//
// Returns:
//   - string: The generated design ID
//   - error: Database error or nil on success
func CreateVoucherDesign(db DBExecutor, url, name, status string) (string, error) {
	// Generate unique design ID
	designID, _ := shortid.Generate()

	// Insert design template
	query := `
		INSERT INTO voucher_designs (design_id, url, name, status)
		VALUES (?, ?, ?, ?)
	`
	_, err := db.Exec(query, designID, url, name, status)
	if err != nil {
		return "", err
	}
	return designID, nil
}

// GetVoucherDesign retrieves a specific design template.
//
// Parameters:
//   - designID: string - The unique design ID
//
// Returns:
//   - dtos.VoucherDesign: Design template data with URL, name, status, created date
//   - error: "design not found", database error, or nil on success
func GetVoucherDesign(db DBExecutor, designID string) (dtos.VoucherDesign, error) {
	// Validate design exists
	err := isVoucherDesignThere(db, designID)
	if err != nil {
		return dtos.VoucherDesign{}, err
	}

	var design dtos.VoucherDesign
	// Retrieve design by ID
	query := `SELECT url, name, status, created_at FROM voucher_designs WHERE design_id = ? LIMIT 1`
	err = db.QueryRow(query, designID).Scan(&design.URL, &design.Name, &design.Status, &design.Created_At)
	if err != nil {
		if err == sql.ErrNoRows {
			return dtos.VoucherDesign{}, fmt.Errorf("design not found")
		}
		return dtos.VoucherDesign{}, err
	}
	return design, nil
}

// EditVoucherDesign updates an existing design template.
//
// The URL is optional - if nil or empty, it won't be updated.
//
// Parameters:
//   - designID: string - The design ID to update
//   - newURL: *string - Optional new design URL (nil to keep existing)
//   - newName: string - New design name
//   - newStatus: string - New status ("active" or "inactive")
//
// Returns:
//   - error: "design not found", database error, or nil on success
func EditVoucherDesign(db DBExecutor, designID string, newURL *string, newName, newStatus string) error {
	// Validate design exists
	err := isVoucherDesignThere(db, designID)
	if err != nil {
		return err
	}

	// Build dynamic query - only update URL if provided
	query := `UPDATE voucher_designs SET name = ?, status = ?`
	args := []interface{}{newName, newStatus}

	if newURL != nil && *newURL != "" {
		query += `, url = ?`
		args = append(args, *newURL)
	}

	query += ` WHERE design_id = ?`
	args = append(args, designID)

	_, err = db.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}

// DeleteVoucherDesign permanently removes a design template.
//
// Parameters:
//   - designID: string - The design ID to delete
//
// Returns:
//   - error: "design not found", database error, or nil on success
func DeleteVoucherDesign(db DBExecutor, designID string) error {
	// Validate design exists
	err := isVoucherDesignThere(db, designID)
	if err != nil {
		return err
	}

	// Delete design template
	query := `DELETE FROM voucher_designs WHERE design_id = ?`
	_, err = db.Exec(query, designID)
	if err != nil {
		return err
	}

	return nil
}

// GetAllVoucherDesigns retrieves design templates with pagination and filtering.
//
// Parameters:
//   - page: int - Page number (1-based)
//   - size: int - Items per page
//   - name: string - Filter by design name (partial match, optional)
//   - status: string - Filter by status (exact match, optional)
//
// Returns:
//   - []dtos.VoucherDesign: Array of design templates
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func GetAllVoucherDesigns(db DBExecutor, page, size int, name, status string) ([]dtos.VoucherDesign, *dtos.PaginationMeta, error) {
	var (
		total int
		args  []interface{}
	)

	// Build count query with optional filters
	countQuery := `SELECT COUNT(*) FROM voucher_designs`
	if name != "" {
		countQuery += " WHERE name LIKE ?"
		args = append(args, "%"+name+"%")
	}
	if status != "" {
		if len(args) == 0 {
			countQuery += " WHERE status = ?"
		} else {
			countQuery += " AND status = ?"
		}
		args = append(args, status)
	}

	// Count total matching designs
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count query failed: %w", err)
	}

	// Build select query with same filters
	selectQuery := `
		SELECT design_id, url, created_at, name, status
		FROM voucher_designs
	`
	var queryArgs []interface{}

	if name != "" {
		selectQuery += " WHERE name LIKE ?"
		queryArgs = append(queryArgs, "%"+name+"%")
	}
	if status != "" {
		if len(queryArgs) == 0 {
			selectQuery += " WHERE status = ?"
		} else {
			selectQuery += " AND status = ?"
		}
		queryArgs = append(queryArgs, status)
	}

	selectQuery += " ORDER BY created_at DESC LIMIT ?, ?"

	// Add pagination
	queryArgs = append(queryArgs, (page-1)*size, size)

	rows, err := db.Query(selectQuery, queryArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("select query failed: %w", err)
	}
	defer rows.Close()

	// ----- SCAN RESULTS -----
	var designs []dtos.VoucherDesign
	for rows.Next() {
		var d dtos.VoucherDesign
		if err := rows.Scan(
			&d.DesignID,
			&d.URL,
			&d.Created_At,
			&d.Name,
			&d.Status,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}
		designs = append(designs, d)
	}

	// ----- PAGINATION META -----
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: (total + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return designs, meta, nil
}

// GetVoucherHistoryByVoucherID retrieves usage history for a voucher.
//
// This function fetches all redemption records including dates, amounts,
// and items purchased (stored as JSON).
//
// Parameters:
//   - voucherID: string - The voucher ID to get history for
//
// Returns:
//   - []map[string]any: Array of history records with items_log unmarshaled from JSON
//   - error: Database error, JSON unmarshal error, or nil on success
func GetVoucherHistoryByVoucherID(db DBExecutor, voucherID string) ([]map[string]any, error) {
	// Fetch all history records for voucher
	query := `
		SELECT history_id, redeemed_date, amount_redeemed, items_log
		FROM vouchers_history
		WHERE voucher_id = ?
		ORDER BY redeemed_date DESC
	`

	rows, err := db.Query(query, voucherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []map[string]any

	// Process each history record
	for rows.Next() {
		var (
			historyID      string
			redeemedDate   time.Time
			amountRedeemed float64
			itemsLog       string
		)

		if err := rows.Scan(&historyID, &redeemedDate, &amountRedeemed, &itemsLog); err != nil {
			return nil, err
		}

		// Unmarshal items log from JSON string
		var itemsLogSlice []map[string]any
		if itemsLog != "" {
			itemsLogSlice = make([]map[string]any, 0)
			if err := json.Unmarshal([]byte(itemsLog), &itemsLogSlice); err != nil {
				return nil, err
			}
		}

		// Build history record
		history := map[string]any{
			"history_id":      historyID,
			"redeemed_date":   redeemedDate,
			"amount_redeemed": amountRedeemed,
			"items_log":       itemsLogSlice,
		}

		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return histories, nil
}

// isVoucherDesignThere checks if a design template exists.
//
// Parameters:
//   - designID: string - The design ID to validate
//
// Returns:
//   - error: "voucher design not found" or database error, nil if exists
func isVoucherDesignThere(db DBExecutor, designID string) error {
	// Check if design exists
	exists, err := RecordExists(db, "voucher_designs", "design_id = ?", designID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("voucher design not found")
	}

	return nil
}

// IsVoucherDesignActive checks if a design template is active.
//
// Parameters:
//   - designID: string - The design ID to check
//
// Returns:
//   - error: "voucher design is not active" or database error, nil if active
func IsVoucherDesignActive(db DBExecutor, designID string) error {
	var status string
	// Retrieve design status
	err := db.QueryRow(`SELECT status FROM voucher_designs WHERE design_id = ?`, designID).Scan(&status)
	if err != nil {
		return err
	}

	// Validate status is active
	if status != "active" {
		return errors.New("voucher design is not active")
	}
	return nil
}

// ValidateDesignID performs complete design validation.
//
// This function checks both existence and active status in one call.
//
// Parameters:
//   - designID: string - The design ID to validate
//
// Returns:
//   - error: "design not found", "design not active", database error, or nil if valid
func ValidateDesignID(db DBExecutor, designID string) error {
	// Check design exists
	err := isVoucherDesignThere(db, designID)
	if err != nil {
		return err
	}

	// Check design is active
	err = IsVoucherDesignActive(db, designID)
	if err != nil {
		return err
	}
	return nil
}

// CreateNewVoucher creates a complete voucher with purchase tracking.
//
// This is the main voucher creation workflow that:
// 1. Validates design exists
// 2. Generates unique voucher ID and code
// 3. Creates voucher record with initial balance = original value
// 4. Creates purchase record with sender/recipient details
//
// Parameters:
//   - v: dtos.VoucherDataCreate containing voucher details (design, amount, expiry, recipient info)
//   - userID: string - The purchasing user ID
//
// Returns:
//   - string: The generated voucher ID
//   - error: "design not found", database error, or nil on success
func CreateNewVoucher(db DBExecutor, v dtos.VoucherDataCreate, userID string) (string, error) {
	// Validate design exists
	err := isVoucherDesignThere(db, v.DesignID)
	if err != nil {
		return "", err
	}

	// Generate unique voucher ID and code
	voucherID, _ := shortid.Generate()
	code, _ := GenerateVoucherCode()

	// Create voucher record with initial balance = original value
	query := `
		INSERT INTO vouchers (voucher_id, design_id, user_id, code, balance, original_value, expiry_date, status, is_redeemed, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(query, voucherID, v.DesignID, userID, code, v.Amount, v.Amount, StringToTime(v.ExpiryDate), "active", false, v.InternalNotes)
	if err != nil {
		return "", err
	}

	// Parse and format delivery time
	deliveryTime := StringToTime(v.DeliveryTime)

	// Create purchase record with sender/recipient details
	err = InsertIntoVoucherPurchases(db, dtos.BuyVoucherData{
		DesignID:     v.DesignID,
		Amount:       v.Amount,
		FromName:     v.FromName,
		ToName:       v.ToName,
		ToEmail:      v.ToEmail,
		Message:      v.Message,
		DeliveryTime: deliveryTime.Format("2006-01-02 15:04:05"),
	}, userID, voucherID)
	if err != nil {
		return "", err
	}
	return voucherID, nil
}

// ListVoucherPurchases retrieves voucher purchases with pagination and filtering.
//
// This function fetches purchase records with participant details and design info.
//
// Parameters:
//   - page: int - Page number (1-based)
//   - size: int - Items per page
//   - name: string - Filter by sender/recipient name or email (partial match, optional)
//
// Returns:
//   - []dtos.VoucherPurchaseData: Array of voucher purchases
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or nil on success
func ListVoucherPurchases(db DBExecutor, page, size int, name string) ([]dtos.VoucherPurchaseData, *dtos.PaginationMeta, error) {
	// Calculate offset for pagination
	offset := (page - 1) * size

	// Build base query with optional name filter
	baseQuery := `
		FROM voucher_purchases vp
		JOIN vouchers v ON vp.voucher_id = v.voucher_id
		LEFT JOIN voucher_designs vd ON v.design_id = vd.design_id
		WHERE 1=1
	`
	args := []interface{}{}

	// Apply name filter (matches sender/recipient name or email)
	if name != "" {
		baseQuery += " AND (vp.from_name LIKE ? OR vp.to_name LIKE ? OR vp.to_email LIKE ?)"
		nameLike := "%" + name + "%"
		args = append(args, nameLike, nameLike, nameLike)
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("failed to count voucher purchases: %w", err)
	}

	var voucherUpdateTime time.Time
	selectQuery := `
		SELECT 
			v.voucher_id, v.code, v.balance, v.original_value, 
			vp.from_name, vp.to_name, vp.to_email, vp.personalized_msg, vp.from_user_id, vd.url,
			vp.created_at, v.updated_at
	` + baseQuery + `
		ORDER BY v.updated_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, size, offset)

	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query voucher purchases: %w", err)
	}
	defer rows.Close()

	var vouchers []dtos.VoucherPurchaseData
	for rows.Next() {
		var v dtos.VoucherPurchaseData
		var personalizedMsg sql.NullString
		var designURL sql.NullString
		var createdAt time.Time
		var fromUserID string
		if err := rows.Scan(
			&v.VoucherID,
			&v.Code,
			&v.Balance,
			&v.Amount,
			&v.FromName,
			&v.ToName,
			&v.ToEmail,
			&personalizedMsg,
			&fromUserID,
			&designURL,
			&createdAt,
			&voucherUpdateTime,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan voucher purchase row: %w", err)
		}
		if personalizedMsg.Valid {
			personalizedMsgStr := personalizedMsg.String
			v.Message = &personalizedMsgStr
		}
		if designURL.Valid {
			designURLStr := designURL.String
			v.DesignURL = &designURLStr
		}
		v.CreatedAt = &createdAt

		v.FromEmail, _, err = getVoucherParticipants(db, v.VoucherID, fromUserID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get voucher participants: %w", err)
		}

		vouchers = append(vouchers, v)
	}

	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: (total + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return vouchers, &meta, nil
}

// GetVoucherPurchases retrieves detailed purchase information for a voucher.
//
// This function fetches purchase details including sender/recipient info,
// delivery time, personalized message, and design URL.
//
// Parameters:
//   - voucherID: string - The voucher ID to get purchase details for
//
// Returns:
//   - dtos.VoucherPurchaseData: Complete purchase data with participant info
//   - error: Database error or nil on success
func GetVoucherPurchases(db DBExecutor, voucherID string) (dtos.VoucherPurchaseData, error) {
	var (
		result          dtos.VoucherPurchaseData
		personalizedMsg sql.NullString
		designURL       sql.NullString
		fromUserID      string
		createdAt       time.Time
		deliveryTime    time.Time
	)

	baseQuery := `
		FROM voucher_purchases vp
		JOIN vouchers v ON vp.voucher_id = v.voucher_id
		LEFT JOIN voucher_designs vd ON v.design_id = vd.design_id
		WHERE v.voucher_id = ?
	`

	selectQuery := `
		SELECT 
			v.voucher_id,
			v.code,
			v.balance,
			v.original_value,
			vp.from_name,
			vp.to_name,
			vp.to_email,
			vp.personalized_msg,
			vp.from_user_id,
			vd.url,
			vp.created_at,
			vp.delivery_time
	` + baseQuery + `
		ORDER BY vp.created_at DESC
	`

	err := db.QueryRow(selectQuery, voucherID).Scan(
		&result.VoucherID,
		&result.Code,
		&result.Balance,
		&result.Amount,
		&result.FromName,
		&result.ToName,
		&result.ToEmail,
		&personalizedMsg,
		&fromUserID,
		&designURL,
		&createdAt,
		&deliveryTime,
	)

	if err != nil {
		return dtos.VoucherPurchaseData{}, fmt.Errorf("failed to scan voucher purchase row: %w", err)
	}

	// Handle nullable fields
	if personalizedMsg.Valid {
		msg := personalizedMsg.String
		result.Message = &msg
	}

	if designURL.Valid {
		url := designURL.String
		result.DesignURL = &url
	}

	result.CreatedAt = &createdAt
	if !deliveryTime.IsZero() {
		deliveryTimeStr := deliveryTime.Format("2006-01-02")
		result.DeliveryTime = &deliveryTimeStr
	}

	// Fetch participants (e.g. sender email)
	result.FromEmail, _, err = getVoucherParticipants(db, result.VoucherID, fromUserID)
	if err != nil {
		return dtos.VoucherPurchaseData{}, fmt.Errorf("failed to get voucher participants: %w", err)
	}

	return result, nil
}

// UpdateVoucher updates voucher balance, status, and design.
//
// This function increments the voucher balance (for adding value) and
// updates status and design ID.
//
// Parameters:
//   - voucherID: string - The voucher ID to update
//   - amount: float64 - Amount to add to balance (can be negative)
//   - status: string - New voucher status
//   - designID: string - New design ID
//
// Returns:
//   - error: "voucher not found", database error, or nil on success
func UpdateVoucher(db DBExecutor, voucherID string, amount float64, status string, designID string) error {
	// Validate voucher exists
	err := IsVoucherThere(db, voucherID)
	if err != nil {
		return err
	}

	// Update balance, status, and design
	query := `
		UPDATE vouchers SET balance = balance + ?, status = ?, design_id = ?
		WHERE voucher_id = ?
	`
	_, err = db.Exec(query, amount, status, designID, voucherID)
	if err != nil {
		return err
	}
	return nil
}

// UpdateVoucherPurchases updates purchase details for a voucher.
//
// This function modifies sender/recipient information, personalized message,
// and delivery time for an existing voucher purchase.
//
// Parameters:
//   - v: dtos.BuyVoucherData containing updated purchase details
//   - voucherID: string - The voucher ID to update
//
// Returns:
//   - error: Database error or nil on success
func UpdateVoucherPurchases(db DBExecutor, v dtos.BuyVoucherData, voucherID string) error {
	// Update purchase details (recipient info, message, delivery time, sender name)
	query := `
		UPDATE voucher_purchases SET  to_name = ?, to_email = ?, personalized_msg = ?, delivery_time = ?, from_name = ?
		WHERE voucher_id = ?
	`
	_, err := db.Exec(query, v.ToName, v.ToEmail, v.Message, v.DeliveryTime, v.FromName, voucherID)
	if err != nil {
		return err
	}
	return nil
}

// helper function to mark voucher status as aactive and update voucher order payment status to paid
// parameters:
//   - voucherID: string - The voucher ID to update
//
// returns:
//   - error: Database error or nil on success
func MarkVoucherAsPaid(db DBExecutor, voucherID string) error {
	// Update voucher status to active and payment status to paid
	query := `
		UPDATE vouchers SET status = 'active' WHERE voucher_id = ?
	`
	_, err := db.Exec(query, voucherID)
	if err != nil {
		return err
	}
	// update voucher order payment status to paid
	orderQuery := `
		UPDATE voucher_orders SET status = 'COMPLETED' WHERE voucher_id = ?
	`
	_, err = db.Exec(orderQuery, voucherID)
	if err != nil {
		return err
	}
	return nil
}
