package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

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
	args := []any{newName, newStatus}

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
		args  []any
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
	var queryArgs []any

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
