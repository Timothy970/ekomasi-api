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

func GetAllProducts(categoryFilter, productFilter, categoryID string, page, limit int) ([]dtos.CategoryWithProducts, *dtos.PaginationMeta, error) {
	// Build queries
	if categoryID != "" {
		err := CategoryExists(categoryID)
		if err != nil {
			return nil, nil, err
		}
	}
	query, args := buildProductQuery(categoryFilter, productFilter, categoryID, page, limit)
	countQuery, countArgs := buildCountQuery(categoryFilter, productFilter, categoryID)

	// Fetch products and categories
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Count total items
	var totalItems int64
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// Map of categories
	categoryMap := make(map[string]*dtos.CategoryWithProducts)

	// Process rows
	for rows.Next() {
		cat, prod, err := scanCategoryAndProduct(rows)
		if err != nil {
			return nil, nil, err
		}

		// Ensure category is initialized in the map
		if _, exists := categoryMap[cat.CategoryID]; !exists {
			categoryMap[cat.CategoryID] = &cat
		}

		// Append product if exists
		if prod != nil {
			categoryMap[cat.CategoryID].Products = append(categoryMap[cat.CategoryID].Products, *prod)
		}
	}

	var result []dtos.CategoryWithProducts
	if categoryID != "" {
		// Find the specific category and build its complete hierarchy including parents
		if targetCat, exists := categoryMap[categoryID]; exists {
			// Build the complete hierarchy from root to the target category
			completeHierarchy := buildCompleteHierarchyWithParents(categoryMap, targetCat)
			result = completeHierarchy
		} else {
			// Category not found, return empty result
			result = []dtos.CategoryWithProducts{}
		}
	} else {
		// No categoryID specified, return full hierarchy
		result = buildCategoryHierarchy(categoryMap)
	}

	// Pagination
	pagination := calculatePagination(page, limit, totalItems)

	return result, &pagination, nil
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
		query += " AND LOWER(c.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(categoryFilter)+"%")
	}
	if productFilter != "" {
		query += " AND LOWER(p.name) LIKE ?"
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
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
		FROM categories c
		LEFT JOIN products p ON c.category_id = p.category_id
		WHERE 1=1`
	var args []interface{}

	if categoryFilter != "" {
		query += " AND LOWER(c.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(categoryFilter)+"%")
	}
	if productFilter != "" {
		query += " AND LOWER(p.name) LIKE ?"
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
		query += " LIMIT ? OFFSET ?"
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
		catID, catName, catDesc string
		parentCatID             *string

		productID, name, desc, sku, categoryID, searchVector sql.NullString
		price                                                sql.NullFloat64
		stockQuantity                                        sql.NullInt64
		createdAt, updatedAt                                 sql.NullTime
	)

	if err := rows.Scan(
		&catID, &catName, &parentCatID, &catDesc,
		&productID, &name, &desc, &sku, &price, &categoryID,
		&stockQuantity, &searchVector, &createdAt, &updatedAt,
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

	product := dtos.Product{
		ID:            productID.String,
		Name:          name.String,
		Description:   desc.String,
		SKU:           sku.String,
		Price:         price.Float64,
		CategoryID:    categoryID.String,
		StockQuantity: stock,
		SearchVector:  searchVector.String,
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
			product_id, name, description, sku, price, category_id,
			stock_quantity, search_vector, created_at, last_updated_at
		FROM products
		WHERE product_id = ?
	`

	var p dtos.Product
	err := DB.QueryRow(query, productID).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
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
func AddNewProduct(input dtos.CreateProduct) (*dtos.CreateProduct, error) {
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

	var parentID *string
	err = DB.QueryRow("SELECT parent_category_id FROM categories WHERE category_id = ?", input.CategoryID).Scan(&parentID)
	if err != nil {
		return nil, err
	}

	// 3. Prevent adding product to parent category
	if parentID == nil {
		return nil, fmt.Errorf("cannot add product to a parent category, choose a subcategory instead")
	}
	productID, _ := shortid.Generate()

	_, err = DB.Exec(`
		INSERT INTO products (product_id, name, description, sku, price, category_id, stock_quantity, search_vector)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		productID, input.Name, input.Description, input.SKU, input.Price, input.CategoryID, input.StockQuantity, input.SearchVector,
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
	errr := isProductThere(productID)
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
		SET name = ?, description = ?, sku = ?, price = ?, stock_quantity = ?, search_vector = ?, last_updated_at = CURRENT_TIMESTAMP
		WHERE product_id = ?`,
		input.Name, input.Description, input.SKU, input.Price, input.StockQuantity, input.SearchVector,
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
	err := isProductThere(productID)
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

func InsertProductImage(productID, imageURL string, isPrimary bool) error {
	imageID, _ := shortid.Generate()
	query := `INSERT INTO product_images (image_id, product_id, url, is_primary) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, imageID, productID, imageURL, isPrimary)
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
            p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
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
		query += fmt.Sprintf(limtOffset)
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
		productID, name, desc, sku, categoryID, searchVector sql.NullString
		price                                                sql.NullFloat64
		stockQuantity                                        sql.NullInt64
		createdAt, updatedAt                                 sql.NullTime
	)

	if err := rows.Scan(
		&productID, &name, &desc, &sku, &price, &categoryID,
		&stockQuantity, &searchVector, &createdAt, &updatedAt,
	); err != nil {
		return dtos.Product{}, err
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
func GetBundleProducts(bundleID, bundleName string, limit, page int) ([]dtos.GetBundleRequest, *dtos.PaginationMeta, error) {
	isPaginated := bundleID == "" && bundleName == ""
	log.Printf("bundle))))id  %s", bundleID)
	if bundleID != "" {
		err := isBundleThere(bundleID)
		if err != nil {
			return nil, nil, err
		}
	}
	baseQuery, args := buildBaseQuery(bundleID, bundleName)
	log.Printf("args1111%s", args)

	var pagination *dtos.PaginationMeta
	if isPaginated {
		var err error
		pagination, args, baseQuery, err = addPagination(baseQuery, args, limit, page)
		if err != nil {
			log.Printf("000000000000000 %s", err)
			return nil, nil, err
		}
	}
	query := buildSelectQuery(baseQuery)
	log.Printf("query.....%s", query)
	log.Printf("args.....%s", args)
	rows, err := DB.Query(query, args...)
	if err != nil {
		log.Printf("111111111111111111111%s", err)
		return nil, nil, err
	}
	defer rows.Close()

	bundles, err := mapBundlesWithProducts(rows)
	if err != nil {
		log.Printf("2222222222222222%s", err)
		return nil, nil, err
	}
	log.Printf("333333333333333333")

	return bundles, pagination, nil
}

func buildBaseQuery(bundleID, bundleName string) (string, []interface{}) {
	query := `
		FROM product_bundles pb
		LEFT JOIN bundle_products bp ON pb.bundle_id = bp.bundle_id
		LEFT JOIN products p ON bp.product_id = p.product_id
		WHERE 1=1
	`
	var args []interface{}

	if bundleID != "" {
		query += " AND pb.bundle_id = ?"
		args = append(args, bundleID)
	}
	if bundleName != "" {
		query += " AND LOWER(pb.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(bundleName)+"%")
	}
	return query, args
}

func addPagination(baseQuery string, args []interface{}, limit, page int) (*dtos.PaginationMeta, []interface{}, string, error) {
	var total int
	countQuery := "SELECT COUNT(DISTINCT pb.bundle_id) " + baseQuery
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, args, "", err
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

	args = append(args, limit, offset)
	baseQuery += " ORDER BY pb.bundle_id LIMIT ? OFFSET ?"
	return pagination, args, baseQuery, nil
}

func buildSelectQuery(baseQuery string) string {
	return `
		SELECT 
			pb.bundle_id, pb.name, pb.description, pb.bundle_price,
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
	` + baseQuery
}

func mapBundlesWithProducts(rows *sql.Rows) ([]dtos.GetBundleRequest, error) {
	bundleMap := make(map[string]*dtos.GetBundleRequest)

	for rows.Next() {
		bundle, product, err := scanBundleAndProduct(rows)
		if err != nil {
			return nil, err
		}

		if existing, ok := bundleMap[bundle.BundleID]; ok {
			if product != nil {
				existing.Products = append(existing.Products, *product)
			}
		} else {
			if product != nil {
				bundle.Products = []dtos.Product{*product}
			} else {
				bundle.Products = []dtos.Product{}
			}
			bundleMap[bundle.BundleID] = &bundle
		}
	}

	var bundles []dtos.GetBundleRequest
	for _, b := range bundleMap {
		bundles = append(bundles, *b)
	}
	return bundles, nil
}

// func buildBundleQuery(bundleID, bundleName string) (string, []interface{}) {
// 	var args []interface{}
// 	query := `
// 		SELECT
// 			pb.bundle_id, pb.name, pb.description, pb.bundle_price,
// 			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
// 			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
// 		FROM product_bundles pb
// 		LEFT JOIN bundle_products bp ON pb.bundle_id = bp.bundle_id
// 		LEFT JOIN products p ON bp.product_id = p.product_id
// 		WHERE 1=1
// 	`
// 	if bundleID != "" {
// 		query += " AND pb.bundle_id = ?"
// 		args = append(args, bundleID)
// 	}
// 	if bundleName != "" {
// 		query += " AND LOWER(pb.name) LIKE ?"
// 		args = append(args, "%"+strings.ToLower(bundleName)+"%")
// 	}
// 	query += " ORDER BY pb.bundle_id"
// 	return query, args
// }

func scanBundleAndProduct(rows *sql.Rows) (dtos.GetBundleRequest, *dtos.Product, error) {
	var (
		bundleID, bundleName, bundleDesc                     sql.NullString
		bundlePrice                                          sql.NullFloat64
		productID, name, desc, sku, categoryID, searchVector sql.NullString
		price                                                sql.NullFloat64
		stockQuantity                                        sql.NullInt64
		createdAt, updatedAt                                 sql.NullTime
	)

	if err := rows.Scan(
		&bundleID, &bundleName, &bundleDesc, &bundlePrice,
		&productID, &name, &desc, &sku, &price, &categoryID,
		&stockQuantity, &searchVector, &createdAt, &updatedAt,
	); err != nil {
		return dtos.GetBundleRequest{}, nil, err
	}

	bundle := dtos.GetBundleRequest{
		BundleID:          bundleID.String,
		BundleName:        bundleName.String,
		BundleDescription: bundleDesc.String,
		BundlePrice:       bundlePrice.Float64,
	}

	// if product_id is NULL, return the bundle with no product
	if !productID.Valid {
		return bundle, nil, nil
	}

	product := &dtos.Product{
		ID:            productID.String,
		Name:          name.String,
		Description:   desc.String,
		SKU:           sku.String,
		Price:         price.Float64,
		CategoryID:    categoryID.String,
		StockQuantity: int(stockQuantity.Int64),
		SearchVector:  searchVector.String,
	}

	if createdAt.Valid {
		product.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		product.LastUpdated = updatedAt.Time
	}

	images, err := fetchProductImages(product.ID)
	if err != nil {
		return dtos.GetBundleRequest{}, nil, err
	}
	product.Images = images
	variants, err := getProductVariants(product.ID)
	if err != nil {
		return dtos.GetBundleRequest{}, nil, err
	}
	product.ProductVariants = variants
	return bundle, product, nil
}

// create bundle
func CreateBundle(req dtos.Bundle) error {
	bundleID, _ := shortid.Generate()
	_, err := DB.Exec(`
		INSERT INTO product_bundles (bundle_id, name, description, bundle_price)
		VALUES (?,?,?,?)
	`, bundleID, req.Name, req.Description, req.Price)
	if err != nil {
		return err
	}
	return nil
}

// update bundle
func UpdateBundle(req dtos.UpdateBundle, bundleID string) error {
	exists, err := RecordExists("product_bundles", fetchbundle, bundleID)
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

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	query += " " + strings.Join(updates, ", ") + " WHERE bundle_id = ?"
	args = append(args, bundleID)

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

func AddProductsToBundle(req dtos.AddProductsToBundle, bundleID string) error {
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
	checkQuery := `SELECT COUNT(1) FROM bundle_products WHERE bundle_id = ? AND product_id = ?`
	insertQuery := `INSERT INTO bundle_products (bundle_product_id, bundle_id, product_id) VALUES (?, ?, ?)`

	for _, productID := range req.ProductIDs {
		bundleProductID, _ := shortid.Generate()
		var count int
		err := DB.QueryRow(checkQuery, bundleID, productID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check existence for product %s: %v", productID, err)
		}

		if count > 0 {
			continue // skip if already exists
		}

		if _, err := DB.Exec(insertQuery, bundleProductID, bundleID, productID); err != nil {
			return fmt.Errorf("failed to insert product to a bundle %s: %v", productID, err)
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
			stock_quantity, search_vector, created_at, last_updated_at
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
			&p.CreatedAt, &p.LastUpdated,
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

func GetCategoriesWithSubcategoriesAndProducts(page, size int, filterCategoryID string) ([]dtos.CategoryResponse, *dtos.PaginationMeta, error) {
	if filterCategoryID != "" {
		ok, err := IsValidCategory(filterCategoryID)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, fmt.Errorf("cannot use a subcategory ID, must be a main category")
		}
	}
	offset := (page - 1) * size

	// Count top-level categories
	totalItems, err := getTotalCategoriesCount(filterCategoryID)
	if err != nil {
		return nil, nil, err
	}

	// Get top-level categories
	categories, err := getMainCategories(filterCategoryID, size, offset)
	if err != nil {
		return nil, nil, err
	}

	// For each category, attach subcategories & products
	for i := range categories {
		subs, subIDs, err := getSubcategoriesProducts(categories[i].ID)
		if err != nil {
			return nil, nil, err
		}
		categories[i].Subcategories = subs

		if len(subIDs) > 0 {
			products, err := getProductsForSubcategories(subIDs)
			if err != nil {
				return nil, nil, err
			}
			categories[i].Products = products
		}
	}

	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: (totalItems + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < totalItems,
	}

	return categories, meta, nil
}
func getTotalCategoriesCount(filterCategoryID string) (int, error) {
	query := `SELECT COUNT(*) FROM categories WHERE parent_category_id IS NULL`
	args := []interface{}{}
	if filterCategoryID != "" {
		query += " AND category_id = ?"
		args = append(args, filterCategoryID)
	}

	var count int
	if err := DB.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func getMainCategories(filterCategoryID string, size, offset int) ([]dtos.CategoryResponse, error) {
	query := `
		SELECT category_id, name, parent_category_id, image, description
		FROM categories
		WHERE parent_category_id IS NULL`
	args := []interface{}{}
	if filterCategoryID != "" {
		query += " AND category_id = ?"
		args = append(args, filterCategoryID)
	}
	query += " ORDER BY category_id DESC LIMIT ? OFFSET ?"
	args = append(args, size, offset)

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

func getSubcategoriesProducts(parentID string) ([]dtos.SubcategoryResponse, []string, error) {
	rows, err := DB.Query(`
		SELECT category_id, name, parent_category_id, image, description
		FROM categories
		WHERE parent_category_id = ?`, parentID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var subs []dtos.SubcategoryResponse
	var ids []string
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

func getProductsForSubcategories(subIDs []string) ([]dtos.CategoryProduct, error) {
	placeholders := strings.Repeat(",?", len(subIDs)-1)
	query := fmt.Sprintf(`
		SELECT p.product_id, p.name, p.description, p.sku, p.price,
		       p.category_id, c.parent_category_id, p.stock_quantity, p.search_vector,
		       p.created_at, p.last_updated_at
		FROM products p
		JOIN categories c ON p.category_id = c.category_id
		WHERE p.category_id IN (?%s)`, placeholders)

	args := make([]interface{}, len(subIDs))
	for i, id := range subIDs {
		args[i] = id
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.CategoryProduct
	for rows.Next() {
		var pr dtos.CategoryProduct
		var subcategoryID, parentCategoryID string

		if err := rows.Scan(
			&pr.ID, &pr.Name, &pr.Description, &pr.SKU, &pr.Price,
			&subcategoryID, &parentCategoryID, &pr.StockQuantity,
			&pr.SearchVector, &pr.CreatedAt, &pr.LastUpdated,
		); err != nil {
			return nil, err
		}

		pr.CategoryID = parentCategoryID // top-level / parent category
		pr.SubcategoryID = subcategoryID // actual subcategory

		images, err := fetchProductImages(pr.ID)
		if err != nil {
			return nil, err
		}
		pr.Images = images
		variants, err := getProductVariants(pr.ID)
		if err != nil {
			return nil, err
		}
		pr.ProductVariants = variants

		products = append(products, pr)
	}
	return products, nil
}

// new fetch products
func SearchProducts(params dtos.SearchParams) ([]dtos.Product, *dtos.PaginationMeta, error) {
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
		product, err := scanProduct(rows)
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
			c.name as category_name
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE 1=1
	`
	var args []interface{}

	// Apply filters
	if params.CategoryName != "" {
		query += " AND LOWER(c.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(params.CategoryName)+"%")
	}

	if params.ProductName != "" {
		query += " AND LOWER(p.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(params.ProductName)+"%")
	}

	if params.VariantName != "" && params.VariantValue != "" {
		query += `
			AND p.product_id IN (
				SELECT pv.product_id 
				FROM product_variants pv
				JOIN variants v ON pv.variant_id = v.variant_id
				WHERE LOWER(v.variant_type) = ? AND LOWER(v.name) = ?
			)
		`
		args = append(args, strings.ToLower(params.VariantName), strings.ToLower(params.VariantValue))
	}

	// Apply sorting
	query += " ORDER BY " + getSortClause(params.SortBy)

	// Apply pagination
	if params.Limit > 0 {
		offset := (params.Page - 1) * params.Limit
		query += " LIMIT ? OFFSET ?"
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

	if params.CategoryName != "" {
		query += " AND LOWER(c.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(params.CategoryName)+"%")
	}

	if params.ProductName != "" {
		query += " AND LOWER(p.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(params.ProductName)+"%")
	}

	if params.VariantName != "" && params.VariantValue != "" {
		query += `
			AND p.product_id IN (
				SELECT pv.product_id 
				FROM product_variants pv
				JOIN variants v ON pv.variant_id = v.variant_id
				WHERE LOWER(v.variant_type) = ? AND LOWER(v.name) = ?
			)
		`
		args = append(args, strings.ToLower(params.VariantName), strings.ToLower(params.VariantValue))
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
func scanProduct(rows *sql.Rows) (dtos.Product, error) {
	var (
		productID, name, desc, sku, categoryID, searchVector, categoryName sql.NullString
		price                                                              sql.NullFloat64
		stockQuantity                                                      sql.NullInt64
		createdAt, updatedAt                                               sql.NullTime
	)

	if err := rows.Scan(
		&productID, &name, &desc, &sku, &price, &categoryID,
		&stockQuantity, &searchVector, &createdAt, &updatedAt, &categoryName,
	); err != nil {
		return dtos.Product{}, err
	}

	stock := 0
	if stockQuantity.Valid {
		stock = int(stockQuantity.Int64)
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

	return product, nil
}
