package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"log"
	"strings"

	"github.com/teris-io/shortid"
)

//func AddNewBulkProduct adds a new bulk product to the database.

// Parameters:
// - product: A dtos.BulkUploadProduct struct containing product details.
// - userID: A string representing the ID of the user creating the product.

// Returns:
// - error: An error object if the operation fails, otherwise nil.
func AddNewBulkProduct(db DBExecutor, product dtos.BulkUploadProduct, userID string) error {
	productID, _ := shortid.Generate()
	_, err := db.Exec(`
		INSERT INTO bulk_products (
			id, name, description, sku, price, sub_category_id,
			stock_quantity, tag, low_stock_quantity_warning, barcode,
			buying_price, weight, weight_limit, dimensions, brand, manufacturer, warranty_period,
			created_by_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, productID, product.Name, product.Description, product.SKU, product.Price, product.CategoryID,
		product.StockQuantity, product.Tag, product.LowStockQuantityWarning, product.Barcode,
		product.BuyingPrice, product.Weight, product.WeightLimit, product.Dimensions, product.Brand, product.Manufacturer,
		product.WarrantyPeriod,
		userID)
	return err
}

// Function to retrieve bulk uploaded products with pagination and optional category and name filtering
// Parameters:
// - categoryID: Optional string to filter products by category ID.
// - name: Optional string to filter products by name (partial match).
// - page: Optional integer for pagination (page number).
// - limit: Optional integer for pagination (items per page).
// Returns:
// - ([]dtos.BulkUploadProduct, Pagination, error): A slice of BulkUploadProduct DTOs and an error if the operation fails.
func GetBulkUploadProducts(db DBExecutor, startDate, endDate, name string, page, limit int) ([]dtos.BulkUploadProduct, *dtos.PaginationMeta, error) {
	var products []dtos.BulkUploadProduct
	offset := (page - 1) * limit
	var count int

	// Build filter conditions and args
	var conditions string
	var args []interface{}

	if startDate != "" && endDate != "" {
		conditions += " AND created_at BETWEEN ? AND ?"
		args = append(args, StringToTime(startDate), StringToTime(endDate))
	}
	if name != "" {
		conditions += " AND name LIKE ?"
		args = append(args, "%"+name+"%")
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM bulk_products WHERE 1=1" + conditions
	err := db.QueryRow(countQuery, args...).Scan(&count)
	if err != nil {
		return products, nil, err
	}

	// Select query
	query := "SELECT id, name, description, sku, price, sub_category_id, stock_quantity, tag, low_stock_quantity_warning, barcode, buying_price, weight, weight_limit, dimensions, brand, manufacturer, warranty_period, created_by_id, created_at FROM bulk_products WHERE 1=1" + conditions + " LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)
	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return products, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var product dtos.BulkUploadProduct
		err := rows.Scan(&product.ProductID, &product.Name, &product.Description, &product.SKU, &product.Price, &product.CategoryID,
			&product.StockQuantity, &product.Tag, &product.LowStockQuantityWarning, &product.Barcode,
			&product.BuyingPrice, &product.Weight, &product.WeightLimit, &product.Dimensions, &product.Brand, &product.Manufacturer,
			&product.WarrantyPeriod, &product.CreatedByID, &product.CreatedAt)
		if err != nil {
			return products, nil, err
		}
		products = append(products, product)
	}
	pagination := &dtos.PaginationMeta{
		TotalItems: count,
		Page:       page,
		Size:       limit,
		TotalPages: (count + limit - 1) / limit,
		HasPrev:    page > 1,
		HasNext:    page*limit < count,
	}
	return products, pagination, nil
}

// Function to get bulk product by ID
// Parameters:
// - productID: string representing the ID of the bulk product to retrieve.
// Returns:
// - (dtos.BulkUploadProduct, error): A BulkUploadProduct DTO and an error if the operation fails.
func GetBulkProductByID(db DBExecutor, productID string) (dtos.BulkUploadProduct, error) {
	err := isBulkProductThere(db, productID)
	if err != nil {
		return dtos.BulkUploadProduct{}, err
	}
	var product dtos.BulkUploadProduct
	err = db.QueryRow(`
		SELECT id, name, description, sku, price, sub_category_id,
			stock_quantity, tag, low_stock_quantity_warning, barcode,
			buying_price, weight, weight_limit, dimensions, brand, manufacturer, warranty_period,
			created_by_id, created_at
		FROM bulk_products WHERE id = ?
	`, productID).Scan(&product.ProductID, &product.Name, &product.Description, &product.SKU, &product.Price, &product.CategoryID,
		&product.StockQuantity, &product.Tag, &product.LowStockQuantityWarning, &product.Barcode,
		&product.BuyingPrice, &product.Weight, &product.WeightLimit, &product.Dimensions, &product.Brand, &product.Manufacturer, &product.WarrantyPeriod, &product.CreatedByID, &product.CreatedAt)
	if err != nil {
		return dtos.BulkUploadProduct{}, err
	}

	return product, nil
}

// helper function to check if bulk product exists
func isBulkProductThere(db DBExecutor, productID string) error {
	exists, err := RecordExists(db, "bulk_products", "id = ?", productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Bulk product with ID %s does not exist", productID)
	}
	return nil
}

// helper function to get Brand ID
func GetVariantID(db DBExecutor, variantName, variantType string) string {
	var variantID string
	err := db.QueryRow(`
		SELECT variant_id FROM variants WHERE LOWER(name) = ? AND variant_type = LOWER(?)
	`, strings.ToLower(variantName), strings.ToLower(variantType)).Scan(&variantID)
	if err != nil {
		log.Printf("Error fetching Brand ID for %s: %v", variantName, err)
		// if not exists, create it
		req := dtos.VariantRequest{
			Name:        variantName,
			VariantType: variantType,
		}
		variantID, err = CreateVariant(db, req)
		if err != nil {
			log.Printf("Error creating Brand Variant for %s: %v", variantName, err)
			return ""
		}
	}
	log.Printf("Brand ID for %s: %s", variantName, variantID)
	return variantID
}

// helper function to get Default Warranty Type
func GetDefaultWarrantyType(db DBExecutor) (string, error) {
	var warrantyTypeID string
	err := db.QueryRow(`
		SELECT warranty_type_id FROM warranty_types LIMIT 1
	`).Scan(&warrantyTypeID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("No warranty types found in the database.")
			return "", fmt.Errorf("please create a warranty type before adding bulk products")
		}
		log.Println("Error fetching Default Warranty Type ID:", err)
		return "", err
	}
	return warrantyTypeID, nil
}

// helper function to delete bulk product by ID
func DeleteBulkProductByID(db DBExecutor, productID string) error {
	_, err := db.Exec(`DELETE FROM bulk_products WHERE id = ?`, productID)
	return err
}
