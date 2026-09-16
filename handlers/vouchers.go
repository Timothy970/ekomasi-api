package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"mime/multipart"
	"net/http"
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
// @Success      201     {object}  map[string]any "Design created successfully"
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
// @Success      201      {object}  map[string]any  "Voucher created successfully"
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
		Payload:   map[string]any{"voucher_id": voucherID},
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
// @Success      200          {object}  map[string]any "Vouchers with pagination metadata"
// @Failure      401          {object}  dtos.ErrorResponse     "Admin authorization required"
// @Failure      500          {object}  dtos.ErrorResponse     "Query failed"
// @Security     BearerAuth
// @Router       /api/admin/vouchers [get]
