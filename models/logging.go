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

	whereSQL, args := buildUserLogsFilter(filters)

	// Separate count query for better MySQL performance
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM logs l
		LEFT JOIN users u ON l.user_id = u.user_id
		%s
	`, whereSQL)

	var total int
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count query failed: %w", err)
	}

	// Main data query without CTE
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

	// Add pagination parameters
	args = append(args, filters.Limit, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var logs []dtos.UserLog

	for rows.Next() {
		var (
			singleLog dtos.UserLog
			user      dtos.Users
			meta      string
			ts        time.Time
			logUserID sql.NullString
			// User fields that can be NULL
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

		// Scan all fields including total
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

		singleLog.CreatedAt = ts.Format("2006-01-02 15:04:05")

		// Parse metadata
		err := parseMetadata(&singleLog, meta)
		if err != nil {
			log.Printf("Failed to parse metadata for log %s: %v", singleLog.LogID, err)
		}

		// Set user info from nullable fields
		if userID.Valid && userID.String != "" && userID.String != "unknown" {
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
			singleLog.User = UnknownUser()
		}

		logs = append(logs, singleLog)
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
func buildUserLogsFilter(filters UserLogFilters) (string, []interface{}) {
	var args []interface{}
	var conditions []string

	if filters.Module != "" && filters.Module != "all" {
		conditions = append(conditions, "l.module = ?")
		args = append(args, filters.Module)
	}

	if filters.Status != "" && filters.Status != "all" {
		conditions = append(conditions, "l.level = ?")
		args = append(args, filters.Status)
	}

	if filters.Role != "" && filters.Role != "all" {
		conditions = append(conditions, "l.role = ?")
		args = append(args, filters.Role)
	}

	// Date range with index usage
	if filters.StartDate != "" && filters.EndDate != "" {
		start := FormatDateTimeString(filters.StartDate)
		//add 23hr 59min to end date
		end := FormatDateTimeString(filters.EndDate + " 23:59:59")
		log.Printf("Start date filter applied: %s", start)
		log.Printf("End date filter applied: %s", end)
		conditions = append(conditions, "l.timestamp BETWEEN ? AND ?")
		args = append(args, start, end)
	} else if filters.StartDate != "" {
		start := FormatDateTimeString(filters.StartDate)
		conditions = append(conditions, "l.timestamp >= ?")
		args = append(args, start)
	} else if filters.EndDate != "" {
		end := FormatDateTimeString(filters.EndDate + " 23:59:59")
		conditions = append(conditions, "l.timestamp <= ?")
		args = append(args, end)
	}

	// Optimized search - use full-text search if available, otherwise be careful with wildcards
	if filters.Search != "" {
		searchTerm := "%" + strings.ToLower(filters.Search) + "%"
		// Only search on indexed columns
		conditions = append(conditions, `
			(l.message LIKE ? OR 
			EXISTS (
				SELECT 1 FROM users u2 
				WHERE u2.user_id = l.user_id 
				AND (LOWER(u2.first_name) LIKE ? OR 
					 LOWER(u2.last_name) LIKE ? OR 
					 LOWER(u2.email) LIKE ?)
			))`)
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
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
