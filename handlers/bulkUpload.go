package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"time"
)

// BulkUploadProductsHandler (already in your file)
func BulkUploadProductsHandler(w http.ResponseWriter, r *http.Request) {
	authuser, ok := middleware.UserFromContext(r.Context())

	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "CSV file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	products, err := utils.ParseProductsCSV(file)
	if err != nil {
		http.Error(w, "Invalid CSV: "+err.Error(), http.StatusBadRequest)
		return
	}
	//check that the skus are unique among the products being uploaded themselves
	skuSet := make(map[string]bool)
	for _, product := range products {
		if _, exists := skuSet[product.SKU]; exists {
			http.Error(w, "Duplicate SKU found in CSV: "+product.SKU, http.StatusBadRequest)
			return
		}
		skuSet[product.SKU] = true
	}

	for _, product := range products {
		//check that the sku is unique among the products being uploaded in the database
		err := models.IsSkuThere(product.SKU)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = models.CategoryExists(product.CategoryID)
		if err != nil {
			http.Error(w, "Category does not exist: "+product.CategoryID, http.StatusBadRequest)
			return
		}
		err = models.IsCategoryParent(product.CategoryID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
			http.Error(w, "Error adding product: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := models.InsertProductImage(productResponse.ID, product.Image, "gallery", true); err != nil {
			http.Error(w, "Error adding product image: "+err.Error(), http.StatusInternalServerError)
			return
		}

	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Products uploaded successfully",
		"count":   len(products),
		// "products": products,
	})
}

// 🚀 New: Download sample CSV handler
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
