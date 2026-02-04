package models

import (
	"adenzo_backend/dtos"
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
	log.Printf("expiry date %v and manufacturer date %v for the product", product.ExpiryDate, product.ManufacturingDate)
	_, err := db.Exec(`
		INSERT INTO bulk_products (
			id, name, description, sku, price, sub_category_id,
			stock_quantity, tag, low_stock_quantity_warning, sell_when_out_of_stock, show_stock_quantity,
			buying_price, weight, weight_limit, dimensions, age_range, brand, manufacturer, material, colors, sizes, warranty_period,
			created_by_id, expiry_date, manufacturing_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, productID, product.Name, product.Description, product.SKU, product.Price, product.CategoryID,
		product.StockQuantity, product.Tag, product.LowStockQuantityWarning, product.SellWhenOutOfStock, product.ShowStockQuantity,
		product.BuyingPrice, product.Weight, product.WeightLimit, product.Dimensions, strings.Join(*product.AgeRange, ","), product.Brand, product.Manufacturer, strings.Join(*product.Material, ","),
		strings.Join(*product.Colors, ","), strings.Join(*product.Sizes, ","), product.WarrantyPeriod,
		userID, product.ExpiryDate, product.ManufacturingDate)
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
	query := "SELECT id, name, description, sku, price, sub_category_id, stock_quantity, tag, low_stock_quantity_warning, sell_when_out_of_stock, show_stock_quantity, buying_price, weight, weight_limit, dimensions, age_range, brand, manufacturer, material, colors, sizes, warranty_period, created_by_id, expiry_date, manufacturing_date, created_at FROM bulk_products WHERE 1=1" + conditions + " LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)
	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return products, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var product dtos.BulkUploadProduct
		var material, colors, sizes, age string
		err := rows.Scan(&product.ProductID, &product.Name, &product.Description, &product.SKU, &product.Price, &product.CategoryID,
			&product.StockQuantity, &product.Tag, &product.LowStockQuantityWarning, &product.SellWhenOutOfStock, &product.ShowStockQuantity,
			&product.BuyingPrice, &product.Weight, &product.WeightLimit, &product.Dimensions, &age, &product.Brand, &product.Manufacturer,
			&material, &colors, &sizes, &product.WarrantyPeriod, &product.CreatedByID, &product.ExpiryDate, &product.ManufacturingDate, &product.CreatedAt)
		if err != nil {
			return products, nil, err
		}
		materialSlice := strings.Split(material, ",")
		colorsSlice := strings.Split(colors, ",")
		sizesSlice := strings.Split(sizes, ",")
		ageSlice := strings.Split(age, ",")
		product.Material = &materialSlice
		product.Colors = &colorsSlice
		product.Sizes = &sizesSlice
		product.AgeRange = &ageSlice
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
	var material, colors, sizes, age string
	err = db.QueryRow(`
		SELECT id, name, description, sku, price, sub_category_id,
			stock_quantity, tag, low_stock_quantity_warning, sell_when_out_of_stock, show_stock_quantity,
			buying_price, weight, weight_limit, dimensions, age_range, brand, manufacturer, material, colors, sizes, warranty_period,
			created_by_id, expiry_date, manufacturing_date, created_at
		FROM bulk_products WHERE id = ?
	`, productID).Scan(&product.ProductID, &product.Name, &product.Description, &product.SKU, &product.Price, &product.CategoryID,
		&product.StockQuantity, &product.Tag, &product.LowStockQuantityWarning, &product.SellWhenOutOfStock, &product.ShowStockQuantity,
		&product.BuyingPrice, &product.Weight, &product.WeightLimit, &product.Dimensions, &age, &product.Brand, &product.Manufacturer,
		&material, &colors, &sizes, &product.WarrantyPeriod, &product.CreatedByID, &product.ExpiryDate, &product.ManufacturingDate, &product.CreatedAt)
	if err != nil {
		return dtos.BulkUploadProduct{}, err
	}
	materialSlice := strings.Split(material, ",")
	colorsSlice := strings.Split(colors, ",")
	sizesSlice := strings.Split(sizes, ",")
	ageSlice := strings.Split(age, ",")
	product.Material = &materialSlice
	product.Colors = &colorsSlice
	product.Sizes = &sizesSlice
	product.AgeRange = &ageSlice

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

// helper function to get Manufacturing Warranty ID
func GetManufacturingWarrantyID(db DBExecutor) string {
	var warrantyID string
	err := db.QueryRow(`
		SELECT warranty_type_id FROM warranty_types WHERE name LIKE ?
	`, "Manufacturing Warranty").Scan(&warrantyID)
	if err != nil {
		log.Println("Error fetching Manufacturing Warranty ID:", err)
		return ""
	}
	return warrantyID
}

// helper function to getAge Variant IDs
func GetAgeVariantIDs(db DBExecutor, ageRanges []string) []string {
	var ageIDs []string
	for _, ageRange := range ageRanges {
		var ageID string
		// first check if the age variant exists
		err := db.QueryRow(`
			SELECT variant_id FROM variants WHERE LOWER(name) = ? AND variant_type = LOWER('age')
		`, strings.ToLower(ageRange)).Scan(&ageID)
		if err != nil {
			log.Printf("Error fetching Age Variant ID for %s: %v", ageRange, err)
			// if not exists, create it
			req := dtos.VariantRequest{
				Name:        ageRange,
				VariantType: "Age",
			}
			ageID, err = CreateVariant(db, req)
			if err != nil {
				log.Printf("Error creating Age Variant for %s: %v", ageRange, err)
				continue
			}
		}
		ageIDs = append(ageIDs, ageID)
	}
	return ageIDs
}

// helper function to get Brand ID
func GetBrandID(db DBExecutor, brandName string) string {
	var brandID string
	err := db.QueryRow(`
		SELECT variant_id FROM variants WHERE LOWER(name) = ? AND variant_type = LOWER('brand')
	`, strings.ToLower(brandName)).Scan(&brandID)
	if err != nil {
		log.Printf("Error fetching Brand ID for %s: %v", brandName, err)
		// if not exists, create it
		req := dtos.VariantRequest{
			Name:        brandName,
			VariantType: "Brand",
		}
		brandID, err = CreateVariant(db, req)
		if err != nil {
			log.Printf("Error creating Brand Variant for %s: %v", brandName, err)
			return ""
		}
	}
	return brandID
}

// helper function to get Material Variant IDs
func GetMaterialIDs(db DBExecutor, materials []string) []string {
	var materialIDs []string
	for _, material := range materials {
		var materialID string
		// first check if the material variant exists
		err := db.QueryRow(`
			SELECT variant_id FROM variants WHERE LOWER(name) = ? AND variant_type = LOWER('material')
		`, strings.ToLower(material)).Scan(&materialID)
		if err != nil {
			log.Printf("Error fetching Material Variant ID for %s: %v", material, err)
			// if not exists, create it
			req := dtos.VariantRequest{
				Name:        material,
				VariantType: "Material",
			}
			materialID, err = CreateVariant(db, req)
			if err != nil {
				log.Printf("Error creating Material Variant for %s: %v", material, err)
				continue
			}
		}
		materialIDs = append(materialIDs, materialID)
	}
	return materialIDs
}

// helper function to get Color Variant IDs
func GetColorIDs(db DBExecutor, colors []string) []string {
	var colorIDs []string
	for _, color := range colors {
		var colorID string
		// first check if the color variant exists
		err := db.QueryRow(`
			SELECT variant_id FROM variants WHERE LOWER(name) = ? AND variant_type = LOWER('color')
		`, strings.ToLower(color)).Scan(&colorID)
		if err != nil {
			log.Printf("Error fetching Color Variant ID for %s: %v", color, err)
			// if not exists, create it
			req := dtos.VariantRequest{
				Name:        color,
				VariantType: "Color",
			}
			colorID, err = CreateVariant(db, req)
			if err != nil {
				log.Printf("Error creating Color Variant for %s: %v", color, err)
				continue
			}
		}
		colorIDs = append(colorIDs, colorID)
	}
	return colorIDs
}

// helper function to get Size Variant IDs
func GetSizeIDs(db DBExecutor, sizes []string) []string {
	var sizeIDs []string
	for _, size := range sizes {
		var sizeID string
		// first check if the size variant exists
		err := db.QueryRow(`
			SELECT variant_id FROM variants WHERE LOWER(name) = ? AND variant_type = LOWER('size')
		`, strings.ToLower(size)).Scan(&sizeID)
		if err != nil {
			log.Printf("Error fetching Size Variant ID for %s: %v", size, err)
			// if not exists, create it
			req := dtos.VariantRequest{
				Name:        size,
				VariantType: "Size",
			}
			sizeID, err = CreateVariant(db, req)
			if err != nil {
				log.Printf("Error creating Size Variant for %s: %v", size, err)
				continue
			}
		}
		sizeIDs = append(sizeIDs, sizeID)
	}
	return sizeIDs
}

// helper function to get Default Warranty Type
func GetDefaultWarrantyType(db DBExecutor) string {
	var warrantyTypeID string
	err := db.QueryRow(`
		SELECT warranty_type_id FROM warranty_types WHERE LOWER(name) = LOWER('Manufacturing Warranty') LIMIT 1
	`).Scan(&warrantyTypeID)
	if err != nil {
		log.Println("Error fetching Default Warranty Type ID:", err)
		return ""
	}
	return warrantyTypeID
}

// helper function to delete bulk product by ID
func DeleteBulkProductByID(db DBExecutor, productID string) error {
	_, err := db.Exec(`DELETE FROM bulk_products WHERE id = ?`, productID)
	return err
}
