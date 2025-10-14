package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"strings"

	"github.com/teris-io/shortid"
)

// CreateRole inserts a new role record
func CreateRole(name, description string) error {
	roleID, _ := shortid.Generate()
	query := `
		INSERT INTO roles (role_id,name, description)
		VALUES (?, ?, ?)
	`
	_, err := DB.Exec(query, roleID, name, description)
	if err != nil {
		return err
	}
	return nil
}

// GetRoles retrieves roles, optionally filtered by name and/or created_at range
func GetRoles(name, startDate, endDate string) ([]dtos.Role, error) {
	query := `
		SELECT role_id, name, description, created_at, updated_at
		FROM roles 
	`
	var args []interface{}
	var conditions []string

	// Filter by name
	if name != "" {
		conditions = append(conditions, "LOWER(name) LIKE ?")
		args = append(args, "%"+strings.ToLower(name)+"%")
	}

	// Filter by created_at range
	if startDate != "" && endDate != "" {
		conditions = append(conditions, "DATE(created_at) BETWEEN ? AND ?")
		args = append(args, startDate, endDate)
	}

	// Apply WHERE clause if filters exist
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return []dtos.Role{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var roles []dtos.Role
	for rows.Next() {
		var r dtos.Role
		if err := rows.Scan(&r.RoleID, &r.Name, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.Count, err = GetRoleCount(r.Name)
		if err != nil {
			return nil, err

		}
		roles = append(roles, r)
	}

	return roles, nil
}
func GetRoleCount(role string) (int, error) {
	query := `
		SELECT COUNT(*) AS count
		FROM users
		WHERE LOWER(role) = LOWER(?)
	`

	var count int
	err := DB.QueryRow(query, role).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func isRoleThere(id string) error {
	exists, err := RecordExists("roles", "role_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("role not found")
	}
	return nil
}

// UpdateRole updates role description by name
func UpdateRole(name, newDescription, roleID string) error {
	err := isRoleThere(roleID)
	if err != nil {
		return err
	}
	query := `
		UPDATE roles
		SET description = ?, name = ?
		WHERE role_id = ?
	`
	_, err = DB.Exec(query, newDescription, name, roleID)
	if err != nil {
		return err
	}

	return err
}

// DeleteRole removes a role by name
func DeleteRole(roleID string) error {
	err := isRoleThere(roleID)
	if err != nil {
		return err
	}
	query := `DELETE FROM roles WHERE role_ = ?`
	_, err = DB.Exec(query, roleID)

	return err
}
