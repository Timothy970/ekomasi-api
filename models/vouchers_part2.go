package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"log"
)

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
