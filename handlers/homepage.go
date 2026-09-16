package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	blogWithID = "Blog with ID "
)

// HomePageData returns homepage footer, social links, and menu links.
//
// @Summary      Homepage Data
// @Description  Get footer, social, and menu links for homepage.
// @Tags         Home
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      404  {object}  dtos.ErrorResponse
// @Router       /api/home/data [get]
func HomePageData(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	cacheKey := fmt.Sprintf("home_data:tenant_%d", tenantID)

	if utils.RedisClient != nil {
		cached, err := utils.RedisClient.Get(c.Request.Context(), cacheKey).Result()
		if err == nil && cached != "" {
			c.Data(http.StatusOK, "application/json", []byte(cached))
			return
		}
	}

	footer, _ := models.GetFooterData(models.DB)
	socials, _ := models.GetSocialsData(models.DB)
	menu, _ := models.GetMenuData(models.DB)

	var copyrightText, companyAddress, contactEmail, phoneNumber string
	if len(footer) > 0 {
		if footer[0].CopyrightText != nil {
			copyrightText = *footer[0].CopyrightText
		}
		if footer[0].CompanyAddress != nil {
			companyAddress = *footer[0].CompanyAddress
		}
		if footer[0].ContactEmail != nil {
			contactEmail = *footer[0].ContactEmail
		}
		if footer[0].PhoneNumber != nil {
			phoneNumber = *footer[0].PhoneNumber
		}
	}

	response := map[string]any{
		"data": map[string]any{
			"copyright_text":  copyrightText,
			"company_address": companyAddress,
			"contact_email":   contactEmail,
			"phone_number":    phoneNumber,
			"social_links": func() []map[string]string {
				var links []map[string]string
				for _, link := range socials {
					if link.Platform != nil && link.URL != nil && link.IconClass != nil {
						links = append(links, map[string]string{
							"platform":   *link.Platform,
							"url":        *link.URL,
							"icon_class": *link.IconClass,
						})
					}
				}
				return links
			}(),
			"links": menu,
		},
		"meta": map[string]string{
			"version":     "1.0",
			"api_version": "v1",
		},
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Homepage data fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Sucess",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetHomeBannersData returns homepage banner sliders.
//
// @Summary      Slider Data
// @Description  Get banner/slider data for homepage.
// @Tags         Home
// @Produce      json
// @Success      200  {object}  map[string]any
// @Router       /api/home/banners [get]
func GetHomeBannersData(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	banners, _ := models.GetBannersData(models.DB, "homebanner")
	var data []map[string]any
	for _, b := range banners {
		data = append(data, map[string]any{
			"id":            b.ID,
			"image_url":     b.ImageURL,
			"text":          b.Text,
			"heading":       b.Heading,
			"button_text":   b.ButtonText,
			"button_url":    b.ButtonURL,
			"display_order": b.DisplayOrder,
			"is_active":     b.IsActive,
			"type":          b.Type,
		})
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Home",
			Description: "Homepage banners fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   data,
		Message:   "Success",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetSliderData returns homepage banner sliders.
//
// @Summary      Slider Data
// @Description  Get banner/slider data for homepage.
// @Tags         Home
// @Produce      json
// @Success      200  {object}  map[string]any
// @Router       /api/home/sliders [get]
func GetSliderData(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	banners, _ := models.GetBannersData(models.DB, "banner")
	var data []map[string]any
	for _, b := range banners {
		data = append(data, map[string]any{
			"id":          b.ID,
			"image_url":   b.ImageURL,
			"text":        b.Text,
			"heading":     b.Heading,
			"button_text": b.ButtonText,
			"button_url":  b.ButtonURL,
		})
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Homepage sliders fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   data,
		Message:   "Success",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetSliderData returns homepage banner sliders for admin.
//
// @Summary      Slider Data
// @Description  Get banner/slider data for homepage.
// @Tags         Home
// @Produce      json
// @Success      200  {object}  map[string]any
// @Router       /api/home/sliders [get]
func AdminGetSliderData(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	banners, _ := models.AdminGetBannersData(models.DB, "banner")
	var data []map[string]any
	for _, b := range banners {
		data = append(data, map[string]any{
			"id":          b.ID,
			"image_url":   b.ImageURL,
			"text":        b.Text,
			"heading":     b.Heading,
			"button_text": b.ButtonText,
			"button_url":  b.ButtonURL,
			"is_active":   b.IsActive,
			"type":        b.Type,
		})
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Homepage sliders fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   data,
		Message:   "Success",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// // GetCategories returns featured and deal categories.
// // @Summary Categories Data
// // @Description Get featured and deal-based product categories.
// // @Tags Home
// // @Produce json
// // @Success 200 {object} map[string]any
// // @Router /api/home/categories [get]
// func GetCategories(c *gin.Context) {
// 	start := time.Now()
// 	// Read and restore body FIRST
// 	requestSummary := utils.GetRequestSummary(c.Request)
// 	categories, err := models.GetCategoriesWithProducts()
// 	if err != nil {
// 		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
// 			Code:      http.StatusInternalServerError,
// 			Message:   "Failed to fetch categories",
// 			TimeTaken: time.Since(start),
// 			Function:  utils.GetCurrentFuncName(),
// 			Request: c.Request,
// 			RawBody:   requestSummary})
// 		return
// 	}

// 	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
// 		Code:      http.StatusOK,
// 		Payload:   categories,
// 		Message:   "Categories",
// 		TimeTaken: time.Since(start),
// 		Function:  utils.GetCurrentFuncName(),
// 		Request: c.Request,
// 		RawBody:   requestSummary})
// }

// GetPromotionsHandler handles GET /api/promotions
//
// @Summary      Get active promotions
// @Description  Retrieve all active promotions with products and category info
// @Tags         Promotions
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      500  {object}  dtos.ErrorResponse
// @Router       /api/home/promotions [get]
func GetPromotionsHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	var promotions []dtos.Promotion
	var cachedPromotions []dtos.Promotion
	// first try using cache
	_ = utils.GetCache("promotions", &cachedPromotions)

	if cachedPromotions == nil {
		var err error
		promotions, err = models.GetPromotions(models.DB)
		if err != nil {
			log.Printf("promotiones error::%s", err)
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Promotions",
					Description: "Failed to fetch promotions",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache("promotions", promotions)
	} else {
		promotions = cachedPromotions
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promotions fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   promotions,
		Message:   "Promotions fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AddBannerInfo adds new banner information.
// This endpoint is restricted to administrators.
//
// @Summary      Add banner info
// @Description  Add banner info
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        banner_image   formData  file    true   "Banner Image"
// @Param        text           formData  string  false  "Banner Text"
// @Param        heading        formData  string  false  "Banner Heading"
// @Param        button_text    formData  string  false  "Button Text"
// @Param        button_url     formData  string  false  "Button URL"
// @Param        display_order  formData  int     false  "Display Order"
// @Param        is_active      formData  bool    false  "Is Active"
// @Param        type           formData  string  false  "Banner Type"
// @Success      201            {object}  map[string]any
// @Failure      400            {object}  dtos.ErrorResponse
// @Failure      500            {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/banners [post]
