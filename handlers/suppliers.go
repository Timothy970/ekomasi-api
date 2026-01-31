// Package handlers provides HTTP request handlers for supplier management.
// This file contains handlers for managing suppliers in the e-commerce platform,
// including CRUD operations for supplier information, contact details, and business relationships.
// Suppliers are critical for inventory management and purchase order fulfillment.
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

// supplierWithID is a constant prefix for supplier-related log messages
var supplierWithID = "Supplier with ID "

// CreateSupplier creates a new supplier in the system.
// Admin-only operation for adding new suppliers to support purchase orders and inventory management.
// Validates supplier information including name, contact details, and business information.
//
// @Summary      Create supplier
// @Description  Create a new supplier with business and contact information (admin only)
// @Tags         Suppliers
// @Accept       json
// @Produce      json
// @Param        supplier  body      dtos.Supplier           true  "Supplier details"
// @Success      200       {object}  dtos.SuccessResponse    "Supplier created successfully"
// @Failure      400       {object}  dtos.ErrorResponse      "Invalid request or validation failed"
// @Failure      401       {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/suppliers [post]
func CreateSupplier(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can create suppliers)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Suppliers", "suppliers.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with supplier details
	req, ok := DecodeRequestBody[dtos.Supplier](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (name, contact info, etc.)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Suppliers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Create supplier record in database
	if err := models.CreateSupplier(models.DB, *req); err != nil {
		// Supplier creation failed (duplicate, constraint violation, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Suppliers",
				Description: "Failed to create supplier",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate supplier caches to ensure fresh data
	utils.DeleteCacheByPrefix("suppliers_")
	utils.DeleteCacheByPrefix("suppliers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Suppliers",
			Description: "Supplier created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Supplier created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ListSuppliers retrieves a paginated list of all suppliers.
// Admin-only operation with Redis caching for performance optimization.
// Essential for viewing supplier directory and managing business relationships.
//
// @Summary      List suppliers
// @Description  Retrieve paginated list of all suppliers with caching (admin only)
// @Tags         Suppliers
// @Produce      json
// @Param        page  query     int                       false  "Page number (default: 1)"
// @Param        size  query     int                       false  "Page size (default: 10)"
// @Success      200   {object}  map[string]interface{}    "Suppliers with pagination metadata"
// @Failure      401   {object}  dtos.ErrorResponse        "Admin authorization required"
// @Failure      500   {object}  dtos.ErrorResponse        "Failed to list suppliers"
// @Security     BearerAuth
// @Router       /api/suppliers [get]
func ListSuppliers(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can view suppliers)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Suppliers", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Generate unique cache keys for suppliers data and pagination metadata
	cacheKeySuppliers := fmt.Sprintf("suppliers_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("suppliers_pagination_%d_size_%d", page, size)
	var suppliers []dtos.Supplier
	var cachedSupplier []dtos.Supplier
	var meta dtos.PaginationMeta
	var cachedPagination dtos.PaginationMeta
	// Attempt to retrieve suppliers from Redis cache
	_ = utils.GetCache(cacheKeySuppliers, &cachedSupplier)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	// If cache miss, fetch from database
	if cachedSupplier == nil {
		var err error
		// Fetch paginated suppliers from database
		suppliers, meta, err = models.ListSuppliers(models.DB, page, size)
		if err != nil {
			// Database query failed
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Suppliers",
					Description: "Failed to list suppliers",
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
		_ = utils.SetCache(cacheKeySuppliers, cachedSupplier)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		// Cache hit - use cached data
		suppliers = cachedSupplier
		meta = cachedPagination
	}
	// Construct response with suppliers and pagination metadata
	resp := map[string]interface{}{
		"suppliers":  suppliers,
		"pagination": meta,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Suppliers",
			Description: "Suppliers fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Suppliers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetSupplierByID retrieves detailed information for a specific supplier.
// Returns complete supplier profile including contact information and business details.
// Admin-only operation for managing supplier relationships.
//
// @Summary      Get supplier by ID
// @Description  Retrieve detailed information for a specific supplier (admin only)
// @Tags         Suppliers
// @Produce      json
// @Param        supplier_id  path      string                 true  "Supplier ID"
// @Success      200          {object}  dtos.Supplier          "Supplier details"
// @Failure      401          {object}  dtos.ErrorResponse     "Admin authorization required"
// @Failure      404          {object}  dtos.ErrorResponse     "Supplier not found"
// @Security     BearerAuth
// @Router       /api/suppliers/{supplier_id} [get]
func GetSupplierByID(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can view supplier details)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Suppliers", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract supplier ID from URL path parameters
	id := mux.Vars(r)["supplier_id"]

	// Fetch supplier details from database
	supplier, err := models.GetSupplierByID(models.DB, id)
	if err != nil {
		// Supplier not found or database error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Suppliers",
				Description: "Failed to fetch supplier with ID " + id,
				Code:        http.StatusNotFound,
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
			Module:      "Suppliers",
			Description: supplierWithID + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   supplier,
		Message:   "Supplier fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateSupplier modifies an existing supplier's information.
// Admin-only operation for updating supplier contact details, business information, and status.
// Validates phone number format (Kenyan format) before applying changes.
//
// @Summary      Update supplier
// @Description  Update supplier information including contact and business details (admin only)
// @Tags         Suppliers
// @Accept       json
// @Produce      json
// @Param        supplier_id  path      string                  true  "Supplier ID"
// @Param        supplier     body      dtos.Supplier           true  "Updated supplier details"
// @Success      200          {object}  dtos.SuccessResponse    "Supplier updated successfully"
// @Failure      400          {object}  dtos.ErrorResponse      "Invalid request or validation failed"
// @Failure      401          {object}  dtos.ErrorResponse      "Admin authorization required"
// @Failure      404          {object}  dtos.ErrorResponse      "Supplier not found"
// @Security     BearerAuth
// @Router       /api/suppliers/{supplier_id} [patch]
func UpdateSupplier(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can update suppliers)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Suppliers", "suppliers.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with updated supplier data
	req, ok := DecodeRequestBody[dtos.Supplier](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Suppliers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Validate contact phone number format (must be valid Kenyan phone format)
	if !utils.IsValidKenyanPhone(req.ContactPhone) {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Suppliers",
				Description: "Invalid phone number format",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid phone number",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Extract supplier ID from URL path parameters
	id := mux.Vars(r)["supplier_id"]

	// Update supplier information in database
	if err := models.UpdateSupplier(models.DB, *req, id); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Suppliers",
				Description: "Failed to update supplier with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate supplier caches to ensure fresh data after update
	utils.DeleteCacheByPrefix("suppliers_")
	utils.DeleteCacheByPrefix("suppliers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Suppliers",
			Description: supplierWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Supplier updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteSupplier removes a supplier from the system.
// Admin-only operation for deactivating or removing suppliers.
// @Summary      Delete supplier
// @Description  Delete or deactivate a supplier (admin only)
// @Tags         Suppliers
// @Produce      json
// @Param        supplier_id  path      string                  true  "Supplier ID"
// @Success      200          {object}  dtos.SuccessResponse    "Supplier deleted successfully"
// @Failure      401          {object}  dtos.ErrorResponse      "Admin authorization required"
// @Failure      404          {object}  dtos.ErrorResponse      "Supplier not found"
// @Security     BearerAuth
// @Router       /api/suppliers/{supplier_id} [delete]
func DeleteSupplier(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can delete suppliers)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Suppliers", "suppliers.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Extract supplier ID from URL path parameters
	id := mux.Vars(r)["supplier_id"]

	// Delete supplier from database (may be soft delete)
	if err := models.DeleteSupplier(models.DB, id); err != nil {
		// Supplier not found or database error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Suppliers",
				Description: "Failed to delete supplier with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate supplier caches to ensure fresh data after deletion
	utils.DeleteCacheByPrefix("suppliers_")
	utils.DeleteCacheByPrefix("suppliers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Suppliers",
			Description: supplierWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Supplier deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
