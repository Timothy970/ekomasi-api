package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
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

// DeleteDealHandler deletes a deal.
// This endpoint is restricted to administrators.
//
// @Summary      Delete a deal
// @Description  Delete a deal by ID
// @Tags         Deals
// @Produce      json
// @Param        deal_id  path      string  true  "Deal ID"
// @Success      200      {object}  map[string]interface{}
// @Failure      500      {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/deals/{deal_id} [delete]
func DeleteDealHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Deals", "promotions.delete")
	if !ok {
		return
	}
	dealID := c.Param("deal_id")
	if err := models.DeleteDeal(models.DB, dealID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to delete deal with ID " + dealID,
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
			Description: dealWithID + dealID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Deal deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AddProductToDealHandler adds a product to a deal.
// This endpoint is restricted to administrators.
//
// @Summary      Add product to deal
// @Description  Associate a product with a deal
// @Tags         Deals
// @Accept       json
// @Produce      json
// @Param        deal_product  body      dtos.ProductDeal  true  "Deal Product Details"
// @Success      200           {object}  map[string]interface{}
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      500           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/deals/products [post]
func AddProductToDealHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Deals", "products.create")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.ProductDeal](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Deals") {
		return
	}
	err := models.AddProductToDeal(models.DB, req.ID, req.ProductID, nil, nil)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to add product with product ID " + req.ProductID + " to deal with deal ID " + req.ID,
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
			Description: "Product with ID " + req.ProductID + " added to deal with ID " + req.ID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product added to deal successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// RemoveProductFromDealHandler removes a product from a deal.
// This endpoint is restricted to administrators.
//
// @Summary      Remove product from deal
// @Description  Remove a product association from a deal
// @Tags         Deals
// @Accept       json
// @Produce      json
// @Param        deal_product  body      dtos.ProductDeal  true  "Deal Product Details"
// @Success      200           {object}  map[string]interface{}
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      500           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/deals/products [delete]
func RemoveProductFromDealHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Deals", "products.delete")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.ProductDeal](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Deals") {
		return
	}
	if err := models.RemoveProductFromDeal(models.DB, req.ID, req.ProductID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to remove product with product ID " + req.ProductID + " from deal with deal ID " + req.ID,
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
			Description: "Product with ID " + req.ProductID + " removed from deal with ID " + req.ID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product removed from deal successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetDealWithProductsHandler retrieves a deal and its associated products.
//
// @Summary      Get deal with products
// @Description  Retrieve a deal and its products by deal ID
// @Tags         Deals
// @Produce      json
// @Param        deal_id  path      string  true   "Deal ID"
// @Param        page     query     int     false  "Page number"
// @Param        size     query     int     false  "Page size"
// @Success      200      {object}  map[string]interface{}
// @Failure      500      {object}  dtos.ErrorResponse
// @Router       /api/deals/{deal_id}/products [get]
func GetDealWithProductsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	page, limit := parsePagination(c.Query("page"), c.Query("size"))

	dealID := c.Param("deal_id")
	deals, pagination, err := models.GetDealWithProducts(models.DB, dealID, page, limit)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to get deal with ID " + dealID,
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
			Description: "Deal of ID " + dealID + " with products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"deals": deals, "pagination": pagination},
		Message:   "Deal with products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// CreateDealProductHandler creates a new deal with associated products.
// This endpoint is restricted to administrators.
//
// @Summary      Create deal with products
// @Description  Create a new deal and associate products with it
// @Tags         Deals
// @Accept       multipart/form-data
// @Produce      json
// @Param        title     formData  string  true   "Deal Title"
// @Param        image     formData  file    true   "Deal Image"
// @Param        duration  formData  string  true   "Duration (YYYY-MM-DD to YYYY-MM-DD)"
// @Param        products  formData  string  true   "JSON array of products"
// @Success      201       {object}  map[string]interface{}
// @Failure      400       {object}  dtos.ErrorResponse
// @Failure      500       {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/deals/create [post]
func CreateDealProductHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Deals", "products.create"); !ok {
		return
	}

	req, err := parseDealProductRequest(c.Request)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to parse deal product request",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Deals") {
		return
	}

	if err := validateProductsExist(req.Products, c, start, requestSummary); err != nil {
		return
	}

	startDate, endDate, err := parseDuration(req.Duration)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Deals",
				Description: "Failed to parse duration when creating deal",
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

	dealID, err := createDeal(req, startDate, endDate)
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
			RawBody:   requestSummary,
		})
		return
	}

	if err := addProductsToDeal(dealID, req.Products, c, start, requestSummary); err != nil {
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Deals",
			Description: dealWithID + dealID + " created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   fmt.Sprintf("%s Deal created successfully", req.Title),
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
func parseDealProductRequest(r *http.Request) (*dtos.FlashDealProducts, error) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, fmt.Errorf("image is required")
	}
	defer file.Close()

	url := "https://example.com/image.jpg" // Placeholder URL since GCS upload is not implemented here

	dealType := r.FormValue("deal_type")
	brandID := r.FormValue("brand_id")
	if dealType == "" {
		dealType = "product"
	}
	brandDiscount := r.FormValue("discount")
	brandDiscountType := r.FormValue("discount_type")
	if dealType == "brand" && (brandID == "" || brandDiscount == "" || brandDiscountType == "") {
		return nil, fmt.Errorf("brand ID, discount, and discount type are required for brand deals")
	}
	products, err := parseProducts(r.FormValue("products"), brandID, dealType, brandDiscount, brandDiscountType)
	if err != nil {
		return nil, err
	}
	return &dtos.FlashDealProducts{
		Title:    r.FormValue("title"),
		Image:    url,
		Duration: r.FormValue("duration"),
		Products: products,
		DealType: dealType,
		BrandID:  &brandID,
	}, nil
}
func parseProducts(productsStr string, brandID, dealType, brandDiscount, brandDiscountType string) ([]dtos.ProductsDeal, error) {
	var products []dtos.ProductsDeal
	if dealType == "product" {
		if productsStr == "" {
			return products, nil
		}

		if err := json.Unmarshal([]byte(productsStr), &products); err != nil {
			return nil, fmt.Errorf("invalid products format: %w", err)
		}
		if len(products) == 0 {
			return products, fmt.Errorf("products array cannot be empty for product deals")
		}
	} else if dealType == "brand" {
		var err error
		products, err = models.GetProductIDsByBrandID(models.DB, brandID, brandDiscount, brandDiscountType)
		if err != nil {
			return nil, fmt.Errorf("failed to get products by brand ID: %w", err)
		}
	} else {
		return nil, fmt.Errorf("invalid deal type: %s", dealType)
	}
	return products, nil
}
func validateProductsExist(products []dtos.ProductsDeal, c *gin.Context, start time.Time, requestSummary string) error {
	for _, p := range products {
		err := models.IsProductThere(models.DB, p.ProductID)
		if err != nil {
			if err.Error() == "product not found" {
				err = fmt.Errorf("product with ID %s not found", p.ProductID)
			}
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Deals",
					Description: "Failed to validate product existence when creating deal",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return err
		}
	}
	return nil
}
func parseDuration(duration string) (time.Time, time.Time, error) {
	var startStr, endStr string
	if _, err := fmt.Sscanf(duration, "%s to %s", &startStr, &endStr); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format. expected 'YYYY-MM-DD to YYYY-MM-DD'")
	}
	return models.StringToTime(startStr), models.StringToTime(endStr), nil
}
func createDeal(req *dtos.FlashDealProducts, startDate, endDate time.Time) (string, error) {
	dealData := dtos.CreateDeal{
		Name:      req.Title,
		StartDate: startDate,
		EndDate:   endDate,
		Image:     req.Image,
		DealType:  req.DealType,
		BrandID:   req.BrandID,
	}
	return models.CreateDeal(models.DB, dealData)
}

func addProductsToDeal(dealID string, products []dtos.ProductsDeal, c *gin.Context, start time.Time, requestSummary string) error {
	for _, p := range products {
		discount := float64(p.Discount)
		if err := models.AddProductToDeal(models.DB, dealID, p.ProductID, &p.DiscountType, &discount); err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Deals",
					Description: "Failed to add product to deal when creating deal with deal ID " + dealID,
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return err
		}
	}
	return nil
}
