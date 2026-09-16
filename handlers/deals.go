package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var dealWithID = "Deal with ID "

// Create deals eg today's deal, flash sale, etc
// Get deals
// Update deals
// Delete deals
// Add and remove products from deals
// Get deals with their products

// CreateDealHandler creates a new deal.
// This endpoint is restricted to administrators.
//
// @Summary      Create a new deal
// @Description  Create a new promotional deal
// @Tags         Deals
// @Accept       json
// @Produce      json
// @Param        deal  body      dtos.CreateDeal  true  "Deal Details"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/deals [post]
func CreateDealHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Deals", "promotions.create")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateDeal](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Deals") {
		return
	}
	_, err := models.CreateDeal(models.DB, *req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to create deal",
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
			Module:      "Deals",
			Description: "Deal created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Deal created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetDealsHandler retrieves all deals.
//
// @Summary      Get all deals
// @Description  Retrieve a list of all deals with pagination
// @Tags         Deals
// @Produce      json
// @Param        page  query     int     false  "Page number"
// @Param        size  query     int     false  "Page size"
// @Success      200   {object}  map[string]interface{}
// @Failure      404   {object}  dtos.ErrorResponse
// @Router       /api/deals [get]
func GetDealsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	admin := c.Query("isAdmin")
	isAdmin := false
	if admin != "" {
		isAdmin = true
	}

	deals, meta, err := models.GetAllDeals(models.DB, page, limit, isAdmin)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to get deals",
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
			Module:      "Deals",
			Description: "All deals fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"deals": deals, "pagination": meta},
		Message:   "Deals fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// UpdateDealHandler updates an existing deal.
// This endpoint is restricted to administrators.
//
// @Summary      Update a deal
// @Description  Update an existing deal by ID
// @Tags         Deals
// @Accept       multipart/form-data
// @Produce      json
// @Param        deal_id    path      string  true   "Deal ID"
// @Param        name       formData  string  false  "Deal Name"
// @Param        image      formData  file    false  "Deal Image"
// @Param        start_date formData  string  false  "Start Date"
// @Param        end_date   formData  string  false  "End Date"
// @Param        is_active  formData  bool    false  "Is Active"
// @Success      200        {object}  map[string]interface{}
// @Failure      400        {object}  dtos.ErrorResponse
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/deals/{deal_id} [patch]
func UpdateDealHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Deals", "promotions.update")
	if !ok {
		return
	}
	dealID := c.Param("deal_id")
	file, header, err := c.Request.FormFile("image")
	var url string

	if err == nil && header != nil {
		// Remember to close the file if it exists
		defer file.Close()

		// Upload to GCS
		url, err = utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Deals",
					Description: err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   "Failed to upload image",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return
		}
	} else if err != http.ErrMissingFile && err != nil {
		// Handle any other unexpected error
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   "Error reading image file",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Continue even if no image was uploaded
	isActive := models.StringToBool(c.Request.FormValue("is_active"))
	dealType := c.Request.FormValue("deal_type")
	brandID := c.Request.FormValue("brand_id")
	if dealType == "" {
		dealType = "product"
	}
	brandDiscount := c.Request.FormValue("discount")
	brandDiscountType := c.Request.FormValue("discount_type")
	if dealType == "brand" && (brandID == "" || brandDiscount == "" || brandDiscountType == "") {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Brand ID, discount, and discount type are required for brand deals",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Brand ID, discount, and discount type are required for brand deals",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	products, err := parseProducts(c.Request.FormValue("deal_products"), brandID, dealType, brandDiscount, brandDiscountType)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to update deal with ID " + dealID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	req := &dtos.UpdateDeal{
		Name:      c.Request.FormValue("name"),
		StartDate: models.StringToTime(c.Request.FormValue("start_date")),
		EndDate:   models.StringToTime(c.Request.FormValue("end_date")),
		IsActive:  &isActive,
		Products:  products,
		DealType:  dealType,
		BrandID:   &brandID,
	}

	// Only set image if it was uploaded
	if url != "" {
		req.Image = &url
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Deals") {
		return
	}
	err = models.UpdateDeal(models.DB, dealID, *req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to update deal with ID " + dealID,
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
			Module:      "Deals",
			Description: dealWithID + dealID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Deal updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
