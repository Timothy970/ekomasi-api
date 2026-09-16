package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func UpdateInventory(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.update")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateInventoryRequest](c, requestSummary, start)
	if !ok {
		return
	}
	if req.LowStockThreshold == nil && req.Quantity == nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "LowStockThreshold or Quantity must be provided",
				Code:        http.StatusBadRequest,
			},
			Message:   "Request cannot be empty",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Inventory") {
		return
	}
	id := c.Param("inventory_id")

	if err := models.UpdateInventory(models.DB, id, req.Quantity, req.LowStockThreshold); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to update inventory with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("inventories_")
	utils.DeleteCacheByPrefix("inventories_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Inventory updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteInventory deletes an inventory item.
// This endpoint is restricted to administrators.
//
// @Summary      Delete inventory by ID
// @Description  Delete inventory by ID
// @Tags         Inventories
// @Produce      json
// @Param        inventory_id  path      string  true  "Inventory ID"
// @Success      200           {object}  map[string]any
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      409           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/inventories/{inventory_id} [delete]
func DeleteInventory(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.delete")
	if !ok {
		return
	}
	id := c.Param("inventory_id")

	if err := models.DeleteInventory(models.DB, id); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to delete inventory with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("inventories_")
	utils.DeleteCacheByPrefix("inventories_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory with ID " + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Inventory deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetInventoryTurnover retrieves inventory turnover report.
//
// @Summary      Get inventory turnover report
// @Description  Get inventory turnover report
// @Tags         Reports
// @Produce      json
// @Param        period  query     string  false  "Period (weekly, monthly, etc.)"
// @Param        start   query     string  false  "Start Date (YYYY-MM-DD)"
// @Param        end     query     string  false  "End Date (YYYY-MM-DD)"
// @Success      200     {object}  dtos.InventoryTurnoverResponse
// @Failure      404     {object}  dtos.ErrorResponse
// @Router       /api/reports/inventory/turnover [get]
func GetInventoryTurnover(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	groupBy := "weekly"
	periodStr := c.Query("period")
	if periodStr != "" {
		groupBy = periodStr
	}
	startDate, endDate, _ := ParseDateRange(c.Request)
	data, err := models.GetInventoryTurnover(models.DB, startDate, endDate, groupBy)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get inventory turnover report",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.InventoryTurnoverResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
			Type  string    `json:"type"`
		}{Start: startDate, End: endDate, Type: groupBy},
		Data: data,
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Summary inventory turnover report generated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Summary inventory turnover report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetInventoryTurnoverByProduct retrieves inventory turnover report for a specific product.
//
// @Summary      Get inventory turnover report by product
// @Description  Get inventory turnover report by product
// @Tags         Reports
// @Produce      json
// @Param        product_id  path      string  true   "Product ID"
// @Param        period      query     string  false  "Period (weekly, monthly, etc.)"
// @Param        start       query     string  false  "Start Date (YYYY-MM-DD)"
// @Param        end         query     string  false  "End Date (YYYY-MM-DD)"
// @Success      200         {object}  dtos.InventoryTurnoverResponse
// @Failure      404         {object}  dtos.ErrorResponse
// @Router       /api/reports/inventory/turnover/{product_id} [get]
func GetInventoryTurnoverByProduct(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	productID := c.Param("product_id")
	groupBy := "weekly"
	periodStr := c.Query("period")
	if periodStr != "" {
		groupBy = periodStr
	}
	startDate, endDate, _ := ParseDateRange(c.Request)

	data, err := models.GetInventoryTurnoverByProduct(models.DB, productID, startDate, endDate, groupBy)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get inventory turnover report for product " + productID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.InventoryTurnoverResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
			Type  string    `json:"type"`
		}{Start: startDate, End: endDate, Type: groupBy},
		Data: data,
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Product inventory turnover report generated successfully for product " + productID,
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Product inventory turnover report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// StockEntry manages inventory stock entry.
// It handles batch details, inspection details, inventory updates, and handling notes.
// This endpoint is restricted to administrators.
//
// @Summary      Manage inventory stock entry
// @Description  Create a new stock entry with batch and inspection details
// @Tags         Inventory
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id           formData  string  true   "Product ID"
// @Param        batch_number         formData  string  true   "Batch Number"
// @Param        expiry_date          formData  string  true   "Expiry Date (YYYY-MM-DD)"
// @Param        manufacturing_date   formData  string  true   "Manufacturing Date (YYYY-MM-DD)"
// @Param        inspection_date      formData  string  true   "Inspection Date (YYYY-MM-DD)"
// @Param        inspector_id         formData  string  true   "Inspector ID"
// @Param        inspection_notes     formData  string  false  "Inspection Notes"
// @Param        quantity_received    formData  int     true   "Quantity Received"
// @Param        minimum_stock_level  formData  int     true   "Minimum Stock Level"
// @Param        store_quantity       formData  string  true   "Store Quantity JSON"
// @Param        supplier_id          formData  string  true   "Supplier ID"
// @Param        purchase_order_id    formData  string  true   "Purchase Order ID"
// @Param        buying_price         formData  number  true   "Buying Price"
// @Param        condition_id         formData  string  true   "Condition ID"
// @Param        handling_notes       formData  string  false  "Handling Notes"
// @Param        batch_images         formData  file    false  "Batch Images"
// @Param        inspection_images    formData  file    false  "Inspection Images"
// @Success      201                  {object}  map[string]any
// @Failure      400                  {object}  dtos.ErrorResponse
// @Failure      404                  {object}  dtos.ErrorResponse
// @Failure      500                  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/inventory/stock-entry [post]
