// Package handlers provides HTTP request handlers for promotion and promo code management.
// This file contains handlers for creating, updating, retrieving, and deleting promotional campaigns,
// promo codes, and managing product-promotion associations.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
// @Success      200                   {object}  map[string]interface{}  "Promo code created successfully"
// @Failure      400                   {object}  dtos.ErrorResponse      "Invalid request data"
// @Failure      500                   {object}  dtos.ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes [post]
func AddPromoCodeHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for creating promo codes)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Promotions", "promotions.create"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse minimum order value from form data
	minimumOrderValue := models.StringToFloat64(r.FormValue("minimum_order_value"))

	// Build promo code request object from form data
	req := &dtos.PromoCodeRequest{
		Discount_Code:     r.FormValue("discount_code"),                            // Unique promo code
		DiscountType:      r.FormValue("discount_type"),                            // percentage or fixed
		DiscountValue:     models.StringToFloat64(r.FormValue("discount_value")),   // Discount amount
		ExpiresAt:         r.FormValue("expires_at"),                               // Expiration date
		MinimumOrderValue: &minimumOrderValue,                                      // Min order requirement
		MaximumUse:        int(models.StringToFloat64(r.FormValue("maximum_use"))), // Usage limit
		IsActive:          models.StringToBool(r.FormValue("is_active")),           // Active status
	}
	// Validate all required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Promotions") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Insert promo code into database
	promo, err := models.AddPromoCode(models.DB, *req)
	if err != nil {
		// Database insertion failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to add promo code",
				Code:        http.StatusInternalServerError,
			},
			Message: err.Error(), TimeTaken: time.Since(start),
			Function: utils.GetCurrentFuncName(),
			Request:  r,
			RawBody:  requestSummary,
		})
		return
	}
	// Return successful response with created promo code data
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promo code added successfully",
			Code:        http.StatusOK,
		},
		Payload: promo, Message: "Promo code added successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
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
// @Success      200                   {object}  map[string]interface{}  "Promo code updated successfully"
// @Failure      400                   {object}  dtos.ErrorResponse      "Invalid request data"
// @Failure      404                   {object}  dtos.ErrorResponse      "Promo code not found"
// @Failure      500                   {object}  dtos.ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes/{promo_id} [put]
func UpdatePromoCodeHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for updating promo codes)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Promotions", "promotions.update"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Extract promo code ID from URL path parameters
	id := mux.Vars(r)["promo_id"]
	// Parse minimum order value from form data
	minimumOrderValue := models.StringToFloat64(r.FormValue("minimum_order_value"))
	// Build promo code update request object from form data
	req := &dtos.PromoCodeRequest{
		Discount_Code:     r.FormValue("discount_code"),                            // Updated promo code
		DiscountType:      r.FormValue("discount_type"),                            // Updated discount type
		DiscountValue:     models.StringToFloat64(r.FormValue("discount_value")),   // Updated discount value
		ExpiresAt:         r.FormValue("expires_at"),                               // Updated expiration
		MinimumOrderValue: &minimumOrderValue,                                      // Updated min order value
		MaximumUse:        int(models.StringToFloat64(r.FormValue("maximum_use"))), // Updated usage limit
		IsActive:          models.StringToBool(r.FormValue("is_active")),           // Updated active status
	}
	// Validate all required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Promotions") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Update promo code in database
	promo, err := models.UpdatePromoCode(models.DB, id, *req)
	if err != nil {
		// Database update failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to update promo code with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message: err.Error(), TimeTaken: time.Since(start),
			Function: utils.GetCurrentFuncName(),
			Request:  r,
			RawBody:  requestSummary,
		})
		return
	}
	// Return successful response with updated promo code data
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: promoCodeWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload: promo, Message: "Promo code updated successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
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
// @Success      200       {object}  map[string]interface{}  "Promo code details"
// @Failure      404       {object}  dtos.ErrorResponse      "Promo code not found"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes/{promo_id} [get]
func GetPromoCodeByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for viewing promo codes)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Promotions", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract promo code ID from URL path parameters
	id := mux.Vars(r)["promo_id"]
	// Fetch promo code details from database
	promo, err := models.GetPromoCodeByID(models.DB, id)
	if err != nil {
		// Promo code not found or database error, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Failed to get promo code with ID " + id,
			Code:        http.StatusNotFound,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return successful response with promo code details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Promotions",
		Description: promoCodeWithID + id + " retrieved successfully",
		Code:        http.StatusOK,
	},
		Payload: promo, Message: "Promo code retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200    {object}  map[string]interface{}  "List of promo codes with pagination"
// @Failure      500    {object}  dtos.ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes [get]
func GetAllPromoCodesHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for viewing promo codes)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Promotions", ""); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Fetch paginated promo codes from database
	promos, pagination, err := models.GetAllPromoCodes(models.DB, page, limit)
	if err != nil {
		// Database query failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Failed to retrieve all promo codes",
			Code:        http.StatusInternalServerError,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return successful response with promo codes list and pagination metadata
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Promotions",
		Description: "All promo codes retrieved successfully",
		Code:        http.StatusOK,
	},
		Payload: map[string]any{"promocodes": promos, "pagination": pagination}, Message: "Promo codes retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeletePromoCodeHandler permanently removes a promotional discount code from the system.
// This action is irreversible and should be used with caution. Consider deactivating
// promo codes instead of deleting them to preserve historical data.
//
// @Summary      Delete a promo code
// @Description  Permanently remove a promotional discount code from the system
// @Tags         Promotions
// @Produce      json
// @Param        promo_id  path      string  true  "Promo code ID"
// @Success      200       {object}  map[string]interface{}  "Promo code deleted successfully"
// @Failure      404       {object}  dtos.ErrorResponse      "Promo code not found"
// @Failure      500       {object}  dtos.ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes/{promo_id} [delete]
func DeletePromoCodeHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for deleting promo codes)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Promotions", "promotions.delete"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Extract promo code ID from URL path parameters
	id := mux.Vars(r)["promo_id"]

	// Delete promo code from database
	err := models.DeletePromoCode(models.DB, id)
	if err != nil {
		// Deletion failed or promo code not found, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Failed to delete promo code with ID " + id,
			Code:        http.StatusInternalServerError,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return successful response confirming deletion
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Promotions",
		Description: promoCodeWithID + id + " deleted successfully",
		Code:        http.StatusOK,
	},
		Payload:   nil,
		Message:   "Promo code deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// TogglePromoCodeStatusHandler activates or deactivates a promotional discount code.
// This allows administrators to enable or disable promo codes without deleting them,
// preserving historical data while controlling availability.
//
// @Summary      Toggle promo code active status
// @Description  Activate or deactivate a promotional discount code
// @Tags         Promotions
// @Accept       json
// @Produce      json
// @Param        promo_id   path      string                         true  "Promo code ID"
// @Param        request    body      dtos.PromoCodeStatusRequest    true  "Status change request"
// @Success      200        {object}  map[string]interface{}         "Status updated successfully"
// @Failure      400        {object}  dtos.ErrorResponse             "Invalid request data"
// @Failure      404        {object}  dtos.ErrorResponse             "Promo code not found"
// @Failure      500        {object}  dtos.ErrorResponse             "Internal server error"
// @Security     BearerAuth
// @Router       /admin/promotions/promo-codes/{promo_id}/status [patch]
func TogglePromoCodeStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for changing promo code status)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Promotions", "promotions.update"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Extract promo code ID from URL path parameters
	id := mux.Vars(r)["promo_id"]
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.PromoCodeStatusRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}

	// Validate request structure
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Promotions") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Update promo code active status in database
	err := models.SetPromoCodeActiveStatus(models.DB, id, req.IsActive)
	if err != nil {
		// Status update failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to set promo code active status for ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Prepare appropriate success message based on the new status
	msg := "Promo code deactivated successfully"
	if req.IsActive {
		msg = "Promo code activated successfully"
	}

	// Return successful response with status update confirmation
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: msg + " for ID " + id,
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// AddPromotionToProductHandler associates a promotional discount with a specific product.
// This creates a product-promotion relationship allowing the product to participate
// in the specified promotional campaign.
//
// @Summary      Add promotion to product
// @Description  Associate a promotional discount or campaign with a specific product
// @Tags         Promotions
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.AddPromotionToProductRequest  true  "Product-promotion association request"
// @Success      200      {object}  map[string]interface{}             "Promotion added to product successfully"
// @Failure      400      {object}  dtos.ErrorResponse                 "Invalid request data"
// @Failure      404      {object}  dtos.ErrorResponse                 "Product or promotion not found"
// @Failure      500      {object}  dtos.ErrorResponse                 "Internal server error"
// @Security     BearerAuth
// @Router       /admin/promotions/products [post]
func AddPromotionToProductHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for managing product promotions)
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Promotions", "promotions.create"); !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.AddPromotionToProductRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate request structure (product ID and promotion type ID required)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Promotions") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Create product-promotion association in database
	if err := models.AddPromotionToProduct(models.DB, *req); err != nil {
		// Association creation failed, return error response
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Failed to add promotion to product with ID " + req.ProductID,
			Code:        http.StatusInternalServerError,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return successful response confirming promotion was added to product
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Promotions",
		Description: "Promotion added to product with ID " + req.ProductID + " successfully",
		Code:        http.StatusOK,
	},
		Payload:   nil,
		Message:   "Promotion added to product successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
