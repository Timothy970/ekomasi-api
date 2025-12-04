package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

var nobundle = "bundle not found"
var fetchbundle = "bundle_id = ?"
var limtOffset = " LIMIT ? OFFSET ?"
var lowerCname = " AND LOWER(c.name) LIKE ?"
var lowerPname = " AND LOWER(p.name) LIKE ?"
var lowerVariant = "(LOWER(v.variant_type) = ?)"
var lowerVariantTypeName = "(LOWER(v.variant_type) = ? AND LOWER(v.name) = ?)"
var whereBundleID = " WHERE bundle_id = ?"

func GetAllProducts(categoryFilter, productFilter, categoryID string, page, limit int) ([]dtos.Product, *dtos.PaginationMeta, error) {
	if categoryID != "" {
		if err := CategoryExists(categoryID); err != nil {
			return nil, nil, err
		}
	}
	query, args := buildProductQuery(categoryFilter, productFilter, categoryID, page, limit)
	countQuery, countArgs := buildCountQuery(categoryFilter, productFilter, categoryID)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var totalItems int64
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	var products []dtos.Product
	for rows.Next() {
		var product dtos.Product
		var tag sql.NullString

		err := rows.Scan(
			&product.CategoryID, &product.CategoryName, &product.ID, &product.Name, &product.Description, &product.SKU, &product.Price, &product.CategoryID, &product.StockQuantity, &product.SearchVector, &product.CreatedAt, &product.LastUpdated, &tag, &product.Discount, &product.DiscountType, &product.Weight, &product.Dimensions, &product.Manufacturer, &product.WeightLimit,
		)
		if err != nil {
			return nil, nil, err
		}
		if tag.Valid {
			product.Tag = &tag.String
		}
		product.Images, err = fetchProductImages(product.ID)
		if err != nil {
			return nil, nil, err
		}
		warranty, err := FetchProductWarranties(product.ID)
		if err != nil {
			return nil, nil, err
		}
		product.Warranty = &warranty
		features, err := fetchProductFeatures(product.ID)
		if err != nil {
			return nil, nil, err
		}
		product.Features = features
		variants, err := getProductVariants(product.ID)
		if err != nil {
			return nil, nil, err
		}
		product.ProductVariants = variants
		tax, err := fetchProductTax(product.ID)
		if err != nil {
			return nil, nil, err
		}
		product.Tax = &tax
		products = append(products, product)
	}

	pagination := calculatePagination(page, limit, totalItems)
	return products, &pagination, nil
}

// Update the queries to fetch the complete hierarchy including parents
func buildCountQuery(categoryFilter, productFilter, categoryID string) (string, []interface{}) {
	query := `
        SELECT COUNT(DISTINCT p.product_id)
        FROM products p
        JOIN categories c ON p.category_id = c.category_id
        WHERE p.product_type = 'single'`
	var args []interface{}

	if categoryFilter != "" {
		query += lowerCname
		args = append(args, "%"+strings.ToLower(categoryFilter)+"%")
	}
	if productFilter != "" {
		query += lowerPname
		args = append(args, "%"+strings.ToLower(productFilter)+"%")
	}
	if categoryID != "" {
		query = `
        WITH RECURSIVE ancestors AS (
            SELECT category_id, parent_category_id
            FROM categories
            WHERE category_id = ?
            UNION ALL
            SELECT c.category_id, c.parent_category_id
            FROM categories c
            INNER JOIN ancestors a ON c.category_id = a.parent_category_id
        ),
        descendants AS (
            SELECT category_id, parent_category_id
            FROM categories
            WHERE category_id = ?
            UNION ALL
            SELECT c.category_id, c.parent_category_id
            FROM categories c
            INNER JOIN descendants d ON c.parent_category_id = d.category_id
        )
        SELECT COUNT(DISTINCT p.product_id)
        FROM products p
        JOIN categories c ON p.category_id = c.category_id
        WHERE p.product_type = 'single'
          AND c.category_id IN (
              SELECT category_id FROM ancestors
              UNION
              SELECT category_id FROM descendants
          )`
		args = append(args, categoryID, categoryID)
	}

	return query, args
}

func buildProductQuery(categoryFilter, productFilter, categoryID string, page, limit int) (string, []interface{}) {
	query := `
        SELECT 
            c.category_id, c.name,
            p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
            p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
        FROM categories c
        JOIN products p ON c.category_id = p.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
        WHERE p.product_type = 'single'`
	var args []interface{}

	if categoryFilter != "" {
		query += lowerCname
		args = append(args, "%"+strings.ToLower(categoryFilter)+"%")
	}
	if productFilter != "" {
		query += lowerPname
		args = append(args, "%"+strings.ToLower(productFilter)+"%")
	}
	if categoryID != "" {
		query += " AND (c.category_id = ? OR c.parent_category_id = ? OR c.category_id IN (SELECT parent_category_id FROM categories WHERE category_id = ? AND parent_category_id IS NOT NULL))"
		args = append(args, categoryID, categoryID, categoryID)
	}

	query += " ORDER BY c.parent_category_id IS NULL DESC, c.parent_category_id, c.category_id"

	if limit > 0 {
		offset := (page - 1) * limit
		query += limtOffset
		args = append(args, limit, offset)
	}

	return query, args
}

func calculatePagination(page, limit int, totalItems int64) dtos.PaginationMeta {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))

	return dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: int(totalItems),
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
}

// GetProductVariants fetches all variants for a given productID
func getProductVariants(productID string) ([]dtos.ProductVariants, error) {
	query := `
		SELECT 
			v.variant_id,
			v.variant_type,
			v.name,
			v.hex_code,
			pv.additional_price,
			pv.stock_quantity
		FROM product_variants pv
		INNER JOIN variants v ON pv.variant_id = v.variant_id
		WHERE pv.product_id = ?
	`

	rows, err := DB.Query(query, productID)
	if err != nil {
		return nil, fmt.Errorf("querying product variants: %w", err)
	}
	defer rows.Close()

	var variants []dtos.ProductVariants
	for rows.Next() {
		var pv dtos.ProductVariants
		if err := rows.Scan(
			&pv.VariantID,
			&pv.VariantType,
			&pv.Name,
			&pv.HexCode,
			&pv.AdditionalPrice,
			&pv.StockQuantity,
		); err != nil {
			return nil, fmt.Errorf("scanning product variant: %w", err)
		}
		variants = append(variants, pv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating product variants: %w", err)
	}

	return variants, nil
}

// func buildCategoryHierarchy(categoryMap map[string]*dtos.CategoryWithProducts) []dtos.CategoryWithProducts {
// 	var topLevel []dtos.CategoryWithProducts
// 	for _, cat := range categoryMap {
// 		if cat.ParentCategoryID != nil {
// 			if parent, ok := categoryMap[*cat.ParentCategoryID]; ok {
// 				parent.Subcategories = append(parent.Subcategories, *cat)
// 			}
// 		} else {
// 			topLevel = append(topLevel, *cat)
// 		}
// 	}
// 	return topLevel
// }

func GetProductByID(productID string) (*dtos.Product, error) {
	err := IsProductThere(productID)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, c.name, p.tag, p.details, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE p.product_id = ?
	`

	var (
		p           dtos.Product
		detailsData []byte
	)
	err = DB.QueryRow(query, productID).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&p.CategoryName, &p.Tag, &detailsData, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
	)
	if err != nil {
		return nil, err
	}
	if len(detailsData) > 0 {
		err := json.Unmarshal(detailsData, &p.Details)
		if err != nil {
			return nil, err
		}
	} else {
		p.Details = []string{}
	}
	// Fetch product images
	images, err := fetchProductImages(p.ID)
	if err != nil {
		return nil, err
	}
	p.Images = images
	// Fetch product warranties
	warranties, err := FetchProductWarranties(p.ID)
	if err != nil {
		return nil, err
	}
	p.Warranty = &warranties
	features, err := fetchProductFeatures(p.ID)
	if err != nil {
		return nil, err
	}
	p.Features = features
	variants, err := getProductVariants(p.ID)
	if err != nil {
		return nil, err
	}
	p.ProductVariants = variants
	tax, err := fetchProductTax(p.ID)
	if err != nil {
		return nil, err
	}
	p.Tax = &tax
	return &p, nil
}
func IsSkuThere(sku string) error {
	skuExists, err := RecordExists("products", "sku = ?", sku)
	if err != nil {
		return err
	}
	if skuExists {
		return fmt.Errorf("duplicate SKU found: %s", sku)
	}
	return nil
}
func IsCategoryParent(categoryID string) error {
	var parentID *string
	err := DB.QueryRow("SELECT parent_category_id FROM categories WHERE category_id = ?", categoryID).Scan(&parentID)
	if err != nil {
		return err
	}

	// 3. Prevent adding product to parent category
	if parentID == nil {
		return fmt.Errorf("cannot add product to a parent category with ID %s, choose a subcategory instead", categoryID)
	}
	return nil

}
func AddNewProduct(input dtos.CreateProduct, userID string) (*dtos.CreateProduct, error) {
	skuExists, err := RecordExists("products", "sku = ?", input.SKU)
	if err != nil {
		return nil, err
	}
	if skuExists {
		return nil, fmt.Errorf("duplicate SKU")
	}
	err = CategoryExists(input.CategoryID)
	if err != nil {
		return nil, err
	}

	err = IsCategoryParent(input.CategoryID)
	if err != nil {
		return nil, err
	}
	productID, _ := shortid.Generate()
	sellWhenOOs := false
	showStock := false
	if input.SellWhenOOS != nil {
		sellWhenOOs = *input.SellWhenOOS
	}
	if input.ShowStock != nil {
		showStock = *input.ShowStock
	}
	var detailsJSON []byte

	if input.Details != nil {
		detailsJSON, err = json.Marshal(input.Details)
		if err != nil {
			return nil, err
		}
	} else {
		detailsJSON = nil
	}
	_, err = DB.Exec(`
		INSERT INTO products (product_id, name, description, sku, price, category_id, stock_quantity, search_vector, tag, low_stock_quantity_warning, sell_when_out_of_stock, show_stock_quantity, created_by_id, buying_price, details)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		productID, input.Name, input.Description, input.SKU, input.Price, input.CategoryID, input.StockQuantity, input.SearchVector, input.Tag, input.LowStockAlert, sellWhenOOs, showStock, userID, input.BuyingPrice, detailsJSON,
	)
	if err != nil {
		return nil, err
	}

	return &dtos.CreateProduct{
		ID:            productID,
		Name:          input.Name,
		Description:   input.Description,
		SKU:           input.SKU,
		Price:         input.Price,
		CategoryID:    input.CategoryID,
		StockQuantity: input.StockQuantity,
		SearchVector:  input.SearchVector,
	}, nil
}
func UpdateProductByID(productID string, input dtos.CreateProduct) (*dtos.CreateProduct, error) {
	errr := IsProductThere(productID)
	if errr != nil {
		return nil, errr
	}
	// Check if SKU exists in another product (exclude current product)
	var exists bool
	err := DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM products
			WHERE sku = ? AND product_id != ?
		)`, input.SKU, productID,
	).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf("failed to check SKU uniqueness: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("duplicate SKU")
	}
	_, err = DB.Exec(`
		UPDATE products
		SET name = ?, description = ?, sku = ?, price = ?, stock_quantity = ?, search_vector = ?, last_updated_at = CURRENT_TIMESTAMP, tag = ?, low_stock_quantity_warning = ?, sell_when_out_of_stock = ?, show_stock_quantity = ?
		WHERE product_id = ?`,
		input.Name, input.Description, input.SKU, input.Price, input.StockQuantity, input.SearchVector, input.Tag, input.LowStockAlert, input.SellWhenOOS, input.ShowStock,
		productID,
	)

	if err != nil {
		return nil, err
	}

	// Return the updated product info
	return &dtos.CreateProduct{
		ID:            productID,
		Name:          input.Name,
		Description:   input.Description,
		SKU:           input.SKU,
		Price:         input.Price,
		CategoryID:    input.CategoryID,
		StockQuantity: input.StockQuantity,
		SearchVector:  input.SearchVector,
	}, nil
}

func DeleteProductByID(productID string) error {
	err := IsProductThere(productID)
	if err != nil {
		return err
	}
	// 1. Check if product is used in any uncollected order
	query := `
		SELECT COUNT(*) 
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE oi.product_id = ? AND o.status != 'collected'
	`
	var count int
	err = DB.QueryRow(query, productID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check product usage in orders: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete product; it's used in active orders")
	}

	// 2. Check if product is part of any bundles
	var bundleCount int
	err = DB.QueryRow(`SELECT COUNT(*) FROM bundle_products WHERE product_id = ?`, productID).Scan(&bundleCount)
	if err != nil {
		return fmt.Errorf("failed to check product usage in bundles: %w", err)
	}
	if bundleCount > 0 {
		return fmt.Errorf("cannot delete product; it's part of a bundle")
	}

	_, err = DB.Exec("DELETE FROM products WHERE product_id = ?", productID)
	return err
}

func InsertProductImage(productID, imageURL, fileType string, isPrimary bool) error {
	imageID, _ := shortid.Generate()
	query := `INSERT INTO product_images (image_id, product_id, url, is_primary, type) VALUES (?, ?, ?, ?, ?)`
	_, err := DB.Exec(query, imageID, productID, imageURL, isPrimary, fileType)
	return err
}

func DeleteProductImage(imageID string) error {
	_, err := DB.Exec("DELETE FROM product_images WHERE image_id = ?", imageID)
	return err
}

func GetProductImages(productID string) ([]dtos.Image, error) {
	query := `SELECT image_id, product_id, url, is_primary, type FROM product_images WHERE product_id = ?`
	rows, err := DB.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []dtos.Image
	for rows.Next() {
		var img dtos.Image
		var productID string
		err := rows.Scan(&img.ImageID, &productID, &img.URL, &img.IsPrimary, &img.Type)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}
func GetRelatedProducts(categoryID, excludeProductID string, limit, page int) ([]dtos.Product, *dtos.PaginationMeta, error) {
	// Build query to get related products from the same category, excluding the specified product
	query, args := buildRelatedProductsQuery(categoryID, excludeProductID, limit, page)

	// Build count query for total items
	countQuery, countArgs := buildRelatedProductsCountQuery(categoryID, excludeProductID)

	// Execute main query
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Execute count query
	var totalItems int64
	err = DB.QueryRow(countQuery, countArgs...).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Process products
	var relatedProducts []dtos.Product
	for rows.Next() {
		product, err := scanRelatedProduct(rows)
		if err != nil {
			return nil, nil, err
		}
		relatedProducts = append(relatedProducts, product)
	}

	// Calculate pagination metadata
	pagination := calculatePagination(page, limit, totalItems)

	return relatedProducts, &pagination, nil
}

func buildRelatedProductsQuery(categoryID, excludeProductID string, limit, page int) (string, []interface{}) {
	var args []interface{}

	query := `
        SELECT 
            p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
            p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
        FROM products p
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
        WHERE p.category_id = ?
		AND p.product_type = 'single'
    `
	args = append(args, categoryID)

	if excludeProductID != "" {
		query += " AND p.product_id != ?"
		args = append(args, excludeProductID)
	}

	// Order by most recent or most relevant
	query += " ORDER BY p.created_at DESC"

	// Pagination
	if limit > 0 {
		offset := (page - 1) * limit
		query += limtOffset
		args = append(args, limit, offset)
	}

	return query, args
}

func buildRelatedProductsCountQuery(categoryID, excludeProductID string) (string, []interface{}) {
	var args []interface{}

	query := "SELECT COUNT(*) FROM products p WHERE p.category_id = ? AND p.product_type = 'single'"
	args = append(args, categoryID)

	if excludeProductID != "" {
		query += " AND p.product_id != ?"
		args = append(args, excludeProductID)
	}

	return query, args
}

func scanRelatedProduct(rows *sql.Rows) (dtos.Product, error) {
	// Null-safe scan variables
	var (
		productID, name, desc, sku, categoryID, searchVector, tag sql.NullString
		price                                                     sql.NullFloat64
		stockQuantity                                             sql.NullInt64
		createdAt, updatedAt                                      sql.NullTime
		weightLimit, weight, discount                             sql.NullFloat64
		discountType, dimensions, manufacturer                    sql.NullString
	)

	// Scan database row
	if err := rows.Scan(
		&productID, &name, &desc, &sku, &price, &categoryID,
		&stockQuantity, &searchVector, &createdAt, &updatedAt,
		&tag, &discount, &discountType, &weight, &dimensions, &manufacturer, &weightLimit,
	); err != nil {
		return dtos.Product{}, err
	}

	// Prepare tag pointer
	tagPtr := ""
	if tag.Valid {
		tagPtr = tag.String
	}

	// Build product struct
	product := dtos.Product{
		ID:            productID.String,
		Name:          name.String,
		Description:   desc.String,
		SKU:           sku.String,
		Price:         price.Float64,
		CategoryID:    categoryID.String,
		StockQuantity: int(stockQuantity.Int64),
		SearchVector:  searchVector.String,
		Tag:           &tagPtr,
	}

	if createdAt.Valid {
		product.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		product.LastUpdated = updatedAt.Time
	}
	if discount.Valid {
		product.Discount = &discount.Float64
	}

	if discountType.Valid {
		product.DiscountType = &discountType.String
	}
	if weight.Valid {
		product.Weight = &weight.Float64
	}
	if weightLimit.Valid {
		product.WeightLimit = &weightLimit.Float64
	}
	if dimensions.Valid {
		product.Dimensions = &dimensions.String
	}
	if manufacturer.Valid {
		product.Manufacturer = &manufacturer.String
	}

	// Fetch related data
	if images, err := fetchProductImages(product.ID); err == nil {
		product.Images = images
	} else {
		return product, err
	}

	if warranties, err := FetchProductWarranties(product.ID); err == nil {
		product.Warranty = &warranties
	} else {
		return product, err
	}
	tax, err := fetchProductTax(product.ID)
	if err == nil {
		product.Tax = &tax
	} else {
		return product, err
	}

	if features, err := fetchProductFeatures(product.ID); err == nil {
		product.Features = features
	} else {
		return product, err
	}

	if variants, err := getProductVariants(product.ID); err == nil {
		product.ProductVariants = variants
	} else {
		return product, err
	}

	return product, nil
}

// products reports
func GetProductPerformance(start, end time.Time, categoryID string) ([]map[string]interface{}, error) {
	rows, err := DB.Query(`
		SELECT 
			p.product_id,
			p.name,
			c.name AS category,
			COALESCE(SUM(oi.quantity), 0) AS sales_volume,
			COALESCE(SUM(r.quantity), 0) AS total_returns,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.price)) / NULLIF(SUM(oi.quantity * oi.unit_price), 0), 0) AS profit_margin,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.price)), 0) AS net_profit
		FROM products p
		JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN order_items oi ON p.product_id = oi.product_id
		LEFT JOIN returns r ON p.product_id = r.product_id
		JOIN orders o ON oi.order_id = o.order_id
		WHERE o.created_at BETWEEN ? AND ?
		  AND c.category_id = ?
		GROUP BY p.product_id, p.name, c.name
		ORDER BY net_profit DESC
	`, start, end, categoryID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summary []map[string]interface{}
	for rows.Next() {
		var productID, name, category string
		var salesVolume, totalReturns int
		var profitMargin, netProfit float64

		if err := rows.Scan(&productID, &name, &category, &salesVolume, &totalReturns, &profitMargin, &netProfit); err != nil {
			return nil, err
		}

		summary = append(summary, map[string]interface{}{
			"product_id":    productID,
			"name":          name,
			"category":      category,
			"sales_volume":  salesVolume,
			"total_returns": totalReturns,
			"profit_margin": profitMargin,
			"net_profit":    netProfit,
		})
	}
	return summary, nil
}

func GetSingleProductPerformance(productID string, start, end time.Time) (map[string]interface{}, error) {

	var name, category string
	var salesVolume, totalReturns int
	var returnRate, profitMargin, netProfit float64

	err := DB.QueryRow(`
		SELECT 
			p.product_id,
			p.name,
			c.name AS category,
			COALESCE(SUM(oi.quantity), 0) AS sales_volume,
			COALESCE(SUM(r.quantity), 0) AS total_returns,
			(COALESCE(SUM(r.quantity), 0) / NULLIF(SUM(oi.quantity), 0)) * 100 AS return_rate,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.price)) / NULLIF(SUM(oi.quantity * oi.unit_price), 0), 0) AS profit_margin,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.price)), 0) AS net_profit
		FROM products p
		JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN order_items oi ON p.product_id = oi.product_id
		LEFT JOIN returns r ON p.product_id = r.product_id
		JOIN orders o ON oi.order_id = o.order_id
		WHERE o.created_at BETWEEN ? AND ?
		  AND p.product_id = ?
		GROUP BY p.product_id, p.name, c.name
	`, start, end, productID).Scan(&productID, &name, &category, &salesVolume, &totalReturns, &returnRate, &profitMargin, &netProfit)

	if err != nil {
		return nil, err
	}

	response := map[string]interface{}{
		"product_id":    productID,
		"name":          name,
		"category":      category,
		"sales_volume":  salesVolume,
		"total_returns": totalReturns,
		"return_rate":   returnRate,
		"profit_margin": profitMargin,
		"net_profit":    netProfit,
		"period": map[string]time.Time{
			"start": start,
			"end":   end,
		},
	}
	return response, nil
}

// FetchSubcategoryProducts retrieves products under a given subcategory with pagination
func FetchSubcategoryProducts(subcategoryID string, page, size int) (*dtos.SubcategoryProducts, *dtos.PaginationMeta, error) {
	valid, err := IsValidSubcategory(subcategoryID)
	if err != nil {
		return nil, nil, err
	}
	if !valid {
		return nil, nil, fmt.Errorf("cannot use a main category ID, must be a subcategory")
	}

	offset := (page - 1) * size

	// Fetch subcategory but make sure it's not a main category (parent_category_id must NOT be NULL)
	var sub dtos.SubcategoryProducts
	err = DB.QueryRow(`
		SELECT 
			c.category_id, c.name, c.image, c.parent_category_id,
			p.name as parent_name, p.image as parent_image
		FROM categories c
		LEFT JOIN categories p ON c.parent_category_id = p.category_id
		WHERE c.category_id = ? AND c.parent_category_id IS NOT NULL`, subcategoryID).
		Scan(&sub.ID, &sub.Name, &sub.ImageURL, &sub.ParentID, &sub.ParentCategoryName, &sub.ParentCategoryImageURL)

	if err != nil {
		return nil, nil, err
	}

	// Count total products for pagination
	var totalItems int
	err = DB.QueryRow(`SELECT COUNT(*) FROM products WHERE category_id = ? AND product_type = 'single'`, subcategoryID).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Fetch products for the subcategory with pagination
	rows, err := DB.Query(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id, 
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, p.tag, p.details, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM products p
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE category_id = ? AND product_type = 'single'
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, subcategoryID, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var products []dtos.Product
	for rows.Next() {
		var (
			p           dtos.Product
			detailsData []byte
		)
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price,
			&p.CategoryID, &p.StockQuantity, &p.SearchVector,
			&p.CreatedAt, &p.LastUpdated, &p.Tag, &detailsData, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
		); err != nil {
			return nil, nil, err
		}

		if len(detailsData) > 0 {
			err := json.Unmarshal(detailsData, &p.Details)
			if err != nil {
				return nil, nil, err
			}
		} else {
			p.Details = []string{}
		}

		// fetch product images (reusable helper)
		images, err := fetchProductImages(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Images = images
		// fetch product warranties
		warranties, err := FetchProductWarranties(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Warranty = &warranties
		// fetch product features
		features, err := fetchProductFeatures(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Features = features
		variants, err := getProductVariants(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.ProductVariants = variants
		tax, err := fetchProductTax(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Tax = &tax

		products = append(products, p)
	}
	sub.Products = products

	// build pagination metadata
	totalPages := int(math.Ceil(float64(totalItems) / float64(size)))
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return &sub, meta, nil
}

// IsValidSubcategory checks whether a category is a valid subcategory (not a main category).
func IsValidSubcategory(categoryID string) (bool, error) {
	var parentID sql.NullString
	err := DB.QueryRow(`SELECT parent_category_id FROM categories WHERE category_id = ?`, categoryID).Scan(&parentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("category not found")
		}
		return false, err
	}

	// If parent_category_id is NULL → it's a main category → invalid
	if !parentID.Valid {
		return false, nil
	}
	return true, nil
}
func IsValidCategory(categoryID string) (bool, error) {
	var parentID sql.NullString
	err := DB.QueryRow(`
		SELECT parent_category_id 
		FROM categories 
		WHERE category_id = ?`, categoryID).
		Scan(&parentID)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("category not found")
		}
		return false, err
	}

	// Only valid if it's a parent (i.e., has no parent itself)
	if !parentID.Valid {
		return true, nil // main category
	}
	return false, nil // subcategory, not valid
}

// GetCategoriesWithSubcategoriesAndProducts retrieves main categories (or filtered one),
// their subcategories, and products under each subcategory.
func GetCategoriesWithSubcategoriesAndProducts(
	searchParams dtos.SearchParams,
	filterCategoryID string,
) ([]dtos.CategoryResponse, *dtos.PaginationMeta, error) {

	// --- Validate category filter ---
	if filterCategoryID != "" {
		valid, err := IsValidCategory(filterCategoryID)
		if err != nil {
			return nil, nil, err
		}
		if !valid {
			return nil, nil, fmt.Errorf("cannot use a subcategory ID, must be a main category")
		}
	}

	// --- Fetch main categories ---
	categories, err := getMainCategories(filterCategoryID, searchParams)
	if err != nil {
		return nil, nil, err
	}

	var paginationMeta *dtos.PaginationMeta

	// --- For each category, attach subcategories and products ---
	for i := range categories {
		subs, subIDs, err := getSubcategoriesWithParentID(categories[i].ID)
		if err != nil {
			return nil, nil, err
		}
		categories[i].Subcategories = subs

		products, meta, err := getProductsForSubcategories(subIDs, searchParams.Page, searchParams.Limit, searchParams)
		if err != nil {
			return nil, nil, err
		}
		categories[i].Products = products
		paginationMeta = meta
	}

	return categories, paginationMeta, nil
}

//
// --- MAIN CATEGORY QUERY ---
//

func getMainCategories(filterCategoryID string, params dtos.SearchParams) ([]dtos.CategoryResponse, error) {
	var (
		query = `
            SELECT c.category_id, c.name, c.parent_category_id, c.image, c.description
            FROM categories c
            WHERE c.parent_category_id IS NULL
            AND EXISTS (
                SELECT 1 FROM categories sub
                LEFT JOIN products p ON sub.category_id = p.category_id
                WHERE sub.parent_category_id = c.category_id
                AND p.product_id IS NOT NULL
				AND p.product_type = 'single'
            )`
		args []interface{}
	)

	if filterCategoryID != "" {
		query += " AND c.category_id = ?"
		args = append(args, filterCategoryID)
	}

	if params.Q != "" {
		query += lowerPname
		args = append(args, "%"+strings.ToLower(params.Q)+"%")
	}

	if params.CategoryName != "" {
		query += lowerCname
		args = append(args, "%"+strings.ToLower(params.CategoryName)+"%")
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []dtos.CategoryResponse
	for rows.Next() {
		var cat dtos.CategoryResponse
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.ParentID, &cat.ImageURL, &cat.Description); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}

	return categories, nil
}

//
// --- SUBCATEGORIES QUERY ---
//

func getSubcategoriesWithParentID(parentID string) ([]dtos.SubcategoryResponse, []string, error) {
	rows, err := DB.Query(`
        SELECT c.category_id, c.name, c.parent_category_id, c.image, c.description
        FROM categories c
        LEFT JOIN products p ON c.category_id = p.category_id
        WHERE c.parent_category_id = ?
        AND p.product_id IS NOT NULL
		AND p.product_type = 'single'
        GROUP BY c.category_id, c.name, c.parent_category_id, c.image, c.description`, parentID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var (
		subs []dtos.SubcategoryResponse
		ids  []string
	)

	for rows.Next() {
		var sub dtos.SubcategoryResponse
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.ParentID, &sub.ImageURL, &sub.Description); err != nil {
			return nil, nil, err
		}
		subs = append(subs, sub)
		ids = append(ids, sub.ID)
	}

	return subs, ids, nil
}

//
// --- PRODUCTS FOR SUBCATEGORIES ---
//

func getProductsForSubcategories(subIDs []string, page, size int, params dtos.SearchParams) ([]dtos.CategoryProduct, *dtos.PaginationMeta, error) {
	if len(subIDs) == 0 {
		return nil, nil, nil
	}

	offset := (page - 1) * size
	placeholders := strings.Repeat(",?", len(subIDs)-1)

	baseWhere := fmt.Sprintf(`WHERE p.category_id IN (?%s)
		AND p.product_type = 'single'`, placeholders)

	args := make([]interface{}, len(subIDs))
	for i, id := range subIDs {
		args[i] = id
	}

	// Apply filters
	filterQuery, filterArgs := buildProductFilters(params)
	args = append(args, filterArgs...)
	whereClause := baseWhere + filterQuery

	joinClause := `FROM products p
		JOIN categories c ON p.category_id = c.category_id`

	countQuery := "SELECT COUNT(*) " + joinClause + " " + whereClause
	var totalItems int
	if err := DB.QueryRow(countQuery, args...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	sortClause := getSortClause(params.SortBy)
	dataQuery := fmt.Sprintf(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price,
			p.category_id, c.parent_category_id, p.stock_quantity,
			p.search_vector, p.created_at, p.last_updated_at, p.tag, p.details, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?`, whereClause, sortClause)

	args = append(args, size, offset)

	rows, err := DB.Query(dataQuery, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var products []dtos.CategoryProduct
	for rows.Next() {
		var pr dtos.CategoryProduct
		var subcategoryID, parentCategoryID string
		var detailsData []byte

		if err := rows.Scan(
			&pr.ID, &pr.Name, &pr.Description, &pr.SKU, &pr.Price,
			&subcategoryID, &parentCategoryID, &pr.StockQuantity,
			&pr.SearchVector, &pr.CreatedAt, &pr.LastUpdated, &pr.Tag, &detailsData, &pr.Discount, &pr.DiscountType, &pr.Weight, &pr.Dimensions, &pr.Manufacturer, &pr.WeightLimit,
		); err != nil {
			return nil, nil, err
		}

		pr.CategoryID = parentCategoryID
		pr.SubcategoryID = subcategoryID

		if len(detailsData) > 0 {
			err := json.Unmarshal(detailsData, &pr.Details)
			if err != nil {
				return nil, nil, err
			}
		} else {
			pr.Details = []string{}
		}

		// Fetch related images and variants
		if pr.Images, err = fetchProductImages(pr.ID); err != nil {
			return nil, nil, err
		}
		warranty, err := FetchProductWarranties(pr.ID)
		if err != nil {
			return nil, nil, err
		}
		pr.Warranty = &warranty
		if pr.Features, err = fetchProductFeatures(pr.ID); err != nil {
			return nil, nil, err
		}
		if pr.ProductVariants, err = getProductVariants(pr.ID); err != nil {
			return nil, nil, err
		}
		tax, err := fetchProductTax(pr.ID)
		if err != nil {
			return nil, nil, err
		}
		pr.Tax = &tax

		products = append(products, pr)
	}

	// --- Pagination metadata ---
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: int(math.Ceil(float64(totalItems) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < totalItems,
	}

	return products, meta, nil
}

//
// --- FILTER BUILDER ---
//

func buildProductFilters(params dtos.SearchParams) (string, []interface{}) {
	var (
		conditions []string
		args       []interface{}
	)

	if params.ProductName != "" {
		conditions = append(conditions, "LOWER(p.name) LIKE ?")
		args = append(args, "%"+strings.ToLower(params.ProductName)+"%")
	}

	if params.SKU != "" {
		conditions = append(conditions, "LOWER(p.sku) = ?")
		args = append(args, strings.ToLower(params.SKU))
	}

	if params.Tag != "" {
		conditions = append(conditions, "LOWER(p.tag) = ?")
		args = append(args, strings.ToLower(params.Tag))
	}

	if params.MinPrice >= 0 && params.MaxPrice > 0 {
		conditions = append(conditions, "p.price BETWEEN ? AND ?")
		args = append(args, params.MinPrice, params.MaxPrice)
	}

	// --- Variants filtering (fixed: NO leading "AND") ---
	if len(params.Variants) > 0 {
		var variantConds []string
		for _, v := range params.Variants {
			if strings.ToLower(v.Value) == "all" {
				// lowerVariant should be something like: "LOWER(v.type) = ?"
				variantConds = append(variantConds, lowerVariant)
				args = append(args, strings.ToLower(v.Type))
			} else {
				// lowerVariantTypeName should be something like: "LOWER(v.type) = ? AND LOWER(pv.value) = ?"
				variantConds = append(variantConds, lowerVariantTypeName)
				args = append(args, strings.ToLower(v.Type), strings.ToLower(v.Value))
			}
		}

		// IMPORTANT: no leading AND here — the full condition is just the IN(...) expression
		variantQuery := fmt.Sprintf(
			`p.product_id IN (
				SELECT pv.product_id
				FROM product_variants pv
				JOIN variants v ON pv.variant_id = v.variant_id
				WHERE %s
			)`,
			strings.Join(variantConds, " OR "),
		)

		// append the single condition (no leading AND)
		conditions = append(conditions, variantQuery)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " AND " + strings.Join(conditions, " AND "), args
}

// new fetch products
func SearchProducts(params dtos.SearchParams, isAdmin bool) ([]dtos.Product, *dtos.PaginationMeta, error) {
	// Build the main query
	query, args := buildSearchQuery(params)
	countQuery, countArgs := buildCountQuerySearch(params)

	// Count total items
	var totalItems int64
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Fetch products
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var products []dtos.Product
	for rows.Next() {
		product, err := scanProduct(rows, isAdmin)
		if err != nil {
			return nil, nil, err
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Pagination
	pagination := calculatePagination(params.Page, params.Limit, totalItems)

	return products, &pagination, nil
}
func buildSearchQuery(params dtos.SearchParams) (string, []interface{}) {
	query := `
		SELECT DISTINCT
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at,
			c.name as category_name, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE p.product_type = 'single'
	`
	var args []interface{}
	// Apply search query (q parameter) - searches both category name and product name
	if params.Q != "" {
		query += " AND (LOWER(p.name) LIKE ? OR LOWER(c.name) LIKE ?)"
		searchTerm := "%" + strings.ToLower(params.Q) + "%"
		args = append(args, searchTerm, searchTerm)
	}
	// Apply filters
	if params.CategoryName != "" {
		query += lowerCname
		args = append(args, "%"+strings.ToLower(params.CategoryName)+"%")
	}

	if params.ProductName != "" {
		query += lowerPname
		args = append(args, "%"+strings.ToLower(params.ProductName)+"%")
	}
	if params.SKU != "" {
		query += " AND LOWER(p.sku) = ?"
		args = append(args, strings.ToLower(params.SKU))
	}
	if params.Tag != "" {
		query += " AND LOWER(p.tag) = ?"
		args = append(args, strings.ToLower(params.Tag))
	}
	if params.MinPrice >= 0 && params.MaxPrice > 0 {
		query += " AND p.price BETWEEN ? AND ?"
		args = append(args, params.MinPrice, params.MaxPrice)
	}
	if params.StartDate != "" && params.EndDate != "" {
		query += " AND DATE(p.created_at) BETWEEN ? AND ?"
		args = append(args, params.StartDate, params.EndDate)
	}
	// Apply multiple variant filters
	if len(params.Variants) > 0 {
		variantSubquery := `
        AND p.product_id IN (
            SELECT pv.product_id 
            FROM product_variants pv
            JOIN variants v ON pv.variant_id = v.variant_id
            WHERE `

		variantConditions := []string{}
		for _, variant := range params.Variants {
			if strings.ToLower(variant.Value) == "all" {
				// Only match by type if "all"
				variantConditions = append(variantConditions, lowerVariant)
				args = append(args, strings.ToLower(variant.Type))
			} else {
				// Match by both type and value
				variantConditions = append(variantConditions, lowerVariantTypeName)
				args = append(args, strings.ToLower(variant.Type), strings.ToLower(variant.Value))
			}
		}

		variantSubquery += strings.Join(variantConditions, " OR ")
		variantSubquery += ")"
		query += variantSubquery
	}

	// Apply sorting
	query += " ORDER BY " + getSortClause(params.SortBy)

	// Apply pagination
	if params.Limit > 0 {
		offset := (params.Page - 1) * params.Limit
		query += limtOffset
		args = append(args, params.Limit, offset)
	}

	return query, args
}

func buildCountQuerySearch(params dtos.SearchParams) (string, []interface{}) {
	query := `
		SELECT COUNT(DISTINCT p.product_id)
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE p.product_type = 'single'
	`
	var args []interface{}
	// Apply search query
	if params.Q != "" {
		query += " AND (LOWER(p.name) LIKE ? OR LOWER(c.name) LIKE ?)"
		searchTerm := "%" + strings.ToLower(params.Q) + "%"
		args = append(args, searchTerm, searchTerm)
	}
	if params.CategoryName != "" {
		query += lowerCname
		args = append(args, "%"+strings.ToLower(params.CategoryName)+"%")
	}

	if params.ProductName != "" {
		query += lowerPname
		args = append(args, "%"+strings.ToLower(params.ProductName)+"%")
	}
	if params.SKU != "" {
		query += " AND LOWER(p.sku) = ?"
		args = append(args, strings.ToLower(params.SKU))
	}
	if params.Tag != "" {
		query += " AND LOWER(p.tag) = ?"
		args = append(args, strings.ToLower(params.Tag))
	}
	if params.MinPrice >= 0 && params.MaxPrice > 0 {
		query += " AND p.price BETWEEN ? AND ?"
		args = append(args, params.MinPrice, params.MaxPrice)
	}
	if params.StartDate != "" && params.EndDate != "" {
		query += " AND DATE(p.created_at) BETWEEN ? AND ?"
		args = append(args, params.StartDate, params.EndDate)
	}
	// Apply multiple variant filters
	if len(params.Variants) > 0 {
		variantSubquery := `
        AND p.product_id IN (
            SELECT pv.product_id 
            FROM product_variants pv
            JOIN variants v ON pv.variant_id = v.variant_id
            WHERE `

		variantConditions := []string{}
		for _, variant := range params.Variants {
			if strings.ToLower(variant.Value) == "all" {
				// Only match by type if "all"
				variantConditions = append(variantConditions, lowerVariant)
				args = append(args, strings.ToLower(variant.Type))
			} else {
				// Match by both type and value
				variantConditions = append(variantConditions, lowerVariantTypeName)
				args = append(args, strings.ToLower(variant.Type), strings.ToLower(variant.Value))
			}
		}

		variantSubquery += strings.Join(variantConditions, " OR ")
		variantSubquery += ")"
		query += variantSubquery
	}

	return query, args
}

const (
	SortPriceHighToLow   = "price:high-to-low"
	SortPriceLowToHigh   = "price:low-to-high"
	SortDateOldToNew     = "date:old-to-new"
	SortDateNewToOld     = "date:new-to-old"
	SortFeatured         = "featured"
	SortBestSellers      = "best_sellers"
	SortAlphabeticallyAZ = "alphabetically:a-z"
	SortAlphabeticallyZA = "alphabetically:z-a"
)

func getSortClause(sortBy string) string {
	switch sortBy {
	case SortPriceHighToLow:
		return "p.price DESC"
	case SortPriceLowToHigh:
		return "p.price ASC"
	case SortDateOldToNew:
		return "p.created_at ASC"
	case SortDateNewToOld:
		return "p.created_at DESC"
	case SortFeatured:
		return `
			CASE WHEN p.product_id IN (SELECT product_id FROM featured_products) THEN 0 ELSE 1 END,
			p.created_at DESC
		`
	case SortBestSellers:
		return `
			(SELECT COALESCE(SUM(oi.quantity), 0) 
			 FROM order_items oi 
			 WHERE oi.product_id = p.product_id) DESC,
			p.created_at DESC
		`
	case SortAlphabeticallyAZ:
		return "p.name ASC"
	case SortAlphabeticallyZA:
		return "p.name DESC"
	default:
		return "p.created_at DESC"
	}
}

func scanProduct(rows *sql.Rows, isAdmin bool) (dtos.Product, error) {
	var (
		productID, name, desc, sku, categoryID, searchVector, categoryName, tag sql.NullString
		price                                                                   sql.NullFloat64
		stockQuantity                                                           sql.NullInt64
		createdAt, updatedAt                                                    sql.NullTime
		discount, weight, weightLimit                                           sql.NullFloat64
		discountType, dimensions, manufacturer                                  sql.NullString
	)

	if err := rows.Scan(
		&productID, &name, &desc, &sku, &price, &categoryID,
		&stockQuantity, &searchVector, &createdAt, &updatedAt, &categoryName, &tag, &discount, &discountType, &weight, &dimensions, &manufacturer, &weightLimit,
	); err != nil {
		return dtos.Product{}, err
	}

	stock := 0
	if stockQuantity.Valid {
		stock = int(stockQuantity.Int64)
	}
	tagStr := ""
	if tag.Valid {
		tagStr = strings.TrimSpace(tag.String)
	}
	product := dtos.Product{
		ID:            productID.String,
		Name:          name.String,
		Description:   desc.String,
		SKU:           sku.String,
		Price:         price.Float64,
		CategoryID:    categoryID.String,
		CategoryName:  categoryName.String,
		StockQuantity: stock,
		SearchVector:  searchVector.String,
		Tag:           &tagStr,
	}
	if discount.Valid {
		product.Discount = &discount.Float64
	}
	if discountType.Valid {
		product.DiscountType = &discountType.String
	}
	if weight.Valid {
		product.Weight = &weight.Float64
	}
	if weightLimit.Valid {
		product.WeightLimit = &weightLimit.Float64
	}
	if dimensions.Valid {
		product.Dimensions = &dimensions.String
	}
	if manufacturer.Valid {
		product.Manufacturer = &manufacturer.String
	}

	if createdAt.Valid {
		product.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		product.LastUpdated = updatedAt.Time
	}

	// Fetch additional data
	images, err := fetchProductImages(product.ID)
	if err != nil {
		return product, err
	}
	product.Images = images
	warranties, err := FetchProductWarranties(product.ID)
	if err != nil {
		return product, err
	}
	product.Warranty = &warranties
	features, err := fetchProductFeatures(product.ID)
	if err != nil {
		return product, err
	}
	product.Features = features
	variants, err := getProductVariants(product.ID)
	if err != nil {
		return product, err
	}
	product.ProductVariants = variants
	tax, err := fetchProductTax(product.ID)
	if err != nil {
		return product, err
	}
	product.Tax = &tax

	if isAdmin {
		if err := enrichProductAdmin(&product); err != nil {
			return product, err
		}
	}

	return product, nil
}

// enrichProductAdmin adds admin-specific fields to the product.
func enrichProductAdmin(product *dtos.Product) error {
	var err error
	product.CreatedBy, err = getProductCreator(product.ID)
	if err != nil {
		return err
	}
	product.IsInTodaysDeals, err = isProductInTodaysDeal(product.ID)
	if err != nil {
		return err
	}
	product.MaxStockQuantity, err = getMaxQuantity(product.ID)
	if product.MaxStockQuantity < product.StockQuantity {
		product.MaxStockQuantity = product.StockQuantity
	}
	return err
}
func getProductCreator(productID string) (string, error) {
	var createdByID sql.NullString
	query := `SELECT created_by_id FROM products WHERE product_id = ?`

	err := DB.QueryRow(query, productID).Scan(&createdByID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // product not found → treat as no creator
		}
		return "", fmt.Errorf("failed to get product creator ID: %v", err)
	}

	// If created_by_id is NULL → return empty string
	if !createdByID.Valid || createdByID.String == "" {
		return "", nil
	}

	// Fetch user info
	var firstName, lastName sql.NullString
	userQuery := `SELECT first_name, last_name FROM users WHERE user_id = ?`
	err = DB.QueryRow(userQuery, createdByID.String).Scan(&firstName, &lastName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // user not found → return empty
		}
		return "", fmt.Errorf("failed to get user details: %v", err)
	}

	// Handle possible NULLs
	fn := ""
	ln := ""
	if firstName.Valid {
		fn = firstName.String
	}
	if lastName.Valid {
		ln = lastName.String
	}

	fullName := strings.TrimSpace(fn + " " + ln)
	return fullName, nil
}
func isProductInTodaysDeal(productID string) (bool, error) {
	var dealID string

	// Get deal_id for a deal whose name matches 'today'
	dealQuery := `SELECT deal_id FROM deals WHERE name REGEXP '(?i)today' LIMIT 1`
	err := DB.QueryRow(dealQuery).Scan(&dealID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // No "today" deal exists
		}
		return false, fmt.Errorf("failed to get today's deal: %v", err)
	}

	// 2Check if this product is linked to that deal
	var exists bool
	checkQuery := `SELECT EXISTS(
		SELECT 1 FROM deal_products WHERE product_id = ? AND deal_id = ?
	)`
	err = DB.QueryRow(checkQuery, productID, dealID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check product-deal link: %v", err)
	}

	return exists, nil
}
func getMaxQuantity(productID string) (int, error) {
	var totalQuantity sql.NullInt64

	query := `
		SELECT SUM(quantity)
		FROM inventory
		WHERE product_id = ?
	`
	err := DB.QueryRow(query, productID).Scan(&totalQuantity)
	if err != nil {
		return 0, fmt.Errorf("failed to get quantity for product %s: %v", productID, err)
	}

	if !totalQuantity.Valid {
		return 0, nil // no entries found for this product
	}

	return int(totalQuantity.Int64), nil
}

func InsertProductSpecs(req dtos.ProductSpecs) error {
	err := IsProductThere(req.ProductID)
	if err != nil {
		return err
	}
	insertQuery := `INSERT INTO product_specifications (specifications_id, product_id, weight, weight_limit, dimensions, manufacturer) VALUES (?, ?, ?, ?,?,?)`

	// Generate bundle_product_id
	specificationsID, _ := shortid.Generate()

	// Insert product into bundle
	if _, err := DB.Exec(insertQuery, specificationsID, req.ProductID, req.Weight, req.WeightLimit, req.Dimensions, req.Manufacturer); err != nil {
		return err
	}

	return nil
}

// Fetch most expensive and cheapest products
func GetExpensiveAndCheapProducts() (*dtos.ExpensiveCheapProduct, error) {
	cheapestProduct, err := getProductByPriceType("cheapest")
	if err != nil {
		return nil, err
	}
	expensiveProduct, err := getProductByPriceType("expensive")
	if err != nil {
		return nil, err
	}
	combinedProducts := &dtos.ExpensiveCheapProduct{
		CheapestProduct:  *cheapestProduct,
		ExpensiveProduct: *expensiveProduct,
	}
	return combinedProducts, nil
}

// getProductByPriceType fetches a single product with either the lowest or highest price.
func getProductByPriceType(priceType string) (*dtos.Product, error) {
	var orderClause string
	switch strings.ToLower(priceType) {
	case "cheapest":
		orderClause = "ASC"
	case "expensive":
		orderClause = "DESC"
	default:
		return nil, fmt.Errorf("invalid price type: %s", priceType)
	}

	query := fmt.Sprintf(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, 
			p.category_id, p.stock_quantity, p.search_vector, 
			p.created_at, p.last_updated_at, c.name AS category_name, p.tag
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE p.product_type = 'single'
		ORDER BY p.price %s
		LIMIT 1
	`, orderClause)

	var p dtos.Product
	err := DB.QueryRow(query).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&p.CategoryName, &p.Tag,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // no products found
		}
		return nil, err
	}

	// Fetch product images
	if p.Images, err = fetchProductImages(p.ID); err != nil {
		return nil, err
	}
	// Fetch product warranties
	warranty, err := FetchProductWarranties(p.ID)
	if err != nil {
		return nil, err
	}
	p.Warranty = &warranty
	if p.Features, err = fetchProductFeatures(p.ID); err != nil {
		return nil, err
	}
	// Fetch product variants
	if p.ProductVariants, err = getProductVariants(p.ID); err != nil {
		return nil, err
	}

	tax, err := fetchProductTax(p.ID)
	if err != nil {
		return nil, err
	}
	p.Tax = &tax
	return &p, nil
}

// check if products exists in wishlist with userID
func IsProductInUserWishlist(userID, productID string) (bool, error) {
	var exists bool

	query := `
		SELECT EXISTS(
			SELECT 1
			FROM wishlists w
			JOIN wishlist_items witems ON w.wishlist_id = witems.wishlist_id
			WHERE w.user_id = ? AND witems.product_id = ?
		)
	`

	err := DB.QueryRow(query, userID, productID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check wishlist: %w", err)
	}

	return exists, nil
}

func RemoveHeldProductSpecs(specID string) error {
	_, err := DB.Exec(`DELETE FROM product_specifications WHERE specifications_id = ?`, specID)
	return err
}

func HoldProductSpecs(productID string) ([]string, error) {
	rows, err := DB.Query(`SELECT specifications_id FROM product_specifications WHERE product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var specIDs []string
	for rows.Next() {
		var specID string
		if err := rows.Scan(&specID); err != nil {
			return nil, err
		}
		specIDs = append(specIDs, specID)
	}
	return specIDs, nil
}

func RemoveAllProductVariants(productID string) error {
	_, err := DB.Exec(`DELETE FROM product_variants WHERE product_id = ?`, productID)
	return err
}
