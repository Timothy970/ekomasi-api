package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

func DownloadOrderInvoicePDF(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure the user has permission to view orders
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "")
	if !ok {
		return
	}

	// Extract order ID from path variables
	orderID := c.Param("order_id")

	// Fetch order details
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	params := models.AdminOrderParameters{
		OrderID: orderID,
		Page:    page,
		Limit:   limit,
	}
	orders, _, err := models.ListOrdersByAdmin(models.DB, params)
	if err != nil {
		log.Printf("%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to get order to download invoice PDF",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if len(orders) == 0 {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Order not found for invoice PDF download",
				Code:        http.StatusNotFound,
			},
			Message:   "Order not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Generate PDF invoice
	pdfBytes, err := utils.GenerateInvoicePDF(orders[0])
	if err != nil {
		fmt.Println("Failed to generate invoice PDF error :", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to generate invoice PDF " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to generate invoice PDF",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
