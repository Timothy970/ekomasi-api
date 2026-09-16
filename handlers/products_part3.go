package handlers

import (
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

func UpdateProductImageHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure admin access
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.update")
	if !ok {
		return
	}

	// Parse upload request parameters
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

	// Fetch existing media to keep URLs that aren't being updated
	existingMedia, err := models.GetProductImages(models.DB, productID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch existing video links for product ID " + productID,
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
	var uploadedResults []map[string]string
	mediaToDelete := make(map[string]bool) // Track which media IDs to delete

	// Handle optional video link
	if videoLink != "" {
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
		// Mark existing video links for deletion
		for _, media := range existingMedia {
			if media.Type == "video" {
				mediaToDelete[media.ImageID] = true
			}
		}
	}

	fileTypes := []string{"gallery", "thumbnail", "video"}
	// Check and upload files for each type
	for _, fileType := range fileTypes {
		// Check if actual files were uploaded
		if c.Request.MultipartForm != nil && len(c.Request.MultipartForm.File[fileType]) > 0 {
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
			// Mark existing media of this type for deletion since new files uploaded
		} else if len(c.Request.Form[fileType]) > 0 {
			// URL strings were passed as form values (not files) - keep existing media
			log.Printf("URL values passed for %s, keeping existing media", fileType)
			for _, media := range existingMedia {
				if media.Type == fileType {
					uploadedResults = append(uploadedResults, map[string]string{
						"type": fileType,
						"url":  media.URL,
					})
				}
			}
		} else {
			// No files and no form values - keep existing media for this type
			for _, media := range existingMedia {
				if media.Type == fileType {
					uploadedResults = append(uploadedResults, map[string]string{
						"type": fileType,
						"url":  media.URL,
					})
				}
			}
		}
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

	clearProductCache()
	// Delete only the media marked for deletion
	for _, media := range existingMedia {
		if mediaToDelete[media.ImageID] {
			err := models.DeleteProductImage(models.DB, media.ImageID)
			if err != nil {
				utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Products",
						Description: "Failed to delete existing media for product ID " + productID,
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
		}
	}

	// Respond with success
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

// handleFileUploads processes file uploads for a specific file type (gallery, thumbnail, video).
// It uploads files to Google Cloud Storage and inserts records into the database.
// Returns a list of uploaded file metadata or an error if any upload fails.
// handleFileUploads processes file uploads for a specific file type (gallery, thumbnail, video).
// It uploads files to Google Cloud Storage and inserts records into the database.
// For gallery and thumbnail, it checks for is_primary flags per file using form values like gallery_is_primary[0], thumbnail_is_primary[0], etc.
func handleFileUploads(db models.DBExecutor, r *http.Request, productID, fileType string) ([]map[string]string, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	formFiles := r.MultipartForm.File[fileType]
	if len(formFiles) == 0 {
		return nil, nil
	}

	var uploaded []map[string]string

	for idx, fileHeader := range formFiles {
		// Determine isPrimary for gallery and thumbnail types
		isPrimary := false
		if fileType == "gallery" || fileType == "thumbnail" {
			key := fileType + "_is_primary[" + strconv.Itoa(idx) + "]"
			val := r.FormValue(key)
			if val == "true" || val == "1" {
				isPrimary = true
			}
		}

		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{fileHeader})
		if err != nil {
			log.Printf("error uploading %s: %v", fileType, err)
			// log.Printf("using hardcoded url")
		}
		// url := "https://cdn.pixabay.com/photo/2018/05/18/15/30/web-design-3411373_1280.jpg"

		if err := models.InsertProductImage(db, productID, url, fileType, isPrimary); err != nil {
			return nil, fmt.Errorf("failed to insert %s into DB: %w", fileType, err)
		}

		uploaded = append(uploaded, map[string]string{
			"type": fileType,
			"url":  url,
		})
	}
	return uploaded, nil
}

// clearProductCache invalidates all product-related cache entries.
// This ensures data consistency after product modifications by clearing
// cached product lists, pagination data, and category-product associations.
func clearProductCache() {
	// Define all product-related cache key prefixes
	prefixes := []string{
		"products_page_",                 // Product listing pages
		"pagination_page_",               // Pagination metadata
		"categories_products",            // Category-product associations
		"categories_products_pagination", // Category pagination
		"expensiveandcheapproducts",      // Price extremes cache
	}

	// Clear cache for each prefix
	for _, prefix := range prefixes {
		_ = utils.DeleteCacheByPrefix(prefix)
	}
}

// parseUploadRequest extracts and validates product media upload parameters from the request.
// It returns the product ID, whether the media is primary, video link (if provided), and any validation errors.
func parseUploadRequest(r *http.Request) (string, bool, string, error) {
	// Extract product ID from form (required field)
	productID := r.FormValue("product_id")
	if productID == "" {
		return "", false, "", fmt.Errorf("product_id is required")
	}
	// Extract optional video link
	videoLink := r.FormValue("video_link")

	return productID, false, videoLink, nil
}

// GetRelatedProductsHandler retrieves related products based on category and other criteria.
// It utilizes Redis caching for performance.
//
// @Summary      Get related products
// @Description  Retrieve a list of related products for a specific product
// @Tags         Products
// @Produce      json
// @Param        product_id  query     string  true   "Product ID"
// @Param        page        query     int     false  "Page number"
// @Param        size        query     int     false  "Page size"
// @Success      200         {object}  map[string]any
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      404         {object}  dtos.ErrorResponse
// @Router       /api/products/related [get]
