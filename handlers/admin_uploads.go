package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/csv"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// UploadImageHandler uploads an image to GCS and returns the URL.
// This endpoint is restricted to administrators.
//
// @Summary      Upload an image
// @Description  Upload an image file to Google Cloud Storage
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image File"
// @Success      200    {object}  map[string]string
// @Failure      400    {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/upload [post]
func UploadImageHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}
	url, err := utils.ParseAndUploadFile(c.Request, "image", 10)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to upload image: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Image with URL " + url + " uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload:   url,
		Message:   "Image uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// UploadImageHandler2 uploads an image to GCS and returns the URL.
// This is a duplicate of UploadImageHandler, likely for testing or legacy reasons.
// This endpoint is restricted to administrators.
//
// @Summary      Upload an image (Alternative)
// @Description  Upload an image file to Google Cloud Storage
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image File"
// @Success      200    {object}  map[string]string
// @Failure      400    {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/upload2 [post]
func UploadImageHandler2(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}
	url, err := utils.ParseAndUploadFile(c.Request, "image", 10)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to upload image: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Image with URL " + url + " uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload:   url,
		Message:   "Image uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetAllSubscribersHandler retrieves a paginated list of newsletter subscribers.
//
// @Summary      Get all subscribers
// @Description  Retrieve a paginated list of newsletter subscribers with optional filtering
// @Tags         Admin
// @Produce      json
// @Param        page        query     int     false  "Page number"
// @Param        size        query     int     false  "Page size"
// @Param        q           query     string  false  "Search query"
// @Param        start_date  query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end_date    query     string  false  "End date (YYYY-MM-DD)"
// @Success      200         {object}  map[string]any
// @Failure      401         {object}  dtos.ErrorResponse
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/subscribers [get]
func GetAllSubscribersHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Subscribers", "subscribers.view"); !ok {
		return
	}

	// Parse pagination and filters
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	q := c.Query("q")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	offset := (page - 1) * size

	// Fetch subscribers
	subscribers, meta, err := models.GetAllSubscribers(models.DB, size, offset, q, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching subscribers: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Subscribers",
				Description: "Failed to fetch subscribers",
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

	// Respond with data
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Subscribers",
			Description: "Subscribers fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"subscribers": subscribers,
			"pagination":  meta,
		},
		Message:   "Subscribers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// DownloadSubscribersCSVHandler downloads the subscriber list as a CSV.
//
// @Summary      Download subscribers CSV
// @Description  Download a CSV file containing all subscribers or filtered results
// @Tags         Admin
// @Produce      text/csv
// @Param        q           query     string  false  "Search query"
// @Param        start_date  query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end_date    query     string  false  "End date (YYYY-MM-DD)"
// @Success      200         {file}    file
// @Security     BearerAuth
// @Router       /api/admin/subscribers/csv [get]
func DownloadSubscribersCSVHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Subscribers", "subscribers.view"); !ok {
		return
	}

	// Parse filters (ignore pagination for CSV export)
	q := c.Query("q")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Fetch subscribers - using a large limit for export
	subscribers, _, err := models.GetAllSubscribers(models.DB, 1000000, 0, q, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching subscribers for CSV: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Subscribers",
				Description: "Failed to fetch subscribers for CSV",
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

	// Set headers for CSV download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=subscribers.csv")

	// Initialize CSV writer
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header row
	header := []string{"Email", "Date Subscribed"}
	if err := writer.Write(header); err != nil {
		log.Printf("Error writing CSV header: %v", err)
		return
	}

	// Write data rows
	for _, s := range subscribers {
		row := []string{
			s.Email,
			s.CreatedAt,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Error writing CSV row: %v", err)
			return
		}
	}
}
