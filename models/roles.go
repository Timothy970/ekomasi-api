// Package models provides data access functions for the Ekomasi e-commerce backend.
//
// roles.go handles role-based access control (RBAC) including:
//   - Role management (CRUD operations)
//   - Permission management (CRUD operations)
//   - Role-permission associations
//   - Available permissions master list
//   - User count per role
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"strings"

	"github.com/teris-io/shortid"
)

// CreateRole creates a new role with assigned permissions.
//
// This function performs the following:
//  1. Validates role doesn't already exist (case-insensitive)
//  2. Creates role with generated ID
//  3. Associates permissions with the role
//
// Parameters:
//   - name: string - Role name (must be unique, case-insensitive)
//   - description: string - Role description
//   - permissionIDs: []string - Array of permission IDs to assign to role
//
// Returns:
//   - error: "role {name} already exists", database error, or nil on success
func CreateRole(db DBExecutor, name, description string, permissions []dtos.AvailablePermission) error {
	// Validate role doesn't exist (case-insensitive check)
	exists, err := RecordExists(db, "roles", "LOWER(name) = LOWER(?)", name)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("role " + name + " already exists")
	}

	// Create role with generated ID
	roleID, _ := shortid.Generate()
	query := `
		INSERT INTO roles (role_id,name, description)
		VALUES (?, ?, ?)
	`
	_, err = db.Exec(query, roleID, name, description)
	if err != nil {
		return err
	}

	// Process and attach permissions to the role
	return processRolePermissions(db, roleID, permissions)
}

// processRolePermissions handles the creation and assignment of permissions to a role.
func processRolePermissions(db DBExecutor, roleID string, permissions []dtos.AvailablePermission) error {
	for _, pm := range permissions {
		permissionMasterID, err := getOrCreatePermissionMaster(db, pm)
		if err != nil {
			return err
		}

		// Attach the role to the permission using permission_master_id
		err = addRolePermission(db, roleID, permissionMasterID)
		if err != nil {
			return err
		}
	}
	return nil
}

// getOrCreatePermissionMaster retrieves or creates a permission in the master table.
func getOrCreatePermissionMaster(db DBExecutor, pm dtos.AvailablePermission) (string, error) {
	var permissionMasterID string

	// Check if permission exists in permissions_master table
	// Check if permission exists in permissions_master table
	err := db.QueryRow("SELECT permission_master_id FROM permissions_master WHERE LOWER(category) = LOWER(?) AND LOWER(permission_key) = LOWER(?)", pm.Category, pm.Key).Scan(&permissionMasterID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Permission doesn't exist in master, create it
			return createPermissionMaster(db, pm)
		}
		return "", err
	}
	return permissionMasterID, nil
}

// createPermissionMaster creates a new permission in the master table.
func createPermissionMaster(db DBExecutor, pm dtos.AvailablePermission) (string, error) {
	permissionMasterID, _ := shortid.Generate()
	query := `
		INSERT INTO permissions_master (permission_master_id, category, permission_key, description)
		VALUES (?, ?, ?, ?)
	`
	_, err := db.Exec(query, permissionMasterID, pm.Category, pm.Key, pm.Description)
	if err != nil {
		return "", err
	}
	return permissionMasterID, nil
}

// GetRoles retrieves roles with optional filtering and enrichment.
//
// This comprehensive function provides:
//   - Optional name filtering (case-insensitive partial match)
//   - Optional date range filtering (created_at)
//   - User count per role
//   - Permission list per role
//   - Results ordered by creation date (newest first)
//
// Parameters:
//   - name: string - Filter by role name (partial match, case-insensitive). Empty = no filter
//   - startDate: string - Start date for created_at filter (format: YYYY-MM-DD). Empty = no start filter
//   - endDate: string - End date for created_at filter (format: YYYY-MM-DD). Empty = no end filter
//
// Returns:
//   - []dtos.Role: Array of roles containing:
//   - RoleID, Name, Description, CreatedAt, UpdatedAt
//   - Count: Number of users with this role
//   - Permissions: Array of permissions assigned to role
//   - error: Database error or nil on success
//
// Notes:
//   - Returns empty array if no roles match filters (not an error)
//   - Date range requires both startDate and endDate
//   - Uses parameterized queries to prevent SQL injection
func GetRoles(db DBExecutor, name, startDate, endDate string) ([]dtos.Role, error) {
	// Build base query
	query := `
		SELECT role_id, name, description, created_at, updated_at
		FROM roles 
	`
	var args []any
	var conditions []string

	// Apply name filter (case-insensitive partial match)
	if name != "" {
		conditions = append(conditions, "LOWER(name) LIKE ?")
		args = append(args, "%"+strings.ToLower(name)+"%")
	}

	// Apply date range filter (requires both dates)
	if startDate != "" && endDate != "" {
		conditions = append(conditions, "DATE(created_at) BETWEEN ? AND ?")
		args = append(args, startDate, endDate)
	}

	// Build WHERE clause if filters exist
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Order by newest first
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return []dtos.Role{}, nil // No roles found, return empty array
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

		// Get count of users with this role
		r.Count, err = GetRoleCount(db, r.Name)
		if err != nil {
			return nil, err
		}

		// Get permissions assigned to this role
		r.Permissions, err = GetRolePermissions(db, r.RoleID)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, nil
}

// GetRolePermissions retrieves all permissions assigned to a specific role.
//
// Parameters:
//   - roleID: string - The role_id to fetch permissions for
//
// Returns:
//   - *[]dtos.Permission: Pointer to array of permissions containing ID, Name, Description
//     Returns nil if role has no permissions (not an error)
//   - error: Database error or nil on success
func GetRolePermissions(db DBExecutor, roleID string) (*[]dtos.Permission, error) {
	// Join role_permissions with permissions table
	query := `
		SELECT pm.permission_master_id, pm.permission_key, pm.description, pm.category
		FROM permissions_master pm
		JOIN role_permissions rp ON pm.permission_master_id = rp.permission_id
		WHERE rp.role_id = ?
	`
	rows, err := db.Query(query, roleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Role has no permissions
		}
		return nil, err
	}
	defer rows.Close()

	var permissions []dtos.Permission
	for rows.Next() {
		var p dtos.Permission
		if err := rows.Scan(&p.ID, &p.Key, &p.Description, &p.Category); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}

	return &permissions, nil
}

// GetRoleCount returns the number of users assigned to a specific role.
//
// Parameters:
//   - role: string - Role name to count users for (case-insensitive)
//
// Returns:
//   - int: Number of users with this role (0 if none)
//   - error: Database error or nil on success
func GetRoleCount(db DBExecutor, role string) (int, error) {
	query := `
		SELECT COUNT(*) AS count
		FROM users
		WHERE LOWER(role) = LOWER(?)
	`

	var count int
	err := db.QueryRow(query, role).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// isRoleThere validates that a role exists in the database.
//
// Parameters:
//   - id: string - The role_id to validate
//
// Returns:
//   - error: "role not found", database error, or nil if role exists
func isRoleThere(db DBExecutor, id string) error {
	exists, err := RecordExists(db, "roles", "role_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("role not found")
	}
	return nil
}
