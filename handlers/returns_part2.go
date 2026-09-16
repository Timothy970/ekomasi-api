// Package handlers provides HTTP request handlers for order returns and refund management.
// This file contains handlers for processing customer return requests, managing return status
// workflows (pending, approved, rejected, completed), and tracking return items. The return
// system helps handle product returns, quality issues, and customer satisfaction.
package handlers

import (
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetReturnByIDHandler retrieves detailed information about a specific return request.
// Provides complete return details including items, status history, and customer information.
// Admin-only access for managing returns across all customers.
//
// @Summary      Get return request details
// @Description  Retrieve detailed information about a specific return request (admin only)
// @Tags         Returns
// @Produce      json
// @Param        return_id  path      string                true  "Return ID"
// @Success      200        {object}  dtos.ReturnResponse   "Return details"
// @Failure      400        {object}  dtos.ErrorResponse    "Return not found or fetch failed"
// @Failure      401        {object}  dtos.ErrorResponse    "User not authorized (admin required)"
// @Security     BearerAuth
// @Router       /api/returns/{return_id} [get]
func GetReturnByIDHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can view all returns)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract return ID from URL path parameters
	returnID := c.Param("return_id")
	// Fetch complete return details from database
	returnRequest, err := models.GetReturnByID(models.DB, returnID)
	if err != nil {
		// Return not found or database error
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch return details " + err.Error(),
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
	// Return success response with complete return details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   returnRequest,
		Message:   "Return fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// DeleteReturnHandler permanently removes a return request from the system.
// This should be used cautiously, typically only for cancelled or invalid returns.
// Admin-only operation to maintain data integrity.
//
// @Summary      Delete return request
// @Description  Permanently delete a return request from the system (admin only)
// @Tags         Returns
// @Produce      json
// @Param        return_id  path      string                true  "Return ID"
// @Success      200        {object}  map[string]interface{}  "Return deleted successfully"
// @Failure      400        {object}  dtos.ErrorResponse    "Return not found or deletion failed"
// @Failure      401        {object}  dtos.ErrorResponse    "User not authorized (admin required)"
// @Security     BearerAuth
// @Router       /api/returns/{return_id} [delete]
func DeleteReturnHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can delete returns)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", "orders.delete"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract return ID from URL path parameters
	returnID := c.Param("return_id")
	// Permanently delete return from database
	err := models.DeleteReturn(models.DB, returnID)
	if err != nil {
		// Deletion failed (return not found or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to delete return " + err.Error(),
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
	// Return success response - return permanently removed from system
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Return deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// ListAllReturnsHandler retrieves a paginated list of all return requests across all customers.
// Supports filtering by status (pending, approved, rejected, completed) and search query.
// Admin-only access for managing returns system-wide.
//
// @Summary      List all return requests
// @Description  Retrieve a paginated list of all return requests with optional filters (admin only)
// @Tags         Returns
// @Produce      json
// @Param        status  query     string  false  "Filter by status (pending, approved, rejected, completed)"
// @Param        q       query     string  false  "Search query for order ID or customer name"
// @Param        page    query     int     false  "Page number (default: 1)"
// @Param        size    query     int     false  "Page size (default: 10)"
// @Success      200     {object}  map[string]interface{}  "Returns list with pagination"
// @Failure      400     {object}  dtos.ErrorResponse      "Fetch failed"
// @Failure      401     {object}  dtos.ErrorResponse      "User not authorized (admin required)"
// @Security     BearerAuth
// @Router       /api/returns [get]
func ListAllReturnsHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can view all returns)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Orders", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract optional filter parameters from query string
	status := c.Query("status")
	q := c.Query("q")
	// Parse pagination parameters
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	// Fetch paginated returns list from database with filters
	returns, pagination, err := models.GetAllReturns(models.DB, page, size, status, q)
	if err != nil {
		// Database query failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch returns " + err.Error(),
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
	// Return success response with paginated returns list and metadata
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "All returns fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"returns":    returns,
			"pagination": pagination,
		},
		Message:   "All returns fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// ListOwnerReturnsHandler retrieves all return requests for the authenticated user.
// Allows customers to view their own return history with optional status and search filters.
// Users can only see their own returns for security and privacy.
//
// @Summary      List user's return requests
// @Description  Retrieve all return requests for the authenticated user
// @Tags         Returns
// @Produce      json
// @Param        status  query     string                   false  "Filter by status (pending, approved, rejected, completed)"
// @Param        q       query     string                   false  "Search query for order ID or product name"
// @Success      200     {array}   dtos.ReturnResponse      "User's returns list"
// @Failure      400     {object}  dtos.ErrorResponse       "Fetch failed"
// @Failure      401     {object}  dtos.ErrorResponse       "User not authenticated"
// @Security     BearerAuth
// @Router       /api/my-returns [get]
func ListOwnerReturnsHandler(c *gin.Context) {
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
				Description: "User not found in context or not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Extract optional filter parameters from query string
	status := c.Query("status")
	q := c.Query("q")
	// Fetch all returns for this user from database with filters
	returns, err := models.GetAllOwnerReturns(models.DB, status, q, authuser.ID)
	if err != nil {
		// Database query failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch returns " + err.Error(),
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
	// Return success response with user's returns list
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Returns fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   returns,
		Message:   "Returns fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
