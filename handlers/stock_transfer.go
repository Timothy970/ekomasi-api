// Package handlers provides HTTP request handlers for stock transfer management.
// This file contains handlers for warehouse-to-warehouse inventory transfers,
// allowing admins to move stock between locations, track transfer status,
// and maintain inventory accuracy across multiple warehouses.
package handlers

import (
	"github.com/gin-gonic/gin"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"
)

// CreateStockTransfer initiates a new stock transfer between warehouses.
// Admin-only operation for moving inventory from one warehouse to another.
// Validates that source and destination warehouses are different and updates inventory levels.
//
// @Summary      Create stock transfer
// @Description  Initiate a new stock transfer between warehouses (admin only)
// @Tags         Stock Transfers
// @Accept       json
// @Produce      json
// @Param        transfer  body      dtos.StockTransferDTO    true  "Stock transfer details"
// @Success      201       {object}  map[string]interface{}     "Transfer created successfully"
// @Failure      400       {object}  dtos.ErrorResponse       "Invalid request or validation failed"
// @Failure      401       {object}  dtos.ErrorResponse       "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/stock_transfers [post]
func CreateStockTransfer(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can create stock transfers)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Warehouse", "warehouse.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with transfer details
	req, ok := DecodeRequestBody[dtos.StockTransferDTO](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (from/to warehouse, product, quantity, etc.)
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Warehouse") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Prevent transfers within the same warehouse (business rule validation)
	if req.FromWarehouseID == req.ToWarehouseID {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "From and To warehouse cannot be the same",
				Code:        http.StatusBadRequest,
			},
			Message:   "From and To warehouse cannot be the same",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary})
		return
	}
	// Create stock transfer record and update inventory levels in both warehouses
	if err := models.CreateStockTransfer(models.DB, *req); err != nil {
		// Transfer creation failed (insufficient stock, invalid warehouses, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to create stock transfer",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate transfer caches to ensure fresh data
	utils.DeleteCacheByPrefix("transfers_")
	utils.DeleteCacheByPrefix("transfers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Stock transfer created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Stock transfer created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary})
}

// ListStockTransfers retrieves a paginated list of all stock transfers.
// Admin-only operation for viewing transfer history with optional search filtering.
// Essential for tracking inventory movements and auditing warehouse operations.
//
// @Summary      List stock transfers
// @Description  Retrieve paginated list of stock transfers with optional search (admin only)
// @Tags         Stock Transfers
// @Produce      json
// @Param        page  query     int                       false  "Page number (default: 1)"
// @Param        size  query     int                       false  "Page size (default: 10)"
// @Param        q     query     string                    false  "Search query"
// @Success      200   {object}  map[string]interface{}    "Stock transfers with pagination"
// @Failure      401   {object}  dtos.ErrorResponse        "Admin authorization required"
// @Failure      500   {object}  dtos.ErrorResponse        "Failed to list transfers"
// @Security     BearerAuth
// @Router       /api/stock_transfers [get]
func ListStockTransfers(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can view stock transfers)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Warehouse", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract optional search query parameter
	q := c.Query("q")
	// Parse pagination parameters from query string
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	// Fetch paginated stock transfers from database with search filtering
	transfers, meta, err := models.ListStockTransfers(models.DB, page, size, q)
	if err != nil {
		// Database query failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to list stock transfers",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary})
		return
	}
	// Return stock transfers list with pagination metadata for inventory tracking
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Stock transfers fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"stock_transfers": transfers, "pagination": meta},
		Message:   "Stock transfers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary})
}

// GetStockTransfer retrieves detailed information for a specific stock transfer.
// Displays transfer details including source/destination warehouses, product, quantity, and status.
// Useful for tracking individual transfer history and verification.
//
// @Summary      Get stock transfer by ID
// @Description  Retrieve detailed information for a specific stock transfer
// @Tags         Stock Transfers
// @Produce      json
// @Param        transfer_id  path      string                  true  "Transfer ID"
// @Success      200          {object}  map[string]interface{}      "Transfer details"
// @Failure      404          {object}  dtos.ErrorResponse      "Transfer not found"
// @Router       /api/stock_transfers/{transfer_id} [get]
func GetStockTransfer(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract transfer ID from URL path parameters
	id := c.Param("transfer_id")

	// Fetch specific stock transfer details from database
	st, err := models.GetStockTransferByID(models.DB, id)
	if err != nil {
		// Transfer not found or database error
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to fetch stock transfer with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return stock transfer details for verification and tracking

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Stock transfer fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   st,
		Message:   "Stock transfer fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary})
}

// UpdateStockTransfer modifies the quantity of an existing stock transfer.
// Admin-only operation for adjusting transfer amounts before completion.
// Updates inventory levels in both warehouses to reflect the new quantity.
//
// @Summary      Update stock transfer
// @Description  Update stock transfer quantity (admin only)
// @Tags         Stock Transfers
// @Accept       json
// @Produce      json
// @Param        transfer_id  path      string                       true  "Transfer ID"
// @Param        update       body      dtos.StockTransferUpdateDTO  true  "Updated quantity"
// @Success      200          {object}  map[string]interface{}         "Transfer updated successfully"
// @Failure      400          {object}  dtos.ErrorResponse           "Invalid request or update failed"
// @Failure      401          {object}  dtos.ErrorResponse           "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/stock_transfers/{transfer_id} [patch]
func UpdateStockTransfer(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can update stock transfers)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Warehouse", "warehouse.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with updated quantity
	req, ok := DecodeRequestBody[dtos.StockTransferUpdateDTO](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate the quantity field in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Warehouse") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract transfer ID from URL path parameters
	id := c.Param("transfer_id")

	// Update stock transfer quantity in database and adjust inventory levels
	if err := models.UpdateStockTransfer(models.DB, *req, id); err != nil {
		// Update failed (insufficient stock, invalid transfer, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to update stock transfer with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate transfer caches to ensure fresh data
	utils.DeleteCacheByPrefix("transfers_")
	utils.DeleteCacheByPrefix("transfers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: "Stock transfer with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Stock transfer updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary})
}
