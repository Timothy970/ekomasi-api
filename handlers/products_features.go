package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func parseFeatureImages(c *gin.Context, start time.Time) (string, []string, error) {
	var mainImageURL string
	file, header, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()
		mainImageURL, err = utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return "", nil, err
		}
	} else {
		log.Printf("[parseFeatureImages] No main image provided (this is OK): %v", err)
	}

	var imageURLs []string
	if c.Request.MultipartForm != nil && c.Request.MultipartForm.File["images"] != nil {
		for _, fh := range c.Request.MultipartForm.File["images"] {
			url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{fh})
			if err != nil {
				utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Products",
						Description: "Failed uploading images: " + err.Error(),
						Code:        http.StatusInternalServerError,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   c.Request,
				})
				return "", nil, err
			}
			imageURLs = append(imageURLs, url)
		}
	} else {
		log.Printf("[parseFeatureImages] No additional images provided")
	}
	return mainImageURL, imageURLs, nil
}

func parseFeatureJSONFields(c *gin.Context, start time.Time) ([]dtos.Section, []string, error) {
	var topSections []dtos.Section
	topSectionStr := c.Request.FormValue("top_section")

	if topSectionStr != "" {
		if err := json.Unmarshal([]byte(topSectionStr), &topSections); err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: invalidTopSectionJSON + err.Error(),
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid top_section format",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return nil, nil, err
		}
	} else {
		log.Printf("[parseFeatureJSONFields] top_section is empty")
	}

	var productSpecs []string
	specStr := c.Request.FormValue("product_specifications")

	if specStr != "" {
		if err := json.Unmarshal([]byte(specStr), &productSpecs); err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Invalid product_specifications JSON: " + err.Error(),
					Code:        http.StatusBadRequest,
				},
				Message:   "Invalid product_specifications format",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return nil, nil, err
		}
	} else {
		log.Printf("[parseFeatureJSONFields] product_specifications is empty")
	}

	return topSections, productSpecs, nil
}

// UpdateProductFeatureHandler updates an existing product feature.
// This endpoint is restricted to administrators.
//
// @Summary      Update product feature
// @Description  Update features, images, and specifications of a product
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        feature_id              path      string  true  "Feature ID"
// @Param        image                   formData  file    false "Main feature image"
// @Param        images                  formData  file    false "Additional feature images"
// @Param        header                  formData  string  false "Feature header"
// @Param        description             formData  string  false "Feature description"
// @Param        image_position          formData  string  false "Image position"
// @Param        top_section             formData  string  false "Top section JSON"
// @Param        product_specifications  formData  string  false "Product specifications JSON"
// @Param        design_type             formData  string  false "Design type"
// @Success      200                     {object}  dtos.ProductFeature
// @Failure      400                     {object}  dtos.ErrorResponse
// @Failure      500                     {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/features/{feature_id} [patch]
func UpdateProductFeatureHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.update"); !ok {
		return
	}

	featureID := c.Param("feature_id")

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

	feature, err := models.UpdateProductFeature(models.DB, req, featureID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product feature: " + err.Error(),
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
			Description: "Product feature updated successfully for feature ID " + featureID,
			Code:        http.StatusOK,
		},
		Payload:   feature,
		Message:   "Product feature updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// UpdateAllProductFeaturesHandler updates all features for a product.
// This endpoint is restricted to administrators.
//
// @Summary      Update all product features
// @Description  Update all features, images, and specifications of a product
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id              path      string  true  "Product ID"
// @Param        image                   formData  file    false "Main feature image"
// @Param        images                  formData  file    false "Additional feature images"
// @Param        header                  formData  string  false "Feature header"
// @Param        description             formData  string  false "Feature description"
// @Param        image_position          formData  string  false "Image position"
// @Param        top_section             formData  string  false "Top section JSON"
// @Param        product_specifications  formData  string  false "Product specifications JSON"
// @Param        design_type             formData  string  false "Design type"
// @Success      200                     {object}  dtos.ProductFeature
// @Failure      400                     {object}  dtos.ErrorResponse
// @Failure      500                     {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/{product_id}/features [patch]
