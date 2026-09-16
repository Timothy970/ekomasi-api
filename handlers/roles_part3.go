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
	"time"

	"github.com/gin-gonic/gin"
)

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
// @Success      200            {object}  map[string]any   "Available permissions"
// @Failure      400            {object}  dtos.ErrorResponse     "Database error"
// @Failure      401            {object}  dtos.ErrorResponse     "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/available [get]
func GetAvailablePermissions(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", ""); !ok {
		return
	}
	// Extract optional category filter
	category := c.Query("category")
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
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Failed to fetch available permissions",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		// Cache the fetched permissions for future requests
		_ = utils.SetCache(redisKey, availablePermissions)
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Available permissions fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: availablePermissions, Message: "Available permissions fetched successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  c.Request,
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
// @Success      200            {object}  map[string]any        "Permission added"
// @Failure      400            {object}  dtos.ErrorResponse          "Validation error or duplicate key"
// @Failure      401            {object}  dtos.ErrorResponse          "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/available [post]
func AddAvailablePermission(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "roles.create"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.AvailablePermission](c, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields (category, key, description)
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}
	// Add to available permissions catalog
	err := models.AddAvailablePermission(models.DB, req.Category, req.Key, req.Description)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to add available permission",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Available permission added successfully",
			Code:        http.StatusOK,
		},
		Payload: req, Message: "Available permission added successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  c.Request,
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
// @Success      200            {object}  map[string]any        "Permission removed"
// @Failure      400            {object}  dtos.ErrorResponse          "Permission not found"
// @Failure      401            {object}  dtos.ErrorResponse          "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/available [delete]
func RemoveAvailablePermission(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "roles.delete"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.AvailablePermission](c, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields (category, key)
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}
	// Remove from available permissions catalog
	err := models.RemoveAvailablePermission(models.DB, req.Category, req.Key)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to remove available permission",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Available permission removed successfully",
			Code:        http.StatusOK,
		},
		Payload: req, Message: "Available permission removed successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  c.Request,
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
// @Success      200            {object}  map[string]any              "Permission updated"
// @Failure      400            {object}  dtos.ErrorResponse                "Permission not found or validation error"
// @Failure      401            {object}  dtos.ErrorResponse                "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/permissions/available [patch]
func UpdateAvailablePermission(c *gin.Context) {
	// Start performance tracking
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", "roles.update"); !ok {
		return
	}
	// Decode JSON request body
	req, ok := DecodeRequestBody[dtos.UpdateAvailablePermission](c, requestSummary, start)
	if !ok {
		return
	}
	// Validate required fields (category, key, and at least one new field)
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Users") {
		return
	}
	// Update permission in catalog (by category and key)
	err := models.UpdateAvailablePermission(models.DB, req.Category, req.Key, req.Description, req.NewDescription, req.NewKey, req.NewCategory)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to update available permission",
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Available permission updated successfully",
			Code:        http.StatusOK,
		},
		Payload: req, Message: "Available permission updated successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  c.Request,
		RawBody:  requestSummary,
	})
}
