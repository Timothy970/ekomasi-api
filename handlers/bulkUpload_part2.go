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

func validateAndCreateProduct(product dtos.BulkUploadProduct, userID string) error {
	if err := models.IsSkuThere(models.DB, product.SKU); err != nil {
		return err
	}

	if err := models.CategoryExists(models.DB, product.CategoryID); err != nil {
		return fmt.Errorf("sub category does not exist: %s", product.CategoryID)
	}

	if err := models.IsCategoryParent(models.DB, product.CategoryID); err != nil {
		return err
	}

	err := models.AddNewBulkProduct(models.DB, product, userID)
	if err != nil {
		return fmt.Errorf("error adding product: %w", err)
	}
	return nil
}

// convertDateFormat converts date from DD/MM/YYYY to YYYY/MM/DD if needed
func convertDateFormat(date string) string {
	// Try to parse as DD/MM/YYYY
	if len(date) == 10 && date[2] == '/' && date[5] == '/' {
		// Split the date
		day := date[0:2]
		month := date[3:5]
		year := date[6:10]
		// Return in YYYY/MM/DD format
		return year + "/" + month + "/" + day
	}
	// If already in correct format or different format, return as is
	return date
}

// Function to list bulk upload products - for admin use only
//
// @Summary      Get Bulk upload products
// @Description  List all bulk upload products with pagination
// @Tags         Products
// @Produce      json
// @Param        category_id   query     string  false  "Filter by Category ID"
// @Param        page          query     int     false  "Page number"
// @Param        limit         query     int     false  "Items per page"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/bulk-upload [get]
func GetBulkUploadProductsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	q := c.Query("q")
	products, pagination, err := models.GetBulkUploadProducts(models.DB, startDate, endDate, q, page, limit)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetching bulk upload products: " + err.Error(),
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Bulk upload products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"products":   products,
			"pagination": pagination,
		},
		Message:   "Bulk upload products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// Function to publish bulk uploaded products to main products table after getting their images and other details
//
// @Summary      Publish Bulk uploaded products
// @Description  Publish bulk uploaded products to main products table
// @Tags         Products
// @Produce      json
// @Param        product_ids  body      []string  true  "List of Bulk Uploaded Product IDs to publish"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/bulk-upload/publish [post]
