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
	total, err := countUserLogs(whereSQL, args)
	if err != nil {
		return nil, nil, err
	}

	argsWithLimit := append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT log_id, user_id, level, metadata, message, timestamp
		FROM logs
		%s
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?`, whereSQL)

	rows, err := DB.Query(query, argsWithLimit...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	logs, err := processUserLogRows(rows)
	if err != nil {
		return nil, nil, err
	}

	meta := buildPagination(limit, offset, total)
	return logs, meta, nil
}

// buildUserLogsFilter creates the WHERE clause and args for filtering logs.
func buildUserLogsFilter(module, status, role, startDate, endDate, q string) (string, []interface{}) {
	var whereClauses []string
	var args []interface{}

	if module != "" && module != "all" {
		whereClauses = append(whereClauses, "LOWER(module) = ?")
		args = append(args, strings.ToLower(module))
	}
	if status != "" && status != "all" {
		whereClauses = append(whereClauses, "level = ?")
		args = append(args, status)
	}
	if role != "" && role != "all" {
		whereClauses = append(whereClauses, "LOWER(role) = ?")
		args = append(args, strings.ToLower(role))
	}
	if startDate != "" && endDate != "" {
		whereClauses = append(whereClauses, "DATE(timestamp) BETWEEN DATE(?) AND DATE(?)")
		args = append(args, startDate, endDate)
	}
	if q != "" {
		whereClauses = append(whereClauses, `user_id IN (
			SELECT user_id FROM users
			WHERE LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR phone_number LIKE ?
		)`)
		qLike := "%" + strings.ToLower(q) + "%"
		args = append(args, qLike, qLike, qLike, qLike)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
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
	// Check if user exists
	if err := isUserThere(userID); err != nil {
		return nil, nil, err
	}

	log.Printf("Fetching logs for user ID: %s", userID)
	offset := (page - 1) * limit
	var logs []dtos.UserLog
	var total int

	// Get total count for pagination
	err := DB.QueryRow(`SELECT COUNT(*) FROM logs WHERE user_id = ?`, userID).Scan(&total)
	if err != nil {
		return nil, nil, err
	}

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
	logs, err = processUserLogRows(rows)
	log.Printf("Fetched %d logs for user ID %s", len(logs), userID)
	if err != nil {
		log.Printf("Error processing log rows for user ID %s: %v", userID, err)
		return nil, nil, err
	}
	log.Printf("Processed %d logs for user ID %s", len(logs), userID)

	meta := buildPagination(limit, offset, total)
	return logs, meta, nil
}
