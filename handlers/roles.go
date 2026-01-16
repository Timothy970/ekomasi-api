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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
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
	// Validate all permission IDs exist before creating role
	for _, permissionID := range req.PermissionIDs {
		// Check if permission exists in database
		err := models.IsPermissionThere(permissionID)
		if err != nil {
			// Permission ID invalid - reject role creation
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Failed to create role due to invalid permission ID " + permissionID,
					Code:        http.StatusBadRequest,
				},
				Message:   fmt.Sprintf("Permission with ID %s does not exist", permissionID),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	// Create role in database with permission associations
	err := models.CreateRole(req.Name, req.Description, req.PermissionIDs)
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	// Extract optional query parameters for filtering
	name := r.URL.Query().Get("name")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Validate date format if date range date range provided
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
	roles, err := models.GetRoles(name, startDate, endDate)
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
// @Router       /api/roles/{role_id} [put]
func UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify user has admin privileges
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
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

	// Update role in database (name and description only)
	if err := models.UpdateRole(req.Name, req.Description, roleID); err != nil {
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	// Extract role ID from URL path
	params := mux.Vars(r)
	roleID := params["role_id"]

	// Delete role from database (cascades to role_permissions)
	if err := models.DeleteRole(roleID); err != nil {
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

// CreatePermissionHandler creates a new permission.
// Admin-only operation for defining granular access control permissions.
// Permissions are organized by category (Inventory, Orders, Products, etc.).
//
// @Summary      Create permission
// @Description  Create a new permission with category and unique key (admin only)
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string               true  "Bearer token"
// @Param        permission     body      dtos.Permission      true  "Permission details"
// @Success      201            {object}  dtos.SuccessResponse "Permission created"
// @Failure      400            {object}  dtos.ErrorResponse   "Validation error or duplicate key"
// @Failure      401            {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions [post]
func CreatePermissionHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.Permission](r, w, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields (name, category, key)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Create permission in database
	err := models.CreatePermission(req.Name, req.Description, req.Category, req.Key)
	if err != nil {
		// Permission creation failed (duplicate key or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create permission",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate permission cache
	_ = utils.DeleteCacheByPrefix(rolePerms)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Permission created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Permission created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// UpdatePermissionHandler updates an existing permission's details.
// Admin-only operation for modifying permission configuration.
// Updates name, description, category, and key.
//
// @Summary      Update permission
// @Description  Update permission details by ID (admin only)
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Param        Authorization   header    string               true  "Bearer token"
// @Param        permission_id   path      string               true  "Permission ID"
// @Param        permission      body      dtos.Permission      true  "Updated permission details"
// @Success      200             {object}  dtos.SuccessResponse "Permission updated"
// @Failure      400             {object}  dtos.ErrorResponse   "Validation error or permission not found"
// @Failure      401             {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/{permission_id} [put]
func UpdatePermissionHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.Permission](r, w, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Extract permission ID from URL path
	permissionID := mux.Vars(r)["permission_id"]
	// Update permission in database
	err := models.UpdatePermission(req.Name, req.Description, permissionID, req.Category, req.Key)
	if err != nil {
		// Update failed (permission not found or duplicate key)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update permission with ID " + permissionID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate permission cache
	_ = utils.DeleteCacheByPrefix(rolePerms)

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: permissionWithID + permissionID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Permission updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeletePermissionHandler removes a permission from the system.
// Admin-only operation for deleting permission configuration.
// Also removes permission from all role associations.
//
// @Summary      Delete permission
// @Description  Delete permission by ID (admin only)
// @Tags         Permissions
// @Produce      json
// @Param        Authorization   header    string                 true  "Bearer token"
// @Param        permission_id   path      string                 true  "Permission ID"
// @Success      200             {object}  dtos.SuccessResponse   "Permission deleted"
// @Failure      400             {object}  dtos.ErrorResponse     "Permission not found or in use"
// @Failure      401             {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/{permission_id} [delete]
func DeletePermissionHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	// Extract permission ID from URL path
	permissionID := mux.Vars(r)["permission_id"]
	// Delete permission from database (cascades to role_permissions)
	err := models.DeletePermission(permissionID)
	if err != nil {
		// Deletion failed (permission not found or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete permission with ID " + permissionID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate permission cache
	_ = utils.DeleteCacheByPrefix(rolePerms)

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: permissionWithID + permissionID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Permission deleted successfully",
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	// Extract optional category filter from query string
	category := r.URL.Query().Get("category")
	// Fetch permissions from database with optional category filter
	permissions, err := models.GetPermissions(category)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch permissions",
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

// GetPermissionByIDHandler retrieves a single permission by its ID.
// Admin-only operation for viewing detailed permission configuration.
//
// @Summary      Get permission by ID
// @Description  Retrieve detailed permission information by ID (admin only)
// @Tags         Permissions
// @Produce      json
// @Param        Authorization   header    string                 true  "Bearer token"
// @Param        permission_id   path      string                 true  "Permission ID"
// @Success      200             {object}  dtos.SuccessResponse   "Permission details"
// @Failure      400             {object}  dtos.ErrorResponse     "Permission not found"
// @Failure      401             {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/{permission_id} [get]
func GetPermissionByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	// Extract permission ID from URL path
	permissionID := mux.Vars(r)["permission_id"]
	// Fetch permission from database
	permission, err := models.GetPermissionByID(permissionID)
	if err != nil {
		// Permission not found
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch permission with ID " + permissionID,
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
			Description: permissionWithID + permissionID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   permission,
		Message:   "Permission fetched successfully",
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	// Decode JSON request body containing permission IDs
	req, ok := DecodeRequestBody[dtos.PermissionIDs](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate request structure
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Extract role ID from URL path
	roleID := mux.Vars(r)["role_id"]
	// Validate all permission IDs exist before adding to role
	for _, permissionID := range req.PermissionIDs {
		// Verify each permission exists in database
		err := models.IsPermissionThere(permissionID)
		if err != nil {
			// Permission ID invalid - reject entire operation
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Failed to add permissions to role due to invalid permission ID " + permissionID,
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	// Add permissions to role (creates role_permission associations)
	err := models.AddPermissionsToRole(roleID, req.PermissionIDs)
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	// Decode JSON request body containing permission IDs
	req, ok := DecodeRequestBody[dtos.PermissionIDs](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate request structure
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	// Extract role ID from URL path
	roleID := mux.Vars(r)["role_id"]
	// Validate all permission IDs exist before removing from role
	for _, permissionID := range req.PermissionIDs {
		// Verify each permission exists in database
		err := models.IsPermissionThere(permissionID)
		if err != nil {
			// Permission ID invalid - reject entire operation
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Failed to remove permissions from role due to invalid permission ID " + permissionID,
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	// Remove permissions from role (deletes role_permission associations)
	err := models.RemovePermissionsFromRole(roleID, req.PermissionIDs)
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

// supportedPermissions defines the complete permission hierarchy for the application.
// Organized by functional categories for granular access control.
// Each permission has a unique key (e.g., "inventory.create"), category, and description.
// Used as fallback when database permissions are unavailable.
var supportedPermissions = []dtos.AvailablePermission{
	// INVENTORY
	{Category: "Inventory", Key: "inventory.create", Description: "Add new inventory item"},
	{Category: "Inventory", Key: "inventory.view", Description: "View inventory list and details"},
	{Category: "Inventory", Key: "inventory.update", Description: "Edit inventory item"},
	{Category: "Inventory", Key: "inventory.delete", Description: "Delete inventory item"},

	// ORDERS
	{Category: "Orders", Key: "orders.create", Description: "Create new order"},
	{Category: "Orders", Key: "orders.view", Description: "View orders and order details"},
	{Category: "Orders", Key: "orders.update", Description: "Update or modify order"},
	{Category: "Orders", Key: "orders.delete", Description: "Cancel or delete order"},

	// PRODUCTS
	{Category: "Products", Key: "products.create", Description: "Add new product"},
	{Category: "Products", Key: "products.view", Description: "View products list"},
	{Category: "Products", Key: "products.update", Description: "Edit product details"},
	{Category: "Products", Key: "products.delete", Description: "Delete product from catalog"},

	// USERS & ROLES
	{Category: "Users", Key: "users.create", Description: "Add new user"},
	{Category: "Users", Key: "users.view", Description: "View users"},
	{Category: "Users", Key: "users.update", Description: "Edit user info"},
	{Category: "Users", Key: "users.delete", Description: "Delete user"},
	{Category: "Users", Key: "roles.create", Description: "Create new role"},
	{Category: "Users", Key: "roles.view", Description: "View roles and permissions"},
	{Category: "Users", Key: "roles.update", Description: "Update existing roles"},
	{Category: "Users", Key: "roles.delete", Description: "Delete a role"},

	// REPORTS
	{Category: "Reports", Key: "reports.view", Description: "View all reports"},
	{Category: "Reports", Key: "reports.download", Description: "Download or export report data"},
	{Category: "Reports", Key: "reports.generate", Description: "Generate reports manually"},

	// PAYMENTS
	{Category: "Payments", Key: "payments.create", Description: "Initiate new payment"},
	{Category: "Payments", Key: "payments.view", Description: "View payment transactions"},
	{Category: "Payments", Key: "payments.refund", Description: "Process payment refund"},
	{Category: "Payments", Key: "payments.update", Description: "Update payment status"},

	// CUSTOMERS
	{Category: "Customers", Key: "customers.create", Description: "Add new customer"},
	{Category: "Customers", Key: "customers.view", Description: "View customers"},
	{Category: "Customers", Key: "customers.update", Description: "Edit customer details"},
	{Category: "Customers", Key: "customers.delete", Description: "Delete customer"},

	// SUPPLIERS
	{Category: "Suppliers", Key: "suppliers.create", Description: "Add new supplier"},
	{Category: "Suppliers", Key: "suppliers.view", Description: "View supplier list"},
	{Category: "Suppliers", Key: "suppliers.update", Description: "Edit supplier details"},
	{Category: "Suppliers", Key: "suppliers.delete", Description: "Remove supplier"},

	// SETTINGS
	{Category: "Settings", Key: "settings.view", Description: "View system settings"},
	{Category: "Settings", Key: "settings.update", Description: "Modify application settings"},

	// WAREHOUSE
	{Category: "Warehouse", Key: "warehouse.view", Description: "View warehouse stock and details"},
	{Category: "Warehouse", Key: "warehouse.update", Description: "Update warehouse information"},

	// LOGS & AUDIT
	{Category: "Logs", Key: "logs.view", Description: "View system or user activity logs"},
	{Category: "Logs", Key: "logs.export", Description: "Export system logs for analysis"},
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
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
		availablePermissions, err = models.GetAvailablePermissions(category)
		if availablePermissions == nil {
			// Database returned nothing - use hardcoded fallback
			availablePermissions = supportedPermissions
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
		Payload: supportedPermissions, Message: "Available permissions fetched successfully", TimeTaken: time.Since(start),
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
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
	err := models.AddAvailablePermission(req.Category, req.Key, req.Description)
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
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
	err := models.RemoveAvailablePermission(req.Category, req.Key)
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
// @Router       /api/permissions/available [put]
func UpdateAvailablePermission(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
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
	err := models.UpdateAvailablePermission(req.Category, req.Key, req.Description, req.NewDescription, req.NewKey, req.NewCategory)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update available permission",
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
			Description: "Available permission updated successfully",
			Code:        http.StatusOK,
		},
		Payload: req, Message: "Available permission updated successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}
