package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
		Payload: map[string]any{
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
// @Success      200         {object}  map[string]any    "Voucher deleted successfully"
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
// @Success      200         {object}  map[string]any     "Voucher updated successfully"
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
