package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DeleteDealHandler deletes a deal.
// This endpoint is restricted to administrators.
//
// @Summary      Delete a deal
// @Description  Delete a deal by ID
// @Tags         Deals
// @Produce      json
// @Param        deal_id  path      string  true  "Deal ID"
// @Success      200      {object}  map[string]any
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
// @Success      200           {object}  map[string]any
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
// @Success      200           {object}  map[string]any
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
// @Success      200      {object}  map[string]any
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
// @Success      201       {object}  map[string]any
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
