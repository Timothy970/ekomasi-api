package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"time"

	"github.com/teris-io/shortid"
)

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
	args := []any{}

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
