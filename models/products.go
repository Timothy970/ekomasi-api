package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"
	"strings"

	"github.com/teris-io/shortid"
)

var nobundle = "bundle not found"
var fetchbundle = "bundle_id = ?"

func GetAllProducts(categoryFilter, productFilter, categoryID string, page, limit int) ([]dtos.CategoryWithProducts, *dtos.PaginationMeta, error) {
	// Build main query for fetching data
	query, args := buildProductQuery(categoryFilter, productFilter, categoryID, page, limit)

	// Build count query for total items
	countQuery, countArgs := buildCountQuery(categoryFilter, productFilter, categoryID)

	// Execute main query
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Execute count query
	var totalItems int
	err = DB.QueryRow(countQuery, countArgs...).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Process rows (your existing code)
	categoryMap := make(map[string]*dtos.CategoryWithProducts)

	for rows.Next() {
		cat, prod, err := scanCategoryAndProduct(rows)
		if err != nil {
			return nil, nil, err
		}

		if _, exists := categoryMap[cat.CategoryID]; !exists {
			categoryMap[cat.CategoryID] = &cat
		}

		if prod != nil {
			categoryMap[cat.CategoryID].Products = append(categoryMap[cat.CategoryID].Products, *prod)
		}
	}

	// Build hierarchy
	var topLevel []dtos.CategoryWithProducts
	for _, cat := range categoryMap {
		if cat.ParentCategoryID != nil {
			parent, ok := categoryMap[*cat.ParentCategoryID]
			if ok {
				parent.Subcategories = append(parent.Subcategories, *cat)
			}
		} else {
			topLevel = append(topLevel, *cat)
		}
	}

	// Calculate pagination metadata
	pagination := calculatePagination(page, limit, totalItems)

	return topLevel, &pagination, nil
}
func buildCountQuery(categoryFilter, productFilter, categoryID string) (string, []interface{}) {
	var args []interface{}
	query := `
        SELECT COUNT(DISTINCT c.category_id)
        FROM categories c
        LEFT JOIN products p ON c.category_id = p.category_id
        WHERE 1=1
    `

	if categoryFilter != "" {
		query += " AND LOWER(c.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(categoryFilter)+"%")
	}
	if productFilter != "" {
		query += " AND LOWER(p.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(productFilter)+"%")
	}
	if categoryID != "" {
		query += " AND c.category_id = ?"
		args = append(args, categoryID)
	}

	return query, args
}

func buildProductQuery(categoryFilter, productFilter, categoryID string, page, limit int) (string, []interface{}) {
	var args []interface{}
	query := `
        SELECT 
            c.category_id, c.name, c.parent_category_id, c.description,
            p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
            p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
        FROM categories c
        LEFT JOIN products p ON c.category_id = p.category_id
        WHERE 1=1
    `

	if categoryFilter != "" {
		query += " AND LOWER(c.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(categoryFilter)+"%")
	}
	if productFilter != "" {
		query += " AND LOWER(p.name) LIKE ?"
		args = append(args, "%"+strings.ToLower(productFilter)+"%")
	}
	if categoryID != "" {
		query += " AND c.category_id = ?"
		args = append(args, categoryID)
	}

	query += " ORDER BY c.category_id"

	// Pagination
	if limit > 0 {
		offset := (page - 1) * limit
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	return query, args
}
func calculatePagination(page, limit, totalItems int) dtos.PaginationMeta {
	if limit <= 0 {
		limit = 10 // default limit
	}
	if page <= 0 {
		page = 1 // default page
	}

	totalPages := 0
	if limit > 0 {
		totalPages = (totalItems + limit - 1) / limit
	}

	return dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
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

	images, err := fetchProductImages(product.ID)
	if err != nil {
		return category, nil, err
	}
	product.Images = images

	return category, &product, nil
}

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
	exists, err := CategoryExists(input.CategoryID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("category not forund")
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
		SET name = ?, description = ?, sku = ?, price = ?, category_id = ?, stock_quantity = ?, search_vector = ?, last_updated_at = CURRENT_TIMESTAMP
		WHERE product_id = ?`,
		input.Name, input.Description, input.SKU, input.Price,
		input.CategoryID, input.StockQuantity, input.SearchVector,
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
	// 1. Check if product is used in any uncollected order
	query := `
		SELECT COUNT(*) 
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE oi.product_id = ? AND o.status != 'collected'
	`
	var count int
	err := DB.QueryRow(query, productID).Scan(&count)
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
	var totalItems int
	err = DB.QueryRow(countQuery, countArgs...).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Process products
	var relatedProducts []dtos.Product
	for rows.Next() {
		product, err := scanProduct(rows)
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
		query += " LIMIT ? OFFSET ?"
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

func scanProduct(rows *sql.Rows) (dtos.Product, error) {
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

	return product, nil
}

// Get bundles
func GetBundleProducts(bundleID, bundleName string, limit, page int) ([]dtos.GetBundleRequest, *dtos.PaginationMeta, error) {
	isPaginated := bundleID == "" && bundleName == ""

	baseQuery, args := buildBaseQuery(bundleID, bundleName)

	var pagination *dtos.PaginationMeta
	if isPaginated {
		var err error
		pagination, args, err = addPagination(baseQuery, args, limit, page)
		if err != nil {
			return nil, nil, err
		}
	}

	query := buildSelectQuery(baseQuery)
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	bundles, err := mapBundlesWithProducts(rows)
	if err != nil {
		return nil, nil, err
	}

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

func addPagination(baseQuery string, args []interface{}, limit, page int) (*dtos.PaginationMeta, []interface{}, error) {
	var total int
	countQuery := "SELECT COUNT(DISTINCT pb.bundle_id) " + baseQuery
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, args, err
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
	baseQuery += " LIMIT ? OFFSET ?"
	return pagination, args, nil
}

func buildSelectQuery(baseQuery string) string {
	return `
		SELECT 
			pb.bundle_id, pb.name, pb.description, pb.bundle_price,
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
	` + baseQuery + " ORDER BY pb.bundle_id"
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
