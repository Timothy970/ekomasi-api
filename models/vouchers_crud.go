package models

import (
	"ekomasi_backend/dtos"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/teris-io/shortid"
)

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
