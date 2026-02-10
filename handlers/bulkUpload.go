package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// BulkUploadProductsHandler handles the bulk upload of products via CSV file.
// This endpoint is restricted to authenticated users.
//
// @Summary      Bulk upload products
// @Description  Upload multiple products using a CSV file
// @Tags         Products
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "CSV File"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/bulk-upload [post]
func BulkUploadProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for media uploads)
	authuser, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error parsing form: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Csv file error: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	defer file.Close()

	products, err := utils.ParseProductsCSV(file)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Invalid CSV: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	if err := validateUniqueSKUs(products); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	for _, product := range products {
		if err := validateAndCreateProduct(product, authuser.ID); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: err.Error(),
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Products uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"message": "Products uploaded successfully",
			"count":   len(products),
		},
		Message:   "Products upload successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func DownloadSampleCSVHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=sample_products.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	headers := utils.BulkUploadHeaders
	var headerNames []string
	for _, header := range headers {
		headerNames = append(headerNames, header.Name)
	}
	writer.Write(headerNames)

	// 2 sample product rows
	sampleData := [][]string{
		{
			"Organic Soap",
			"Natural handmade soap with essential oils",
			"SOAP001",
			"3.50",
			"cat-001",
			"100",
			"",
			"5",
			"false",
			"true",
			"2.00",
			"12",
			"5",
			"3x3x1",
			"1-2 years, 12-60 months",
			"Nature's Best",
			"Nature's Best Co.",
			"Organic",
			"Green, White",
			"Small, Medium",
			"12",
			"2025-12-31",
			"2024-01-01",
		},
		{
			"Herbal Shampoo",
			"Moisturizing herbal shampoo for daily use",
			"SHAMP001",
			"6.99",
			"cat-002",
			"75",
			"beauty",
			"10",
			"true",
			"true",
			"4.50",
			"6",
			"2",
			"6x6x2",
			"18-45 years, 1-12 months",
			"Nature's Best",
			"Nature's Best Co.",
			"Organic",
			"Green, White",
			"Small, Medium",
			"12",
			"2025-11-30",
			"2024-02-01",
		},
	}

	for _, record := range sampleData {
		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		http.Error(w, "Error generating sample CSV", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Generated-At", time.Now().Format(time.RFC3339))
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

	// Convert date format from DD/MM/YYYY to YYYY/MM/DD if needed
	if product.ExpiryDate != nil {
		convertedDate := convertDateFormat(*product.ExpiryDate)
		product.ExpiryDate = &convertedDate
	}
	if product.ManufacturingDate != nil {
		convertedDate := convertDateFormat(*product.ManufacturingDate)
		product.ManufacturingDate = &convertedDate
	}

	//check if expiry date is after manufacturing date
	if product.ExpiryDate != nil && product.ManufacturingDate != nil {
		if models.StringToTime(*product.ExpiryDate).Before(models.StringToTime(*product.ManufacturingDate)) {
			return fmt.Errorf("expiry date cannot be before manufacturing date for product: %s", product.Name)
		}
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
func GetBulkUploadProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Parse pagination parameters from query string
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	q := r.URL.Query().Get("q")
	products, pagination, err := models.GetBulkUploadProducts(models.DB, startDate, endDate, q, page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetching bulk upload products: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
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
func PublishBulkUploadedProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for media uploads)
	authuser, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Start transaction for atomic operations
	tx, err := models.DB.Begin()
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to start transaction: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	defer tx.Rollback() // Rollback if not committed

	// req, ok := DecodeRequestBody[dtos.PublishBulkProduct](r, w, requestSummary, start)
	// if !ok {
	// 	return
	// }
	// if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
	// 	return
	// }
	productID := r.FormValue("product_id")
	videoLink := r.FormValue("video_link")
	productDetails := r.Form["details"]
	bulkProduct, err := models.GetBulkProductByID(tx, productID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetching bulk product: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	createRequest := dtos.CreateProduct{
		Name:          bulkProduct.Name,
		Description:   bulkProduct.Description,
		SKU:           bulkProduct.SKU,
		Price:         bulkProduct.Price,
		CategoryID:    bulkProduct.CategoryID,
		StockQuantity: bulkProduct.StockQuantity,
		SearchVector:  bulkProduct.Name,
		Tag:           bulkProduct.Tag,
		LowStockAlert: bulkProduct.LowStockQuantityWarning,
		SellWhenOOS:   &bulkProduct.SellWhenOutOfStock,
		ShowStock:     &bulkProduct.ShowStockQuantity,
		BuyingPrice:   &bulkProduct.BuyingPrice,
		Details:       productDetails,
	}
	product, err := models.AddNewProduct(tx, createRequest, authuser.ID)
	if err != nil {
		log.Printf("Error for adding new product %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add new product",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//handle product specifications
	specificationsRequest := dtos.ProductSpecification{
		ProductID:        product.ID,
		Age:              models.GetAgeVariantIDs(tx, *bulkProduct.AgeRange), //should be IDS
		Brand:            models.GetBrandID(tx, *bulkProduct.Brand),          //should be ID
		CategoryID:       bulkProduct.CategoryID,
		Color:            models.GetColorIDs(tx, *bulkProduct.Colors), //should be IDS
		Dimensions:       *bulkProduct.Dimensions,
		ExpiryDate:       bulkProduct.ExpiryDate,
		ManufacturerDate: bulkProduct.ManufacturingDate,
		Manufacturer:     *bulkProduct.Manufacturer,
		Material:         models.GetMaterialIDs(tx, *bulkProduct.Material), //should be IDs
		Size:             models.GetSizeIDs(tx, *bulkProduct.Sizes),        //should be IDs
		WarrantyPeriod:   *bulkProduct.WarrantyPeriod,
		Weight:           *bulkProduct.Weight,
		WeightLimit:      *bulkProduct.WeightLimit,
		WarrantyType:     models.GetDefaultWarrantyType(tx),
	}
	state := "add"
	err = handleProductSpecs(tx, specificationsRequest, state)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product specifications: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//handle products variants
	err = handleProductsVariants(tx, specificationsRequest, state)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product variants: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//handle product warranty
	specificationsRequest.WarrantyType = models.GetManufacturingWarrantyID(tx)
	err = handleProductsWarranty(tx, specificationsRequest)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product warranty: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//handle product images
	//first video link if any
	if videoLink != "" {
		if err := models.InsertProductImage(tx, product.ID, videoLink, "video", false); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to insert video link for product ID " + product.ID,
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
	}
	var uploadedResults []map[string]string
	isPrimary := true
	fileTypes := []string{"gallery", "thumbnail", "video"}
	// Check and upload files for each type
	for _, fileType := range fileTypes {
		results, err := handleFileUploads(tx, r, product.ID, fileType, isPrimary)
		if err != nil {
			log.Printf("Error::::%s adding image to product:::::%s", err.Error(), product.ID)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to upload " + fileType + " for product ID " + product.ID,
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
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
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete bulk product: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	// Commit transaction - all operations succeeded
	if err := tx.Commit(); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to commit transaction: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   product,
		Message:   "Product published successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func DeleteBulkUploadProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Get bulk product ID from URL path
	bulkProductID := mux.Vars(r)["product_id"]
	err := models.DeleteBulkProductByID(models.DB, bulkProductID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error deleting bulk product: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
		RawBody:   requestSummary,
	})

}
