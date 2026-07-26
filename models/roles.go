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
	"ekomasi_backend/dtos"
	"database/sql"
	"errors"
	"fmt"
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
	var args []interface{}
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

// UpdateRole updates a role's name, description, and permissions.
//
// Parameters:
//   - req: dtos.RoleRequest - Contains name, description, and permissions
//   - roleID: string - The role_id to update
//
// Returns:
//   - error: "role not found", database error, or nil on success
func UpdateRole(db DBExecutor, req dtos.RoleRequest, permissions []dtos.AvailablePermission, roleID string) error {
	// Validate role exists
	err := isRoleThere(db, roleID)
	if err != nil {
		return err
	}

	// Initiate db transaction
	dbInterface, ok := db.(DatabaseInterface)
	if !ok {
		return errors.New("database does not support transactions")
	}
	tx, err := dbInterface.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update role name and description
	query := `
		UPDATE roles
		SET description = ?, name = ?
		WHERE role_id = ?
	`
	_, err = tx.Exec(query, req.Description, req.Name, roleID)
	if err != nil {
		return err
	}

	//update role name in users table
	updateUsersQuery := `
		UPDATE users
		SET role = ?
		WHERE role_id = ?
	`
	_, err = tx.Exec(updateUsersQuery, req.Name, roleID)
	if err != nil {
		return err
	}

	// Delete current permissions
	deleteQuery := `DELETE FROM role_permissions WHERE role_id = ?`
	_, err = tx.Exec(deleteQuery, roleID)
	if err != nil {
		return err
	}
	// Insert new permissions
	err = processRolePermissions(tx, roleID, permissions)
	if err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}

// DeleteRole permanently removes a role from the database.
//
// Warning: This does not cascade delete role_permissions or update users with this role.
//
//	Consider validating role is not in use before deletion.
//
// Parameters:
//   - roleID: string - The role_id to delete
//
// Returns:
//   - error: "role not found", database error, or nil on success
func DeleteRole(db DBExecutor, roleID string) error {
	// Validate role exists
	err := isRoleThere(db, roleID)
	if err != nil {
		return err
	}

	// Hard delete role
	query := `DELETE FROM roles WHERE role_id = ?`
	_, err = db.Exec(query, roleID)

	return err
}

// GetRoleIDForCustomerRole retrieves the role_id for the "customer" role.
//
// This is a convenience function for getting the default customer role ID
// used during user registration.
//
// Returns:
//   - string: The role_id of the "customer" role
//   - error: Database error or sql.ErrNoRows if "customer" role doesn't exist
func GetRoleIDForCustomerRole(db DBExecutor) (string, error) {
	var roleID string
	query := `SELECT role_id FROM roles WHERE LOWER(name) = 'customer'`
	err := db.QueryRow(query).Scan(&roleID)
	if err != nil {
		return "", err
	}
	return roleID, nil
}

// GetRoleNameByID retrieves the name of a role by its ID.
//
// Parameters:
//   - roleID: string - The role_id to look up
//
// Returns:
//   - string: Role name, or empty string if role not found (not an error)
//   - error: Database error or nil on success
func GetRoleNameByID(db DBExecutor, roleID string) (string, error) {
	var roleName string
	query := `SELECT name FROM roles WHERE role_id = ?`
	err := db.QueryRow(query, roleID).Scan(&roleName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return roleName, nil
}

// AddPermissionsToRole assigns multiple permissions to a role.
//
// This function is idempotent - it skips permissions already assigned to the role.
//
// Parameters:
//   - roleID: string - The role_id to add permissions to
//   - permissionIDs: []string - Array of permission IDs to assign
//
// Returns:
//   - error: Database error or nil on success
func AddPermissionsToRole(db DBExecutor, roleID string, permissionKeys []dtos.AvailablePermission) error {
	// Process each permission
	for _, permissionKey := range permissionKeys {
		permissionMasterID, err := getOrCreatePermissionMaster(db, permissionKey)
		if err != nil {
			return err
		}
		// Check if role already has this permission
		exists, err := RecordExists(db, "role_permissions", "role_id = ? AND permission_id = ?", roleID, permissionMasterID)
		if err != nil {
			return err
		}
		if exists {
			continue // Skip already assigned permissions (idempotent)
		}

		// Add permission to role
		err = addRolePermission(db, roleID, permissionMasterID)
		if err != nil {
			return err
		}
	}
	return nil
}

// addRolePermission is a helper that inserts a single role-permission association.
//
// Parameters:
//   - roleID: string - The role_id
//   - permissionID: string - The permission_id to assign
//
// Returns:
//   - error: Database error or nil on success
func addRolePermission(db DBExecutor, roleID, permissionID string) error {
	rolePermissionID, _ := shortid.Generate()
	query := `
		INSERT INTO role_permissions (role_permission_id, role_id, permission_id)
		VALUES (?, ?, ?)
	`
	_, err := db.Exec(query, rolePermissionID, roleID, permissionID)
	return err
}

// RemovePermissionsFromRole removes multiple permissions from a role.
//
// This function is idempotent - it skips permissions not assigned to the role.
//
// Parameters:
//   - roleID: string - The role_id to remove permissions from
//   - permissionIDs: []string - Array of permission IDs to remove
//
// Returns:
//   - error: Database error or nil on success
func RemovePermissionsFromRole(db DBExecutor, roleID string, permissionKeys []dtos.AvailablePermission) error {
	// Process each permission
	for _, permissionKey := range permissionKeys {
		permissionMasterID, err := getOrCreatePermissionMaster(db, permissionKey)
		if err != nil {
			return err
		}
		// Check if role has this permission
		exists, err := RecordExists(db, "role_permissions", "role_id = ? AND permission_id = ?", roleID, permissionMasterID)
		if err != nil {
			return err
		}
		if !exists {
			continue // Skip permissions not assigned (idempotent)
		}

		// Remove permission from role
		err = removeRolePermission(db, roleID, permissionMasterID)
		if err != nil {
			return err
		}
	}
	return nil
}

// removeRolePermission is a helper that deletes a single role-permission association.
//
// Parameters:
//   - roleID: string - The role_id
//   - permissionID: string - The permission_id to remove
//
// Returns:
//   - error: Database error or nil on success
func removeRolePermission(db DBExecutor, roleID, permissionID string) error {
	query := `DELETE FROM role_permissions WHERE role_id = ? AND permission_id = ?`
	_, err := db.Exec(query, roleID, permissionID)
	return err
}

// GetAvailablePermissions retrieves all available permissions from the master list.
//
// The permissions_master table serves as a template/catalog of all possible
// permissions that can be created in the system.
//
// Parameters:
//   - category: string - Filter by category (case-insensitive). Empty = no filter
//
// Returns:
//   - []dtos.AvailablePermission: Array containing Category, Key, Description
//   - error: Database error or nil on success
func GetAvailablePermissions(db DBExecutor, category string) ([]dtos.AvailablePermission, error) {
	var permissions []dtos.AvailablePermission

	baseQuery := `
		SELECT category, permission_key, description
		FROM permissions_master
	`

	var rows *sql.Rows
	var err error

	// Apply category filter if provided
	if category != "" {
		baseQuery += " WHERE LOWER(category) = LOWER(?)"
		rows, err = db.Query(baseQuery, category)
	} else {
		rows, err = db.Query(baseQuery)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Fetch available permissions
	for rows.Next() {
		var p dtos.AvailablePermission
		if err := rows.Scan(&p.Category, &p.Key, &p.Description); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

// AddAvailablePermission adds a new permission template to the master list.
//
// Parameters:
//   - category: string - Permission category (e.g., "products", "orders")
//   - key: string - Permission key (e.g., "product.create")
//   - description: string - Permission description
//
// Returns:
//   - error: Database error or nil on success
func AddAvailablePermission(db DBExecutor, category, key, description string) error {
	permissionMasterID, _ := shortid.Generate()
	query := `
		INSERT INTO permissions_master (permission_master_id, category, permission_key, description)
		VALUES (?, ?, ?, ?)
	`
	_, err := db.Exec(query, permissionMasterID, category, key, description)
	return err
}

// RemoveAvailablePermission removes a permission template from the master list.
//
// Parameters:
//   - category: string - Permission category (case-insensitive)
//   - key: string - Permission key (case-insensitive)
//
// Returns:
//   - error: "permission not found", database error, or nil on success
func RemoveAvailablePermission(db DBExecutor, category, key string) error {
	// Validate permission exists
	err := IsAvailablePermissionThere(db, category, key)
	if err != nil {
		return err
	}

	// Delete by category and key (case-insensitive)
	query := `
		DELETE FROM permissions_master
		WHERE LOWER(category) = LOWER(?) AND LOWER(permission_key) = LOWER(?)
	`
	_, err = db.Exec(query, category, key)
	return err
}

// IsAvailablePermissionThere validates that a permission exists in the master list.
//
// Parameters:
//   - category: string - Permission category (case-insensitive)
//   - key: string - Permission key (case-insensitive)
//
// Returns:
//   - error: "permission not found", database error, or nil if exists
func IsAvailablePermissionThere(db DBExecutor, category, key string) error {
	exists, err := RecordExists(db, "permissions_master", "LOWER(category) = LOWER(?) AND LOWER(permission_key) = LOWER(?)", category, key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("permission not found")
	}
	return nil
}

// UpdateAvailablePermission updates an existing permission template in the master list.
//
// Parameters:
//   - category: string - Current category (for lookup, case-insensitive)
//   - key: string - Current key (for lookup, case-insensitive)
//   - description: string - Unused parameter (legacy)
//   - newDescription: string - New description
//   - newKey: string - New permission key
//   - newCategory: string - New category
//
// Returns:
//   - error: "permission not found", database error, or nil on success
func UpdateAvailablePermission(db DBExecutor, category, key, description, newDescription, newKey, newCategory string) error {
	// Validate permission exists
	err := IsAvailablePermissionThere(db, category, key)
	if err != nil {
		return err
	}

	// Update all fields based on old category and key (case-insensitive)
	query := `
		UPDATE permissions_master
		SET category = ?, permission_key = ?, description = ?
		WHERE LOWER(category) = LOWER(?) AND LOWER(permission_key) = LOWER(?)
	`
	_, err = db.Exec(query, newCategory, newKey, newDescription, category, key)
	return err
}
