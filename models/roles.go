package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"strings"

	"github.com/teris-io/shortid"
)

// CreateRole inserts a new role record
func CreateRole(name, description string, permissionIDs []string) error {
	roleID, _ := shortid.Generate()
	query := `
		INSERT INTO roles (role_id,name, description)
		VALUES (?, ?, ?)
	`
	_, err := DB.Exec(query, roleID, name, description)
	if err != nil {
		return err
	}
	// Assign permissions to the role
	for _, pid := range permissionIDs {
		rolePermissionID, _ := shortid.Generate()
		query := `
			INSERT INTO role_permissions (role_permission_id, role_id, permission_id)
			VALUES (?, ?, ?)
		`
		_, err := DB.Exec(query, rolePermissionID, roleID, pid)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetRoles retrieves roles, optionally filtered by name and/or created_at range
// GetRoles retrieves a list of roles from the database, optionally filtered by name and creation date range.
// The function supports case-insensitive partial matching for the role name and filtering by a date range for the created_at field.
// Results are ordered by creation date in descending order.
// For each role, it also fetches the count of associated entities and the permissions assigned to the role.
//
// Parameters:
//   - name:      (string) Optional. Filters roles by name using a case-insensitive LIKE query.
//   - startDate: (string) Optional. The start date (inclusive) for filtering roles by their creation date.
//   - endDate:   (string) Optional. The end date (inclusive) for filtering roles by their creation date.
//
// Returns:
//   - ([]dtos.Role): A slice of Role DTOs containing role details, count, and permissions.
//   - (error):       An error object if any database or processing error occurs; otherwise, nil.
//
// Notes:
//   - If no roles match the filters, an empty slice is returned.
//   - If a database error occurs (other than no rows found), the error is returned.
//   - The function assumes the existence of helper functions GetRoleCount and GetRolePermissions.
//   - The function uses parameterized queries to prevent SQL injection.
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
		//get role permissions
		r.Permissions, err = GetRolePermissions(r.RoleID)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}

	return roles, nil
}
func GetRolePermissions(roleID string) (*[]dtos.Permission, error) {
	query := `
		SELECT p.permission_id, p.name, p.description
		FROM permissions p
		JOIN role_permissions rp ON p.permission_id = rp.permission_id
		WHERE rp.role_id = ?
	`
	rows, err := DB.Query(query, roleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var permissions []dtos.Permission
	for rows.Next() {
		var p dtos.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}

	return &permissions, nil
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
	query := `DELETE FROM roles WHERE role_id = ?`
	_, err = DB.Exec(query, roleID)

	return err
}

func CreatePermission(name string, description *string) error {
	permissionID, _ := shortid.Generate()
	//check id permission exists
	exists, err := RecordExists("permissions", "LOWER(name) = LOWER(?)", name)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("permission already exists")
	}
	query := `
		INSERT INTO permissions (permission_id, name, description)
		VALUES (?, ?, ?)
	`
	_, err = DB.Exec(query, permissionID, name, description)
	if err != nil {
		return err
	}
	return nil
}

func IsPermissionThere(id string) error {
	exists, err := RecordExists("permissions", "permission_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("permission not found")
	}
	return nil
}

func UpdatePermission(name string, newDescription *string, permissionID string) error {
	err := IsPermissionThere(permissionID)
	if err != nil {
		return err
	}
	query := `
		UPDATE permissions
		SET description = ?, name = ?
		WHERE permission_id = ?
	`
	_, err = DB.Exec(query, newDescription, name, permissionID)
	if err != nil {
		return err
	}

	return err
}
func DeletePermission(permissionID string) error {
	err := IsPermissionThere(permissionID)
	if err != nil {
		return err
	}
	query := `DELETE FROM permissions WHERE permission_id = ?`
	_, err = DB.Exec(query, permissionID)

	return err
}
func GetPermissions() ([]dtos.Permission, error) {
	query := `
		SELECT permission_id, name, description
		FROM permissions 
	`
	rows, err := DB.Query(query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []dtos.Permission{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var permissions []dtos.Permission
	for rows.Next() {
		var p dtos.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}

	return permissions, nil
}
func GetPermissionByID(permissionID string) (*dtos.Permission, error) {
	err := IsPermissionThere(permissionID)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT permission_id, name, description
		FROM permissions
		WHERE permission_id = ?
	`
	var p dtos.Permission
	err = DB.QueryRow(query, permissionID).Scan(&p.ID, &p.Name, &p.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func GetRoleIDForCustomerRole() (string, error) {
	var roleID string
	query := `SELECT role_id FROM roles WHERE LOWER(name) = 'customer'`
	err := DB.QueryRow(query).Scan(&roleID)
	if err != nil {
		return "", err
	}
	return roleID, nil
}

// / GetRoleNameByID returns the name of a role for a given role ID.
// If the role does not exist, it returns an empty string and no error.
// If a database error occurs, it returns an empty string and the error.
func GetRoleNameByID(roleID string) (string, error) {
	var roleName string
	query := `SELECT name FROM roles WHERE role_id = ?`
	err := DB.QueryRow(query, roleID).Scan(&roleName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return roleName, nil
}
func AddPermissionsToRole(roleID string, permissionIDs []string) error {
	//if role has the permission skip it
	for _, permissionID := range permissionIDs {
		exists, err := RecordExists("role_permissions", "role_id = ? AND permission_id = ?", roleID, permissionID)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		// If not, add the permission to the role
		err = addRolePermission(roleID, permissionID)
		if err != nil {
			return err
		}
	}
	return nil
}
func addRolePermission(roleID, permissionID string) error {
	rolePermissionID, _ := shortid.Generate()
	query := `
		INSERT INTO role_permissions (role_permission_id, role_id, permission_id)
		VALUES (?, ?, ?)
	`
	_, err := DB.Exec(query, rolePermissionID, roleID, permissionID)
	return err
}
func RemovePermissionsFromRole(roleID string, permissionIDs []string) error {
	//if role has the permission skip it
	for _, permissionID := range permissionIDs {
		exists, err := RecordExists("role_permissions", "role_id = ? AND permission_id = ?", roleID, permissionID)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		// If it exists, remove the permission from the role
		err = removeRolePermission(roleID, permissionID)
		if err != nil {
			return err
		}
	}
	return nil
}
func removeRolePermission(roleID, permissionID string) error {
	query := `DELETE FROM role_permissions WHERE role_id = ? AND permission_id = ?`
	_, err := DB.Exec(query, roleID, permissionID)
	return err
}
