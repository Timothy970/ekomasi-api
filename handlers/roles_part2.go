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
// @Success      200            {object}  map[string]interface{}   "Role deleted"
// @Failure      400            {object}  dtos.ErrorResponse     "Role not found or in use"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles/{role_id} [delete]
func DeleteRoleHandler(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "roles.delete"); !ok {
		return
	}
	// Extract role ID from URL path
	roleID := c.Param("role_id")

	// Delete role from database (cascades to role_permissions)
	if err := models.DeleteRole(models.DB, roleID); err != nil {
		// Deletion failed (role not found or still assigned to users)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete role with ID " + roleID,
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
			Description: "Role with ID " + roleID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Role deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200            {object}  map[string]interface{}   "Permissions list"
// @Failure      400            {object}  dtos.ErrorResponse     "Database error"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions [get]
func GetPermissionsHandler(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", ""); !ok {
		return
	}
	// Extract optional category filter from query string
	category := c.Query("category")
	q := c.Query("q")
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Permissions fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   permissions,
		Message:   "Permissions fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Param        permissions    body      map[string][]string  true  "Permission IDs to add"
// @Success      200            {object}  map[string]interface{} "Permissions added"
// @Failure      400            {object}  dtos.ErrorResponse   "Invalid permission ID or role not found"
// @Failure      401            {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles/{role_id}/permissions [post]
func AddPermissionsToRoleHandler(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "roles.update"); !ok {
		return
	}
	// Decode JSON request body containing permission IDs
	req, ok := DecodeRequestBody[dtos.PermissionKeys](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate request structure
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}
	// Extract role ID from URL path
	roleID := c.Param("role_id")
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
	// Add permissions to role (creates role_permission associations)
	err = models.AddPermissionsToRole(models.DB, roleID, validatedPermissions)
	if err != nil {
		// Association failed (role not found or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to add permissions to role with ID " + roleID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
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
			Description: "Permissions added to role with ID " + roleID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Permissions added to role successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Param        permissions    body      map[string][]string  true  "Permission IDs to remove"
// @Success      200            {object}  map[string]interface{} "Permissions removed"
// @Failure      400            {object}  dtos.ErrorResponse   "Invalid permission ID or role not found"
// @Failure      401            {object}  dtos.ErrorResponse   "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/roles/{role_id}/permissions [delete]
func RemovePermissionsFromRoleHandler(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "roles.update"); !ok {
		return
	}
	// Decode JSON request body containing permission IDs
	req, ok := DecodeRequestBody[dtos.PermissionKeys](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate request structure
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}
	// Extract role ID from URL path
	roleID := c.Param("role_id")
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
	// Remove permissions from role (deletes role_permission associations)
	err = models.RemovePermissionsFromRole(models.DB, roleID, validatedPermissions)
	if err != nil {
		// Disassociation failed (role not found or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to remove permissions from role with ID " + roleID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
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
			Description: "Permissions removed from role with ID " + roleID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Permissions removed from role successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
