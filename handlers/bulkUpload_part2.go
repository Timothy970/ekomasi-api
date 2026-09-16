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
