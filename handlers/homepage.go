package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"database/sql"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// HomePageData returns homepage footer, social links, and menu links.
// @Summary Homepage Data
// @Description Get footer, social, and menu links for homepage.
// @Tags Home
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /api/home/data [get]
func HomePageData(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	footerRows, err := models.GetFooterData()
	if err != nil || len(footerRows) == 0 {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	socialLinks, _ := models.GetSocialsData()
	menuLinks, _ := models.GetMenuData()

	response := map[string]interface{}{
		"data": map[string]interface{}{
			"copyright_text":  footerRows[0].CopyrightText,
			"company_address": footerRows[0].CompanyAddress,
			"contact_email":   footerRows[0].ContactEmail,
			"phone_number":    footerRows[0].PhoneNumber,
			"social_links": func() []map[string]string {
				var links []map[string]string
				for _, link := range socialLinks {
					links = append(links, map[string]string{
						"platform":   link.Platform,
						"url":        link.URL,
						"icon_class": link.IconClass,
					})
				}
				return links
			}(),
			"menu_links": func() []map[string]string {
				var links []map[string]string
				for _, link := range menuLinks {
					links = append(links, map[string]string{
						"id":        link.ID,
						"title":     link.Title,
						"url":       link.URL,
						"parent_id": link.ParentID,
					})
				}
				return links
			}(),
		},
		"meta": map[string]string{
			"version":     "1.0",
			"api_version": "v1",
		},
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   response,
		Message:   "Sucess",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetSliderData returns homepage banner sliders.
// @Summary Slider Data
// @Description Get banner/slider data for homepage.
// @Tags Home
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/home/banners [get]
func GetHomeBannersData(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	banners, _ := models.GetBannersData("homebanner")
	var data []map[string]interface{}
	for _, b := range banners {
		data = append(data, map[string]interface{}{
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
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   data,
		Message:   "Success",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetSliderData returns homepage banner sliders.
// @Summary Slider Data
// @Description Get banner/slider data for homepage.
// @Tags Home
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/home/sliders [get]
func GetSliderData(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	banners, _ := models.GetBannersData("banner")
	var data []map[string]interface{}
	for _, b := range banners {
		data = append(data, map[string]interface{}{
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
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   data,
		Message:   "Success",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// // GetCategories returns featured and deal categories.
// // @Summary Categories Data
// // @Description Get featured and deal-based product categories.
// // @Tags Home
// // @Produce json
// // @Success 200 {object} map[string]interface{}
// // @Router /api/home/categories [get]
// func GetCategories(w http.ResponseWriter, r *http.Request) {
// 	start := time.Now()
// 	// Read and restore body FIRST
// 	requestSummary := utils.GetRequestSummary(r)
// 	categories, err := models.GetCategoriesWithProducts()
// 	if err != nil {
// 		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
// 			Code:      http.StatusInternalServerError,
// 			Message:   "Failed to fetch categories",
// 			TimeTaken: time.Since(start),
// 			Function:  utils.GetCurrentFuncName(),
// 			Request:   r,
// 			RawBody:   requestSummary})
// 		return
// 	}

// 	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
// 		Code:      http.StatusOK,
// 		Payload:   categories,
// 		Message:   "Categories",
// 		TimeTaken: time.Since(start),
// 		Function:  utils.GetCurrentFuncName(),
// 		Request:   r,
// 		RawBody:   requestSummary})
// }

// GetPromotionsHandler handles GET /api/promotions
// @Summary Get active promotions
// @Description Retrieve all active promotions with products and category info
// @Tags Promotions
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/home/promotions [get]
func GetPromotionsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	promotions, err := models.GetPromotions()
	if err != nil {
		log.Printf("promotiones error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   promotions,
		Message:   "Promotions fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// func to add banners
// @Summary Add banner info
// @Description Add banner info
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/banners [POST]
func AddBannerInfo(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Parse the multipart form
	err := r.ParseMultipartForm(20 << 20) // 20 MB
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to parse form: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Get the image
	file, header, err := r.FormFile("banner_image")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "No banner_image uploaded",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	defer file.Close()

	// Upload the image to GCS
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to upload file: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Build the request DTO
	displayOrder, _ := strconv.Atoi(r.FormValue("display_order"))
	isActive := r.FormValue("is_active") == "true"
	req := dtos.BannerInfo{
		Image:        header,
		Text:         utils.StringPtr(r.FormValue("text")),
		Heading:      utils.StringPtr(r.FormValue("heading")),
		ButtonText:   utils.StringPtr(r.FormValue("button_text")),
		ButtonURL:    utils.StringPtr(r.FormValue("button_url")),
		DisplayOrder: utils.IntPtr(displayOrder),
		IsActive:     utils.BoolPtr(isActive),
		Type:         utils.StringPtr(r.FormValue("type")),
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	// Insert banner info into DB
	err = models.InsertBannerDetails(url, req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to insert image into DB: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
		Payload: map[string]interface{}{
			"banner_url": url,
		},
		Message:   "Banner uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
	})
}

// Update Banner Info
// @Summary Update banner info
// @Description Update banner info
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/banners [PATCH]
func UpdateBannerInfo(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	bannerID := mux.Vars(r)["banner_id"]
	//decode request body
	req, ok := DecodeRequestBody[dtos.UpdateBannerInfo](r, w, requestSummary, start)
	if !ok {
		return
	}
	err := models.UpdateBannerDetails(*req, bannerID)
	if err != nil {
		log.Printf("Error updating banner: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Banner updated succesfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete Banner Info
// @Summary Delete banner info
// @Description Delete banner info
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/banners [DELETE]
func DeleteBannerInfo(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	bannerID := mux.Vars(r)["banner_id"]
	err := models.DeleteBanner(bannerID)
	if err != nil {
		log.Printf("Error deleting banner: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Banner deleted succesfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get promotions types
// GetPromotionsHandler handles GET /api/promotions
// @Summary Get active promotions
// @Description Retrieve all active promotions with products and category info
// @Tags Promotions
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/home/promotions/types [get]
func GetPromotionsTypesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	promotions, err := models.GetPromotionsTypes()
	if err != nil {
		log.Printf("promotion types error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to fetch promotion types",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   promotions,
		Message:   "Promotion types fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// create a new promotion
//
// @Summary Create new promotions
// @Description Create Promotion
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/promotions [POST]
func NewPromotionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.NewPromotion](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	if !req.StartDate.Before(req.EndDate) {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Invalid validity period",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	_, err := models.CreateNewPromotion(*req)
	if err != nil {
		log.Printf("Error creating promotion: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to add promotion",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Promotion created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete a promotion
// @Summary Delete a promotion
// @Description Deletes a promotion
// @Tags Admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/admin/promotions/{promotion_id} [delete]
func DeletePromotionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	promotionID := mux.Vars(r)["promotion_id"]
	err := models.DeletePromotion(promotionID)
	if err != nil {
		log.Printf("Error deleting promotion: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to delete promotion",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Promotion deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Edit a promotion
// @Summary Edit a promotion
// @Description Edit a promotion
// @Tags Admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/admin/promotions/{promotion_id} [PATCH]
func EditPromotionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.EditPromotion](r, w, requestSummary, start)
	if !ok {
		return
	}
	err := models.EditPromotion(*req)
	if err != nil {
		log.Printf("Error editing promotion: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to add promotion",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Promotion edited successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Attach a product to a promotion
// @Summary Attach a product to promotion
// @Description Attach a product to promotion
// @Tags Admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/admin/promotions/{promotion_id} [POST]
func AttachProductToPromotionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.AttachProductToPromotion](r, w, requestSummary, start)
	if !ok {
		return
	}

	//check if promotion exists
	ok = checkIfPromotionExists(req.PromotionID, r, w, start)
	if !ok {
		return
	}
	err := models.AttachProductsToPromotion(*req)
	if err != nil {
		log.Printf("Error adding prodct to promotion promotion: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Products added to promotion successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func RemoveProductFromPromotionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.AttachProductToPromotion](r, w, requestSummary, start)
	if !ok {
		return
	}
	// user, ok := middleware.UserFromContext(r.Context())
	// if !ok {
	// 	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
	// 		Code:      http.StatusInternalServerError,
	// 		Message:   noUser,
	// 		TimeTaken: time.Since(start),
	// 		Function:  utils.GetCurrentFuncName(),
	// 		Request:   r,
	// 		RawBody:   requestSummary})
	// 	return
	// }
	//check if promotion exists
	ok = checkIfPromotionExists(req.PromotionID, r, w, start)
	if !ok {
		return
	}
	err := models.RemoveProductFromPromotion(*req)
	if err != nil {
		log.Printf("Error adding prodct to promotion promotion: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Products added to promotion successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func checkIfPromotionExists(promotionID string, r *http.Request, w http.ResponseWriter, start time.Time) bool {
	promotionExists, err := models.CheckPromotionExists(promotionID)
	if err != nil || !promotionExists {
		log.Printf("Error checking promotion existence: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Promotion does not exist",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return false
	}
	return true
}

// Create blog
// @Summary Create new blog
// @Description Create blog
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/blogs [POST]
func CreateBlogHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	blog, ok := DecodeRequestBody[dtos.Blog](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(blog, w, r, requestSummary, start) {
		return
	}
	blog.PublishedAt = time.Now().Format("2006-01-02 15:04:05")
	if err := models.CreateBlog(*blog); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Blog created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get blog
// @Summary Get blog by ID
// @Description Get blog by ID
// @Tags Blogs
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/blogs/{blog_id} [GET]
func GetBlogHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	blogID := mux.Vars(r)["blog_id"]
	blog, err := models.GetBlogByID(blogID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   blog,
		Message:   "Blog fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// update blog
// @Summary Update blog
// @Description Update blog
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/blogs/{blog_id} [PATCH]
func UpdateBlogHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	blog, ok := DecodeRequestBody[dtos.UpdateBlog](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(blog, w, r, requestSummary, start) {
		return
	}
	blogID := mux.Vars(r)["blog_id"]
	if err := models.UpdateBlog(blog.IsPublished, blogID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Blog updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete Blog
// @Summary Delete blog
// @Description Delete blog
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/blogs/{blog_id} [DELETE]
func DeleteBlogHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	blogID := mux.Vars(r)["blog_id"]
	if err := models.DeleteBlog(blogID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Blog deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// List all blogs
// @Summary List blogs
// @Description List blogs
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/blogs [GET]
func ListBlogsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	limit := 0
	page := 1
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("size")
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	} else {
		limit = 10
	}
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}
	blogs, pagination, err := models.ListBlogs(page, limit)
	if err != nil {
		log.Printf("blogs errrrrrrrrrrr#################%s", err)
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   "Blogs not found",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		}
		return
	}
	response := map[string]interface{}{
		"blogs":      blogs,
		"pagination": pagination,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   response,
		Message:   "Blogs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CreateMenuLink - POST api/admin/menu-links
// @Summary  Create Menu links
// @Description Create Menu Links
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/menu-links [POST]
func CreateMenuLink(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.MenuLinkRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	err := models.CreateMenuLink(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Menu link created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateMenuLink - PATCH admin/menu-links/{menulink_id}
// @Summary  Update Menu links
// @Description Update Menu Links
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/menu-links/{menulink_id} [PATCH]
func UpdateMenuLink(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.MenuLinkRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	params := mux.Vars(r)
	menulinkID, _ := strconv.Atoi(params["id"])
	err := models.UpdateMenuLink(*req, menulinkID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Menu link updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteMenuLink - DELETE /menu-links/{id}
// @Summary  Delete Menu links
// @Description Delete Menu Links
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/menu-links/{menulink_id} [DELETE]
func DeleteMenuLink(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	params := mux.Vars(r)
	id, _ := strconv.Atoi(params["menulink_id"])
	err := models.DeleteMenuLink(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Menu link deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary  Create Social
// @Description Create Social
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/socials [POST]
func CreateSocialLinkHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.SocialLinkRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	if err := models.CreateSocialLink(req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Social added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary  Update Social
// @Description Update Social
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/socials/{social_id} [PATCH]
func UpdateSocialLinkHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.SocialLinkRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	idParam := mux.Vars(r)["social_id"]
	socialID, _ := strconv.Atoi(idParam)

	if err := models.UpdateSocialLink(socialID, *req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Social updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary  Delete Social
// @Description Delete Social
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/socials/{social_id} [DELETE]
func DeleteSocialLinkHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	idParam := mux.Vars(r)["social_id"]
	socialID, _ := strconv.Atoi(idParam)

	if err := models.DeleteSocialLink(socialID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Social deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Add product to featured
// @Summary Add product to featured
// @Description Add product to featured
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/products/featured/{product_id} [post]
func AddFeaturedProduct(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]

	if err := models.AddFeaturedProduct(productID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Product added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Remove product from featured
// Remove product from featured
// @Summary Remove product from featured
// @Description Remove product from featured
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/products/featured/{product_id} [delete]
func RemoveFeatured(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]

	if err := models.RemoveFeaturedProduct(productID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Product removed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get all featured products
// @Summary All featured Products Data
// @Description Get all featured product categories.
// @Tags Products
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/products/featured [get]
func GetFeatured(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	featured, err := models.GetFeaturedProducts()
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   featured,
		Message:   "Featured products",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
