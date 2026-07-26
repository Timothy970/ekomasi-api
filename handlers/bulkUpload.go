package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/services"
	"ekomasi_backend/utils"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// BulkUploadProductsHandler handles the bulk upload and validation of products via CSV file.
// Supports query params:
// - validate_only: 'true' to run a dry-run validation without DB writes
// - mode: 'upsert' (default), 'create_only', 'update_stock'
//
// @Summary      Bulk upload and validate products
// @Description  Upload multiple products via CSV with dry-run pre-validation and row-by-row error diagnostics
// @Tags         Products
// @Accept       multipart/form-data
// @Produce      json
// @Param        file           formData  file    true  "CSV File"
// @Param        validate_only  query     bool    false "If true, run dry-run pre-validation without DB writes"
// @Param        mode           query     string  false "Import mode: 'upsert', 'create_only', 'update_stock'"
// @Success      200   {object}  dtos.BulkImportDiagnosticResult
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/bulk-upload [post]
func BulkUploadProductsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}

	validateOnlyStr := c.Query("validate_only")
	validateOnly, _ := strconv.ParseBool(validateOnlyStr)
	mode := c.Query("mode")
	if mode == "" {
		mode = "upsert"
	}

	err := c.Request.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error parsing form: " + err.Error(),
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

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Csv file error: " + err.Error(),
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
	defer file.Close()

	products, err := utils.ParseProductsCSV(file)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Invalid CSV format: " + err.Error(),
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

	tenantID := fmt.Sprintf("%d", middleware.TenantIDFromContext(c.Request.Context()))
	diagnostic, err := services.ValidateAndProcessBulkImport(models.DB, tenantID, products, mode, validateOnly, authuser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Bulk import execution failed: " + err.Error(),
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

	msg := "Bulk product import processed successfully"
	if validateOnly {
		msg = "Bulk product pre-validation complete (Dry-Run)"
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: msg,
			Code:        http.StatusOK,
		},
		Payload:   diagnostic,
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// DownloadSampleCSVHandler generates and serves a sample CSV file for bulk product uploads.
//
// @Summary      Download sample CSV
// @Description  Download a sample CSV file template for bulk product upload
// @Tags         Products
// @Produce      text/csv
// @Success      200  {file}  file
// @Failure      500  {string} string "Error generating sample CSV"
// @Router       /api/products/sample-csv [get]
func DownloadSampleCSVHandler(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=sample_products.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	headers := utils.BulkUploadHeaders
	var headerNames []string
	for _, header := range headers {
		headerNames = append(headerNames, header.Name)
	}
	writer.Write(headerNames)

	// Sample products: 1 simple product & 1 parent-child variant combination product
	sampleData := [][]string{
		{
			"Organic Soap",
			"Natural handmade soap with essential oils",
			"SOAP001",
			"3.50",
			"Beauty",
			"100",
			"",
			"5",
			"123355677",
			"2.00",
			"12",
			"5",
			"3x3x1",
			"Nature's Best",
			"Nature's Best Co.",
			"12",
			"",
			"",
			"0.00",
			"",
			"",
			"",
		},
		{
			"Cotton Crewneck Tee",
			"Premium heavyweight organic cotton t-shirt",
			"TEE001",
			"25.00",
			"Fashion",
			"50",
			"apparel",
			"5",
			"987654321",
			"12.50",
			"0.35",
			"1",
			"10x12x1",
			"UrbanWear",
			"Urban Apparel Ltd",
			"12",
			"Blue / Medium",
			"TEE001-BL-M",
			"0.00",
			"Color: Blue",
			"Size: Medium",
			"",
		},
		{
			"Cotton Crewneck Tee",
			"Premium heavyweight organic cotton t-shirt",
			"TEE001",
			"25.00",
			"Fashion",
			"30",
			"apparel",
			"5",
			"987654321",
			"12.50",
			"0.35",
			"1",
			"10x12x1",
			"UrbanWear",
			"Urban Apparel Ltd",
			"12",
			"Blue / XL",
			"TEE001-BL-XL",
			"5.00",
			"Color: Blue",
			"Size: XL",
			"",
		},
	}

	for _, record := range sampleData {
		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		c.String(http.StatusInternalServerError, "Error generating sample CSV")
		return
	}

	c.Header("X-Generated-At", time.Now().Format(time.RFC3339))
}

// validateUniqueSKUs checks if all SKUs in the products slice are unique
func validateUniqueSKUs(products []dtos.BulkUploadProduct) error {
	skuSet := make(map[string]bool)
	for _, product := range products {
		if _, exists := skuSet[product.SKU]; exists {
			return fmt.Errorf("duplicate SKU found in CSV: %s", product.SKU)
		}
		skuSet[product.SKU] = true
	}
	return nil
}

// validateAndCreateProduct validates a product and creates it in the database
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
// @Success      200   {object}  map[string]interface{}
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
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/bulk-upload/publish [post]
func PublishBulkUploadedProductsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for media uploads)
	authuser, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Start transaction for atomic operations
	tx, err := models.DB.Begin()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to start transaction: " + err.Error(),
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
	defer tx.Rollback() // Rollback if not committed

	// req, ok := DecodeRequestBody[dtos.PublishBulkProduct](c, requestSummary, start)
	// if !ok {
	// 	return
	// }
	// if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
	// 	return
	// }
	productID := c.Request.FormValue("product_id")
	videoLink := c.Request.FormValue("video_link")
	productDetails := c.Request.Form["details"]
	bulkProduct, err := models.GetBulkProductByID(tx, productID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetching bulk product: " + err.Error(),
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
	createRequest := dtos.CreateProduct{
		Name:          bulkProduct.Name,
		Description:   bulkProduct.Description,
		SKU:           bulkProduct.SKU,
		Price:         &bulkProduct.Price,
		CategoryID:    bulkProduct.CategoryID,
		StockQuantity: bulkProduct.StockQuantity,
		SearchVector:  bulkProduct.Name,
		Tag:           bulkProduct.Tag,
		LowStockAlert: bulkProduct.LowStockQuantityWarning,
		BuyingPrice:   &bulkProduct.BuyingPrice,
		Details:       productDetails,
	}
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	product, err := models.AddNewProduct(tx, createRequest, authuser.ID, tenantID)
	if err != nil {
		log.Printf("Error for adding new product %s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add new product",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	warrantyType, err := models.GetDefaultWarrantyType(tx)
	if err != nil {
		log.Printf("Error fetching default warranty type: %s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch default warranty type",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	//handle product specifications
	specificationsRequest := dtos.ProductSpecification{
		ProductID:      product.ID,
		Brand:          models.GetVariantID(tx, *bulkProduct.Brand, "brand"), //should be ID
		CategoryID:     bulkProduct.CategoryID,
		Dimensions:     *bulkProduct.Dimensions,
		Manufacturer:   models.GetVariantID(tx, *bulkProduct.Manufacturer, "manufacturer"),
		WarrantyPeriod: *bulkProduct.WarrantyPeriod,
		Weight:         *bulkProduct.Weight,
		WeightLimit:    *bulkProduct.WeightLimit,
		WarrantyType:   warrantyType,
	}
	state := "add"
	err = handleProductSpecs(tx, specificationsRequest, state)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product specifications: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//handle products variants
	err = handleProductsVariants(tx, specificationsRequest, state)

	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product variants: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//handle product warranty
	err = handleProductsWarranty(tx, specificationsRequest)

	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product warranty: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//handle product images
	//first video link if any
	if videoLink != "" {
		if err := models.InsertProductImage(tx, product.ID, videoLink, "video", false); err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to insert video link for product ID " + product.ID,
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
	var uploadedResults []map[string]string
	fileTypes := []string{"gallery", "thumbnail", "video"}
	// Check and upload files for each type
	for _, fileType := range fileTypes {
		results, err := handleFileUploads(tx, c.Request, product.ID, fileType)
		if err != nil {
			log.Printf("Error::::%s adding image to product:::::%s", err.Error(), product.ID)
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to upload " + fileType + " for product ID " + product.ID,
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
	//delete bulk product after publishing
	err = models.DeleteBulkProductByID(tx, bulkProduct.ProductID)
	if err != nil {
		log.Printf("Error deleting bulk product ID %s: %s", bulkProduct.ProductID, err.Error())
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete bulk product: " + err.Error(),
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

	// Commit transaction - all operations succeeded
	if err := tx.Commit(); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to commit transaction: " + err.Error(),
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

	// Invalidate cache
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	_ = utils.DeleteCache("expensiveandcheapproducts")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   product,
		Message:   "Product published successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteBulkUploadProductsHandler handles the deletion of bulk uploaded products via the bulk product ID.
// This endpoint is restricted to authenticated users.
//
// @Summary      Delete bulk uploaded products
// @Description  Delete multiple products using a bulk product ID
// @Tags         Products
// @Produce      json
// @Param        bulk_product_id  query  string  true  "Bulk Product ID"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/bulk-upload/{bulk_product_id} [delete]
func DeleteBulkUploadProductsHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Get bulk product ID from URL path
	bulkProductID := c.Param("bulk_product_id")
	err := models.DeleteBulkProductByID(models.DB, bulkProductID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error deleting bulk product: " + err.Error(),
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
			Description: "Product deleted successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"message": "Product deleted successfully",
			"count":   nil,
		},
		Message:   "Product deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})

}
