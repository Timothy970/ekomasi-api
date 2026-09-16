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

func DownloadInventoryPDF(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.view"); !ok {
		return
	}
	id := c.Param("inventory_id")

	inv, err := models.GetInventory(models.DB, id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	pdfBytes, err := utils.GenerateInventoryPDF(*inv)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to generate inventory PDF",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return

	}

	filename := fmt.Sprintf("inventory_%s.pdf", inv.InventoryID)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Writer.Write(pdfBytes)
}

// UpdateInventory updates an existing inventory item.
// This endpoint is restricted to administrators.
//
// @Summary      Update inventory by ID
// @Description  Update inventory by ID
// @Tags         Inventories
// @Accept       json
// @Produce      json
// @Param        inventory_id  path      string                      true  "Inventory ID"
// @Param        inventory     body      dtos.UpdateInventoryRequest true  "Inventory Details"
// @Success      200           {object}  map[string]interface{}
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      409           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/inventories/{inventory_id} [patch]
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
// @Success      200           {object}  map[string]interface{}
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
