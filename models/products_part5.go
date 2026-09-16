package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

func enrichProduct(db DBExecutor, product *dtos.Product) error {
	images, err := fetchProductImages(db, product.ID)
	if err != nil {
		return err
	}
	product.Images = images

	warranties, err := FetchProductWarranties(db, product.ID)
	if err != nil {
		return err
	}
	product.Warranty = &warranties

	tax, err := fetchProductTax(db, product.ID)
	if err != nil {
		return err
	}
	product.Tax = &tax

	features, err := fetchProductFeatures(db, product.ID)
	if err != nil {
		return err
	}
	product.Features = features

	variants, err := getProductVariants(db, product.ID)
	if err != nil {
		return err
	}
	product.ProductVariants = variants

	product.VariantSelection, err = GetVariantSelection(db, product.ID)
	if err != nil {
		return err
	}

	return nil
}
func ptr[T any](v T) *T {
	return &v
}

// products reports
func GetProductPerformance(db DBExecutor, start, end time.Time, categoryID string) ([]map[string]any, error) {
	//check if category exists
	err := isCategoryThere(db, categoryID)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`
		SELECT 
			p.product_id,
			p.name,
			c.name AS category,
			COALESCE(SUM(oi.quantity), 0) AS sales_volume,
			COALESCE(SUM(r.quantity), 0) AS total_returns,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.buying_price)) / NULLIF(SUM(oi.quantity * oi.unit_price), 0), 0) AS profit_margin,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.buying_price)), 0) AS net_profit
		FROM products p
		JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN order_items oi ON p.product_id = oi.product_id
		LEFT JOIN return_products r ON p.product_id = r.product_id
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

	var summary []map[string]any
	for rows.Next() {
		var productID, name, category string
		var salesVolume, totalReturns int
		var profitMargin, netProfit float64

		if err := rows.Scan(&productID, &name, &category, &salesVolume, &totalReturns, &profitMargin, &netProfit); err != nil {
			return nil, err
		}

		summary = append(summary, map[string]any{
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

func GetSingleProductPerformance(db DBExecutor, productID string, start, end time.Time) (map[string]any, error) {

	var name, category string
	var salesVolume, totalReturns int
	var returnRate, profitMargin, netProfit float64

	err := db.QueryRow(`
		SELECT 
			p.product_id,
			p.name,
			c.name AS category,
			COALESCE(SUM(oi.quantity), 0) AS sales_volume,
			COALESCE(SUM(r.quantity), 0) AS total_returns,
			(COALESCE(SUM(r.quantity), 0) / NULLIF(SUM(oi.quantity), 0)) * 100 AS return_rate,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.buying_price)) / NULLIF(SUM(oi.quantity * oi.unit_price), 0), 0) AS profit_margin,
			COALESCE(SUM(oi.quantity * (oi.unit_price - p.buying_price)), 0) AS net_profit
		FROM products p
		JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN order_items oi ON p.product_id = oi.product_id
		LEFT JOIN return_products r ON p.product_id = r.product_id
		JOIN orders o ON oi.order_id = o.order_id
		WHERE o.created_at BETWEEN ? AND ?
		  AND p.product_id = ?
		GROUP BY p.product_id, p.name, c.name
	`, start, end, productID).Scan(&productID, &name, &category, &salesVolume, &totalReturns, &returnRate, &profitMargin, &netProfit)

	if err != nil {
		return nil, err
	}

	response := map[string]any{
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
func FetchSubcategoryProducts(db DBExecutor, subcategoryID string, page, size int) (*dtos.SubcategoryProducts, *dtos.PaginationMeta, error) {
	valid, err := IsValidSubcategory(db, subcategoryID)
	if err != nil {
		return nil, nil, err
	}
	if !valid {
		return nil, nil, fmt.Errorf("cannot use a main category ID, must be a subcategory")
	}

	// Fetch subcategory details
	sub, err := fetchSubcategoryDetails(db, subcategoryID)
	if err != nil {
		return nil, nil, err
	}

	// Count total products for pagination
	totalItems, err := countSubcategoryProducts(db, subcategoryID)
	if err != nil {
		return nil, nil, err
	}

	// Fetch products for the subcategory with pagination
	products, err := fetchSubcategoryProductList(db, subcategoryID, page, size)
	if err != nil {
		return nil, nil, err
	}
	sub.Products = products

	// Build pagination metadata
	meta := buildSubcategoryPaginationMeta(page, size, totalItems)

	return sub, meta, nil
}

// fetchSubcategoryDetails retrieves subcategory information with parent category details
func fetchSubcategoryDetails(db DBExecutor, subcategoryID string) (*dtos.SubcategoryProducts, error) {
	var sub dtos.SubcategoryProducts
	err := db.QueryRow(`
		SELECT 
			c.category_id, c.name, c.image, c.parent_category_id,
			p.name as parent_name, p.image as parent_image
		FROM categories c
		LEFT JOIN categories p ON c.parent_category_id = p.category_id
		WHERE c.category_id = ? AND c.parent_category_id IS NOT NULL`, subcategoryID).
		Scan(&sub.ID, &sub.Name, &sub.ImageURL, &sub.ParentID, &sub.ParentCategoryName, &sub.ParentCategoryImageURL)

	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// countSubcategoryProducts counts total products in a subcategory
func countSubcategoryProducts(db DBExecutor, subcategoryID string) (int, error) {
	var totalItems int
	err := db.QueryRow(`SELECT COUNT(*) FROM products WHERE category_id = ? AND product_type = 'single'`, subcategoryID).Scan(&totalItems)
	return totalItems, err
}

// fetchSubcategoryProductList retrieves paginated products for a subcategory
func fetchSubcategoryProductList(db DBExecutor, subcategoryID string, page, size int) ([]dtos.Product, error) {
	offset := (page - 1) * size

	rows, err := db.Query(`
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
		return nil, err
	}
	defer rows.Close()

	var products []dtos.Product
	for rows.Next() {
		product, err := scanSubcategoryProduct(db, rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

// scanSubcategoryProduct scans a product row and enriches it with related data
func scanSubcategoryProduct(db DBExecutor, rows *sql.Rows) (dtos.Product, error) {
	var (
		p           dtos.Product
		detailsData []byte
	)

	if err := rows.Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price,
		&p.CategoryID, &p.StockQuantity, &p.SearchVector,
		&p.CreatedAt, &p.LastUpdated, &p.Tag, &detailsData, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
	); err != nil {
		return dtos.Product{}, err
	}

	// Unmarshal details
	if len(detailsData) > 0 {
		if err := json.Unmarshal(detailsData, &p.Details); err != nil {
			return dtos.Product{}, err
		}
	} else {
		p.Details = []string{}
	}

	// Enrich with related data
	if err := enrichSubcategoryProduct(db, &p); err != nil {
		return dtos.Product{}, err
	}

	return p, nil
}

// enrichSubcategoryProduct fetches and attaches related data to a product
func enrichSubcategoryProduct(db DBExecutor, p *dtos.Product) error {
	images, err := fetchProductImages(db, p.ID)
	if err != nil {
		return err
	}
	p.Images = images

	warranties, err := FetchProductWarranties(db, p.ID)
	if err != nil {
		return err
	}
	p.Warranty = &warranties

	features, err := fetchProductFeatures(db, p.ID)
	if err != nil {
		return err
	}
	p.Features = features

	variants, err := getProductVariants(db, p.ID)
	if err != nil {
		return err
	}
	p.ProductVariants = variants

	// tax, err := fetchProductTax(db, p.ID)
	// if err != nil {
	// 	return err
	// }
	// p.Tax = &tax
	p.VariantSelection, err = GetVariantSelection(db, p.ID)
	if err != nil {
		return err
	}

	return nil
}

// buildSubcategoryPaginationMeta creates pagination metadata for subcategory products
func buildSubcategoryPaginationMeta(page, size, totalItems int) *dtos.PaginationMeta {
	totalPages := int(math.Ceil(float64(totalItems) / float64(size)))
	return &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
}

// IsValidSubcategory checks whether a category is a valid subcategory (not a main category).
