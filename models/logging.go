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
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// UserLogFilters holds filtering criteria for user log queries.
//
// This struct encapsulates all filter parameters used to narrow down
// log results with pagination support.
//
// Fields:
//   - Module: Filter by log module ("all" for no filter)
//   - Status: Filter by log level/status ("all" for no filter)
//   - Role: Filter by user role ("all" for no filter)
//   - StartDate: Start date for date range filter (empty for no start limit)
//   - EndDate: End date for date range filter (empty for no end limit)
//   - Search: Search term for message and user attributes (empty for no search)
//   - Page: Current page number (1-indexed)
//   - Limit: Number of items per page
type UserLogFilters struct {
	Module    string
	Status    string
	Role      string
	StartDate string
	EndDate   string
	Search    string
	Page      int
	Limit     int
}

// GetUserLogsOptimized retrieves user activity logs with advanced filtering and pagination.
//
// This function is optimized for MySQL performance using separate count and data queries,
// avoiding CTEs. It enriches logs with user details and parses JSON metadata.
//
// Parameters:
//   - filters: UserLogFilters containing:
//   - Module: Log module filter ("all" for no filter)
//   - Status: Log level/status filter ("all" for no filter)
//   - Role: User role filter ("all" for no filter)
//   - StartDate: Date range start (empty for no start limit)
//   - EndDate: Date range end (empty for no end limit, auto-adds 23:59:59)
//   - Search: Search term for message and user attributes
//   - Page: Page number (1-indexed)
//   - Limit: Items per page
//
// Returns:
//   - []dtos.UserLog: Array of logs with:
//   - LogID, Status, Action, Module, CreatedAt
//   - User: Full user details or "Unknown User" placeholder
//   - Metadata fields: IPAddress, Description (parsed from JSON)
//   - *dtos.PaginationMeta: Pagination metadata
//   - error: Database error or metadata parsing error
//
// Performance:
//   - Separate count query for better MySQL optimization
//   - Indexed date range filtering
//   - Optimized search with EXISTS subquery
func GetUserLogsOptimized(db DBExecutor, filters UserLogFilters) ([]dtos.UserLog, *dtos.PaginationMeta, error) {
	// Calculate pagination offset
	offset := (filters.Page - 1) * filters.Limit

	// Build WHERE clause and arguments from filters
	whereSQL, args := buildUserLogsFilter(filters)

	// Separate count query for better MySQL performance (avoids CTE overhead)
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM logs l
		LEFT JOIN users u ON l.user_id = u.user_id
		%s
	`, whereSQL)

	// Execute count query for pagination metadata
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count query failed: %w", err)
	}

	// Main data query without CTE (better for MySQL optimizer)
	query := fmt.Sprintf(`
		SELECT 
			l.log_id,
			l.user_id,
			l.level,
			l.metadata,
			l.message,
			l.timestamp,
			l.module,
			u.user_id as u_user_id,
			u.first_name,
			u.last_name,
			u.email,
			u.role,
			u.status as user_status,
			u.last_login,
			u.created_at,
			u.phone_number
		FROM logs l
		LEFT JOIN users u ON l.user_id = u.user_id
		%s
		ORDER BY l.timestamp DESC
		LIMIT ? OFFSET ?
	`, whereSQL)

	// Add pagination parameters to argument list
	args = append(args, filters.Limit, offset)

	// Execute main data query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var logs []dtos.UserLog

	// Iterate through result rows
	for rows.Next() {
		var (
			singleLog dtos.UserLog
			user      dtos.Users
			meta      string
			ts        time.Time
			logUserID sql.NullString
			// User fields that can be NULL (LEFT JOIN may not match)
			userID     sql.NullString
			firstName  sql.NullString
			lastName   sql.NullString
			email      sql.NullString
			role       sql.NullString
			userStatus sql.NullString
			lastLogin  sql.NullString
			createdAt  sql.NullString
			phone      sql.NullString
		)

		// Scan all fields from JOIN query
		if err := rows.Scan(
			&singleLog.LogID,
			&logUserID,
			&singleLog.Status,
			&meta,
			&singleLog.Action,
			&ts,
			&singleLog.Module,
			&userID,
			&firstName,
			&lastName,
			&email,
			&role,
			&userStatus,
			&lastLogin,
			&createdAt,
			&phone,
		); err != nil {
			return nil, nil, fmt.Errorf("scan failed: %w", err)
		}

		// Format timestamp for output
		singleLog.CreatedAt = ts.Format("2006-01-02 15:04:05")

		// Parse JSON metadata (function, IP address, description, module)
		err := parseMetadata(&singleLog, meta)
		if err != nil {
			log.Printf("Failed to parse metadata for log %s: %v", singleLog.LogID, err)
		}

		// Populate user details from nullable fields or use unknown user placeholder
		if userID.Valid && userID.String != "" && userID.String != "unknown" {
			// Valid user found - populate full user details
			user.ID = userID.String
			user.FirstName = firstName.String
			user.LastName = lastName.String
			user.Email = email.String
			user.Role = role.String
			user.Status = userStatus.String
			user.LastLogin = lastLogin.String
			user.DateJoined = createdAt.String
			user.Phone = phone.String
			singleLog.User = &user
		} else {
			// No user or unknown user - use placeholder
			singleLog.User = UnknownUser()
		}

		logs = append(logs, singleLog)
	}

	// Check for row iteration errors
	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	// Build pagination metadata
	meta := buildPagination(filters.Limit, offset, total)
	return logs, meta, nil
}

// parseMetadata extracts fields from JSON metadata into UserLog struct.
//
// This is an internal helper function that unmarshals JSON metadata and
// populates log fields with type-safe extraction.
//
// Parameters:
//   - log: Pointer to UserLog to populate
//   - meta: JSON string containing metadata
//
// Returns:
//   - error: JSON unmarshal error if metadata is invalid
//
// Extracted Fields:
//   - Function → Action: Function/method name
//   - Address → IPAddress: Client IP address
//   - Description → Description: Action description
//   - Module → Module: Application module
func parseMetadata(log *dtos.UserLog, meta string) error {
	// Skip parsing if metadata is empty
	if meta == "" {
		return nil
	}

	// Unmarshal JSON to map
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(meta), &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Extract fields with type assertions (safely handles missing keys)
	if fn, ok := metadata["Function"].(string); ok {
		log.Action = fn
	}
	if addr, ok := metadata["Address"].(string); ok {
		log.IPAddress = addr
	}
	if desc, ok := metadata["Description"].(string); ok {
		log.Description = desc
	}
	if mod, ok := metadata["Module"].(string); ok {
		log.Module = mod
	}

	return nil
}

// UnknownUser creates a placeholder user object for logs without valid user association.
//
// This function returns a standardized "Unknown User" placeholder used when
// logs cannot be associated with a real user (e.g., system logs, deleted users).
//
// Returns:
//   - *dtos.Users: Pointer to user with all fields set to "unknown"
//
// Fields:
