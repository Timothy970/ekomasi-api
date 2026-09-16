package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
)

func getPromotionProductGroups(db DBExecutor, promotionID string) ([]dtos.PromotionProductGroup, error) {
	// Query promotion-product associations
	query := `
		SELECT promotion_product_id, promotion_id, product_id 
		FROM promotion_products 
		WHERE promotion_id = ?
	`

	rows, err := db.Query(query, promotionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []dtos.PromotionProductGroup

	// Iterate through promotion products
	for rows.Next() {
		var pp dtos.PromotionProductGroup
		if err := rows.Scan(&pp.ID, &pp.PromotionID, &pp.ProductID); err != nil {
			return nil, err
		}

		// Fetch full product details with category
		product, category, err := getProductWithCategory(db, pp.ProductID)
		if err != nil {
			return nil, err
		}

		// Group product under its category
		catMap := make(map[string]*dtos.CategoryGroup)
		catID := category.CategoryID
		if _, exists := catMap[catID]; !exists {
			catMap[catID] = &dtos.CategoryGroup{
				CategoryID:       category.CategoryID,
				Name:             category.Name,
				ParentCategoryID: category.ParentCategoryID,
				Description:      category.Description,
				Products:         []dtos.Product{},
			}
		}
		catMap[catID].Products = append(catMap[catID].Products, product)

		// Convert category map to slice
		for _, cat := range catMap {
			pp.Categories = append(pp.Categories, *cat)
		}

		groups = append(groups, pp)
	}

	return groups, nil
}

// getProductWithCategory retrieves a product with its category information and images.
//
// This is an internal helper function that fetches full product details including
// its associated category and product images.
//
// Parameters:
//   - productID: The product_id to retrieve
//
// Returns:
//   - dtos.Product: Product with images
//   - dtos.CategoryGroup: Associated category information
//   - error: Database error if queries fail
//
// Product Fields:
//   - Basic: ID, Name, Description, SKU, Price, CategoryID
//   - Inventory: StockQuantity
//   - Metadata: SearchVector, CreatedAt, LastUpdated
//   - Images: Array of product images with primary flag
//
// Category Fields:
//   - CategoryID, Name, ParentCategoryID, Description
func getProductWithCategory(db DBExecutor, productID string) (dtos.Product, dtos.CategoryGroup, error) {
	// Query product with joined category data
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id, 
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at,
			c.category_id, c.name, c.parent_category_id, c.description
		FROM products p
		JOIN categories c ON p.category_id = c.category_id
		WHERE p.product_id = ?
	`

	var product dtos.Product
	var category dtos.CategoryGroup

	// Scan product and category data
	row := db.QueryRow(query, productID)
	err := row.Scan(
		&product.ID, &product.Name, &product.Description, &product.SKU, &product.Price,
		&product.CategoryID, &product.StockQuantity, &product.SearchVector,
		&product.CreatedAt, &product.LastUpdated,
		&category.CategoryID, &category.Name, &category.ParentCategoryID, &category.Description,
	)
	if err != nil {
		return dtos.Product{}, dtos.CategoryGroup{}, err
	}

	// Fetch associated product images
	imageRows, err := db.Query(`
		SELECT image_id, url, is_primary 
		FROM product_images 
		WHERE product_id = ?`, productID,
	)
	if err != nil {
		return dtos.Product{}, dtos.CategoryGroup{}, err
	}
	defer imageRows.Close()

	var images []dtos.Image
	// Collect product images
	for imageRows.Next() {
		var img dtos.Image
		if err := imageRows.Scan(&img.ImageID, &img.URL, &img.IsPrimary); err != nil {
			return dtos.Product{}, dtos.CategoryGroup{}, err
		}
		images = append(images, img)
	}
	product.Images = images

	return product, category, nil
}

// func groupByCategory(details []struct {
// 	dtos.PromotionProduct
// 	Product  dtos.Product
// 	Category dtos.Category
// }) []dtos.CategoryGroup {
// 	categoryMap := make(map[string]*dtos.CategoryGroup)

// 	for _, d := range details {
// 		catID := d.Category.ID
// 		if _, exists := categoryMap[catID]; !exists {
// 			categoryMap[catID] = &dtos.CategoryGroup{
// 				CategoryID:       d.Category.ID,
// 				Name:             d.Category.Name,
// 				ParentCategoryID: d.Category.ParentCategoryID,
// 				Description:      d.Category.Description,
// 				Products:         []dtos.Product{},
// 			}
// 		}
// 		categoryMap[catID].Products = append(categoryMap[catID].Products, d.Product)
// 	}

// 	var categories []dtos.CategoryGroup
// 	for _, c := range categoryMap {
// 		categories = append(categories, *c)
// 	}

// 	return categories
// }

// GetCategoriesWithProducts retrieves all categories with their associated products.
//
// This function fetches the complete category-product hierarchy, enriching each product
// with images, warranties, and tax information. Products may include deal discounts.
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.CategoryWithProducts: Array of categories with nested product arrays
//   - error: Database error if queries fail
//
// Product Enrichment:
//   - Images: Product images with primary flag
//   - Warranty: Warranty details with type and dates
//   - Tax: Associated tax/charge information
//   - Discount: Deal-specific discount if applicable
//   - Specifications: Weight, dimensions, manufacturer, weight limit
func GetCategoriesWithProducts(db DBExecutor) ([]dtos.CategoryWithProducts, error) {
	// Query categories with products, deals, and specifications
	rows, err := db.Query(`
		SELECT 
			c.category_id, c.name, c.parent_category_id, c.description,
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM categories c
		LEFT JOIN products p ON c.category_id = p.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		ORDER BY c.category_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Use map to group products by category
	categoryMap := make(map[string]*dtos.CategoryWithProducts)

	for rows.Next() {
		// Scan category and product data
		catID, catName, catDesc, parentCatID, product, err := scanCategoryProductRow(rows)
		if err != nil {
			return nil, err
		}

		// Ensure category exists in map
		if _, exists := categoryMap[catID]; !exists {
			categoryMap[catID] = &dtos.CategoryWithProducts{
				CategoryID:       catID,
				Name:             catName,
				ParentCategoryID: parentCatID,
				Description:      catDesc,
				Products:         []dtos.Product{},
			}
		}

		// If product exists, enrich with images, warranty, and tax
		if product.ID != "" {
			enrichedProduct, err := enrichProductDetails(db, product)
			if err != nil {
				return nil, err
			}
			categoryMap[catID].Products = append(categoryMap[catID].Products, enrichedProduct)
		}
	}

	// Convert category map to slice
	var result []dtos.CategoryWithProducts
	for _, cat := range categoryMap {
		result = append(result, *cat)
	}

	return result, nil
}

// enrichProductDetails enriches a product with images, warranty, and tax information.
//
// This is an internal helper function that fetches and attaches supplementary
// product data to reduce cognitive complexity in parent functions.
//
// Parameters:
//   - product: Base product data to enrich
//
// Returns:
//   - dtos.Product: Enriched product with images, warranty, and tax
//   - error: Database error if any enrichment query fails
func enrichProductDetails(db DBExecutor, product dtos.Product) (dtos.Product, error) {
	// Fetch product images
	images, err := fetchProductImages(db, product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Images = images

	// Fetch warranty information
	warranty, err := FetchProductWarranties(db, product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Warranty = &warranty

	// Fetch tax/charge information
	tax, err := fetchProductTax(db, product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Tax = &tax

	return product, nil
}

// scanCategoryProductRow scans a row from the category-product JOIN query.
//
// This is an internal helper function that extracts category and product data
// from a database row.
//
// Parameters:
//   - rows: SQL rows iterator
//
// Returns:
//   - string: Category ID
//   - string: Category name
//   - string: Category description
//   - *string: Parent category ID (nullable)
//   - dtos.Product: Product data (may have empty ID if no product)
//   - error: Scan error if field extraction fails
func scanCategoryProductRow(rows *sql.Rows) (string, string, string, *string, dtos.Product, error) {
	var (
		catID, catName, catDesc string
		parentCatID             *string
		p                       dtos.Product
	)
	err := rows.Scan(
		&catID, &catName, &parentCatID, &catDesc,
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
	)
	return catID, catName, catDesc, parentCatID, p, err
}

// fetchProductImages retrieves all images for a specific product.
//
// This is an internal helper function that fetches product images including
// the primary image and additional images.
//
// Parameters:
//   - productID: The product_id to fetch images for
//
// Returns:
//   - []dtos.Image: Array of images with URL, primary flag, and type
//   - error: Database error if query fails
//
// Image Fields:
//   - ImageID: Unique image identifier
//   - URL: Image URL path
//   - IsPrimary: Boolean indicating main product image
//   - Type: Image type/category
