// Package models provides data access functions for the Ekomasi e-commerce logging system.
//
// This file handles audit logging and activity tracking including:
//   - User activity log retrieval with advanced filtering (module, status, role, date range, search)
//   - Optimized pagination with separate count queries for performance
//   - Metadata parsing from JSON (function, IP address, description, module)
//   - User enrichment with full user details or unknown user placeholder
//   - Log filtering and search across user attributes
//   - Date range filtering with proper timestamp handling
package models

import (
	"ekomasi_backend/dtos"
	"encoding/json"
	"log"
)

// Parameters:
//   - data: JSON string to unmarshal
//
// Returns:
//   - map[string]any: Unmarshaled map
//   - error: JSON unmarshal error if data is invalid
func ConvertStringToMap(data string) (map[string]any, error) {
	var result map[string]any
	err := json.Unmarshal([]byte(data), &result)
	return result, err
}

// GetUserLogsByUserID retrieves paginated logs for a specific user.
//
// This function fetches activity logs for a single user with pagination,
// enriching logs with metadata and full user details.
//
// Parameters:
//   - userID: The user_id to fetch logs for (validated for existence)
//   - limit: Number of logs per page
//   - page: Page number (1-indexed)
//
// Returns:
//   - []dtos.UserLog: Array of logs for the user with:
//   - LogID, Status, Action, Module, CreatedAt
//   - User: Full user details
//   - Metadata fields: IPAddress, Description (parsed from JSON)
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: "user not found" if user doesn't exist,
//     or database error
func GetUserLogsByUserID(db DBExecutor, userID string, limit, page int) ([]dtos.UserLog, *dtos.PaginationMeta, error) {
	// Validate user exists
	if err := isUserThere(db, userID); err != nil {
		return nil, nil, err
	}

	log.Printf("Fetching logs for user ID: %s", userID)

	// Calculate pagination offset
	offset := (page - 1) * limit

	// Count total logs for pagination metadata
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM logs WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, nil, err
	}

	// Fetch log rows ordered by most recent
	query := `
		SELECT log_id, user_id, level, metadata, message, timestamp
		FROM logs
		WHERE user_id = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?`

	rows, err := db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Process rows into DTOs with metadata parsing and user enrichment
	logs, err := processUserLogRows(db, rows)
	if err != nil {
		log.Printf("Error processing logs for user %s: %v", userID, err)
		return nil, nil, err
	}

	log.Printf("Fetched %d logs for user ID %s", len(logs), userID)

	// Build pagination metadata
	meta := buildPagination(limit, offset, total)
	return logs, meta, nil
}
