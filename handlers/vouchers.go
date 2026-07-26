// Package handlers provides HTTP request handlers for voucher management.
// This file contains handlers for managing discount vouchers, gift cards, and promotional codes
// in the e-commerce platform. Vouchers can be purchased by users or created by admins for marketing.
// Includes functionality for voucher creation, redemption, design management, and automated email delivery.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	voucherNoUser      = "no user found"
	voucherNoUserFound = "Unauthorized Access"
	voucherWithID      = "Voucher with ID "
)

// CreateVoucherDesign creates a new visual template for vouchers.
// Admin-only operation for managing available voucher styles/designs.
// Requires an image upload for the design's background or branding.
//
// @Summary      Create voucher design
// @Description  Create a new voucher visual design with image upload (admin only)
// @Tags         Vouchers
// @Accept       multipart/form-data
// @Produce      json
// @Param        image   formData  file     true  "Design image"
// @Param        name    formData  string   true  "Design name"
// @Param        status  formData  string   true  "Design status (active/inactive)"
// @Success      201     {object}  map[string]interface{} "Design created successfully"
// @Failure      400     {object}  dtos.ErrorResponse   "Invalid request or upload failed"
// @Security     BearerAuth
// @Router       /api/admin/vouchers/design [post]
func CreateVoucherDesign(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify user has admin privileges (only admins can create designs)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "promotions.create"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Parse multipart form for image upload (max 20 MB)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Extract design name and status from form values
	name := c.Request.FormValue("name")
	status := c.Request.FormValue("status")

	// Get image file from form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		// Image is required for new design
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Design image is required",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}
	defer file.Close()

	// Upload design image to Google Cloud Storage
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		// GCS upload failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to upload design image",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Insert voucher design record into database
	_, err = models.CreateVoucherDesign(models.DB, url, name, status)
	if err != nil {
		// Database insertion failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to save design record",
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

	// Respond with success
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher design created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Voucher design created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// CreateNewVoucher creates a new voucher record.
// Admin-only operation for generating promotional or manual vouchers.
//
// @Summary      Create voucher
// @Description  Create a new voucher (admin only)
// @Tags         Vouchers
// @Accept       json
// @Produce      json
// @Param        voucher  body      dtos.VoucherDataCreate  true  "Voucher creation details"
// @Success      201      {object}  map[string]interface{}  "Voucher created successfully"
// @Failure      401      {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/vouchers [post]
func CreateVoucherHandlerTest(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "promotions.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode which authenticated user is creating the voucher
	authuser, _ := middleware.UserFromContext(c.Request.Context())
	// Decode JSON request body with voucher details
	req, ok := DecodeRequestBody[dtos.VoucherDataCreate](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate the request payload
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Vouchers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Start transaction
	tx, err := models.DB.Begin()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to start transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	defer tx.Rollback()

	// Create new voucher record in database
	voucherID, err := models.CreateNewVoucher(tx, *req, authuser.ID)
	if err != nil {
		// Voucher creation failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Map request data for purchase record
	data := dtos.BuyVoucherData{
		ToEmail:      req.ToEmail,
		ToName:       req.ToName,
		Message:      req.Message,
		DeliveryTime: req.DeliveryTime,
	}
	// Record voucher purchase details in database
	err = models.InsertIntoVoucherPurchases(tx, data, voucherID, authuser.ID)
	if err != nil {
		// Purchase recording failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to record voucher purchase",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	if err := tx.Commit(); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to commit transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate voucher caches
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	// Respond with the created voucher ID
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   map[string]interface{}{"voucher_id": voucherID},
		Message:   "Voucher created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListVouchersHandler retrieves a paginated list of all vouchers.
// Admin-only operation with extensive filtering (redeemed status, active status, search).
// Essential for voucher monitoring and customer support.
//
// @Summary      List all vouchers
// @Description  Retrieve paginated list of all vouchers with filters (admin only)
// @Tags         Vouchers
// @Produce      json
// @Param        page         query     int     false  "Page number (default: 1)"
// @Param        size         query     int     false  "Page size (default: 10)"
// @Param        is_redeemed  query     string  false  "Filter by redemption status (true/false)"
// @Param        status       query     string  false  "Filter by status (active/inactive)"
// @Param        q            query     string  false  "Search query (code, customer name)"
// @Success      200          {object}  map[string]interface{} "Vouchers with pagination metadata"
// @Failure      401          {object}  dtos.ErrorResponse     "Admin authorization required"
// @Failure      500          {object}  dtos.ErrorResponse     "Query failed"
// @Security     BearerAuth
// @Router       /api/admin/vouchers [get]
func ListVouchersHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract filter parameters from query string
	isRedeemed := c.Query("is_redeemed")
	status := c.Query("status")
	code := c.Query("q")
	customer := c.Query("customer")

	// Parse pagination parameters
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	// Fetch filtered and paginated vouchers from database
	vouchers, pagination, err := models.ListVouchers(models.DB, page, size, isRedeemed, status, code, customer)
	if err != nil {
		// Database query failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to list vouchers",
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
			Module:      "Vouchers",
			Description: "All vouchers fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"vouchers":   vouchers,
			"pagination": pagination,
		},
		Message:   "Vouchers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetVoucherHandler retrieves detailed information for a specific voucher.
// Admin-only operation for viewing complete voucher details including purchase info.
// Essential for voucher verification and customer support.
//
// @Summary      Get voucher by ID
// @Description  Retrieve detailed information for a specific voucher (admin only)
// @Tags         Vouchers
// @Produce      json
// @Param        voucher_id  path      string               true  "Voucher ID"
// @Success      200         {object}  dtos.Voucher         "Voucher details"
// @Failure      401         {object}  dtos.ErrorResponse   "Admin authorization required"
// @Failure      404         {object}  dtos.ErrorResponse   "Voucher not found"
// @Security     BearerAuth
// @Router       /api/admin/vouchers/{voucher_id} [get]
func GetVoucherHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract voucher ID from URL path parameters
	voucherID := c.Param("voucher_id")

	// Fetch specific voucher details from database
	voucher, err := models.GetVoucherByID(models.DB, voucherID)
	if err != nil {
		// Voucher not found or database error
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher with ID " + voucherID,
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
			Module:      "Vouchers",
			Description: voucherWithID + voucherID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   voucher,
		Message:   "Voucher fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteVoucherHandler removes a voucher from the system.
// Admin-only operation for deleting vouchers (may perform soft delete).
// Essential for managing invalid or expired vouchers.
//
// @Summary      Delete voucher
// @Description  Delete or deactivate a voucher (admin only)
// @Tags         Vouchers
// @Produce      json
// @Param        voucher_id  path      string                  true  "Voucher ID"
// @Success      200         {object}  map[string]interface{}    "Voucher deleted successfully"
// @Failure      401         {object}  dtos.ErrorResponse      "Admin authorization required"
// @Failure      500         {object}  dtos.ErrorResponse      "Delete operation failed"
// @Security     BearerAuth
// @Router       /api/admin/vouchers/{voucher_id} [delete]
func DeleteVoucherHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can delete vouchers)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "promotions.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract voucher ID from URL path parameters
	voucherID := c.Param("voucher_id")

	// Delete voucher from database (may be soft delete)
	err := models.DeleteVoucher(models.DB, voucherID)

	if err != nil {
		// Deletion failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to delete voucher",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate voucher caches to ensure fresh data
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// UpdateVoucherHandler modifies an existing voucher's information.
// Admin-only operation for updating voucher details like amount, status, or expiry.
// Essential for voucher management and corrections.
//
// @Summary      Update voucher
// @Description  Update voucher information (admin only)
// @Tags         Vouchers
// @Accept       json
// @Produce      json
// @Param        voucher_id  path      string                   true  "Voucher ID"
// @Param        voucher     body      dtos.VoucherDataUpdate   true  "Updated voucher details"
// @Success      200         {object}  map[string]interface{}     "Voucher updated successfully"
// @Failure      400         {object}  dtos.ErrorResponse       "Invalid request or update failed"
// @Failure      401         {object}  dtos.ErrorResponse       "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/vouchers/{voucher_id} [patch]
func UpdateVoucherHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can update vouchers)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "promotions.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract voucher ID from URL path parameters
	voucherID := c.Param("voucher_id")
	// Decode and parse JSON request body with updated voucher data
	req, ok := DecodeRequestBody[dtos.VoucherDataUpdate](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Vouchers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	err := models.VoucherUpdate(models.DB, *req, voucherID)

	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher with ID " + voucherID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: voucherWithID + voucherID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetUserVoucherHandler retrieves detailed information for a specific voucher owned by the user.
// User-level operation for viewing their own gift cards or promotional codes.
// Verifies user ownership before returning voucher details.
//
// @Summary      View User Voucher Details
// @Description  Get detailed information for a specific voucher owned by the authenticated user
// @Tags         Users
// @Produce      json
// @Param        voucher_id  path      string               true  "Voucher ID"
// @Success      200         {object}  dtos.Voucher         "Voucher details"
// @Failure      401         {object}  dtos.ErrorResponse   "User authentication required"
// @Failure      404         {object}  dtos.ErrorResponse   "Voucher not found or not owned by user"
// @Security     BearerAuth
// @Router       /api/user/vouchers/{voucher_id} [get]
func GetUserVoucherHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: voucherNoUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	voucherID := c.Param("voucher_id")

	voucher, err := models.GetUserVoucherByID(models.DB, voucherID, authuser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher with ID " + voucherID + " for user with ID " + fmt.Sprint(authuser.ID),
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
			Module:      "Vouchers",
			Description: voucherWithID + voucherID + " fetched successfully for user with ID " + fmt.Sprint(authuser.ID),
			Code:        http.StatusOK,
		},
		Payload:   voucher,
		Message:   "Voucher fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListUserVoucherHandler retrieves all vouchers owned by the authenticated user.
// Returns paginated list of available gift cards and promotional codes for the user.
// Essential for user wallet/profile view of their available discounts.
//
// @Summary      List user Vouchers
// @Description  Retrieve a paginated list of all vouchers owned by the authenticated user
// @Tags         Users
// @Produce      json
// @Param        page  query     int                      false  "Page number (default: 1)"
// @Param        size  query     int                      false  "Page size (default: 10)"
// @Success      200   {object}  map[string]interface{}   "User vouchers list"
// @Failure      401   {object}  dtos.ErrorResponse       "User authentication required"
// @Security     BearerAuth
// @Router       /api/user/vouchers [get]
func ListUserVoucherHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: voucherNoUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	voucher, pagination, err := models.GetUserVouchers(models.DB, authuser.ID, page, limit)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers for user with ID " + fmt.Sprint(authuser.ID),
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
			Module:      "Vouchers",
			Description: "Vouchers fetched successfully for user with ID " + fmt.Sprint(authuser.ID),
			Code:        http.StatusOK,
		},
		Payload:   map[string]interface{}{"vouchers": voucher, "pagination": pagination},
		Message:   "Vouchers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// BuyVoucherHandler allows a user to purchase a new voucher.
// Users can buy vouchers for themselves or as gifts for others (sent via email).
// Triggers payment processing (e.g., M-Pesa) before voucher activation.
//
// @Summary      Purchase a voucher
// @Description  Process a request to buy a new voucher, initiating payment
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        voucher_purchase  body      dtos.BuyVoucherData    true  "Voucher purchase details"
// @Success      201               {object}  map[string]interface{} "Purchase initiated, payment pending"
// @Failure      400               {object}  dtos.ErrorResponse     "Invalid purchase request"
// @Failure      401               {object}  dtos.ErrorResponse     "User authentication required"
// @Security     BearerAuth
// @Router       /api/user/vouchers/buy [post]
func BuyVoucherHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: voucherNoUserFound,
				Code:        http.StatusUnauthorized,
			},
			Message:   voucherNoUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.BuyVoucherData](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Vouchers") {
		return
	}
	// create voucher data
	var voucher dtos.Voucher
	voucher.Amount = req.Amount
	voucher.DesignID = &req.DesignID
	active := "scheduled"
	voucher.Status = &active

	deliveryTime := models.StringToTime(req.DeliveryTime)
	//check delivery time is in the past (but allow today's date)
	today := time.Now().Truncate(24 * time.Hour)
	if deliveryTime.Before(today) {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Delivery time cannot be in the past",
				Code:        http.StatusBadRequest,
			},
			Message:   "Delivery time cannot be in the past",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	expiryEnv := os.Getenv("VOUCHER_EXPIRY_DATE")
	if expiryEnv == "" {
		// Add 90 days from the delivery date
		expiryDate := deliveryTime.AddDate(0, 0, 90)
		voucher.ExpiryDate = expiryDate.Format("2006-01-02 15:04:05")
	} else {
		// Use expiry from environment variable
		expiryTime := models.StringToTime(expiryEnv)
		voucher.ExpiryDate = expiryTime.Format("2006-01-02 15:04:05")
	}

	// Start transaction
	tx, err := models.DB.Begin()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to start transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	defer tx.Rollback()

	voucherID, err := models.AddNewVoucher(tx, voucher, authuser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//insert into voucher purchases
	err = models.InsertIntoVoucherPurchases(tx, *req, authuser.ID, voucherID)
	if err != nil {
		// Purchase recording failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to record voucher purchase",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//create voucher order
	voucherOrderID, err := models.CreateVoucherOrder(tx, req.Amount, voucherID, req.PaymentMethod)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher order",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	err = voucherPaymentProcessor(tx, req.PaymentMethod, voucherOrderID, req.PhoneNumber, req.Amount, voucherID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to process voucher payment",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	if err := tx.Commit(); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to commit transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher created and added successfully",
			Code:        http.StatusCreated,
		},
		Payload: map[string]interface{}{
			"voucher_order_id": voucherOrderID,
		},
		Message:   "Voucher created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func BuyVoucherUpdateHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)

	req, ok := DecodeRequestBody[dtos.BuyVoucherData](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Vouchers") {
		return
	}
	voucherID := c.Param("voucher_id")
	// create voucher data
	amount := req.Amount
	status := "scheduled"
	// Start transaction
	tx, err := models.DB.Begin()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to start transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	defer tx.Rollback()

	err = models.ValidateDesignID(tx, req.DesignID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Invalid Design ID",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if req.Amount > 0 {
		err := models.UpdateVoucher(tx, voucherID, amount, status, req.DesignID)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: "Failed to update voucher",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}
	//insert into voucher purchases
	err = models.UpdateVoucherPurchases(tx, *req, voucherID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher purchase",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//create voucher order
	voucherOrderID, err := models.UpdateVoucherPurchaseAmount(tx, req.Amount, voucherID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher purchase amount",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if req.Amount > 0 || req.PaymentMethod != "none" {
		err = voucherPaymentProcessor(tx, req.PaymentMethod, voucherOrderID, req.PhoneNumber, req.Amount, voucherID)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: "Failed to process voucher payment",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to commit transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher updated and added successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"voucher_order_id": voucherOrderID,
		},
		Message:   "Voucher updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
func voucherPaymentProcessor(db models.DBExecutor, paymentMethod string, voucherOrderID, phoneNumber string, amount float64, voucherID string) error {
	switch paymentMethod {
	//where method is mpesa or empty use mpesa
	case "mpesa", "":
		// Initiate Mpesa payment
		err := HandleMpesaVoucherPayment(db, voucherOrderID, phoneNumber, amount)
		if err != nil {
			return err
		}
	case "cash":
		// For cash payments, we can directly mark the order as paid and activate the voucher without external processing to active
		err := models.MarkVoucherAsPaid(db, voucherID)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported payment method: %s", paymentMethod)
	}
	return nil
}

// RedeemVoucherHandler allows a user to redeem a voucher to their account.
// User must be authenticated and provide a valid voucher code.
// Transfers voucher ownership to the user for future use on purchases.
//
// @Summary      Redeem voucher
// @Description  Redeem a voucher code to user's account
// @Tags         Vouchers
// @Accept       json
// @Produce      json
// @Param        voucher  body      dtos.RedeemVoucherRequest  true  "Voucher redemption details"
// @Success      200      {object}  map[string]interface{}       "Voucher redeemed successfully"
// @Failure      400      {object}  dtos.ErrorResponse         "Invalid code or redemption failed"
// @Failure      401      {object}  dtos.ErrorResponse         "User authentication required"
// @Security     BearerAuth
// @Router       /api/vouchers/redeem [post]
func RedeemVoucherHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Get authenticated user from context
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		// User not authenticated
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "User not validated or unauthorized",
				Code:        http.StatusInternalServerError,
			},
			Message:   voucherNoUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Decode and parse JSON request body with voucher code
	req, ok := DecodeRequestBody[dtos.RedeemVoucherRequest](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate voucher code format
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Vouchers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Redeem voucher code to user's account
	_, err := models.RedeemVoucher(models.DB, req.Code, authuser.ID)
	if err != nil {
		// Redemption failed (invalid code, already redeemed, expired, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to redeem voucher",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate voucher caches to reflect redemption status
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher redeemed successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher redeemed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// StartVoucherEmailScheduler initializes a background scheduler for sending voucher emails.
// Runs at specified intervals to process scheduled voucher deliveries.
// The scheduler can run indefinitely or for a specified number of iterations.
func StartVoucherEmailScheduler(interval time.Duration, repeat int) {
	// Create ticker for periodic execution
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		// Execute on each tick
		for range ticker.C {
			// Process and send scheduled voucher emails
			SendBoughtForVoucherEmails()

			// Decrement repeat counter if limited execution
			if repeat > 0 {
				repeat--
				if repeat == 0 {
					// Stop scheduler after specified iterations
					return
				}
			}
		}
	}()
}

// SendBoughtForVoucherEmails processes and sends scheduled voucher delivery emails.
// Fetches vouchers with unsent emails and sends them to recipients.
// Updates email sent status after successful delivery.
func SendBoughtForVoucherEmails() {
	// Fetch all users with vouchers pending email delivery
	users, err := models.GetUsersWithUnsentVoucherEmails(models.DB)
	if err != nil {
		fmt.Printf("error fetching users with unsent voucher emails: %v", err)
		return
	}
	// Process each voucher email
	for _, u := range users {
		// Send email if recipient email is provided
		if u.ToEmail != "" {
			// Generate email subject and HTML body
			subject, htmlBody := utils.SendVoucherEmail(u)
			// Send email notification
			notification.SendEmail(u.ToEmail, subject, htmlBody)

			// Mark voucher email as sent in database
			err := models.MarkVoucherEmailAsSent(models.DB, u.VoucherID)
			if err != nil {
				log.Printf("error marking voucher email as sent for user: %v", err)
			}
		}
	}
}

func EditVoucherDesign(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "promotions.update"); !ok {
		return
	}
	designID := c.Param("voucher_id")

	// Parse multipart form (20 MB max)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Get image file (optional)
	var url string
	file, header, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()
		// Upload image to GCS
		url, err = utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return
		}
	}
	// Insert category into DB
	err = models.EditVoucherDesign(models.DB, designID, &url, c.Request.FormValue("name"), c.Request.FormValue("status"))
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher design",
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
			Module:      "Vouchers",
			Description: "Voucher design updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher design updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func DeleteVoucherDesign(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "promotions.delete"); !ok {
		return
	}
	designID := c.Param("voucher_id")

	err := models.DeleteVoucherDesign(models.DB, designID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to delete voucher design",
				Code:        http.StatusBadRequest,
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
			Module:      "Vouchers",
			Description: "Voucher design deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher design deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func GetAllVoucherDesigns(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	status := c.Query("status")
	q := c.Query("name")

	designs, pagination, err := models.GetAllVoucherDesigns(models.DB, page, limit, q, status)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher designs",
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
			Module:      "Vouchers",
			Description: "Voucher designs fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]interface{}{"designs": designs, "pagination": pagination},
		Message:   "Voucher designs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func GetVoucherDesignByID(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	designID := c.Param("voucher_id")

	design, err := models.GetVoucherDesign(models.DB, designID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher design",
				Code:        http.StatusNotFound,
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
			Module:      "Vouchers",
			Description: "Voucher design fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   design,
		Message:   "Voucher design fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func ListVoucherPurchasesHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", ""); !ok {
		return
	}

	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	q := c.Query("name")

	purchases, pagination, err := models.ListVoucherPurchases(models.DB, page, limit, q)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher purchases",
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
			Module:      "Vouchers",
			Description: "Voucher purchases fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]interface{}{"purchases": purchases, "pagination": pagination},
		Message:   "Voucher purchases fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func GetVoucherPurchasesHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", ""); !ok {
		return
	}

	purchaseID := c.Param("voucher_id")

	purchase, err := models.GetVoucherPurchases(models.DB, purchaseID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher purchase",
				Code:        http.StatusNotFound,
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
			Module:      "Vouchers",
			Description: "Voucher purchase fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   purchase,
		Message:   "Voucher purchase fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
