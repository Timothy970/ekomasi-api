// Package handlers provides HTTP request handlers for purchase order management.
// This file contains handlers for creating, updating, retrieving, and deleting purchase orders,
// as well as managing purchase order items (products) within orders.
// Purchase orders track inventory procurement from suppliers.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
func DeletePurchaseOrder(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for deleting purchase orders)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract purchase order ID from URL path parameters
	id := c.Param("po_id")
	// Delete purchase order from database
	err := models.DeletePurchaseOrder(models.DB, id)
	if err != nil {
		// Deletion failed or purchase order not found, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to delete purchase order with ID " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect deletion
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming deletion
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: pOrderWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Purchase order deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func AddPurchaseOrderItem(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for adding items to purchase orders)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.PurchaseOrderItem](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Add product to purchase order in database
	err := models.AddProductToPurchaseOrder(models.DB, *req)
	if err != nil {
		// Database insertion failed, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to add product to purchase order with ID " + req.PoID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect new line item
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming product addition
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Product added to purchase order with ID " + req.PoID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product added to purchase order successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func RemovePurchaseOrderItem(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for removing items from purchase orders)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract purchase order item ID from URL path parameters
	itemID := c.Param("po_item_id")
	// Remove product from purchase order in database
	err := models.RemoveProductFromPurchaseOrder(models.DB, itemID)
	if err != nil {
		// Deletion failed or item not found, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to remove product from purchase order item with ID " + itemID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Clear purchase orders cache to reflect removed line item
	utils.DeleteCacheByPrefix("purchase_orders_")
	utils.DeleteCacheByPrefix("purchase_orders_pagination_")
	// Return successful response confirming removal
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Product removed from purchase order with ID " + itemID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product removed from purchase order successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
