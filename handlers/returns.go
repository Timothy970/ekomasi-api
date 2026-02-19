// Package handlers provides HTTP request handlers for order returns and refund management.
// This file contains handlers for processing customer return requests, managing return status
// workflows (pending, approved, rejected, completed), and tracking return items. The return
// system helps handle product returns, quality issues, and customer satisfaction.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
func CreateReturnsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract authenticated user from request context
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated, return unauthorized error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: notAuthenticated,
				Code:        http.StatusUnauthorized,
			},
			Message:   noUserFound,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.ReturnRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Validate return request against business rules (return window, order status, etc.)
	if err := models.ValidateReturnRequest(models.DB, *req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Invalid return request " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Create return request in database with "pending" status
	// Links return to user ID for ownership verification
	err := models.CreateReturns(models.DB, *req, authuser.ID)
	if err != nil {
		// Database operation failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to register return " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update order status " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return success response - return request created and awaiting admin approval
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return registered successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Return registered successfully, pending approval",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func UpdateReturnStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can update return status)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Orders", "orders.update"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract return ID from URL path parameters
	returnID := mux.Vars(r)["return_id"]
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.ReturnStatusUpdate](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Orders") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Update return status in database (may trigger inventory/refund actions)
	err := models.UpdateReturnStatus(models.DB, returnID, *req)
	if err != nil {
		// Status update failed (invalid status transition or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to update return status " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return success response - status updated (customer may be notified)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return status updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Return status updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

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
func GetReturnByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can view all returns)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Orders", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract return ID from URL path parameters
	returnID := mux.Vars(r)["return_id"]
	// Fetch complete return details from database
	returnRequest, err := models.GetReturnByID(models.DB, returnID)
	if err != nil {
		// Return not found or database error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch return details " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return success response with complete return details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   returnRequest,
		Message:   "Return fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func DeleteReturnHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can delete returns)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Orders", "orders.delete"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract return ID from URL path parameters
	returnID := mux.Vars(r)["return_id"]
	// Permanently delete return from database
	err := models.DeleteReturn(models.DB, returnID)
	if err != nil {
		// Deletion failed (return not found or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to delete return " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return success response - return permanently removed from system
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Return deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func ListAllReturnsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can view all returns)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Orders", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract optional filter parameters from query string
	status := r.URL.Query().Get("status")
	q := r.URL.Query().Get("q")
	// Parse pagination parameters
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Fetch paginated returns list from database with filters
	returns, pagination, err := models.GetAllReturns(models.DB, page, size, status, q)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch returns " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return success response with paginated returns list and metadata
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
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
func ListOwnerReturnsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract authenticated user from request context
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated, return unauthorized error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context or not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Extract optional filter parameters from query string
	status := r.URL.Query().Get("status")
	q := r.URL.Query().Get("q")
	// Fetch all returns for this user from database with filters
	returns, err := models.GetAllOwnerReturns(models.DB, status, q, authuser.ID)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch returns " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return success response with user's returns list
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Returns fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   returns,
		Message:   "Returns fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// GetOwnerReturnsHandler retrieves details of a specific return request for the authenticated user.
// Allows customers to view their own return details. Access is restricted to the return owner
// for security - users cannot view other users' returns.
//
// @Summary      Get user's return request details
// @Description  Retrieve detailed information about a specific return request owned by the user
// @Tags         Returns
// @Produce      json
// @Param        return_id  path      string                true  "Return ID"
// @Success      200        {object}  dtos.ReturnResponse   "Return details"
// @Failure      400        {object}  dtos.ErrorResponse    "Return not found or not owned by user"
// @Failure      401        {object}  dtos.ErrorResponse    "User not authenticated"
// @Security     BearerAuth
// @Router       /api/my-returns/{return_id} [get]
func GetOwnerReturnsHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract authenticated user from request context
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated, return unauthorized error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context or not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Extract return ID from URL path parameters
	returnID := mux.Vars(r)["return_id"]
	// Fetch return details from database (verifies user ownership)
	ret, err := models.GetOwnerReturnByID(models.DB, returnID, authuser.ID)
	if err != nil {
		// Return not found or not owned by user
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch return details " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return success response with return details for user's own return
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return details fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   ret,
		Message:   "Return details fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
