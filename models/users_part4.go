// Package models provides data access functions for the Ekomasi e-commerce platform.
//
// This file contains functions for managing users and related data:
//   - User CRUD operations (activate, deactivate, delete)
//   - User address management (create, retrieve, update, delete)
//   - User listing with pagination, search, and role filtering
//   - Token-based user retrieval
//   - Purchase history and wishlist category analysis
//   - Product recommendations based on user behavior
//   - Newsletter subscription management
package models

import (
	"ekomasi_backend/dtos"
	"fmt"
	"math"
	"strings"

	"github.com/teris-io/shortid"
)

// Create partner
// Parameters:
// - name: string - The name of the partner
// - image: string - The URL of the partner's image
// Returns:
// - error: Database error or nil on success
func CreatePartner(db DBExecutor, req dtos.Partner) error {
	// Generate unique partner ID
	partnerID, _ := shortid.Generate()
	// Insert new partner record
	_, err := db.Exec(`
		INSERT INTO partners (partner_id, name, image)
		VALUES (?, ?, ?)`,
		partnerID, req.Name, req.Image,
	)
	return err
}

// GetAllPartners retrieves all partners from the database.
//
// This function returns partners ordered by creation date (newest first).
func GetAllPartners(db DBExecutor) ([]dtos.Partner, error) {
	rows, err := db.Query(`
		SELECT partner_id, name, image
		FROM partners
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var partners []dtos.Partner
	for rows.Next() {
		var p dtos.Partner
		if err := rows.Scan(&p.PartnerID, &p.Name, &p.ImageURL); err != nil {
			return nil, err
		}
		partners = append(partners, p)
	}
	return partners, nil
}

// Delete a partner by ID
// Parameters:
// - partnerID: string - The unique ID of the partner to delete
// Returns:
// - error: "partner not found", database error, or nil on success
func DeletePartnerByID(db DBExecutor, partnerID string) error {
	// Check if partner exists
	exists, err := RecordExists(db, "partners", "partner_id = ?", partnerID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("partner not found")
	}
	// Delete partner record
	_, err = db.Exec("DELETE FROM partners WHERE partner_id = ?", partnerID)
	return err
}

// GetAllSubscribers retrieves all newsletter subscribers.
//
// This function returns subscribers ordered by creation date (newest first).
//
// Parameters:
//   - db: DBExecutor - The database executor
//
// filters:
//
//	q - string - Search query (case-insensitive) across email
//
// startDate - string - Filter subscribers created after this date (YYYY-MM-DD)
// endDate - string - Filter subscribers created before this date (YYYY-MM-DD)
//
// Returns:
//   - []dtos.Subscriber: List of subscribers
//   - error: Database error or nil on success
func GetAllSubscribers(db DBExecutor, size, limit int, q string, startDate string, endDate string) ([]dtos.Subscriber, *dtos.PaginationMeta, error) {
	var conditions []string
	var args []interface{}

	countQuery := "SELECT COUNT(*) FROM subscribers"
	selectQuery := `
		SELECT subscriber_id, email, created_at
		FROM subscribers`

	// Add search filter
	if q != "" {
		conditions = append(conditions, "LOWER(email) LIKE ?")
		args = append(args, "%"+strings.ToLower(q)+"%")
	}
	// Add date filters
	if startDate != "" && endDate != "" {
		newStartDate := StringToTime(startDate)
		newEndDate := StringToTime(endDate)
		conditions = append(conditions, "created_at BETWEEN ? AND ?")
		args = append(args, newStartDate, newEndDate)
	}
	// Combine conditions
	if len(conditions) > 0 {
		whereClause := " WHERE " + strings.Join(conditions, " AND ")
		countQuery += whereClause
		selectQuery += whereClause
	}
	// Add ordering and pagination
	selectQuery += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	argsWithPagination := append(args, size, limit)

	// Execute count query
	var totalItems int
	if err := db.QueryRow(countQuery, args...).Scan(&totalItems); err != nil {
		return nil, nil, fmt.Errorf("failed to count subscribers: %w", err)
	}
	// Execute select query
	rows, err := db.Query(selectQuery, argsWithPagination...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query subscribers: %w", err)
	}
	defer rows.Close()

	var subscribers []dtos.Subscriber
	for rows.Next() {
		var s dtos.Subscriber
		if err := rows.Scan(&s.SubscriberID, &s.Email, &s.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("failed to scan subscriber: %w", err)
		}
		subscribers = append(subscribers, s)
	}

	page := (limit / size) + 1
	totalPages := int(math.Ceil(float64(totalItems) / float64(size)))
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
	return subscribers, meta, nil
}
