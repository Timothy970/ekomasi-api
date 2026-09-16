package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func AddBannerInfo(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Parse the multipart form
	err := c.Request.ParseMultipartForm(20 << 20) // 20 MB
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to parse form: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Get the image
	file, header, err := c.Request.FormFile("banner_image")
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "No banner_image uploaded",
				Code:        http.StatusBadRequest,
			},
			Message:   "No banner_image uploaded",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}
	defer file.Close()

	// Upload the image to GCS
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to upload file: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Build the request DTO
	displayOrder, _ := strconv.Atoi(c.Request.FormValue("display_order"))
	isActive := c.Request.FormValue("is_active") == "true"
	req := dtos.BannerInfo{
		Image:        &url,
		Text:         utils.StringPtr(c.Request.FormValue("text")),
		Heading:      utils.StringPtr(c.Request.FormValue("heading")),
		ButtonText:   utils.StringPtr(c.Request.FormValue("button_text")),
		ButtonURL:    utils.StringPtr(c.Request.FormValue("button_url")),
		DisplayOrder: utils.IntPtr(displayOrder),
		IsActive:     utils.BoolPtr(isActive),
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Homepage") {
		return
	}
	// Insert banner info into DB
	if err := models.InsertBannerDetails(models.DB, url, req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to insert image into DB: " + err.Error(),
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
			Module:      "Homepage",
			Description: "Banner with url " + url + " uploaded successfully",
			Code:        http.StatusCreated,
		},
		Payload: map[string]any{
			"banner_url": url,
		},
		Message:   "Banner uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
	})
}

// UpdateBannerInfo updates existing banner information.
// This endpoint is restricted to administrators.
//
// @Summary      Update banner info
// @Description  Update banner info
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        banner_id  path      string                true  "Banner ID"
// @Param        banner     body      dtos.UpdateBannerInfo true  "Banner Details"
// @Success      200        {object}  map[string]any
// @Failure      400        {object}  dtos.ErrorResponse
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/banners/{banner_id} [patch]
func UpdateBannerInfo(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "promotions.update")
	if !ok {
		return
	}
	bannerID := c.Param("banner_id")
	// Parse the multipart form
	err := c.Request.ParseMultipartForm(20 << 20) // 20 MB
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to parse form: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Try to get the image (optional)
	file, header, err := c.Request.FormFile("banner_image")

	var imageURL *string

	if err == nil {
		defer file.Close()

		// Upload only if file exists
		url, uploadErr := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if uploadErr != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Homepage",
					Description: "Failed to upload file: " + uploadErr.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   uploadErr.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return
		}

		imageURL = &url
	}

	// Build the request DTO
	displayOrder, _ := strconv.Atoi(c.Request.FormValue("display_order"))
	isActive := c.Request.FormValue("is_active") == "true"
	//decode request body
	req := dtos.BannerInfo{
		Image:        imageURL,
		Text:         utils.StringPtr(c.Request.FormValue("text")),
		Heading:      utils.StringPtr(c.Request.FormValue("heading")),
		ButtonText:   utils.StringPtr(c.Request.FormValue("button_text")),
		ButtonURL:    utils.StringPtr(c.Request.FormValue("button_url")),
		DisplayOrder: utils.IntPtr(displayOrder),
		IsActive:     utils.BoolPtr(isActive),
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Homepage") {
		return
	}
	if err := models.UpdateBannerDetails(models.DB, req, bannerID); err != nil {
		log.Printf("Error updating banner: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to update banner with ID " + bannerID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Banner with ID " + bannerID + " has been updated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Banner updated succesfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteBannerInfo deletes a banner.
// This endpoint is restricted to administrators.
//
// @Summary      Delete banner info
// @Description  Delete banner info
// @Tags         Admin
// @Produce      json
// @Param        banner_id  path      string  true  "Banner ID"
// @Success      200        {object}  map[string]any
// @Failure      400        {object}  dtos.ErrorResponse
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/banners/{banner_id} [delete]
func DeleteBannerInfo(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "promotions.delete")
	if !ok {
		return
	}
	bannerID := c.Param("banner_id")
	err := models.DeleteBanner(bannerID)
	if err != nil {
		log.Printf("Error deleting banner: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to delete banner with ID " + bannerID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Banner with ID " + bannerID + " has been deleted successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Banner deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetPromotionsTypesHandler handles GET /api/promotions/types
//
// @Summary      Get active promotions types
// @Description  Retrieve all active promotions types
// @Tags         Promotions
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      500  {object}  dtos.ErrorResponse
// @Router       /api/home/promotions/types [get]
