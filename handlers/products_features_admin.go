package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func UpdateAllProductFeaturesHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.update"); !ok {
		return
	}

	productID := c.Param("product_id")

	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse request data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	mainImageURL, imageURLs, err := parseFeatureImages(c, start)
	if err != nil {
		return
	}

	topSections, productSpecs, err := parseFeatureJSONFields(c, start)
	if err != nil {
		return
	}

	designType := c.Request.FormValue("design_type")
	imagePosition := c.Request.FormValue("image_position")
	//default image position to left if not provided
	if imagePosition == "" {
		imagePosition = "left"
	}
	req := dtos.ProductFeature{
		Image:                 &mainImageURL,
		Header:                c.Request.FormValue("header"),
		Description:           c.Request.FormValue("description"),
		ImagePosition:         imagePosition,
		Images:                &imageURLs,
		TopSection:            &topSections,
		ProductSpecifications: &productSpecs,
		DesignType:            &designType,
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	feature, err := models.UpdateProductFeatures(models.DB, req, productID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product feature: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product features updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   feature,
		Message:   "Product features updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetFeaturesByProductHandler retrieves features for a specific product.
//
// @Summary      Get product features
// @Description  Retrieve features, images, and specifications for a product
// @Tags         Products
// @Produce      json
// @Param        product_id  path      string  true  "Product ID"
// @Success      200         {object}  []dtos.ProductFeature
// @Failure      500         {object}  dtos.ErrorResponse
// @Router       /api/products/{product_id}/features [get]
func GetFeaturesByProductHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	productID := c.Param("product_id")
	features, err := models.GetProductFeaturesByProductID(models.DB, productID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch product features: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product features fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   features,
		Message:   "Product features fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// DeleteProductFeatureHandler deletes a product feature.
// This endpoint is restricted to administrators.
//
// @Summary      Delete product feature
// @Description  Delete a specific feature of a product
// @Tags         Admin
// @Produce      json
// @Param        feature_id  path      string  true  "Feature ID"
// @Success      200         {object}  map[string]any
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/features/{feature_id} [delete]
func DeleteProductFeatureHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.delete"); !ok {
		return
	}

	featureID := c.Param("feature_id")

	err := models.DeleteProductFeature(models.DB, featureID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete product feature: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to delete feature: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product feature deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product feature deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// HandleProductSpecifications adds specifications, variants, warranty, tax, and discounts to a product.
// This endpoint is restricted to administrators.
//
// @Summary      Add product specifications
// @Description  Add specifications, variants, warranty, tax, and discounts to a product
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        specifications  body      dtos.ProductSpecification  true  "Product Specifications"
// @Success      200             {object}  map[string]any
// @Failure      400             {object}  dtos.ErrorResponse
// @Failure      500             {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/specifications [post]
func HandleProductSpecifications(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure the user has the required permissions
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}
	state := "add"
	req, ok := DecodeRequestBody[dtos.ProductSpecification](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	//handle products specifications
	err := handleProductSpecs(models.DB, *req, state)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product specifications: " + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//handle products variants
	err = handleProductsVariants(models.DB, *req, state)
	log.Printf("handleProductsVariants ***** %s", err)

	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product variants: " + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//handle product warranty
	err = handleProductsWarranty(models.DB, *req)

	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product warranty: " + err.Error(),
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
			Module:      "Products",
			Description: "Product specifications added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product specifications added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// HandleProductSpecificationsUpdate updates specifications, variants, warranty, tax, and discounts for a product.
// This endpoint is restricted to administrators.
//
// @Summary      Update product specifications
// @Description  Update specifications, variants, warranty, tax, and discounts for a product
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        specifications  body      dtos.ProductSpecification  true  "Product Specifications"
// @Success      200             {object}  map[string]any
// @Failure      400             {object}  dtos.ErrorResponse
// @Failure      500             {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/specifications [patch]
