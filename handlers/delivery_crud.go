package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func DeleteDeliveryHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.delete")
	if !ok {
		return
	}
	deliveryID := c.Param("delivery_id")
	err := models.DeleteDelivery(deliveryID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to delete delivery with ID " + deliveryID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("deliveries_")
	utils.DeleteCacheByPrefix("deliveries_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: deliveryWithID + deliveryID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Delivery deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})

}
