// Package handlers provides HTTP request handlers for the Ekomasi backend API.
// This file contains payment processing, refund management, voucher handling,
// and payment option configuration handlers.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary List all refund requests with pagination
// @Description Retrieves a paginated list of all refund requests in the system
// @Tags Payments
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param size query int false "Page size (default: 10)"
// @Success 200 {object} map[string]interface{} "Refunds retrieved successfully with pagination"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/refunds [get]
// @Security BearerAuth
func ListRefundsHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse pagination parameters from query string
	page, size := parsePagination(c.Query("page"), c.Query("size"))

	// Generate cache keys for refunds data and pagination metadata
	cacheKeyRefunds := fmt.Sprintf("refunds_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("refunds_pagination_%d_size_%d", page, size)

	// Initialize variables for refunds data and cached versions
	var refunds []dtos.Refund
	var cachedRefunds []dtos.Refund
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta

	// Attempt to retrieve cached data
	_ = utils.GetCache(cacheKeyRefunds, &cachedRefunds)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)

	// If cache miss, fetch from database
	if cachedRefunds == nil {
		var err error
		// Fetch refunds from database with pagination
		refunds, pagination, err = models.ListRefunds(models.DB, page, size)
		if err != nil {
			// Return error response if database query fails
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: "Failed to list refunds",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		// Cache the fetched data for future requests
		_ = utils.SetCache(cacheKeyRefunds, cachedRefunds)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		// Use cached data if available
		refunds = cachedRefunds
		pagination = cachedPagination
	}

	// Build response with refunds and pagination metadata
	response := dtos.PaginatedRefundsResponse{
		Refunds: refunds,
		Meta:    *pagination,
	}

	// Return success response with refund data
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refunds fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Refunds fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetRefundByIDHandler retrieves a single refund request by its unique identifier.
// This endpoint returns detailed information about a specific refund request.
//
// @Summary Get refund by ID
// @Description Retrieves detailed information about a specific refund request by its ID
// @Tags Payments
// @Produce json
// @Param refund_id path string true "Refund ID"
// @Success 200 {object} map[string]interface{} "Refund details retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Refund not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/refunds/{refund_id} [get]
// @Security BearerAuth
func GetRefundByIDHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract refund ID from URL path parameters and convert to integer
	idStr := c.Param("refund_id")
	id, _ := strconv.Atoi(idStr)

	// Retrieve refund record from database by ID
	refund, err := models.GetRefundByID(models.DB, id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get refund by ID " + idStr,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund with ID " + idStr + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   refund,
		Message:   "Refund fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetRefundByUserIDHandler retrieves refund requests associated with a specific user.
// This endpoint returns refund information filtered by user ID.
//
// @Summary Get refunds by user ID
// @Description Retrieves refund requests for a specific user identified by user ID
// @Tags Payments
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]interface{} "Refund details retrieved successfully"
// @Failure 404 {object} map[string]interface{} "User or refund not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/refunds/user/{user_id} [get]
// @Security BearerAuth
func GetRefundByUserIDHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract user ID from URL path parameters and convert to integer
	idStr := c.Param("user_id")
	id, _ := strconv.Atoi(idStr)

	// Retrieve refund records from database by user ID
	refund, err := models.GetRefundByID(models.DB, id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get refund by User ID " + idStr,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund with User ID " + idStr + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   refund,
		Message:   "Refund fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// CreateVoucherHandler creates a new voucher for promotional or discount purposes.
// This endpoint is restricted to admin users and allows creation of vouchers that can be
// applied during checkout to provide discounts or special offers.
//
// @Summary Create a new voucher
// @Description Creates a new promotional voucher with discount details and usage rules
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dtos.Voucher true "Voucher creation request"
// @Success 201 {object} map[string]interface{} "Voucher created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/vouchers [post]
// @Security BearerAuth
func CreateVoucherHandler(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "promotions.create")
	if !ok {
		return
	}

	// Retrieve authenticated admin user from context
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		// Return error if user is not found in context
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "User not found in context while creating voucher",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Decode and validate the request body
	req, ok := DecodeRequestBody[dtos.Voucher](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Vouchers") {
		return
	}

	// Create voucher in database associated with admin user
	_, err := models.AddNewVoucher(models.DB, *req, authuser.ID)
	if err != nil {
		// Return error response if voucher creation fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached voucher data to ensure data consistency
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")

	// Return success response confirming voucher creation
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Voucher created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
