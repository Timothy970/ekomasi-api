package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/csv"
	"net/http"
	"time"
)

// BulkUploadProductsHandler (already in your file)
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
	//check that the skus are unique among the products being uploaded themselves
	skuSet := make(map[string]bool)
	for _, product := range products {
		if _, exists := skuSet[product.SKU]; exists {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Duplicate SKU found in CSV: " + product.SKU + "" + err.Error(),
					Code:        http.StatusBadRequest,
				},
				Message:   "Duplicate SKU found in CSV: " + product.SKU,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
		skuSet[product.SKU] = true
	}

	for _, product := range products {
		//check that the sku is unique among the products being uploaded in the database
		err := models.IsSkuThere(product.SKU)
		if err != nil {
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

		err = models.CategoryExists(product.CategoryID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Error adding product: " + err.Error(),
					Code:        http.StatusBadRequest,
				},
				Message:   "Category does not exist: " + product.CategoryID,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
		err = models.IsCategoryParent(product.CategoryID)
		if err != nil {
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
		productResponse, err := models.AddNewProduct(*newProduct, authuser.ID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Error adding product: " + err.Error(),
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
		if err := models.InsertProductImage(productResponse.ID, product.Image, "gallery", true); err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Error adding product image: " + err.Error(),
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
