// Package handlers provides HTTP request handlers for order returns and refund management.
// This file contains handlers for processing customer return requests, managing return status
// workflows (pending, approved, rejected, completed), and tracking return items. The return
// system helps handle product returns, quality issues, and customer satisfaction.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateReturnsHandler allows authenticated customers to create a return request for order items.
// Customers can initiate returns for various reasons (defective, wrong item, not as described, etc.).
// The return must be validated against return policies and order eligibility before creation.
//
// @Summary      Create a return request
// @Description  Submit a return request for one or more items from an order
// @Tags         Returns
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.ReturnRequest       true  "Return request details"
// @Success      201      {object}  map[string]interface{}     "Return request created successfully"
// @Failure      400      {object}  dtos.ErrorResponse       "Invalid request or validation failed"
// @Failure      401      {object}  dtos.ErrorResponse       "User not authenticated"
// @Security     BearerAuth
// @Router       /api/returns [post]
func CreateReturnsHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract authenticated user from request context
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		// User not authenticated, return unauthorized error
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: notAuthenticated,
				Code:        http.StatusUnauthorized,
			},
			Message:   noUserFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.ReturnRequest](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Validate return request against business rules (return window, order status, etc.)
	if err := models.ValidateReturnRequest(models.DB, *req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Invalid return request " + err.Error(),
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

	// Create return request in database with "pending" status
	// Links return to user ID for ownership verification
	err := models.CreateReturns(models.DB, *req, authuser.ID)
	if err != nil {
		// Database operation failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to register return " + err.Error(),
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
	//update the order  status to returned
	returnStatus := "RETURNED"
	deliveryStatus := "RETURNED"
	updateRequest := dtos.UpdateOrderStatusRequest{
		Status:         &returnStatus,
		DeliveryStatus: &deliveryStatus,
	}
	err = models.UpdateOrderStatus(models.DB, req.OrderID, updateRequest)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update order status " + err.Error(),
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
	// Return success response - return request created and awaiting admin approval
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return registered successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Return registered successfully, pending approval",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// UpdateReturnStatusHandler allows admins to update the status of a return request.
// Status transitions: pending → approved/rejected, approved → completed/refunded.
// Status changes may trigger inventory adjustments, refund processing, and customer notifications.
//
// @Summary      Update return request status
// @Description  Update the status of a return request (approve, reject, complete)
// @Tags         Returns
// @Accept       json
// @Produce      json
// @Param        return_id  path      string                    true  "Return ID"
// @Param        request    body      dtos.ReturnStatusUpdate   true  "Status update details"
// @Success      200        {object}  map[string]interface{}      "Status updated successfully"
// @Failure      400        {object}  dtos.ErrorResponse        "Invalid request or update failed"
// @Failure      401        {object}  dtos.ErrorResponse        "User not authorized (admin required)"
// @Security     BearerAuth
// @Router       /api/returns/{return_id}/status [patch]
func UpdateReturnStatusHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can update return status)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.update"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract return ID from URL path parameters
	returnID := c.Param("return_id")
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.ReturnStatusUpdate](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	if strings.ToLower(req.Status) == "approved" {
		if req.PhoneNumber == nil || *req.PhoneNumber == "" {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Orders",
					Description: "Phone number is required for approved returns",
					Code:        http.StatusBadRequest,
				},
				Message:   "Phone number is required for approved returns",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return
		}

	}
	// Update return status in database (may trigger inventory/refund actions)
	err := models.UpdateReturnStatus(models.DB, returnID, *req)
	if err != nil {
		// Status update failed (invalid status transition or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update return status " + err.Error(),
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
	//handle retrun approval - update inventory and process refund if needed
	err = handleReturnRefunding(models.DB, returnID, req.PhoneNumber)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to handle return refunding " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	// Return success response - status updated (customer may be notified)
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return status updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Return status updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// helper function to handle b2c refunding after items retuning have been approved
func handleReturnRefunding(db models.DBExecutor, returnID string, phoneNumber *string) error {
	//fetch return details to get order and customer info
	ret, err := models.GetOrderItemsForReturn(db, returnID)
	if err != nil {
		return err
	}
	client, err := NewMpesaClient()
	if err != nil {
		return err
	}
	order, err := models.GetOrderByID(db, ret.OrderID)
	if err != nil {
		return err
	}

	// Filter order.Items to include only products in ret.Products
	returnedProductIDs := make(map[string]bool)
	for _, product := range ret.Products {
		returnedProductIDs[product.ID] = true
	}

	var filteredItems []dtos.OrderProduct
	for _, item := range order.Items {
		if returnedProductIDs[item.ID] {
			filteredItems = append(filteredItems, item)
		}
	}
	order.Items = filteredItems

	result, err := client.HandleMoneyReturn(ret.TotalRefund, *phoneNumber)
	if err != nil {
		return err
	}
	if result.ResultCode != "0" {
		return fmt.Errorf("mpesa refund failed")
	}

	//mark the order as refunded and restock items
	err = models.HandleMpesaMoneyReturnRefunds(db, *order)
	if err != nil {
		return err
	}

	return nil
}
