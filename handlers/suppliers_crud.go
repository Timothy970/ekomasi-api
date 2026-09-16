package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func DeleteSupplier(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can delete suppliers)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Suppliers", "suppliers.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Extract supplier ID from URL path parameters
	id := c.Param("supplier_id")

	// Delete supplier from database (may be soft delete)
	if err := models.DeleteSupplier(models.DB, id); err != nil {
		// Supplier not found or database error
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Suppliers",
				Description: "Failed to delete supplier with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate supplier caches to ensure fresh data after deletion
	utils.DeleteCacheByPrefix("suppliers_")
	utils.DeleteCacheByPrefix("suppliers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Suppliers",
			Description: supplierWithID + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Supplier deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
