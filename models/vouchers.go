package models

import (
	"crypto/rand"
	"ekomasi_backend/dtos"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

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
func buildVoucherFilters(isRedeemed, status, code, customer string) (string, []any) {
	query := ""
	args := []any{}

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
