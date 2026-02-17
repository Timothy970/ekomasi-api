// Package handlers provides HTTP request handlers for role-based access control (RBAC).
// Implements role and permission management for granular authorization control.
// Supports multi-level permission hierarchy with categories (Inventory, Orders, Products, Users, etc.).
// All operations require admin privileges and include cache invalidation for permission changes.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Package-level cache key prefixes for Redis caching
var (
	// rolePerms is the Redis cache prefix for role-permission associations
	rolePerms = "role_permissions:"
	// permissionWithID is a reusable string template for permission messages
	permissionWithID = "Permission with ID "
)

// CreateRoleHandler creates a new role with associated permissions.
// Admin-only operation for defining new access control roles.
// Validates all permission IDs before creating role.
//
// @Summary      Create new role
// @Description  Create a new role with name, description, and permission assignments (admin only)
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string               true  "Bearer token"
// @Param        role           body      dtos.RoleRequest     true  "Role details with permissions"
// @Success      201            {object}  dtos.SuccessResponse "Role created"
// @Failure      400            {object}  dtos.ErrorResponse   "Invalid permission ID or validation error"
// @Failure      401            {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles [post]
func CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify user has admin privileges (only admins can create roles)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "roles.create"); !ok {
		return
	}
	// Decode JSON request body into RoleRequest DTO
	req, ok := DecodeRequestBody[dtos.RoleRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields (name, description)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Validate all permission keys exist in supported permissions
	availablePermissions := utils.SupportedPermissions
	found, err, validatedPermissions := isValidPermissionKeys(req.PermissionKeys, availablePermissions)

	if !found {
		// Permission key invalid - reject role creation
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create role due to invalid permission key",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Create role in database with validated permissions
	err = models.CreateRole(models.DB, req.Name, req.Description, validatedPermissions)
	if err != nil {
		// Role creation failed (e.g., duplicate name, database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create role",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate role-permission cache to ensure fresh data
	_ = utils.DeleteCacheByPrefix(rolePerms)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Role created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Role created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// isValidPermissionKeys validates that all permission keys exist in supported permissions.
// Returns true if all keys are valid, false otherwise.
func isValidPermissionKeys(permissionKeys []string, supportedPermissions []dtos.AvailablePermission) (bool, error, []dtos.AvailablePermission) {
	var validatedPermissions []dtos.AvailablePermission
	for _, permissionKey := range permissionKeys {
		found := false
		for _, availablePerm := range supportedPermissions {
			if strings.ToLower(availablePerm.Key) == strings.ToLower(permissionKey) {
				validatedPermissions = append(validatedPermissions, availablePerm)
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Errorf("permission key %s not found in supported permissions", permissionKey), nil
		}
	}
	return true, nil, validatedPermissions
}

// GetRolesHandler retrieves roles with optional filtering.
// Admin-only operation for viewing role configuration.
// Supports filtering by role name and creation date range.
//
// @Summary      List roles
// @Description  Retrieve all roles with optional name and date range filters (admin only)
// @Tags         Roles
// @Produce      json
// @Param        Authorization  header    string                 true   "Bearer token"
// @Param        name           query     string                 false  "Filter by role name"
// @Param        start_date     query     string                 false  "Filter from date (YYYY-MM-DD)"
// @Param        end_date       query     string                 false  "Filter to date (YYYY-MM-DD)"
// @Success      200            {object}  dtos.SuccessResponse   "Roles list"
// @Failure      400            {object}  dtos.ErrorResponse     "Invalid date format"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles [get]
func GetRolesHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", ""); !ok {
		return
	}
	// Extract optional query parameters for filtering
	name := r.URL.Query().Get("name")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Validate date format if date range provided
	if startDate != "" && endDate != "" {
		// Validate start_date format (YYYY-MM-DD)
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			// Invalid start_date format
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Invalid start_date format. Use YYYY-MM-DD",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid start_date format. Use YYYY-MM-DD",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Validate end_date format (YYYY-MM-DD)
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			// Invalid end_date format
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Invalid end_date format. Use YYYY-MM-DD",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid end_date format. Use YYYY-MM-DD",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}

	// Fetch roles from database with optional filters
	roles, err := models.GetRoles(models.DB, name, startDate, endDate)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch roles",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return filtered roles list

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Roles fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   roles,
		Message:   "Role fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateRoleHandler updates an existing role's name and description.
// Admin-only operation for modifying role configuration.
// Does not modify permission associations (use AddPermissionsToRoleHandler/RemovePermissionsFromRoleHandler).
//
// @Summary      Update role
// @Description  Update role name and description by ID (admin only)
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string               true  "Bearer token"
// @Param        role_id        path      string               true  "Role ID"
// @Param        role           body      dtos.RoleRequest     true  "Updated role details"
// @Success      200            {object}  dtos.SuccessResponse "Role updated"
// @Failure      400            {object}  dtos.ErrorResponse   "Validation error or role not found"
// @Failure      401            {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles/{role_id} [patch]
func UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "roles.update"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.RoleRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Extract role ID from URL path
	params := mux.Vars(r)
	roleID := params["role_id"]
	permissions := []dtos.AvailablePermission{}
	for _, key := range req.PermissionKeys {
		// Find the permission in SupportedPermissions by matching the key
		for _, supportedPerm := range utils.SupportedPermissions {
			if strings.EqualFold(supportedPerm.Key, key) {
				permissions = append(permissions, supportedPerm)
				break
			}
		}
	}
	// Update role in database (name and description only)
	if err := models.UpdateRole(models.DB, *req, permissions, roleID); err != nil {
		// Update failed (role not found or duplicate name)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update role with ID " + roleID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate role-permission cache
	_ = utils.DeleteCacheByPrefix(rolePerms)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Role with ID " + roleID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Role updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// DeleteRoleHandler removes a role from the system.
// Admin-only operation for deleting role configuration.
// Also removes all role-permission associations.
//
// @Summary      Delete role
// @Description  Delete role by ID (admin only)
// @Tags         Roles
// @Produce      json
// @Param        Authorization  header    string                 true  "Bearer token"
// @Param        role_id        path      string                 true  "Role ID"
// @Success      200            {object}  dtos.SuccessResponse   "Role deleted"
// @Failure      400            {object}  dtos.ErrorResponse     "Role not found or in use"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles/{role_id} [delete]
func DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "roles.delete"); !ok {
		return
	}
	// Extract role ID from URL path
	params := mux.Vars(r)
	roleID := params["role_id"]

	// Delete role from database (cascades to role_permissions)
	if err := models.DeleteRole(models.DB, roleID); err != nil {
		// Deletion failed (role not found or still assigned to users)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete role with ID " + roleID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate role-permission cache
	_ = utils.DeleteCacheByPrefix(rolePerms)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Role with ID " + roleID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Role deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetPermissionsHandler retrieves all permissions with optional category filter.
// Admin-only operation for viewing permission configuration.
// Categories include: Inventory, Orders, Products, Users, Reports, Payments, etc.
//
// @Summary      List permissions
// @Description  Retrieve all permissions with optional category filter (admin only)
// @Tags         Permissions
// @Produce      json
// @Param        Authorization  header    string                 true   "Bearer token"
// @Param        category       query     string                 false  "Filter by category"
// @Success      200            {object}  dtos.SuccessResponse   "Permissions list"
// @Failure      400            {object}  dtos.ErrorResponse     "Database error"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions [get]
func GetPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", ""); !ok {
		return
	}
	// Extract optional category filter from query string
	category := r.URL.Query().Get("category")
	q := r.URL.Query().Get("q")
	// Fetch permissions from database with optional category filter
	permissions := utils.SupportedPermissions
	if category != "" {
		// Filter permissions by category
		var filteredPermissions []dtos.AvailablePermission
		for _, perm := range permissions {
			if strings.Contains(strings.ToLower(perm.Category), strings.ToLower(category)) {
				filteredPermissions = append(filteredPermissions, perm)
			}
		}
		permissions = filteredPermissions
	}
	if q != "" {
		// Filter permissions by category or key
		var filteredPermissions []dtos.AvailablePermission
		for _, perm := range permissions {
			if strings.Contains(strings.ToLower(perm.Category), strings.ToLower(q)) || strings.Contains(strings.ToLower(perm.Key), strings.ToLower(q)) {
				filteredPermissions = append(filteredPermissions, perm)
			}
		}
		permissions = filteredPermissions
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Permissions fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   permissions,
		Message:   "Permissions fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// AddPermissionsToRoleHandler adds permissions to an existing role.
// Admin-only operation for expanding role capabilities.
// Validates all permission IDs before adding to role.
//
// @Summary      Add permissions to role
// @Description  Associate multiple permissions with an existing role (admin only)
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string               true  "Bearer token"
// @Param        role_id        path      string               true  "Role ID"
// @Param        permissions    body      dtos.PermissionIDs   true  "Permission IDs to add"
// @Success      200            {object}  dtos.SuccessResponse "Permissions added"
// @Failure      400            {object}  dtos.ErrorResponse   "Invalid permission ID or role not found"
// @Failure      401            {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles/{role_id}/permissions [post]
func AddPermissionsToRoleHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "roles.update"); !ok {
		return
	}
	// Decode JSON request body containing permission IDs
	req, ok := DecodeRequestBody[dtos.PermissionKeys](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate request structure
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Extract role ID from URL path
	roleID := mux.Vars(r)["role_id"]
	// Validate all permission keys exist in supported permissions
	availablePermissions := utils.SupportedPermissions
	found, err, validatedPermissions := isValidPermissionKeys(req.PermissionKeys, availablePermissions)

	if !found {
		// Permission key invalid - reject role creation
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create role due to invalid permission key",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Add permissions to role (creates role_permission associations)
	err = models.AddPermissionsToRole(models.DB, roleID, validatedPermissions)
	if err != nil {
		// Association failed (role not found or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to add permissions to role with ID " + roleID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate role-permission cache
	_ = utils.DeleteCacheByPrefix(rolePerms)

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Permissions added to role with ID " + roleID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Permissions added to role successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// RemovePermissionsFromRoleHandler removes permissions from an existing role.
// Admin-only operation for restricting role capabilities.
// Validates all permission IDs before removing from role.
//
// @Summary      Remove permissions from role
// @Description  Disassociate multiple permissions from an existing role (admin only)
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string               true  "Bearer token"
// @Param        role_id        path      string               true  "Role ID"
// @Param        permissions    body      dtos.PermissionIDs   true  "Permission IDs to remove"
// @Success      200            {object}  dtos.SuccessResponse "Permissions removed"
// @Failure      400            {object}  dtos.ErrorResponse   "Invalid permission ID or role not found"
// @Failure      401            {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles/{role_id}/permissions [delete]
func RemovePermissionsFromRoleHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "roles.update"); !ok {
		return
	}
	// Decode JSON request body containing permission IDs
	req, ok := DecodeRequestBody[dtos.PermissionKeys](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate request structure
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Extract role ID from URL path
	roleID := mux.Vars(r)["role_id"]
	// Validate all permission keys exist in supported permissions
	availablePermissions := utils.SupportedPermissions
	found, err, validatedPermissions := isValidPermissionKeys(req.PermissionKeys, availablePermissions)

	if !found {
		// Permission key invalid - reject role creation
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create role due to invalid permission key",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Remove permissions from role (deletes role_permission associations)
	err = models.RemovePermissionsFromRole(models.DB, roleID, validatedPermissions)
	if err != nil {
		// Disassociation failed (role not found or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to remove permissions from role with ID " + roleID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate role-permission cache
	_ = utils.DeleteCacheByPrefix(rolePerms)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Permissions removed from role with ID " + roleID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Permissions removed from role successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetAvailablePermissions retrieves the complete permission catalog.
// Admin-only operation for viewing all available permissions.
// Returns from Redis cache if available, otherwise fetches from database with fallback to hardcoded list.
//
// @Summary      Get available permissions
// @Description  Retrieve complete catalog of available permissions with optional category filter (admin only)
// @Tags         Permissions
// @Produce      json
// @Param        Authorization  header    string                 true   "Bearer token"
// @Param        category       query     string                 false  "Filter by category"
// @Success      200            {object}  dtos.SuccessResponse   "Available permissions"
// @Failure      400            {object}  dtos.ErrorResponse     "Database error"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/available [get]
func GetAvailablePermissions(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", ""); !ok {
		return
	}
	// Extract optional category filter
	category := r.URL.Query().Get("category")
	// Generate Redis cache key based on category
	redisKey := "available_permissions"
	if category != "" {
		redisKey = fmt.Sprintf("%s:%s", redisKey, category)
	}
	var availablePermissions []dtos.AvailablePermission
	var cachedAvailablePermissions []dtos.AvailablePermission
	var err error
	// Try to fetch from Redis cache first
	_ = utils.GetCache(redisKey, &cachedAvailablePermissions)
	if len(cachedAvailablePermissions) > 0 {
		// Cache hit - use cached permissions
		availablePermissions = cachedAvailablePermissions
	} else {
		// Cache miss - fetch from database
		availablePermissions, err = models.GetAvailablePermissions(models.DB, category)
		if availablePermissions == nil {
			// Database returned nothing - use hardcoded fallback
			availablePermissions = utils.SupportedPermissions
		}
		if err != nil {
			// Database query failed
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Failed to fetch available permissions",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Cache the fetched permissions for future requests
		_ = utils.SetCache(redisKey, availablePermissions)
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Available permissions fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: availablePermissions, Message: "Available permissions fetched successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}

// AddAvailablePermission adds a new permission to the available permissions catalog.
// Admin-only operation for extending the permission system.
// Used to dynamically add custom permissions beyond the hardcoded list.
//
// @Summary      Add available permission
// @Description  Add a new permission to the system catalog (admin only)
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                      true  "Bearer token"
// @Param        permission     body      dtos.AvailablePermission    true  "Permission details"
// @Success      200            {object}  dtos.SuccessResponse        "Permission added"
// @Failure      400            {object}  dtos.ErrorResponse          "Validation error or duplicate key"
// @Failure      401            {object}  dtos.ErrorResponse          "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/available [post]
func AddAvailablePermission(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "roles.create"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.AvailablePermission](r, w, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields (category, key, description)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Add to available permissions catalog
	err := models.AddAvailablePermission(models.DB, req.Category, req.Key, req.Description)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to add available permission",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Available permission added successfully",
			Code:        http.StatusOK,
		},
		Payload: req, Message: "Available permission added successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}

// RemoveAvailablePermission removes a permission from the available permissions catalog.
// Admin-only operation for removing custom permissions.
// Does not affect existing role-permission associations.
//
// @Summary      Remove available permission
// @Description  Remove a permission from the system catalog (admin only)
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                      true  "Bearer token"
// @Param        permission     body      dtos.AvailablePermission    true  "Permission category and key"
// @Success      200            {object}  dtos.SuccessResponse        "Permission removed"
// @Failure      400            {object}  dtos.ErrorResponse          "Permission not found"
// @Failure      401            {object}  dtos.ErrorResponse          "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/available [delete]
func RemoveAvailablePermission(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "roles.delete"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.AvailablePermission](r, w, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields (category, key)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Remove from available permissions catalog
	err := models.RemoveAvailablePermission(models.DB, req.Category, req.Key)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to remove available permission",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Available permission removed successfully",
			Code:        http.StatusOK,
		},
		Payload: req, Message: "Available permission removed successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}

// UpdateAvailablePermission updates an existing permission in the catalog.
// Admin-only operation for modifying permission metadata.
// Can update category, key, and description.
//
// @Summary      Update available permission
// @Description  Update permission metadata in the system catalog (admin only)
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                            true  "Bearer token"
// @Param        permission     body      dtos.UpdateAvailablePermission    true  "Permission update details"
// @Success      200            {object}  dtos.SuccessResponse              "Permission updated"
// @Failure      400            {object}  dtos.ErrorResponse                "Permission not found or validation error"
// @Failure      401            {object}  dtos.ErrorResponse                "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/available [patch]
func UpdateAvailablePermission(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Users", "roles.update"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.UpdateAvailablePermission](r, w, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields (category, key, and at least one new field)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Update permission in catalog (by category and key)
	err := models.UpdateAvailablePermission(models.DB, req.Category, req.Key, req.Description, req.NewDescription, req.NewKey, req.NewCategory)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update available permission",
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Available permission updated successfully",
			Code:        http.StatusOK,
		},
		Payload: req, Message: "Available permission updated successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}
