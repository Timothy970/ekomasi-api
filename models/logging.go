package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func GetUserLogs(
	page, limit int,
	module, status, role, startDate, endDate, q string,
) ([]dtos.UserLog, *dtos.PaginationMeta, error) {

	offset := (page - 1) * limit

	whereSQL, args := buildUserLogsFilter(module, status, role, startDate, endDate, q)

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM logs l
		LEFT JOIN users u ON l.user_id = u.user_id
		%s
	`, whereSQL)

	var total int
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, err
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT 
			l.log_id,
			l.user_id,        
			l.level,
			l.metadata,
			l.message,
			l.timestamp,

			u.user_id,         
			u.first_name,
			u.last_name,
			u.email,
			u.role,
			u.status,
			u.last_login,
			u.created_at,
			u.phone_number

		FROM logs l
		LEFT JOIN users u ON l.user_id = u.user_id
		%s
		ORDER BY l.timestamp DESC
		LIMIT ? OFFSET ?
	`, whereSQL)

	args = append(args, limit, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var logs []dtos.UserLog

	for rows.Next() {

		var log dtos.UserLog
		var meta string
		var ts sql.NullTime

		var user dtos.Users

		// Raw scanned database values
		var (
			logUserID sql.NullString
			dbUserID  sql.NullString
			firstName sql.NullString
			lastName  sql.NullString
			email     sql.NullString
			roleStr   sql.NullString
			statusStr sql.NullString
			lastLogin sql.NullTime
			createdAt sql.NullTime
			phone     sql.NullString
		)

		if err := rows.Scan(
			&log.LogID,
			&logUserID,
			&log.Status,
			&meta,
			&log.Action,
			&ts,

			&dbUserID,
			&firstName,
			&lastName,
			&email,
			&roleStr,
			&statusStr,
			&lastLogin,
			&createdAt,
			&phone,
		); err != nil {
			return nil, nil, err
		}

		// Timestamp
		if ts.Valid {
			log.CreatedAt = ts.Time.Format("2006-01-02 15:04:05")
		}

		// Metadata JSON
		if err := mapMetadataToLogger(&log, meta); err != nil {
			return nil, nil, err
		}

		// If user exists (not "unknown")
		if dbUserID.Valid && dbUserID.String != "unknown" {

			user.ID = dbUserID.String
			user.FirstName = firstName.String
			user.LastName = lastName.String
			user.Email = email.String
			user.Role = roleStr.String
			user.Status = statusStr.String
			user.LastLogin = formatNullTime(lastLogin)
			user.DateJoined = formatNullTime(createdAt)
			user.Phone = phone.String

			log.User = &user

		} else {
			// Unknown user
			log.User = UnknownUser()
		}

		logs = append(logs, log)
	}

	meta := buildPagination(limit, offset, total)
	return logs, meta, nil
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
func formatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

// buildUserLogsFilter creates the WHERE clause and args for filtering logs.
func buildUserLogsFilter(module, status, role, startDate, endDate, q string) (string, []interface{}) {
	var where []string
	var args []interface{}

	if module != "" && module != "all" {
		where = append(where, "l.module = ?")
		args = append(args, module)
	}

	if status != "" && status != "all" {
		where = append(where, "l.level = ?")
		args = append(args, status)
	}

	if role != "" && role != "all" {
		where = append(where, "u.role = ?")
		args = append(args, role)
	}

	if startDate != "" && endDate != "" {
		startDate = FormatDateTimeString(startDate)
		endDate = FormatDateTimeString(endDate)
		where = append(where, "l.timestamp BETWEEN ? AND ?")
		args = append(args, startDate+" 00:00:00", endDate+" 23:59:59")
	}

	// Optimized user search
	if q != "" {
		qLike := "%" + strings.ToLower(q) + "%"
		where = append(where,
			`(
				LOWER(u.first_name) LIKE ? OR
				LOWER(u.last_name)  LIKE ? OR
				LOWER(u.email)      LIKE ? OR
				u.phone_number      LIKE ?
			)`)
		args = append(args, qLike, qLike, qLike, qLike)
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	return whereSQL, args
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
