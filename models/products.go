package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"
	"log"
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

func GetAllProducts(categoryFilter, productFilter, categoryID string, page, limit int) ([]dtos.CategoryWithProducts, *dtos.PaginationMeta, error) {
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

	categoryMap, err := processProductRows(rows)
	if err != nil {
		return nil, nil, err
	}

	result := buildResultFromCategoryMap(categoryMap, categoryID)

	pagination := calculatePagination(page, limit, totalItems)
	return result, &pagination, nil
}

// Helper to process product rows and build category map
func processProductRows(rows *sql.Rows) (map[string]*dtos.CategoryWithProducts, error) {
	categoryMap := make(map[string]*dtos.CategoryWithProducts)
	for rows.Next() {
		cat, prod, err := scanCategoryAndProduct(rows)
		if err != nil {
			return nil, err
		}
		if _, exists := categoryMap[cat.CategoryID]; !exists {
			categoryMap[cat.CategoryID] = &cat
		}
		if prod != nil {
			categoryMap[cat.CategoryID].Products = append(categoryMap[cat.CategoryID].Products, *prod)
		}
	}
	return categoryMap, nil
}

// Helper to build result from category map
func buildResultFromCategoryMap(categoryMap map[string]*dtos.CategoryWithProducts, categoryID string) []dtos.CategoryWithProducts {
	if categoryID != "" {
		if targetCat, exists := categoryMap[categoryID]; exists {
			return buildCompleteHierarchyWithParents(categoryMap, targetCat)
		}
		return []dtos.CategoryWithProducts{}
	}
	return buildCategoryHierarchy(categoryMap)
}

// Helper function to build complete hierarchy including parents for a specific category
func buildCompleteHierarchyWithParents(categoryMap map[string]*dtos.CategoryWithProducts, targetCat *dtos.CategoryWithProducts) []dtos.CategoryWithProducts {
	// First, build the hierarchy from the target category down to its children
	buildCompleteHierarchy(categoryMap, targetCat)

	// Then, build the hierarchy upwards to include all parents
	// var hierarchy []dtos.CategoryWithProducts
	currentCat := targetCat

	// Build the chain of parents
	parentChain := []*dtos.CategoryWithProducts{currentCat}
	for currentCat.ParentCategoryID != nil {
		if parent, exists := categoryMap[*currentCat.ParentCategoryID]; exists {
			parentChain = append([]*dtos.CategoryWithProducts{parent}, parentChain...)
			currentCat = parent
		} else {
			break
		}
	}

	// Now build the nested hierarchy structure
	for i := 0; i < len(parentChain)-1; i++ {
		// Clear any existing subcategories to avoid duplication
		parentChain[i].Subcategories = []*dtos.CategoryWithProducts{parentChain[i+1]}
	}

	// Return the top-level category (root of the hierarchy)
	if len(parentChain) > 0 {
		return []dtos.CategoryWithProducts{*parentChain[0]}
	}

	return []dtos.CategoryWithProducts{*targetCat}
}

// Helper function to build hierarchy downwards (children)
func buildCompleteHierarchy(categoryMap map[string]*dtos.CategoryWithProducts, targetCat *dtos.CategoryWithProducts) {
	// Clear existing subcategories to avoid duplication
	targetCat.Subcategories = []*dtos.CategoryWithProducts{}

	// Attach all direct subcategories
	for _, cat := range categoryMap {
		if cat.ParentCategoryID != nil && *cat.ParentCategoryID == targetCat.CategoryID {
			// Recursively build hierarchy for this subcategory
			buildCompleteHierarchy(categoryMap, cat)
			targetCat.Subcategories = append(targetCat.Subcategories, cat)
		}
	}
}

func buildCategoryHierarchy(categoryMap map[string]*dtos.CategoryWithProducts) []dtos.CategoryWithProducts {
	// First attach subcategories
	for _, cat := range categoryMap {
		if cat.ParentCategoryID != nil {
			if parent, ok := categoryMap[*cat.ParentCategoryID]; ok {
				parent.Subcategories = append(parent.Subcategories, cat)
			}
		}
	}

	// Then collect only top-level categories
	var topLevel []dtos.CategoryWithProducts
	for _, cat := range categoryMap {
		if cat.ParentCategoryID == nil {
			topLevel = append(topLevel, *cat)
		}
	}
	return topLevel
}

// Update the queries to fetch the complete hierarchy including parents
func buildCountQuery(categoryFilter, productFilter, categoryID string) (string, []interface{}) {
	query := `
		SELECT COUNT(DISTINCT c.category_id)
		FROM categories c
		LEFT JOIN products p ON c.category_id = p.category_id
		WHERE 1=1`
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
        SELECT COUNT(DISTINCT c.category_id)
        FROM categories c
        LEFT JOIN products p ON c.category_id = p.category_id
        WHERE 1=1
          AND c.category_id IN (
              SELECT category_id FROM ancestors
              UNION
              SELECT category_id FROM descendants
          )`
		args = append(args, categoryID, categoryID)
	}

	return query, args
}

// Alternative approach: split into two separate queries
func buildProductQuery(categoryFilter, productFilter, categoryID string, page, limit int) (string, []interface{}) {
	query := `
		SELECT 
			c.category_id, c.name, c.parent_category_id, c.description,
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, p.tag
		FROM categories c
		LEFT JOIN products p ON c.category_id = p.category_id
		WHERE 1=1`
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
		// For MySQL, we might need to handle this differently
		// Option 1: Use application logic to get all related category IDs first
		// Option 2: Use a simpler approach if hierarchy depth is limited
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

func scanCategoryAndProduct(rows *sql.Rows) (dtos.CategoryWithProducts, *dtos.Product, error) {
	var (
		catID, catName, catDesc                              string
		parentCatID                                          *string
		tag                                                  sql.NullString
		productID, name, desc, sku, categoryID, searchVector sql.NullString
		price                                                sql.NullFloat64
		stockQuantity                                        sql.NullInt64
		createdAt, updatedAt                                 sql.NullTime
	)

	if err := rows.Scan(
		&catID, &catName, &parentCatID, &catDesc,
		&productID, &name, &desc, &sku, &price, &categoryID,
		&stockQuantity, &searchVector, &createdAt, &updatedAt, &tag,
	); err != nil {
		return dtos.CategoryWithProducts{}, nil, err
	}

	category := dtos.CategoryWithProducts{
		CategoryID:       catID,
		Name:             catName,
		ParentCategoryID: parentCatID,
		Description:      catDesc,
		Products:         []dtos.Product{},
	}

	if !productID.Valid {
		return category, nil, nil
	}

	stock := 0
	if stockQuantity.Valid {
		stock = int(stockQuantity.Int64)
	}
	tagPtr := ""
	if tag.Valid {
		tagPtr = tag.String
	}
	product := dtos.Product{
		ID:            productID.String,
		Name:          name.String,
		Description:   desc.String,
		SKU:           sku.String,
		Price:         price.Float64,
		CategoryID:    categoryID.String,
		StockQuantity: stock,
		SearchVector:  searchVector.String,
		Tag:           &tagPtr,
	}

	if createdAt.Valid {
		product.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		product.LastUpdated = updatedAt.Time
	}

	images, err := fetchProductImages(product.ID)
	if err != nil {
		return category, nil, err
	}
	product.Images = images
	variants, err := getProductVariants(product.ID)
	if err != nil {
		return category, nil, err
	}
	product.ProductVariants = variants
	return category, &product, nil
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
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, c.name, p.tag
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE product_id = ?
	`

	var p dtos.Product
	err := DB.QueryRow(query, productID).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&p.CategoryName, &p.Tag,
	)
	if err != nil {
		return nil, err
	}
	// Fetch product images
	images, err := fetchProductImages(p.ID)
	if err != nil {
		return nil, err
	}
	p.Images = images
	variants, err := getProductVariants(p.ID)
	if err != nil {
		return nil, err
	}
	p.ProductVariants = variants
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
	_, err = DB.Exec(`
		INSERT INTO products (product_id, name, description, sku, price, category_id, stock_quantity, search_vector, tag, low_stock_quantity_warning, sell_when_out_of_stock, show_stock_quantity, created_by_id, buying_price)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		productID, input.Name, input.Description, input.SKU, input.Price, input.CategoryID, input.StockQuantity, input.SearchVector, input.Tag, input.LowStockAlert, sellWhenOOs, showStock, userID, input.BuyingPrice,
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
            p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, p.tag
        FROM products p
        WHERE p.category_id = ?
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

	query := "SELECT COUNT(*) FROM products p WHERE p.category_id = ?"
	args = append(args, categoryID)

	if excludeProductID != "" {
		query += " AND p.product_id != ?"
		args = append(args, excludeProductID)
	}

	return query, args
}

func scanRelatedProduct(rows *sql.Rows) (dtos.Product, error) {
	var (
		productID, name, desc, sku, categoryID, searchVector, tag sql.NullString
		price                                                     sql.NullFloat64
		stockQuantity                                             sql.NullInt64
		createdAt, updatedAt                                      sql.NullTime
	)

	if err := rows.Scan(
		&productID, &name, &desc, &sku, &price, &categoryID,
		&stockQuantity, &searchVector, &createdAt, &updatedAt, &tag,
	); err != nil {
		return dtos.Product{}, err
	}
	tagPtr := ""
	if tag.Valid {
		tagPtr = tag.String
	}
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

	// Fetch product images
	images, err := fetchProductImages(product.ID)
	if err != nil {
		return product, err
	}
	product.Images = images
	variants, err := getProductVariants(product.ID)
	if err != nil {
		return product, err
	}
	product.ProductVariants = variants
	return product, nil
}

// Get bundles
func GetBundleProducts(bundleID string, limit, page int) ([]dtos.GetBundleRequest, *dtos.PaginationMeta, error) {
	// Validate optional bundle ID
	if bundleID != "" {
		if err := isBundleThere(bundleID); err != nil {
			return nil, nil, err
		}
	}

	// Count total bundles for pagination
	countQuery := "SELECT COUNT(*) FROM product_bundles"
	var args []interface{}

	if bundleID != "" {
		countQuery += whereBundleID
		args = append(args, bundleID)
	}

	var total int
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, err
	}

	offset := (page - 1) * limit
	totalPages := (total + limit - 1) / limit

	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	// Fetch paginated bundles
	query := `
		SELECT 
			bundle_id, name, description, bundle_price, bundle_image, 
			category_id, compare_at_price, keep_selling_when_out_of_stock
		FROM product_bundles
	`
	if bundleID != "" {
		query += whereBundleID
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"

	args = append(args, limit, offset)
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var bundles []dtos.GetBundleRequest
	for rows.Next() {
		var b dtos.GetBundleRequest
		var compareAtPrice sql.NullFloat64
		var keepSelling sql.NullBool

		if err := rows.Scan(
			&b.BundleID, &b.BundleName, &b.BundleDescription, &b.BundlePrice,
			&b.BundleImage, &b.CategoryID, &compareAtPrice, &keepSelling,
		); err != nil {
			return nil, nil, err
		}

		b.CompareAtPrice = nullFloat64ToPtr(compareAtPrice)
		b.KeepSelling = &keepSelling.Bool

		//Fetch products belonging to this bundle
		products, err := getProductsForBundle(b.BundleID)
		if err != nil {
			return nil, nil, err
		}
		b.Products = products

		bundles = append(bundles, b)
	}
	log.Printf("count of bundles************************** %v", len(bundles))

	return bundles, pagination, nil
}
func getProductsForBundle(bundleID string) ([]dtos.Product, error) {
	query := `
		SELECT product_id, quantity
		FROM bundle_products
		WHERE bundle_id = ?
	`
	rows, err := DB.Query(query, bundleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.Product

	for rows.Next() {
		var productID string
		var quantity int
		if err := rows.Scan(&productID, &quantity); err != nil {
			return nil, err
		}

		// Reuse your existing reusable product function
		product, err := GetProductByID(productID)
		if err != nil {
			// Skip missing products instead of failing the entire bundle
			log.Printf("warning: failed to fetch product %s for bundle %s: %v", productID, bundleID, err)
			continue
		}

		// Override product quantity with bundle_products.quantity
		product.StockQuantity = quantity
		products = append(products, *product)
	}

	return products, nil
}

// create bundle
func CreateBundle(req dtos.Bundle) error {
	err := isCategoryThere(req.CategoryID)
	if err != nil {
		return err
	}
	var parentID *string
	err = DB.QueryRow("SELECT parent_category_id FROM categories WHERE category_id = ?", req.CategoryID).Scan(&parentID)
	if err != nil {
		return err
	}

	// 3. Prevent adding product to parent category
	if parentID == nil {
		return fmt.Errorf("cannot create bundle in a parent category, choose a subcategory instead")
	}

	bundleID, _ := shortid.Generate()
	_, err = DB.Exec(`
		INSERT INTO product_bundles (bundle_id, name, description, bundle_price, bundle_image, category_id, compare_at_price, keep_selling_when_out_of_stock)
		VALUES (?,?,?,?,?,?,?,?)
	`, bundleID, req.Name, req.Description, req.Price, req.Image, req.CategoryID, req.CompareAtPrice, req.KeepSelling)
	if err != nil {
		return err
	}
	log.Printf("already addedd bundle")
	//add products to bundle
	if len(req.Products) > 0 {
		err = AddProductsToBundle(req.Products, bundleID)
		if err != nil {
			return err
		}
	}
	return nil
}

// update bundle
func UpdateBundle(req dtos.UpdateBundle) error {
	exists, err := RecordExists("product_bundles", fetchbundle, req.ID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nobundle)
	}
	query := "UPDATE product_bundles SET"
	args := []interface{}{}
	updates := []string{}

	if req.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, req.Name)
	}
	if req.Description != "" {
		updates = append(updates, "description = ?")
		args = append(args, req.Description)
	}
	if req.Price != 0 {
		updates = append(updates, "bundle_price = ?")
		args = append(args, req.Price)
	}
	if req.Image != nil {
		updates = append(updates, "bundle_image = ?")
		args = append(args, req.Image)
	}
	if req.CategoryID != "" {
		err = isCategoryThere(req.CategoryID)
		if err != nil {
			return err
		}
		updates = append(updates, "category_id = ?")
		args = append(args, req.CategoryID)
	}
	if req.KeepSelling != nil {
		updates = append(updates, "keep_selling_when_out_of_stock = ?")
		args = append(args, *req.KeepSelling)
	}
	if req.CompareAtPrice != nil {
		updates = append(updates, "compare_at_price = ?")
		args = append(args, *req.CompareAtPrice)
	}
	if len(updates) == 0 {
		return nil // Nothing to update
	}

	query += " " + strings.Join(updates, ", ") + whereBundleID
	args = append(args, req.ID)

	if _, err := DB.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update bundle: %v", err)
	}

	return nil
}

// Delete bundle
func DeleteBundle(bundleID string) error {
	exists, err := RecordExists("product_bundles", fetchbundle, bundleID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nobundle)
	}
	query := `DELETE FROM product_bundles WHERE bundle_id = ?`
	_, err = DB.Exec(query, bundleID)
	if err != nil {
		return err
	}

	return nil
}

func AddProductsToBundle(req []dtos.BundleProducts, bundleID string) error {
	// check if bundle exists
	exists, err := RecordExists("product_bundles", fetchbundle, bundleID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nobundle)
	}

	checkQuery := `SELECT COUNT(1) FROM bundle_products WHERE bundle_id = ? AND product_id = ?`
	insertQuery := `INSERT INTO bundle_products (bundle_product_id, bundle_id, product_id, quantity) VALUES (?, ?, ?, ?)`

	for _, product := range req {
		// Check if this product already exists in the bundle
		var count int
		if err := DB.QueryRow(checkQuery, bundleID, product.ProductID).Scan(&count); err != nil {
			return err
		}

		if count > 0 {
			continue // skip if already exists
		}

		// Generate bundle_product_id
		bundleProductID, _ := shortid.Generate()

		// Insert product into bundle
		if _, err := DB.Exec(insertQuery, bundleProductID, bundleID, product.ProductID, product.Quantity); err != nil {
			return err
		}
	}

	return nil
}

func RemoveProductsFromBundle(req dtos.AddProductsToBundle, bundleID string) error {
	exists, err := RecordExists("product_bundles", fetchbundle, bundleID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nobundle)
	}
	if len(req.ProductIDs) == 0 {
		return fmt.Errorf("no products provided")
	}

	deleteQuery := `DELETE FROM bundle_products WHERE bundle_id = ? AND product_id = ?`

	for _, productID := range req.ProductIDs {
		if _, err := DB.Exec(deleteQuery, bundleID, productID); err != nil {
			return fmt.Errorf("failed to remove product %s from bundle: %v", productID, err)
		}
	}

	return nil
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
	err = DB.QueryRow(`SELECT COUNT(*) FROM products WHERE category_id = ?`, subcategoryID).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Fetch products for the subcategory with pagination
	rows, err := DB.Query(`
		SELECT 
			product_id, name, description, sku, price, category_id, 
			stock_quantity, search_vector, created_at, last_updated_at, tag
		FROM products
		WHERE category_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, subcategoryID, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var products []dtos.Product
	for rows.Next() {
		var p dtos.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price,
			&p.CategoryID, &p.StockQuantity, &p.SearchVector,
			&p.CreatedAt, &p.LastUpdated, &p.Tag,
		); err != nil {
			return nil, nil, err
		}

		// fetch product images (reusable helper)
		images, err := fetchProductImages(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Images = images
		variants, err := getProductVariants(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.ProductVariants = variants
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
			SELECT category_id, name, parent_category_id, image, description
			FROM categories
			WHERE parent_category_id IS NULL`
		args []interface{}
	)

	if filterCategoryID != "" {
		query += " AND category_id = ?"
		args = append(args, filterCategoryID)
	}

	if params.Q != "" {
		query += " AND LOWER(name) LIKE ?"
		args = append(args, "%"+strings.ToLower(params.Q)+"%")
	}

	if params.CategoryName != "" {
		query += " AND LOWER(name) LIKE ?"
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
		SELECT category_id, name, parent_category_id, image, description
		FROM categories
		WHERE parent_category_id = ?`, parentID)
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

	// Base query
	baseQuery := fmt.Sprintf(`FROM products p
		JOIN categories c ON p.category_id = c.category_id
		WHERE p.category_id IN (?%s)`, placeholders)

	args := make([]interface{}, len(subIDs))
	for i, id := range subIDs {
		args[i] = id
	}

	// Apply filters
	filterQuery, filterArgs := buildProductFilters(params)
	args = append(args, filterArgs...)
	whereClause := baseQuery + filterQuery

	// --- Count total items ---
	countQuery := "SELECT COUNT(*) " + whereClause
	var totalItems int
	if err := DB.QueryRow(countQuery, args...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// --- Fetch products ---
	sortClause := getSortClause(params.SortBy)
	dataQuery := fmt.Sprintf(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price,
			p.category_id, c.parent_category_id, p.stock_quantity,
			p.search_vector, p.created_at, p.last_updated_at, p.tag
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

		if err := rows.Scan(
			&pr.ID, &pr.Name, &pr.Description, &pr.SKU, &pr.Price,
			&subcategoryID, &parentCategoryID, &pr.StockQuantity,
			&pr.SearchVector, &pr.CreatedAt, &pr.LastUpdated, &pr.Tag,
		); err != nil {
			return nil, nil, err
		}

		pr.CategoryID = parentCategoryID
		pr.SubcategoryID = subcategoryID

		// Fetch related images and variants
		if pr.Images, err = fetchProductImages(pr.ID); err != nil {
			return nil, nil, err
		}
		if pr.ProductVariants, err = getProductVariants(pr.ID); err != nil {
			return nil, nil, err
		}

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

	if params.MinPrice > 0 && params.MaxPrice > 0 {
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
			c.name as category_name, p.tag
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE 1=1
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
	if params.MinPrice > 0 && params.MaxPrice > 0 {
		query += " AND p.price BETWEEN ? AND ?"
		args = append(args, params.MinPrice, params.MaxPrice)
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
		WHERE 1=1
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
	if params.MinPrice > 0 && params.MaxPrice > 0 {
		query += " AND p.price BETWEEN ? AND ?"
		args = append(args, params.MinPrice, params.MaxPrice)
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

// Helper to convert sql.NullFloat64 to *float64
func nullFloat64ToPtr(n sql.NullFloat64) *float64 {
	if n.Valid {
		return &n.Float64
	}
	return nil
}

func scanProduct(rows *sql.Rows, isAdmin bool) (dtos.Product, error) {
	var (
		productID, name, desc, sku, categoryID, searchVector, categoryName, tag sql.NullString
		price                                                                   sql.NullFloat64
		stockQuantity                                                           sql.NullInt64
		createdAt, updatedAt                                                    sql.NullTime
	)

	if err := rows.Scan(
		&productID, &name, &desc, &sku, &price, &categoryID,
		&stockQuantity, &searchVector, &createdAt, &updatedAt, &categoryName, &tag,
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

	variants, err := getProductVariants(product.ID)
	if err != nil {
		return product, err
	}
	product.ProductVariants = variants

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

	// Fetch product variants
	if p.ProductVariants, err = getProductVariants(p.ID); err != nil {
		return nil, err
	}

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
