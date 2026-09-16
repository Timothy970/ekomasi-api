package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListSocialLinksHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	socialLinks, err := models.ListSocialLinks()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to list social links: " + err.Error(),
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
			Module:      "Homepage",
			Description: "Social links fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   socialLinks,
		Message:   "Social links fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// UpdateSocialLinkHandler updates an existing social link.
// This endpoint is restricted to administrators.
//
// @Summary      Update Social
// @Description  Update Social
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        social_id    path      string                  true  "Social Link ID"
// @Param        social_link  body      dtos.SocialLinkRequest  true  "Social Link Details"
// @Success      200          {object}  map[string]any
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/socials/{social_id} [patch]
func UpdateSocialLinkHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.SocialLinkRequest](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Homepage") {
		return
	}

	idParam := c.Param("social_id")
	socialID, _ := strconv.Atoi(idParam)

	if err := models.UpdateSocialLink(socialID, *req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to update social link with ID " + strconv.Itoa(socialID) + ": " + err.Error(),
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
			Module:      "Homepage",
			Description: "Social link with ID " + strconv.Itoa(socialID) + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Social updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteSocialLinkHandler deletes a social link.
// This endpoint is restricted to administrators.
//
// @Summary      Delete Social
// @Description  Delete Social
// @Tags         Admin
// @Produce      json
// @Param        social_id  path      string  true  "Social Link ID"
// @Success      200        {object}  map[string]any
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/socials/{social_id} [delete]
func DeleteSocialLinkHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "")
	if !ok {
		return
	}
	idParam := c.Param("social_id")
	socialID, _ := strconv.Atoi(idParam)

	if err := models.DeleteSocialLink(socialID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to delete social link with ID " + strconv.Itoa(socialID) + ": " + err.Error(),
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
			Module:      "Homepage",
			Description: "Social link with ID " + strconv.Itoa(socialID) + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Social deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AddFeaturedProduct adds a product to the featured list.
// This endpoint is restricted to administrators.
//
// @Summary      Add product to featured
// @Description  Add product to featured
// @Tags         Products
// @Produce      json
// @Param        product_id  path      string  true  "Product ID"
// @Success      200         {object}  map[string]any
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/featured/{product_id} [post]
func AddFeaturedProduct(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "products.create")
	if !ok {
		return
	}
	productID := c.Param("product_id")

	if err := models.AddFeaturedProduct(productID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to add product with ID " + productID + " to featured: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	_ = utils.DeleteCache("featured_products")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Product with ID " + productID + " added to featured successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Product added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// RemoveFeatured removes a product from the featured list.
// This endpoint is restricted to administrators.
//
// @Summary      Remove product from featured
// @Description  Remove product from featured
// @Tags         Products
// @Produce      json
// @Param        product_id  path      string  true  "Product ID"
// @Success      200         {object}  map[string]any
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/featured/{product_id} [delete]
func RemoveFeatured(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "products.delete")
	if !ok {
		return
	}
	productID := c.Param("product_id")

	if err := models.RemoveFeaturedProduct(productID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to remove product with ID " + productID + " from featured: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	_ = utils.DeleteCache("featured_products")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Product with ID " + productID + " removed from featured successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product removed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// Get all featured products
// @Summary All featured Products Data
// @Description Get all featured product categories.
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]any
// @Router /api/products/featured [get]
func GetFeatured(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	var featured []dtos.Product
	var cachedFeatured []dtos.Product
	_ = utils.GetCache("featured_products", &cachedFeatured)
	if cachedFeatured == nil {
		var err error
		featured, err = models.GetFeaturedProducts()
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Homepage",
					Description: "Failed to get featured products: " + err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache("featured_products", featured)
	} else {
		featured = cachedFeatured
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Featured products retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   featured,
		Message:   "Featured products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
