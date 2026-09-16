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

	"github.com/teris-io/shortid"
)

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
