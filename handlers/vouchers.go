// Package handlers provides HTTP request handlers for voucher/gift card management.
// This file contains handlers for creating, purchasing, redeeming, and managing vouchers
// in the e-commerce platform. Vouchers allow customers to purchase gift cards with custom
// designs, messages, and scheduled delivery. Essential for gift-giving and promotional campaigns.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/utils"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// voucherWithID is a constant prefix for voucher-related log messages
var voucherWithID = "Voucher with ID "

// CreateVoucherDesign creates a new voucher design template.
// Admin-only operation for adding visual designs that customers can choose when purchasing vouchers.
// Uploads design image to GCS and stores design metadata in database.
//
// @Summary      Create voucher design
// @Description  Create a new voucher design template with image upload (admin only)
// @Tags         Voucher Designs
// @Accept       multipart/form-data
// @Produce      json
// @Param        image   formData  file                    true  "Voucher design image"
// @Param        name    formData  string                  true  "Design name"
// @Param        status  formData  string                  true  "Design status (active/inactive)"
// @Success      201     {object}  dtos.SuccessResponse    "Design created successfully"
// @Failure      400     {object}  dtos.ErrorResponse      "Invalid form data or image required"
// @Failure      401     {object}  dtos.ErrorResponse      "Admin authorization required"
// @Failure      500     {object}  dtos.ErrorResponse      "Image upload or database error"
// @Security     BearerAuth
// @Router       /api/admin/voucher-designs [post]
func CreateVoucherDesign(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify user has admin privileges (only admins can create voucher designs)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Parse multipart form data (20 MB maximum file size)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		// Form parsing failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to parse form data when creating voucher design: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Extract image file from form data
	file, header, err := r.FormFile("image")
	if err != nil {
		// Image file is required
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to get image file from form data when creating voucher design: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   "Image is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	defer file.Close()

	// Upload design image to Google Cloud Storage
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		// Image upload to GCS failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to upload image to GCS when creating voucher design: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	// Extract design name and status from form values
	name := r.FormValue("name")
	status := r.FormValue("status")
	// Construct design DTO for validation
	req := dtos.VoucherDesign{
		URL:    url,
		Name:   &name,
		Status: &status,
	}
	// Validate required fields (name, status)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Insert voucher design record into database
	_, err = models.CreateVoucherDesign(url, name, status)
	if err != nil {
		// Database insertion failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher design in DB: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Return success response with image URL
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher design added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   url,
		Message:   "Voucher design added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// CreateVoucherHandlerTest creates a new voucher with purchase record.
// Admin-only operation for creating vouchers directly (test/admin mode).
// Automatically sets expiry dates based on IsToExpire flag and records purchase details.
//
// @Summary      Create voucher (admin)
// @Description  Create voucher with purchase record and automatic expiry calculation (admin only)
// @Tags         Vouchers
// @Accept       json
// @Produce      json
// @Param        voucher  body      dtos.VoucherDataCreate  true  "Voucher creation details"
// @Success      201      {object}  dtos.SuccessResponse    "Voucher created successfully"
// @Failure      400      {object}  dtos.ErrorResponse      "Invalid request or creation failed"
// @Failure      401      {object}  dtos.ErrorResponse      "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/vouchers [post]
func CreateVoucherHandlerTest(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can create vouchers)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Get authenticated user from context for creator tracking
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "User not validated/unauthorized",
				Code:        http.StatusUnauthorized,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Decode and parse JSON request body with voucher details
	req, ok := DecodeRequestBody[dtos.VoucherDataCreate](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Calculate expiry date based on IsToExpire flag
	if req.IsToExpire {
		// Set expiry date to 90 days from now for limited-time vouchers
		req.ExpiryDate = time.Now().Add(90 * 24 * time.Hour).Format("2006-01-02")
	} else {
		// Set expiry date to 30 years from now for permanent vouchers
		req.ExpiryDate = time.Now().Add(30 * 365 * 24 * time.Hour).Format("2006-01-02")
	}
	// Validate all required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Create new voucher record in database
	voucherID, err := models.CreateNewVoucher(*req, authuser.ID)
	if err != nil {
		// Voucher creation failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Prepare purchase data for recording
	data := dtos.BuyVoucherData{
		Amount:       req.Amount,
		FromName:     req.FromName,
		ToName:       req.ToName,
		ToEmail:      req.ToEmail,
		Message:      req.Message,
		DeliveryTime: req.DeliveryTime,
	}
	// Record voucher purchase details in database
	err = models.InsertIntoVoucherPurchases(data, authuser.ID, voucherID)
	if err != nil {
		// Purchase recording failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to record voucher purchase",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate voucher caches to ensure fresh data
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   voucherID,
		Message:   "Voucher created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ListVouchersHandler retrieves a paginated list of all vouchers with filtering.
// Admin-only operation with multiple filter options: redemption status, voucher status, code, and customer.
// Essential for voucher management and tracking.
//
// @Summary      List vouchers
// @Description  Retrieve paginated list of vouchers with multiple filters (admin only)
// @Tags         Vouchers
// @Produce      json
// @Param        page         query     int                       false  "Page number (default: 1)"
// @Param        size         query     int                       false  "Page size (default: 10)"
// @Param        is_redeemed  query     string                    false  "Filter by redemption status (true/false)"
// @Param        status       query     string                    false  "Filter by voucher status"
// @Param        code         query     string                    false  "Filter by voucher code"
// @Param        customer     query     string                    false  "Filter by customer name"
// @Success      200          {object}  map[string]interface{}    "Vouchers with pagination metadata"
// @Failure      401          {object}  dtos.ErrorResponse        "Admin authorization required"
// @Failure      500          {object}  dtos.ErrorResponse        "Failed to fetch vouchers"
// @Security     BearerAuth
// @Router       /api/admin/vouchers [get]
func ListVouchersHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can list all vouchers)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract filter parameters from query string
	isRedeemed := r.URL.Query().Get("is_redeemed")
	status := r.URL.Query().Get("status")
	r.URL.Query().Get("page")
	code := r.URL.Query().Get("code")
	customer := r.URL.Query().Get("customer")
	// Parse pagination parameters
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Fetch filtered and paginated vouchers from database
	vouchers, pagination, err := models.ListVouchers(page, size, isRedeemed, status, code, customer)
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers list: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
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
func GetVoucherHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can view voucher details)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract voucher ID from URL path parameters
	voucherID := mux.Vars(r)["voucher_id"]

	// Fetch specific voucher details from database
	voucher, err := models.GetVoucherByID(voucherID)
	if err != nil {
		// Voucher not found or database error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher with ID " + voucherID,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: voucherWithID + voucherID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   voucher,
		Message:   "Voucher fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200         {object}  dtos.SuccessResponse    "Voucher deleted successfully"
// @Failure      401         {object}  dtos.ErrorResponse      "Admin authorization required"
// @Failure      500         {object}  dtos.ErrorResponse      "Delete operation failed"
// @Security     BearerAuth
// @Router       /api/admin/vouchers/{voucher_id} [delete]
func DeleteVoucherHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can delete vouchers)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract voucher ID from URL path parameters
	voucherID := mux.Vars(r)["voucher_id"]

	// Delete voucher from database (may be soft delete)
	err := models.DeleteVoucher(voucherID)

	if err != nil {
		// Deletion failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to delete voucher",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate voucher caches to ensure fresh data
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200         {object}  dtos.SuccessResponse     "Voucher updated successfully"
// @Failure      400         {object}  dtos.ErrorResponse       "Invalid request or update failed"
// @Failure      401         {object}  dtos.ErrorResponse       "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/vouchers/{voucher_id} [patch]
func UpdateVoucherHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can update vouchers)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract voucher ID from URL path parameters
	voucherID := mux.Vars(r)["voucher_id"]
	// Decode and parse JSON request body with updated voucher data
	req, ok := DecodeRequestBody[dtos.VoucherDataUpdate](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	err := models.VoucherUpdate(*req, voucherID)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher with ID " + voucherID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: voucherWithID + voucherID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary List user Voucher by ID
// @Description List user Voucher details
// @Tags Vouchers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/vouchers/{voucher_id} [get]
func GetUserVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	voucherID := mux.Vars(r)["voucher_id"]

	voucher, err := models.GetUserVoucherByID(voucherID, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher with ID " + voucherID + " for user with ID " + fmt.Sprint(authuser.ID),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: voucherWithID + voucherID + " fetched successfully for user with ID " + fmt.Sprint(authuser.ID),
			Code:        http.StatusOK,
		},
		Payload:   voucher,
		Message:   "Voucher fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary List user Vouchers
// @Description List user Voucher details
// @Tags Vouchers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/vouchers/{voucher_id} [get]
func ListUserVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	voucher, pagination, err := models.GetUserVouchers(authuser.ID, page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers for user with ID " + fmt.Sprint(authuser.ID),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Vouchers fetched successfully for user with ID " + fmt.Sprint(authuser.ID),
			Code:        http.StatusOK,
		},
		Payload:   map[string]interface{}{"vouchers": voucher, "pagination": pagination},
		Message:   "Vouchers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

//Endpoint for users to buy vouchers

// @Summary Buy Voucher
// @Description Buy Voucher
// @Tags User
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/vouchers [post]
func BuyVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: noUserFound,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.BuyVoucherData](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}
	// create voucher data
	var voucher dtos.Voucher
	voucher.Amount = req.Amount
	voucher.DesignID = &req.DesignID
	active := "scheduled"
	voucher.Status = &active

	deliveryTime := models.StringToTime(req.DeliveryTime)
	//check devlivery time is in the past
	if deliveryTime.Before(time.Now()) {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Delivery time cannot be in the past",
				Code:        http.StatusBadRequest,
			},
			Message:   "Delivery time cannot be in the past",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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

	voucherID, err := models.AddNewVoucher(voucher, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//insert into voucher purchases
	err = models.InsertIntoVoucherPurchases(*req, authuser.ID, voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to record voucher purchase",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//create voucher order
	voucherOrderID, err := models.CreateVoucherOrder(req.Amount, voucherID, req.PaymentMethod)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher order",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	err = voucherPaymentProcessor(req.PaymentMethod, voucherOrderID, req.PhoneNumber, req.Amount)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to process voucher payment",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
		RawBody:   requestSummary})
}
func BuyVoucherUpdateHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	req, ok := DecodeRequestBody[dtos.BuyVoucherData](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}
	voucherID := mux.Vars(r)["voucher_id"]
	// create voucher data
	amount := req.Amount
	status := "scheduled"
	err := models.ValidateDesignID(req.DesignID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Invalid Design ID",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if req.Amount > 0 {
		err := models.UpdateVoucher(voucherID, amount, status, req.DesignID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: "Failed to update voucher",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	//insert into voucher purchases
	err = models.UpdateVoucherPurchases(*req, voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher purchase",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//create voucher order
	voucherOrderID, err := models.UpdateVoucherPurchaseAmount(req.Amount, voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher purchase amount",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if req.Amount > 0 || req.PaymentMethod != "none" {
		err = voucherPaymentProcessor(req.PaymentMethod, voucherOrderID, req.PhoneNumber, req.Amount)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: "Failed to process voucher payment",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
		RawBody:   requestSummary})
}
func voucherPaymentProcessor(paymentMethod string, voucherOrderID, phoneNumber string, amount float64) error {
	switch paymentMethod {
	//where methdod is mpesa or empty use mpesa
	case "mpesa", "":
		// Initiate Mpesa payment
		err := HandleMpesaVoucherPayment(voucherOrderID, phoneNumber, amount)
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
// @Success      200      {object}  dtos.SuccessResponse       "Voucher redeemed successfully"
// @Failure      400      {object}  dtos.ErrorResponse         "Invalid code or redemption failed"
// @Failure      401      {object}  dtos.ErrorResponse         "User authentication required"
// @Security     BearerAuth
// @Router       /api/vouchers/redeem [post]
func RedeemVoucherHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Get authenticated user from context
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "User not validated or unauthorized",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Decode and parse JSON request body with voucher code
	req, ok := DecodeRequestBody[dtos.RedeemVoucherRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate voucher code format
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Redeem voucher code to user's account
	_, err := models.RedeemVoucher(req.Code, authuser.ID)
	if err != nil {
		// Redemption failed (invalid code, already redeemed, expired, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to redeem voucher",
				Code:        http.StatusBadRequest,
			},
			Message: err.Error(),

			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate voucher caches to reflect redemption status
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher redeemed successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher redeemed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
	users, err := models.GetUsersWithUnsentVoucherEmails()
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
			err := models.MarkVoucherEmailAsSent(u.VoucherID)
			if err != nil {
				log.Printf("error marking voucher email as sent for user: %v", err)
			}
		}
	}
}

func EditVoucherDesign(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers"); !ok {
		return
	}
	designID := mux.Vars(r)["voucher_id"]

	// Parse multipart form (20 MB max)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Get image file (optional)
	var url string
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		// Upload image to GCS
		url, err = utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return
		}
	}
	// Insert category into DB
	err = models.EditVoucherDesign(designID, &url, r.FormValue("name"), r.FormValue("status"))
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher design",
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

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher design updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   url,
		Message:   "Voucher design updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func DeleteVoucherDesign(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers"); !ok {
		return
	}
	designID := mux.Vars(r)["voucher_id"]
	// Insert category into DB
	err := models.DeleteVoucherDesign(designID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to delete voucher design",
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

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher design deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher design deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func GetAllVoucherDesigns(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	name := r.URL.Query().Get("name")
	status := r.URL.Query().Get("status")
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	designs, pagination, err := models.GetAllVoucherDesigns(page, size, name, status)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher designs",
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

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher designs fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"designs": designs, "pagination": pagination},
		Message:   "Voucher designs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func GetVoucherDesignByID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	designID := mux.Vars(r)["voucher_id"]
	designs, err := models.GetVoucherDesign(designID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher design",
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

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher design fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   designs,
		Message:   "Voucher design fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func ListVoucherPurchasesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		return
	}
	customerName := r.URL.Query().Get("name")
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	vouchers, pagination, err := models.ListVoucherPurchases(page, size, customerName)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Vouchers purchases fetched successfully for all users",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"vouchers":   vouchers,
			"pagination": pagination,
		},
		Message:   "Vouchers purchases fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}
func GetVoucherPurchasesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		return
	}
	voucherID := mux.Vars(r)["purchase_id"]
	vouchers, err := models.GetVoucherPurchases(voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Vouchers purchase fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   vouchers,
		Message:   "Vouchers purchase fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}
