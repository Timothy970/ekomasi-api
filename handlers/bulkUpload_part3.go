package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DeleteBulkUploadProductsHandler handles the deletion of bulk uploaded products via the bulk product ID.
// This endpoint is restricted to authenticated users.
//
// @Summary      Delete bulk uploaded products
// @Description  Delete multiple products using a bulk product ID
// @Tags         Products
// @Produce      json
// @Param        bulk_product_id  query  string  true  "Bulk Product ID"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/bulk-upload/{bulk_product_id} [delete]
func DeleteBulkUploadProductsHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Get bulk product ID from URL path
	bulkProductID := c.Param("bulk_product_id")
	err := models.DeleteBulkProductByID(models.DB, bulkProductID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error deleting bulk product: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product deleted successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"message": "Product deleted successfully",
			"count":   nil,
		},
		Message:   "Product deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})

}
