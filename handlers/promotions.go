// Package handlers provides HTTP request handlers for promotion and promo code management.
// This file contains handlers for creating, updating, retrieving, and deleting promotional campaigns,
// promo codes, and managing product-promotion associations.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// promoCodeWithID is a reusable string constant for promo code messages
var promoCodeWithID = "Promo code with ID "

// AddPromoCodeHandler creates a new promotional discount code.
// It allows administrators to create promo codes with specific discount types (percentage or fixed),
// expiration dates, minimum order values, and usage limits.
//
// @Summary      Create a new promo code
// @Description  Create a promotional discount code with configurable discount type, value, expiration, and usage limits
// @Tags         Promotions
// @Accept       multipart/form-data
// @Produce      json
// @Param        discount_code         formData  string   true   "Unique promo code identifier"
// @Param        discount_type         formData  string   true   "Discount type (percentage or fixed)"
// @Param        discount_value        formData  number   true   "Discount value (percentage or amount)"
// @Param        expires_at            formData  string   true   "Expiration date (YYYY-MM-DD)"
// @Param        minimum_order_value   formData  number   false  "Minimum order value to apply promo"
// @Param        maximum_use           formData  integer  false  "Maximum number of uses allowed"
// @Param        is_active             formData  boolean  false  "Active status (default: true)"
// @Success      200                   {object}  map[string]any  "Promo code created successfully"
// @Failure      400                   {object}  dtos.ErrorResponse      "Invalid request data"
// @Failure      500                   {object}  dtos.ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes [post]
func AddPromoCodeHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for creating promo codes)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", "promotions.create"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse minimum order value from form data
	minimumOrderValue := models.StringToFloat64(c.Request.FormValue("minimum_order_value"))
	promoType := c.Request.FormValue("promo_type")
	if promoType == "" {
		promoType = "product"
	}
	brandID := c.Request.FormValue("brand_id")
	// Build promo code request object from form data
	req := &dtos.PromoCodeRequest{
		Discount_Code:     c.Request.FormValue("discount_code"),                            // Unique promo code
		DiscountType:      c.Request.FormValue("discount_type"),                            // percentage or fixed
		DiscountValue:     models.StringToFloat64(c.Request.FormValue("discount_value")),   // Discount amount
		ExpiresAt:         c.Request.FormValue("expires_at"),                               // Expiration date
		MinimumOrderValue: &minimumOrderValue,                                              // Min order requirement
		MaximumUse:        int(models.StringToFloat64(c.Request.FormValue("maximum_use"))), // Usage limit
		IsActive:          models.StringToBool(c.Request.FormValue("is_active")),           // Active status
		PromoType:         promoType,                                                       // Promo type
		BrandID:           &brandID,                                                        // Brand ID
	}
	// Validate all required fields
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Promotions") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Insert promo code into database
	promo, err := models.AddPromoCode(models.DB, *req)
	if err != nil {
		// Database insertion failed, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to add promo code",
				Code:        http.StatusInternalServerError,
			},
			Message: err.Error(), TimeTaken: time.Since(start),
			Function: utils.GetCurrentFuncName(),
			Request:  c.Request,
			RawBody:  requestSummary,
		})
		return
	}
	// Return successful response with created promo code data
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promo code added successfully",
			Code:        http.StatusOK,
		},
		Payload: promo, Message: "Promo code added successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  c.Request,
		RawBody:  requestSummary,
	})
}

// UpdatePromoCodeHandler updates an existing promotional discount code.
// It allows administrators to modify promo code details including discount values,
// expiration dates, usage limits, and active status.
//
// @Summary      Update an existing promo code
// @Description  Update promotional discount code details including discount type, value, expiration, and usage limits
// @Tags         Promotions
// @Accept       multipart/form-data
// @Produce      json
// @Param        promo_id              path      string   true   "Promo code ID"
// @Param        discount_code         formData  string   true   "Unique promo code identifier"
// @Param        discount_type         formData  string   true   "Discount type (percentage or fixed)"
// @Param        discount_value        formData  number   true   "Discount value (percentage or amount)"
// @Param        expires_at            formData  string   true   "Expiration date (YYYY-MM-DD)"
// @Param        minimum_order_value   formData  number   false  "Minimum order value to apply promo"
// @Param        maximum_use           formData  integer  false  "Maximum number of uses allowed"
// @Param        is_active             formData  boolean  false  "Active status"
// @Success      200                   {object}  map[string]any  "Promo code updated successfully"
// @Failure      400                   {object}  dtos.ErrorResponse      "Invalid request data"
// @Failure      404                   {object}  dtos.ErrorResponse      "Promo code not found"
// @Failure      500                   {object}  dtos.ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes/{promo_id} [patch]
func UpdatePromoCodeHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for updating promo codes)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", "promotions.update"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Extract promo code ID from URL path parameters
	id := c.Param("promo_id")
	// Parse minimum order value from form data
	minimumOrderValue := models.StringToFloat64(c.Request.FormValue("minimum_order_value"))
	promoType := c.Request.FormValue("promo_type")
	if promoType == "" {
		promoType = "product"
	}
	brandID := c.Request.FormValue("brand_id")
	// Build promo code update request object from form data
	req := &dtos.PromoCodeRequest{
		Discount_Code:     c.Request.FormValue("discount_code"),                            // Updated promo code
		DiscountType:      c.Request.FormValue("discount_type"),                            // Updated discount type
		DiscountValue:     models.StringToFloat64(c.Request.FormValue("discount_value")),   // Updated discount value
		ExpiresAt:         c.Request.FormValue("expires_at"),                               // Updated expiration
		MinimumOrderValue: &minimumOrderValue,                                              // Updated min order value
		MaximumUse:        int(models.StringToFloat64(c.Request.FormValue("maximum_use"))), // Updated usage limit
		IsActive:          models.StringToBool(c.Request.FormValue("is_active")),           // Updated active status
		PromoType:         promoType,
		BrandID:           &brandID,
	}
	// Validate all required fields
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Promotions") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Update promo code in database
	promo, err := models.UpdatePromoCode(models.DB, id, *req)
	if err != nil {
		// Database update failed, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to update promo code with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message: err.Error(), TimeTaken: time.Since(start),
			Function: utils.GetCurrentFuncName(),
			Request:  c.Request,
			RawBody:  requestSummary,
		})
		return
	}
	// Return successful response with updated promo code data
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: promoCodeWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload: promo, Message: "Promo code updated successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  c.Request,
		RawBody:  requestSummary,
	})
}

// GetPromoCodeByIDHandler retrieves a specific promotional discount code by its ID.
// It provides detailed information about a single promo code including discount details,
// expiration date, usage statistics, and current status.
//
// @Summary      Get promo code by ID
// @Description  Retrieve detailed information for a specific promotional discount code
// @Tags         Promotions
// @Produce      json
// @Param        promo_id  path      string  true  "Promo code ID"
// @Success      200       {object}  map[string]any  "Promo code details"
// @Failure      404       {object}  dtos.ErrorResponse      "Promo code not found"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes/{promo_id} [get]
func GetPromoCodeByIDHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for viewing promo codes)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract promo code ID from URL path parameters
	id := c.Param("promo_id")
	// Fetch promo code details from database
	promo, err := models.GetPromoCodeByID(models.DB, id)
	if err != nil {
		// Promo code not found or database error, return error response
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Failed to get promo code with ID " + id,
			Code:        http.StatusNotFound,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Return successful response with promo code details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Promotions",
		Description: promoCodeWithID + id + " retrieved successfully",
		Code:        http.StatusOK,
	},
		Payload: promo, Message: "Promo code retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetAllPromoCodesHandler retrieves a paginated list of all promotional discount codes.
// It provides an overview of all promo codes in the system with pagination support
// for efficient data loading and display.
//
// @Summary      Get all promo codes
// @Description  Retrieve a paginated list of all promotional discount codes with their details
// @Tags         Promotions
// @Produce      json
// @Param        page   query     int     false  "Page number (default: 1)"
// @Param        size   query     int     false  "Page size (default: 10)"
// @Success      200    {object}  map[string]any  "List of promo codes with pagination"
// @Failure      500    {object}  dtos.ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes [get]
