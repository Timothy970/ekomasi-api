package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetOrderTrackingTimelineHandler retrieves the step-by-step visual timeline status for an order
// @Summary      Visual Order Tracking Timeline
// @Description  Get 5-step visual tracking timeline ('placed', 'processing', 'shipped', 'out_for_delivery', 'delivered')
// @Tags         Orders
// @Produce      json
// @Param        order_id  path      string  true  "Order ID"
// @Param        token     query     string  false "Guest Tracking Token"
// @Success      200       {object}  dtos.OrderTrackingTimelineResponse
// @Router       /api/orders/{order_id}/tracking [get]
func GetOrderTrackingTimelineHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	orderID := c.Param("order_id")
	token := c.Query("token")

	if orderID == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Order ID is required",
				Code:        http.StatusBadRequest,
			},
			Message:   "Order ID is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	timeline, err := models.GetOrderTrackingTimeline(models.DB, orderID, token)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch order tracking: " + err.Error(),
				Code:        http.StatusNotFound,
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
			Module:      "Orders",
			Description: "Order visual tracking timeline retrieved",
			Code:        http.StatusOK,
		},
		Payload:   timeline,
		Message:   "Order tracking details retrieved",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
