package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/csv"
	"fmt"
	"net/http"
	"time"
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
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusUnauthorized,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
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

	headers := []string{
		"name", "description", "sku", "price", "category_id",
		"stock_quantity", "tag", "low_stock_quantity_warning",
		"sell_when_out_of_stock", "show_stock_quantity",
		"buying_price", "image",
	}

	writer.Write(headers)

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
			"https://example.com/images/soap.jpg",
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
			"https://example.com/images/shampoo.jpg",
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
	if err := models.IsSkuThere(product.SKU); err != nil {
		return err
	}

	if err := models.CategoryExists(product.CategoryID); err != nil {
		return fmt.Errorf("category does not exist: %s", product.CategoryID)
	}

	if err := models.IsCategoryParent(product.CategoryID); err != nil {
		return err
	}

	tag := product.Tag
	showStock := product.ShowStockQuantity
	sellWhenOutOfStock := product.SellWhenOutOfStock
	buyingPrice := product.BuyingPrice
	newProduct := &dtos.CreateProduct{
		Name:          product.Name,
		Description:   product.Description,
		SKU:           product.SKU,
		Price:         product.Price,
		CategoryID:    product.CategoryID,
		StockQuantity: product.StockQuantity,
		Tag:           &tag,
		LowStockAlert: product.LowStockQuantityWarning,
		SellWhenOOS:   &sellWhenOutOfStock,
		ShowStock:     &showStock,
		BuyingPrice:   &buyingPrice,
	}

	productResponse, err := models.AddNewProduct(*newProduct, userID)
	if err != nil {
		return fmt.Errorf("error adding product: %w", err)
	}

	if err := models.InsertProductImage(productResponse.ID, product.Image, "gallery", true); err != nil {
		return fmt.Errorf("error adding product image: %w", err)
	}

	return nil
}
