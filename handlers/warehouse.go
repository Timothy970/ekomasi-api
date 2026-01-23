// Package handlers provides HTTP request handlers for warehouse management.
// This file contains handlers for managing warehouse locations in the e-commerce platform,
// including CRUD operations for warehouse facilities, inventory tracking locations, and
// distribution center management. Essential for multi-location inventory control and order fulfillment.
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

// warehouseWithID is a constant prefix for warehouse-related log messages
var warehouseWithID = "Warehouse with ID "

// CreateWarehouse creates a new warehouse location in the system.
// Admin-only operation for adding warehouse facilities to support multi-location inventory.
// Validates warehouse details including name, address, capacity, and operational status.
//
// @Summary      Create warehouse
// @Description  Create a new warehouse facility with location and capacity details (admin only)
// @Tags         Warehouses
// @Accept       json
// @Produce      json
// @Param        warehouse  body      dtos.CreateWarehouseRequest  true  "Warehouse creation details"
// @Success      200        {object}  dtos.SuccessResponse         "Warehouse created successfully"
// @Failure      400        {object}  dtos.ErrorResponse           "Invalid request or creation failed"
// @Failure      401        {object}  dtos.ErrorResponse           "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/warehouses [post]
func CreateWarehouse(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can create warehouses)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Warehouse", "warehouse.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with warehouse details
	req, ok := DecodeRequestBody[dtos.CreateWarehouseRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (name, address, capacity, etc.)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Warehouse") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Create warehouse record in database
	_, err := models.CreateWarehouse(*req)
	if err != nil {
		// Warehouse creation failed (duplicate name, invalid data, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to create warehouse: " + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate warehouse caches to ensure fresh data
	utils.DeleteCacheByPrefix("warehouses_")
	utils.DeleteCacheByPrefix("warehouses_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Warehouse created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Warehouse created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ListWarehouses retrieves a paginated list of all warehouse locations.
// Public endpoint with Redis caching for performance optimization.
// Essential for displaying warehouse locations and inventory distribution.
//
// @Summary      List warehouses
// @Description  Retrieve paginated list of all warehouse facilities with caching
// @Tags         Warehouses
// @Produce      json
// @Param        page  query     int                            false  "Page number (default: 1)"
// @Param        size  query     int                            false  "Page size (default: 10)"
// @Success      200   {object}  dtos.ListWarehousesResponse    "Warehouses with pagination metadata"
// @Failure      500   {object}  dtos.ErrorResponse             "Failed to list warehouses"
// @Router       /api/warehouses [get]
func ListWarehouses(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Parse pagination parameters from query string
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Generate unique cache keys for warehouses data and pagination metadata
	cacheKeyWarehouses := fmt.Sprintf("warehouses_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("warehouses_pagination_%d_size_%d", page, size)
	var warehouses []dtos.Warehouse
	var cachedWarehouses []dtos.Warehouse
	var meta dtos.PaginationMeta
	var cachedPagination dtos.PaginationMeta
	// Attempt to retrieve warehouses from Redis cache
	_ = utils.GetCache(cacheKeyWarehouses, &cachedWarehouses)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	// If cache miss, fetch from database
	if cachedWarehouses == nil {
		var err error
		// Fetch paginated warehouses from database
		warehouses, meta, err = models.ListWarehouses(page, size)
		if err != nil {
			// Database query failed
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Warehouse",
					Description: "Failed to list warehouses",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Store results in Redis cache for faster subsequent requests
		_ = utils.SetCache(cacheKeyWarehouses, cachedWarehouses)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		// Cache hit - use cached data
		warehouses = cachedWarehouses
		meta = cachedPagination
	}

	// Construct response with warehouses list and pagination metadata
	resp := dtos.ListWarehousesResponse{
		Data: warehouses,
		Meta: meta,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Warehouses fetched successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Warehouses fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetWarehouse retrieves detailed information for a specific warehouse.
// Returns complete warehouse details including location, capacity, and operational status.
// Public endpoint for displaying warehouse information.
//
// @Summary      Get warehouse by ID
// @Description  Retrieve detailed information for a specific warehouse facility
// @Tags         Warehouses
// @Produce      json
// @Param        warehouse_id  path      string               true  "Warehouse ID"
// @Success      200           {object}  dtos.Warehouse       "Warehouse details"
// @Failure      404           {object}  dtos.ErrorResponse   "Warehouse not found"
// @Router       /api/warehouses/{warehouse_id} [get]
func GetWarehouse(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract warehouse ID from URL path parameters
	id := mux.Vars(r)["warehouse_id"]
	// Fetch specific warehouse details from database
	warehouse, err := models.GetWarehouseByID(id)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to fetch warehouse with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Verify warehouse exists (additional null check)
	if warehouse == nil {
		// Warehouse not found in database
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: warehouseWithID + id + " not found",
				Code:        http.StatusNotFound,
			},
			Message:   "Warehouse not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: warehouseWithID + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   warehouse,
		Message:   "Warehouse fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateWarehouse modifies an existing warehouse's information.
// Admin-only operation for updating warehouse details like name, address, capacity, or status.
// Essential for maintaining accurate warehouse records and operational data.
//
// @Summary      Update warehouse
// @Description  Update warehouse facility information (admin only)
// @Tags         Warehouses
// @Accept       json
// @Produce      json
// @Param        warehouse_id  path      string                        true  "Warehouse ID"
// @Param        warehouse     body      dtos.UpdateWarehouseRequest   true  "Updated warehouse details"
// @Success      200           {object}  dtos.SuccessResponse          "Warehouse updated successfully"
// @Failure      400           {object}  dtos.ErrorResponse            "Invalid request or update failed"
// @Failure      401           {object}  dtos.ErrorResponse            "Admin authorization required"
// @Failure      404           {object}  dtos.ErrorResponse            "Warehouse not found"
// @Security     BearerAuth
// @Router       /api/admin/warehouses/{warehouse_id} [patch]
func UpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can update warehouses)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Warehouse", "warehouse.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with updated warehouse data
	req, ok := DecodeRequestBody[dtos.UpdateWarehouseRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Warehouse") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract warehouse ID from URL path parameters
	id := mux.Vars(r)["warehouse_id"]
	// Update warehouse information in database
	err := models.UpdateWarehouse(id, *req)
	if err != nil {
		// Update failed (warehouse not found, invalid data, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to update warehouse with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate warehouse caches to ensure fresh data
	utils.DeleteCacheByPrefix("warehouses_")
	utils.DeleteCacheByPrefix("warehouses_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: warehouseWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Warehouse updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteWarehouse removes a warehouse from the system.
// Admin-only operation for deactivating or removing warehouse locations.
// @Summary      Delete warehouse
// @Description  Delete or deactivate a warehouse facility (admin only)
// @Tags         Warehouses
// @Produce      json
// @Param        warehouse_id  path      string                  true  "Warehouse ID"
// @Success      200           {object}  dtos.SuccessResponse    "Warehouse deleted successfully"
// @Failure      401           {object}  dtos.ErrorResponse      "Admin authorization required"
// @Failure      404           {object}  dtos.ErrorResponse      "Warehouse not found"
// @Security     BearerAuth
// @Router       /api/admin/warehouses/{warehouse_id} [delete]
func DeleteWarehouse(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can delete warehouses)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Warehouse", "warehouse.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract warehouse ID from URL path parameters
	id := mux.Vars(r)["warehouse_id"]
	// Delete warehouse from database (may be soft delete)
	err := models.DeleteWarehouse(id)
	if err != nil {
		// Deletion failed (warehouse not found, has dependencies, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to delete warehouse with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate warehouse caches to ensure fresh data
	utils.DeleteCacheByPrefix("warehouses_")
	utils.DeleteCacheByPrefix("warehouses_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: warehouseWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Warehouse deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
