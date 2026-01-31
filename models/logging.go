// Package models provides data access functions for the Adenzo e-commerce logging system.
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
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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
//   - ID, FirstName, LastName, Email, Role, Status: All set to "unknown"
//   - LastLogin, DateJoined: Empty strings
//   - UserAddress: nil
func UnknownUser() *dtos.Users {
	return &dtos.Users{
		ID:        "unknown",
		FirstName: "Unknown",
		LastName:  "User",
		Email:     "unknown",
		Role:      "unknown",
		Status:    "unknown",
	}
}

// buildUserLogsFilter creates the WHERE clause and args for filtering logs.
//
// This is an internal helper function that dynamically builds SQL WHERE conditions
// and parameter arguments based on provided filters.
//
// Parameters:
//   - filters: UserLogFilters with optional filter criteria
//
// Returns:
//   - string: WHERE clause (e.g., "WHERE l.module = ? AND l.level = ?") or empty string
//   - []interface{}: Argument values matching placeholders in WHERE clause
//
// Filter Logic:
//   - Module: Exact match on l.module (skip if "all" or empty)
//   - Status: Exact match on l.level (skip if "all" or empty)
//   - Role: Exact match on l.role (skip if "all" or empty)
//   - Date Range: BETWEEN for both dates, >= for start only, <= for end only
//   - Search: LIKE search on message and user fields (first_name, last_name, email)
//
// Performance:
//   - Date range uses indexed l.timestamp column
//   - Search uses EXISTS subquery to avoid full table scan on users
//   - Wildcards are added to search term (case-insensitive)
func buildUserLogsFilter(filters UserLogFilters) (string, []interface{}) {
	var args []interface{}
	var conditions []string

	// Filter by module (skip "all" or empty)
	if filters.Module != "" && filters.Module != "all" {
		conditions = append(conditions, "l.module = ?")
		args = append(args, filters.Module)
	}

	// Filter by status/level (skip "all" or empty)
	if filters.Status != "" && filters.Status != "all" {
		conditions = append(conditions, "l.level = ?")
		args = append(args, filters.Status)
	}

	// Filter by role (skip "all" or empty)
	if filters.Role != "" && filters.Role != "all" {
		conditions = append(conditions, "l.role = ?")
		args = append(args, filters.Role)
	}

	// Date range filter with index usage on l.timestamp
	if filters.StartDate != "" && filters.EndDate != "" {
		// Both dates provided - use BETWEEN
		start := FormatDateTimeString(filters.StartDate)
		// Add 23:59:59 to end date for inclusive end-of-day
		end := FormatDateTimeString(filters.EndDate + " 23:59:59")
		log.Printf("Start date filter applied: %s", start)
		log.Printf("End date filter applied: %s", end)
		conditions = append(conditions, "l.timestamp BETWEEN ? AND ?")
		args = append(args, start, end)
	} else if filters.StartDate != "" {
		// Only start date - filter from start onwards
		start := FormatDateTimeString(filters.StartDate)
		conditions = append(conditions, "l.timestamp >= ?")
		args = append(args, start)
	} else if filters.EndDate != "" {
		// Only end date - filter up to end (inclusive end-of-day)
		end := FormatDateTimeString(filters.EndDate + " 23:59:59")
		conditions = append(conditions, "l.timestamp <= ?")
		args = append(args, end)
	}

	// Optimized search - uses EXISTS subquery to avoid full table scan
	if filters.Search != "" {
		// Prepare case-insensitive search term with wildcards
		searchTerm := "%" + strings.ToLower(filters.Search) + "%"

		// Search on indexed columns only (l.message) and use EXISTS for user fields
		conditions = append(conditions, `
			(l.message LIKE ? OR 
			EXISTS (
				SELECT 1 FROM users u2 
				WHERE u2.user_id = l.user_id 
				AND (LOWER(u2.first_name) LIKE ? OR 
					 LOWER(u2.last_name) LIKE ? OR 
					 LOWER(u2.email) LIKE ?)
			))`)
		// Add search term 4 times (once for message, three for user fields)
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	// Build final WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}

// processUserLogRows processes the rows and returns the logs.
//
// This is an internal helper function that scans rows, parses metadata,
// and enriches logs with user information.
//
// Parameters:
//   - rows: Result set from log query
//
// Returns:
//   - []dtos.UserLog: Array of processed logs with user details
//   - error: Scan error, metadata parsing error, or user fetch error
func processUserLogRows(db DBExecutor, rows *sql.Rows) ([]dtos.UserLog, error) {
	var logs []dtos.UserLog

	// Iterate through result rows
	for rows.Next() {
		var logger dtos.UserLog
		var userID string
		var metadata string
		var timestamp sql.NullTime

		// Scan basic log fields
		if err := rows.Scan(&logger.LogID, &userID, &logger.Status, &metadata, &logger.Action, &timestamp); err != nil {
			return nil, err
		}

		// Format timestamp if valid
		if timestamp.Valid {
			logger.CreatedAt = timestamp.Time.Format("2006-01-02 15:04:05")
		}

		// Parse JSON metadata into log fields
		if err := mapMetadataToLogger(&logger, metadata); err != nil {
			return nil, err
		}

		// Enrich with user details
		if err := assignUserToLogger(db, &logger, userID); err != nil {
			log.Printf("Error fetching user for log %s: %v", logger.LogID, err)
			return nil, err
		}

		logs = append(logs, logger)
	}
	return logs, nil
}

// mapMetadataToLogger extracts fields from JSON metadata into UserLog struct.
//
// This is an internal helper function similar to parseMetadata but provides
// default values for missing fields.
//
// Parameters:
//   - logger: Pointer to UserLog to populate
//   - metadata: JSON string containing metadata
//
// Returns:
//   - error: JSON conversion error if metadata is invalid
//
// Default Values:
//   - IPAddress: Empty string if missing
//   - Description: "TO DO" if missing
//   - Module: "Module not specified" if missing
func mapMetadataToLogger(logger *dtos.UserLog, metadata string) error {
	// Convert JSON string to map
	mapMetadata, err := ConvertStringToMap(metadata)
	if err != nil {
		return err
	}

	// Extract function name
	if fn, ok := mapMetadata["Function"].(string); ok {
		logger.Action = fn
	}

	// Extract IP address with default
	if addr, ok := mapMetadata["Address"].(string); ok {
		logger.IPAddress = addr
	} else {
		logger.IPAddress = ""
	}

	// Extract description with default
	if desc, ok := mapMetadata["Description"].(string); ok {
		logger.Description = desc
	} else {
		logger.Description = "TO DO"
	}

	// Extract module with default
	if mod, ok := mapMetadata["Module"].(string); ok {
		logger.Module = mod
	} else {
		logger.Module = "Module not specified"
	}

	return nil
}

// assignUserToLogger fetches and assigns user details to a log entry.
//
// This is an internal helper function that retrieves full user information
// or assigns an unknown user placeholder.
//
// Parameters:
//   - logger: Pointer to UserLog to populate
//   - userID: The user_id to fetch ("unknown" for placeholder)
//
// Returns:
//   - error: User fetch error if user lookup fails
func assignUserToLogger(db DBExecutor, logger *dtos.UserLog, userID string) error {
	// Check if user is known
	if userID != "unknown" {
		// Fetch full user details
		user, err := GetUserByUserID(db, userID)
		if err != nil {
			return err
		}
		logger.User = user
	} else {
		// Use unknown user placeholder
		logger.User = &dtos.Users{
			ID:          "unknown",
			FirstName:   "Unknown",
			LastName:    "User",
			Email:       "unknown",
			Role:        "unknown",
			Status:      "unknown",
			LastLogin:   "",
			DateJoined:  "",
			UserAddress: nil,
		}
	}
	return nil
}

// ConvertStringToMap is a helper function to convert JSON string to map.
//
// This is an internal utility function that unmarshals JSON strings into
// generic map structures for flexible metadata parsing.
//
// Parameters:
//   - data: JSON string to unmarshal
//
// Returns:
//   - map[string]interface{}: Unmarshaled map
//   - error: JSON unmarshal error if data is invalid
func ConvertStringToMap(data string) (map[string]interface{}, error) {
	var result map[string]interface{}
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
