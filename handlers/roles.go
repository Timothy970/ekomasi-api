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

var (
	rolePerms        = "role_permissions:"
	permissionWithID = "Permission with ID "
)

// CreateRoleHandler handles POST /api/roles
func CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.RoleRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	for _, permissionID := range req.PermissionIDs {
		err := models.IsPermissionThere(permissionID)
		if err != nil {
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
	err := models.CreateRole(req.Name, req.Description, req.PermissionIDs)
	if err != nil {
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

// GetRolesHandler handles GET /api/roles?name=admin&start_date=2025-01-01&end_date=2025-12-31
func GetRolesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	name := r.URL.Query().Get("name")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Validate date format if provided
	if startDate != "" && endDate != "" {
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
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
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
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

	roles, err := models.GetRoles(name, startDate, endDate)
	if err != nil {
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

// UpdateRoleHandler handles PUT /api/roles/{name}
func UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.RoleRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	params := mux.Vars(r)
	roleID := params["role_id"]

	if err := models.UpdateRole(req.Name, req.Description, roleID); err != nil {
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

// DeleteRoleHandler handles DELETE /api/roles/{name}
func DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	params := mux.Vars(r)
	roleID := params["role_id"]

	if err := models.DeleteRole(roleID); err != nil {
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

// Create permissions handler
func CreatePermissionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.Permission](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	err := models.CreatePermission(req.Name, req.Description, req.Category, req.Key)
	if err != nil {
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

// Update a permission handler
func UpdatePermissionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.Permission](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	permissionID := mux.Vars(r)["permission_id"]
	err := models.UpdatePermission(req.Name, req.Description, permissionID, req.Category, req.Key)
	if err != nil {
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

func DeletePermissionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	permissionID := mux.Vars(r)["permission_id"]
	err := models.DeletePermission(permissionID)
	if err != nil {
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
func GetPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	category := r.URL.Query().Get("category")
	permissions, err := models.GetPermissions(category)
	if err != nil {
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

func GetPermissionByIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	permissionID := mux.Vars(r)["permission_id"]
	permission, err := models.GetPermissionByID(permissionID)
	if err != nil {
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

func AddPermissionsToRoleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.PermissionIDs](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	roleID := mux.Vars(r)["role_id"]
	for _, permissionID := range req.PermissionIDs {
		err := models.IsPermissionThere(permissionID)
		if err != nil {
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
	err := models.AddPermissionsToRole(roleID, req.PermissionIDs)
	if err != nil {
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

// Remove permissions from role handler
func RemovePermissionsFromRoleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.PermissionIDs](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
	roleID := mux.Vars(r)["role_id"]
	for _, permissionID := range req.PermissionIDs {
		err := models.IsPermissionThere(permissionID)
		if err != nil {
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
	err := models.RemovePermissionsFromRole(roleID, req.PermissionIDs)
	if err != nil {
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

func GetAvailablePermissions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	category := r.URL.Query().Get("category")
	redisKey := "available_permissions"
	if category != "" {
		redisKey = fmt.Sprintf("%s:%s", redisKey, category)
	}
	var availablePermissions []dtos.AvailablePermission
	var cachedAvailablePermissions []dtos.AvailablePermission
	var err error
	_ = utils.GetCache(redisKey, &cachedAvailablePermissions)
	if len(cachedAvailablePermissions) > 0 {
		availablePermissions = cachedAvailablePermissions
	} else {
		// availablePermissions = supportedPermissions
		availablePermissions, err = models.GetAvailablePermissions(category)
		if availablePermissions == nil {
			availablePermissions = supportedPermissions
		}
		if err != nil {
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

func AddAvailablePermission(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.AvailablePermission](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
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

func RemoveAvailablePermission(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.AvailablePermission](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
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

func UpdateAvailablePermission(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateAvailablePermission](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Users") {
		return
	}
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
