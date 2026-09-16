// Package handlers provides HTTP request handlers for purchase order management.
// This file contains handlers for creating, updating, retrieving, and deleting purchase orders,
// as well as managing purchase order items (products) within orders.
// Purchase orders track inventory procurement from suppliers.
package handlers

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// PurchaseOrderHandler handles purchase order-related HTTP requests
type PurchaseOrderHandler struct {
	DB *sql.DB
}

// pOrderWithID is a reusable string constant for purchase order messages
var pOrderWithID = "Purchase Order with ID "

// CreatePurchaseOrder creates a new purchase order for inventory procurement.
// It allows administrators to create orders for purchasing products from suppliers,
// including supplier details, expected delivery dates, and order status.
//
// @Summary      Create a new purchase order
// @Description  Create a purchase order for procuring inventory from suppliers with order details and line items
// @Tags         Purchase Orders
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.CreatePurchaseOrderRequest  true  "Purchase order details"
// @Success      200      {object}  map[string]interface{}           "Purchase order created successfully"
// @Failure      400      {object}  dtos.ErrorResponse               "Invalid request data"
// @Failure      500      {object}  dtos.ErrorResponse               "Internal server error"
// @Security     BearerAuth
// @Router       /admin/purchase-orders [post]
func CreatePurchaseOrder(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for creating purchase orders)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.CreatePurchaseOrderRequest](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Insert new purchase order into database
	err := models.AddNewPurchaseOrder(models.DB, *req)
	if err != nil {
		// Database insertion failed, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to add new purchase order",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect new data
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming order creation
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Purchase order added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Purchase order added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListPurchaseOrders retrieves a paginated list of all purchase orders.
// It provides an overview of all purchase orders with pagination support and caching
// for efficient data retrieval and display.
//
// @Summary      List all purchase orders
// @Description  Retrieve a paginated list of all purchase orders with supplier and status information
// @Tags         Purchase Orders
// @Produce      json
// @Param        page   query     int     false  "Page number (default: 1)"
// @Param        size   query     int     false  "Page size (default: 10)"
// @Success      200    {object}  dtos.PaginatedPurchaseOrdersResponse  "List of purchase orders with pagination"
// @Failure      500    {object}  dtos.ErrorResponse                    "Internal server error"
// @Security     BearerAuth
// @Router       /purchase-orders [get]
func ListPurchaseOrders(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for viewing purchase orders)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	// Generate cache keys for purchase orders and pagination metadata
	cacheKeyPurchaseOrders := fmt.Sprintf("purchase_orders_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("purchase_orders_pagination_%d_size_%d", page, size)
	// Initialize variables for orders and pagination metadata
	var orders []dtos.PurchaseOrderResponse
	var cachedOrders []dtos.PurchaseOrderResponse
	var meta *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	// Try to retrieve purchase orders from Redis cache
	_ = utils.GetCache(cacheKeyPurchaseOrders, &cachedOrders)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedOrders == nil {
		// Cache miss - fetch from database
		var err error
		purchaseOrders, pagination, err := models.ListPurchaseOrders(models.DB, page, size)
		if err != nil {
			// Database query failed, return error response
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Orders",
					Description: "Failed to list purchase orders",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		// Cache the fetched purchase orders for future requests
		_ = utils.SetCache(cacheKeyPurchaseOrders, purchaseOrders)
		_ = utils.SetCache(cacheKeyPagination, pagination)
	} else {
		// Cache hit - use cached data
		orders = cachedOrders
		meta = cachedPagination
	}
	// Build paginated response structure
	resp := dtos.PaginatedPurchaseOrdersResponse{Data: orders, Meta: *meta}

	// Return successful response with purchase orders list and pagination
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Purchase orders fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Purchase orders fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// GetPurchaseOrder retrieves a specific purchase order by its ID.
// It provides detailed information about a single purchase order including
// supplier details, order items, status, and delivery information.
//
// @Summary      Get purchase order by ID
// @Description  Retrieve detailed information for a specific purchase order including all line items
// @Tags         Purchase Orders
// @Produce      json
// @Param        po_id  path      string  true  "Purchase order ID"
// @Success      200    {object}  dtos.PurchaseOrderResponse  "Purchase order details"
// @Failure      404    {object}  dtos.ErrorResponse          "Purchase order not found"
// @Security     BearerAuth
// @Router       /purchase-orders/{po_id} [get]
func GetPurchaseOrder(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for viewing purchase orders)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract purchase order ID from URL path parameters
	id := c.Param("po_id")
	// Fetch purchase order details from database
	order, err := models.GetPurchaseOrderByID(models.DB, id)
	if err != nil {
		// Purchase order not found or database error, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch purchase order with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return successful response with purchase order details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: pOrderWithID + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   order,
		Message:   "Purchase order fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}

// UpdatePurchaseOrder updates an existing purchase order.
// It allows administrators to modify purchase order details such as supplier information,
// expected delivery dates, status, and other order attributes.
//
// @Summary      Update an existing purchase order
// @Description  Update purchase order details including supplier, delivery dates, status, and order information
// @Tags         Purchase Orders
// @Accept       json
// @Produce      json
// @Param        po_id    path      string                            true  "Purchase order ID"
// @Param        request  body      dtos.UpdatePurchaseOrderRequest   true  "Updated purchase order details"
// @Success      200      {object}  map[string]interface{}            "Purchase order updated successfully"
// @Failure      400      {object}  dtos.ErrorResponse                "Invalid request data"
// @Failure      404      {object}  dtos.ErrorResponse                "Purchase order not found"
// @Security     BearerAuth
// @Router       /admin/purchase-orders/{po_id} [patch]
func UpdatePurchaseOrder(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for updating purchase orders)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.UpdatePurchaseOrderRequest](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract purchase order ID from URL path parameters
	id := c.Param("po_id")
	// Update purchase order in database
	err := models.UpdatePurchaseOrder(models.DB, *req, id)
	if err != nil {
		// Database update failed, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update purchase order with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect updated data
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming update
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: pOrderWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Purchase order updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}
