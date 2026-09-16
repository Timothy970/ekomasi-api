package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func DeleteWarehouse(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can delete warehouses)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Warehouse", "warehouse.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract warehouse ID from URL path parameters
	id := c.Param("warehouse_id")
	// Delete warehouse from database (may be soft delete)
	err := models.DeleteWarehouse(id)
	if err != nil {
		// Deletion failed (warehouse not found, has dependencies, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Warehouse",
				Description: "Failed to delete warehouse with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate warehouse caches to ensure fresh data
	utils.DeleteCacheByPrefix("warehouses_")
	utils.DeleteCacheByPrefix("warehouses_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Warehouse",
			Description: warehouseWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Warehouse deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
