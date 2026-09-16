// Package handlers provides HTTP request handlers for role-based access control (RBAC).
// Implements role and permission management for granular authorization control.
// Supports multi-level permission hierarchy with categories (Inventory, Orders, Products, Users, etc.).
// All operations require admin privileges and include cache invalidation for permission changes.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
// @Success      201            {object}  map[string]any "Role created"
// @Failure      400            {object}  dtos.ErrorResponse   "Invalid permission ID or validation error"
// @Failure      401            {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles [post]
func CreateRoleHandler(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify user has admin privileges (only admins can create roles)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "roles.create"); !ok {
		return
	}
	// Decode JSON request body into RoleRequest DTO
	req, ok := DecodeRequestBody[dtos.RoleRequest](c, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields (name, description)
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}
	// Validate all permission keys exist in supported permissions
	availablePermissions := utils.SupportedPermissions
	found, err, validatedPermissions := isValidPermissionKeys(req.PermissionKeys, availablePermissions)

	if !found {
		// Permission key invalid - reject role creation
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create role due to invalid permission key",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Create role in database with validated permissions
	err = models.CreateRole(models.DB, req.Name, req.Description, validatedPermissions)
	if err != nil {
		// Role creation failed (e.g., duplicate name, database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create role",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate role-permission cache to ensure fresh data
	_ = utils.DeleteCacheByPrefix(rolePerms)
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Role created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Role created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// isValidPermissionKeys validates that all permission keys exist in supported permissions.
// Returns true if all keys are valid, false otherwise.
func isValidPermissionKeys(permissionKeys []string, supportedPermissions []dtos.AvailablePermission) (bool, error, []dtos.AvailablePermission) {
	var validatedPermissions []dtos.AvailablePermission
	for _, permissionKey := range permissionKeys {
		found := false
		for _, availablePerm := range supportedPermissions {
			if strings.EqualFold(availablePerm.Key, permissionKey) {
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
// @Success      200            {object}  map[string]any   "Roles list"
// @Failure      400            {object}  dtos.ErrorResponse     "Invalid date format"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles [get]
func GetRolesHandler(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", ""); !ok {
		return
	}
	// Extract optional query parameters for filtering
	name := c.Query("name")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Validate date format if date range provided
	if startDate != "" && endDate != "" {
		// Validate start_date format (YYYY-MM-DD)
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			// Invalid start_date format
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Invalid start_date format. Use YYYY-MM-DD",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid start_date format. Use YYYY-MM-DD",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		// Validate end_date format (YYYY-MM-DD)
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			// Invalid end_date format
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Invalid end_date format. Use YYYY-MM-DD",
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid end_date format. Use YYYY-MM-DD",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}

	// Fetch roles from database with optional filters
	roles, err := models.GetRoles(models.DB, name, startDate, endDate)
	if err != nil {
		// Database query failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch roles",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return filtered roles list

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Roles fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   roles,
		Message:   "Role fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200            {object}  map[string]any "Role updated"
// @Failure      400            {object}  dtos.ErrorResponse   "Validation error or role not found"
// @Failure      401            {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles/{role_id} [patch]
func UpdateRoleHandler(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "roles.update"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.RoleRequest](c, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}
	// Extract role ID from URL path
	roleID := c.Param("role_id")
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
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update role with ID " + roleID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate role-permission cache
	_ = utils.DeleteCacheByPrefix(rolePerms)
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Role with ID " + roleID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Role updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}
