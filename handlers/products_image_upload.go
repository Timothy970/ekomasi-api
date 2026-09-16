package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func UploadProductImageHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Parse and validate upload request parameters (product_id, is_primary, video_link)
	productID, isPrimary, videoLink, err := parseUploadRequest(c.Request)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: err.Error(),
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

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse multipart form" + err.Error(),
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

	// Track all successfully uploaded files
	var uploadedResults []map[string]string

	// Handle optional video link (if provided in form data)
	if videoLink != "" {
		// Insert video link directly to database without upload
		if err := models.InsertProductImage(models.DB, productID, videoLink, "video", isPrimary); err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to insert video link for product ID " + productID,
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
	}

	fileTypes := []string{"gallery", "thumbnail", "video"}
	// Check and upload files for each type
	for _, fileType := range fileTypes {
		results, err := handleFileUploads(models.DB, c.Request, productID, fileType)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to upload " + fileType + " for product ID " + productID,
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
		uploadedResults = append(uploadedResults, results...)
	}

	// Ensure at least one file or link was processed
	if len(uploadedResults) == 0 {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "No files uploaded for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   "No files were received. Please upload at least one file using the keys: 'gallery', 'thumbnail', or 'video'.",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Clear product-related caches to ensure data consistency
	clearProductCache()

	// Return success response with uploaded file details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product media for product with ID " + productID + " uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload:   uploadedResults,
		Message:   "Product media uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// UpdateProductImageHandler handles the update of product images.
// It deletes existing media and uploads new media.
// This endpoint is restricted to administrators.
//
// @Summary      Update product media
// @Description  Replace existing product media with new uploads
// @Tags         Products
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id  formData  string  true  "Product ID"
// @Param        is_primary  formData  bool    false "Is primary image"
// @Param        video_link  formData  string  false "Video URL"
// @Param        gallery     formData  file    false "Gallery images"
// @Param        thumbnail   formData  file    false "Thumbnail image"
// @Param        video       formData  file    false "Video file"
// @Success      200         {object}  map[string]any
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/product/image [patch]
