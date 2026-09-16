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
	"fmt"

	"github.com/teris-io/shortid"
)

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
