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

func GetUserLogsOptimized(filters UserLogFilters) ([]dtos.UserLog, *dtos.PaginationMeta, error) {
	offset := (filters.Page - 1) * filters.Limit

	// Use CTE for better performance with complex joins
	query := `
	WITH filtered_logs AS (
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
	),
	total_count AS (
		SELECT COUNT(*) as total FROM filtered_logs
	)
	SELECT 
		fl.*,
		tc.total
	FROM filtered_logs fl
	CROSS JOIN total_count tc
	ORDER BY fl.timestamp DESC
	LIMIT ? OFFSET ?
	`

	whereSQL, args := buildUserLogsFilter(filters)
	finalQuery := fmt.Sprintf(query, whereSQL)

	// Add pagination parameters
	args = append(args, filters.Limit, offset)

	rows, err := DB.Query(finalQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var logs []dtos.UserLog
	var total int

	for rows.Next() {
		var (
			log       dtos.UserLog
			user      dtos.Users
			meta      string
			ts        time.Time
			logUserID sql.NullString
		)

		// Scan all fields including total
		if err := rows.Scan(
			&log.LogID,
			&logUserID,
			&log.Status,
			&meta,
			&log.Action,
			&ts,
			&log.Module,
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.Role,
			&user.Status,
			&user.LastLogin,
			&user.DateJoined,
			&user.Phone,
			&total,
		); err != nil {
			return nil, nil, fmt.Errorf("scan failed: %w", err)
		}

		log.CreatedAt = ts.Format("2006-01-02 15:04:05")

		// Parse metadata
		if err := parseMetadata(&log, meta); err != nil {
			// Log but don't fail the entire request
			// log.Printf("Failed to parse metadata for log %s: %v", log.LogID, err)
			// log.Printf("Failed to parse metadata for log %s: %v", log.LogID, err)
		}

		// Set user info
		if user.ID != "" && user.ID != "unknown" {
			log.User = &user
		} else {
			log.User = UnknownUser()
		}

		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	meta := buildPagination(filters.Limit, offset, total)
	return logs, meta, nil
}
func parseMetadata(log *dtos.UserLog, meta string) error {
	if meta == "" {
		return nil
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(meta), &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Extract fields with type assertions
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

// func GetUserLogs(
// 	filters UserLogFilters,
// ) ([]dtos.UserLog, *dtos.PaginationMeta, error) {

// 	offset := (filters.Page - 1) * filters.Limit

// 	whereSQL, args := buildUserLogsFilter(filters)

// 	// Count query
// 	countQuery := fmt.Sprintf(`
// 		SELECT COUNT(*)
// 		FROM logs l
// 		LEFT JOIN users u ON l.user_id = u.user_id
// 		%s
// 	`, whereSQL)

// 	var total int
// 	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
// 		return nil, nil, err
// 	}

// 	// Main query
// 	query := fmt.Sprintf(`
// 		SELECT
// 			l.log_id,
// 			l.user_id,
// 			l.level,
// 			l.metadata,
// 			l.message,
// 			l.timestamp,

// 			u.user_id,
// 			u.first_name,
// 			u.last_name,
// 			u.email,
// 			u.role,
// 			u.status,
// 			u.last_login,
// 			u.created_at,
// 			u.phone_number

// 		FROM logs l
// 		LEFT JOIN users u ON l.user_id = u.user_id
// 		%s
// 		ORDER BY l.timestamp DESC
// 		LIMIT ? OFFSET ?
// 	`, whereSQL)

// 	args = append(args, limit, offset)

// 	rows, err := DB.Query(query, args...)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	defer rows.Close()

// 	var logs []dtos.UserLog

// 	for rows.Next() {

// 		var log dtos.UserLog
// 		var meta string
// 		var ts sql.NullTime

// 		var user dtos.Users

// 		// Raw scanned database values
// 		var (
// 			logUserID sql.NullString
// 			dbUserID  sql.NullString
// 			firstName sql.NullString
// 			lastName  sql.NullString
// 			email     sql.NullString
// 			roleStr   sql.NullString
// 			statusStr sql.NullString
// 			lastLogin sql.NullTime
// 			createdAt sql.NullTime
// 			phone     sql.NullString
// 		)

// 		if err := rows.Scan(
// 			&log.LogID,
// 			&logUserID,
// 			&log.Status,
// 			&meta,
// 			&log.Action,
// 			&ts,

// 			&dbUserID,
// 			&firstName,
// 			&lastName,
// 			&email,
// 			&roleStr,
// 			&statusStr,
// 			&lastLogin,
// 			&createdAt,
// 			&phone,
// 		); err != nil {
// 			return nil, nil, err
// 		}

// 		// Timestamp
// 		if ts.Valid {
// 			log.CreatedAt = ts.Time.Format("2006-01-02 15:04:05")
// 		}

// 		// Metadata JSON
// 		if err := mapMetadataToLogger(&log, meta); err != nil {
// 			return nil, nil, err
// 		}

// 		// If user exists (not "unknown")
// 		if dbUserID.Valid && dbUserID.String != "unknown" {

// 			user.ID = dbUserID.String
// 			user.FirstName = firstName.String
// 			user.LastName = lastName.String
// 			user.Email = email.String
// 			user.Role = roleStr.String
// 			user.Status = statusStr.String
// 			user.LastLogin = formatNullTime(lastLogin)
// 			user.DateJoined = formatNullTime(createdAt)
// 			user.Phone = phone.String

// 			log.User = &user

// 		} else {
// 			// Unknown user
// 			log.User = UnknownUser()
// 		}

// 		logs = append(logs, log)
// 	}

// 	meta := buildPagination(limit, offset, total)
// 	return logs, meta, nil
// }

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
func formatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

// buildUserLogsFilter creates the WHERE clause and args for filtering logs.
func buildUserLogsFilter(filters UserLogFilters) (string, []interface{}) {
	var args []interface{}
	argIndex := 1
	var conditions []string

	if filters.Module != "" && filters.Module != "all" {
		conditions = append(conditions, fmt.Sprintf("l.module = $%d", argIndex))
		args = append(args, filters.Module)
		argIndex++
	}

	if filters.Status != "" && filters.Status != "all" {
		conditions = append(conditions, fmt.Sprintf("l.level = $%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}

	if filters.Role != "" && filters.Role != "all" {
		conditions = append(conditions, fmt.Sprintf("u.role = $%d", argIndex))
		args = append(args, filters.Role)
		argIndex++
	}

	// Date range with index usage
	if filters.StartDate != "" && filters.EndDate != "" {
		start := FormatDateTimeString(filters.StartDate)
		end := FormatDateTimeString(filters.EndDate)
		conditions = append(conditions, fmt.Sprintf("l.timestamp BETWEEN $%d AND $%d", argIndex, argIndex+1))
		args = append(args, start, end)
		argIndex += 2
	} else if filters.StartDate != "" {
		start := FormatDateTimeString(filters.StartDate)
		conditions = append(conditions, fmt.Sprintf("l.timestamp >= $%d", argIndex))
		args = append(args, start)
		argIndex++
	} else if filters.EndDate != "" {
		end := FormatDateTimeString(filters.EndDate)
		conditions = append(conditions, fmt.Sprintf("l.timestamp <= $%d", argIndex))
		args = append(args, end)
		argIndex++
	}

	// Optimized search - use full-text search if available, otherwise be careful with wildcards
	if filters.Search != "" {
		searchTerm := "%" + strings.ToLower(filters.Search) + "%"
		// Only search on indexed columns
		conditions = append(conditions, fmt.Sprintf(`
			(l.message LIKE $%d OR 
			EXISTS (
				SELECT 1 FROM users u2 
				WHERE u2.user_id = l.user_id 
				AND (LOWER(u2.first_name) LIKE $%d OR 
					 LOWER(u2.last_name) LIKE $%d OR 
					 LOWER(u2.email) LIKE $%d)
			))`, argIndex, argIndex, argIndex, argIndex))
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}

// countUserLogs returns the total count of logs for the given filter.
func countUserLogs(whereSQL string, args []interface{}) (int, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM logs %s", whereSQL)
	var total int
	err := DB.QueryRow(countQuery, args...).Scan(&total)
	return total, err
}

// processUserLogRows processes the rows and returns the logs.
func processUserLogRows(rows *sql.Rows) ([]dtos.UserLog, error) {
	var logs []dtos.UserLog
	for rows.Next() {
		var logger dtos.UserLog
		var userID string
		var metadata string
		var timestamp sql.NullTime
		if err := rows.Scan(&logger.LogID, &userID, &logger.Status, &metadata, &logger.Action, &timestamp); err != nil {
			return nil, err
		}
		if timestamp.Valid {
			logger.CreatedAt = timestamp.Time.Format("2006-01-02 15:04:05")
		}
		if err := mapMetadataToLogger(&logger, metadata); err != nil {
			return nil, err
		}
		if err := assignUserToLogger(&logger, userID); err != nil {
			log.Printf("Error fetching user for log %s: %v", logger.LogID, err)
			return nil, err
		}
		logs = append(logs, logger)
	}
	return logs, nil
}

func mapMetadataToLogger(logger *dtos.UserLog, metadata string) error {
	mapMetadata, err := ConvertStringToMap(metadata)
	if err != nil {
		return err
	}
	if fn, ok := mapMetadata["Function"].(string); ok {
		logger.Action = fn
	}
	if addr, ok := mapMetadata["Address"].(string); ok {
		logger.IPAddress = addr
	} else {
		logger.IPAddress = ""
	}
	if desc, ok := mapMetadata["Description"].(string); ok {
		logger.Description = desc
	} else {
		logger.Description = "TO DO"
	}
	if mod, ok := mapMetadata["Module"].(string); ok {
		logger.Module = mod
	} else {
		logger.Module = "Module not specified"
	}
	return nil
}

func assignUserToLogger(logger *dtos.UserLog, userID string) error {
	if userID != "unknown" {
		user, err := GetUserByUserID(userID)
		if err != nil {
			return err
		}
		logger.User = user
	} else {
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

// helper function to convert string to map
func ConvertStringToMap(data string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(data), &result)
	return result, err
}

func GetUserLogsByUserID(userID string, limit, page int) ([]dtos.UserLog, *dtos.PaginationMeta, error) {
	// Validate user
	if err := isUserThere(userID); err != nil {
		return nil, nil, err
	}

	log.Printf("Fetching logs for user ID: %s", userID)
	offset := (page - 1) * limit

	// Count total logs
	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM logs WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, nil, err
	}

	// Fetch log rows
	query := `
		SELECT log_id, user_id, level, metadata, message, timestamp
		FROM logs
		WHERE user_id = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?`

	rows, err := DB.Query(query, userID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Process rows into DTOs
	logs, err := processUserLogRows(rows)
	if err != nil {
		log.Printf("Error processing logs for user %s: %v", userID, err)
		return nil, nil, err
	}

	log.Printf("Fetched %d logs for user ID %s", len(logs), userID)

	meta := buildPagination(limit, offset, total)
	return logs, meta, nil
}
