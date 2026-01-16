// Package handlers provides HTTP request handlers for purchase order management.
// This file contains handlers for creating, updating, retrieving, and deleting purchase orders,
// as well as managing purchase order items (products) within orders.
// Purchase orders track inventory procurement from suppliers.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
func CreatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for creating purchase orders)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.CreatePurchaseOrderRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Insert new purchase order into database
	err := models.AddNewPurchaseOrder(*req)
	if err != nil {
		// Database insertion failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to add new purchase order",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect new data
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming order creation
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Purchase order added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Purchase order added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func ListPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for viewing purchase orders)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
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
		orders, meta, err = models.ListPurchaseOrders(page, size)
		if err != nil {
			// Database query failed, return error response
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Orders",
					Description: "Failed to list purchase orders",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Cache the fetched purchase orders for future requests
		_ = utils.SetCache(cacheKeyPurchaseOrders, cachedOrders)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		// Cache hit - use cached data
		orders = cachedOrders
		meta = cachedPagination
	}
	// Build paginated response structure
	resp := dtos.PaginatedPurchaseOrdersResponse{Data: orders, Meta: *meta}

	// Return successful response with purchase orders list and pagination
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Purchase orders fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Purchase orders fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func GetPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for viewing purchase orders)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract purchase order ID from URL path parameters
	id := mux.Vars(r)["po_id"]
	// Fetch purchase order details from database
	order, err := models.GetPurchaseOrderByID(id)
	if err != nil {
		// Purchase order not found or database error, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch purchase order with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return successful response with purchase order details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: pOrderWithID + id + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   order,
		Message:   "Purchase order fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func UpdatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for updating purchase orders)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.UpdatePurchaseOrderRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract purchase order ID from URL path parameters
	id := mux.Vars(r)["po_id"]
	// Update purchase order in database
	err := models.UpdatePurchaseOrder(*req, id)
	if err != nil {
		// Database update failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update purchase order with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect updated data
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming update
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: pOrderWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Purchase order updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// DeletePurchaseOrder permanently removes a purchase order from the system.
// This action is irreversible and will delete the purchase order along with all its line items.
// Use with caution as this removes all historical procurement data.
//
// @Summary      Delete a purchase order
// @Description  Permanently remove a purchase order and all associated line items from the system
// @Tags         Purchase Orders
// @Produce      json
// @Param        po_id  path      string                  true  "Purchase order ID"
// @Success      200    {object}  map[string]interface{}  "Purchase order deleted successfully"
// @Failure      400    {object}  dtos.ErrorResponse      "Invalid request"
// @Failure      404    {object}  dtos.ErrorResponse      "Purchase order not found"
// @Security     BearerAuth
// @Router       /admin/purchase-orders/{po_id} [delete]
func DeletePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for deleting purchase orders)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract purchase order ID from URL path parameters
	id := mux.Vars(r)["po_id"]
	// Delete purchase order from database
	err := models.DeletePurchaseOrder(id)
	if err != nil {
		// Deletion failed or purchase order not found, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to delete purchase order with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect deletion
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming deletion
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: pOrderWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Purchase order deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// AddPurchaseOrderItem adds a product line item to an existing purchase order.
// It allows administrators to add products to a purchase order, specifying quantity,
// unit price, and other item details.
//
// @Summary      Add product to purchase order
// @Description  Add a product line item to an existing purchase order with quantity and pricing details
// @Tags         Purchase Orders
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.PurchaseOrderItem      true  "Purchase order item details"
// @Success      200      {object}  map[string]interface{}      "Product added to purchase order successfully"
// @Failure      400      {object}  dtos.ErrorResponse          "Invalid request data"
// @Failure      404      {object}  dtos.ErrorResponse          "Purchase order or product not found"
// @Security     BearerAuth
// @Router       /admin/purchase_order_items [post]
func AddPurchaseOrderItem(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for adding items to purchase orders)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.PurchaseOrderItem](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Add product to purchase order in database
	err := models.AddProductToPurchaseOrder(*req)
	if err != nil {
		// Database insertion failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to add product to purchase order with ID " + req.PoID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect new line item
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming product addition
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Product added to purchase order with ID " + req.PoID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product added to purchase order successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// RemovePurchaseOrderItem removes a product line item from a purchase order.
// This permanently deletes the line item from the purchase order.
//
// @Summary      Remove product from purchase order
// @Description  Remove a product line item from an existing purchase order
// @Tags         Purchase Orders
// @Produce      json
// @Param        po_item_id  path      string                  true  "Purchase order item ID"
// @Success      200         {object}  map[string]interface{}  "Product removed from purchase order successfully"
// @Failure      400         {object}  dtos.ErrorResponse      "Invalid request"
// @Failure      404         {object}  dtos.ErrorResponse      "Purchase order item not found"
// @Security     BearerAuth
// @Router       /admin/purchase_order_items/{po_item_id} [delete]
func RemovePurchaseOrderItem(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for removing items from purchase orders)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Orders")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract purchase order item ID from URL path parameters
	vars := mux.Vars(r)
	itemID := vars["po_item_id"]
	// Remove product from purchase order in database
	err := models.RemoveProductFromPurchaseOrder(itemID)
	if err != nil {
		// Deletion failed or item not found, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to remove product from purchase order item with ID " + itemID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect removed line item
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming removal
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Product removed from purchase order with ID " + itemID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product removed from purchase order successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
