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
	"log"
	"strings"
)

// - ID, FirstName, LastName, Email, Role, Status: All set to "unknown"
// - LastLogin, DateJoined: Empty strings
// - UserAddress: nil
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
