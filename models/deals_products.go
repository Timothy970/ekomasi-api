// Package models provides data access layer for the Ekomasi e-commerce platform.
//
// This file handles deals/promotions management including:
//   - Deal CRUD operations (time-limited promotional campaigns)
//   - Product-deal associations (linking products to deals with specific discounts)
//   - Deal retrieval with product listings
//   - Pagination support for deal browsing
//
// Deals represent promotional campaigns with:
//   - Name uniqueness validation
//   - Start and end dates for time-limited offers
//   - Active/inactive status control
//   - Per-product discount overrides (percentage or fixed amount)
//   - Deep linking for marketing campaigns
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"os"
)

// RemoveProductFromDeal removes a product association from a deal.
//
// This function deletes the product-deal relationship, removing the product
// from the promotional deal.
//
// Parameters:
//   - dealID: The deal_id containing the product
//   - productID: The product_id to remove from the deal
//
// Returns:
//   - error: Validation error if deal or product doesn't exist, or database error
func RemoveProductFromDeal(db DBExecutor, dealID, productID string) error {
	// Validate deal exists
	if err := isDealThere(db, dealID); err != nil {
		return err
	}

	// Validate product exists
	if err := IsProductThere(db, productID); err != nil {
		return err
	}

	// Delete product-deal association
	_, err := db.Exec(`DELETE FROM deal_products WHERE deal_id = ? AND product_id = ?`, dealID, productID)
	return err
}

// GetDealWithProducts retrieves a specific deal with paginated products.
//
// This function fetches deal details and associated products with full product information
// including images, variants, warranties, and tax data.
//
// Parameters:
//   - dealID: The deal_id to retrieve
//   - page: Page number for product pagination (1-based)
//   - limit: Number of products per page
//
// Returns:
//   - *dtos.DealWithProducts: Pointer to deal with paginated products
//   - dtos.PaginationMeta: Pagination metadata for products
//   - error: "deal not found" if deal doesn't exist, or database error
//
// Deal Information:
//   - Includes deal_id, name, start/end dates, active status, and image
//   - Products include full details: variants, images, warranties, tax, specifications
//   - Link field contains deep link for marketing ({BASE_URL}/products/deals/{deal_id})
func GetDealWithProducts(db DBExecutor, dealID string, page, limit int) (*dtos.DealWithProducts, dtos.PaginationMeta, error) {
	// Validate deal exists
	if err := isDealThere(db, dealID); err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Query deal details
	query := `
		SELECT 
			d.deal_id, d.name, d.start_date, d.end_date, d.is_active, d.image, d.deal_type, d.brand_id
		FROM deals d
		WHERE d.deal_id = ?
	`

	row := db.QueryRow(query, dealID)

	var deal dtos.DealWithProducts
	var (
		name      sql.NullString
		startDate sql.NullTime
		endDate   sql.NullTime
	)

	// Scan deal fields, handling nullable columns
	if err := row.Scan(&deal.DealID, &name, &startDate, &endDate, &deal.IsActive, &deal.Image, &deal.DealType, &deal.BrandID); err != nil {
		if err == sql.ErrNoRows {
			return nil, dtos.PaginationMeta{}, fmt.Errorf("deal not found")
		}
		return nil, dtos.PaginationMeta{}, err
	}

	// Convert nullable fields to regular types
	deal.Name = name.String
	deal.StartDate = startDate.Time
	deal.EndDate = endDate.Time

	// Fetch paginated deal products with full details
	products, pagination, err := GetProductsByDealIDWithPagination(db, dealID, page, limit)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	deal.Products = products

	// Construct deep link for marketing
	baseUrl := os.Getenv("BASE_URL")
	deal.Link = fmt.Sprintf("%s/products/deals/%s", baseUrl, deal.DealID)

	return &deal, *pagination, nil
}

// GetProductsByDealID retrieves all products associated with a deal.
//
// This function fetches complete product information for all products in a deal,
// including images, variants, warranties, tax, and specifications. No pagination applied.
//
// Parameters:
//   - dealID: The deal_id to fetch products for
//
// Returns:
//   - []dtos.DealProduct: Array of products with full details and deal-specific discount
//   - error: Database error if queries fail
//
// Product Details Include:
//   - Basic info: ID, name, description, SKU, price, stock, category
//   - Deal-specific: discount and discount_type from deal_products table
//   - Specifications: weight, dimensions, manufacturer, weight_limit
//   - Related data: images, variants, warranties, tax (fetched via helper functions)
//
// TODO1: Use products pagination for all deals
func GetProductsByDealID(db DBExecutor, dealID string) ([]dtos.DealProduct, error) {
	// Query products with deal associations and specifications
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at,
			c.name AS category_name, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM deal_products dp
		INNER JOIN products p ON p.product_id = dp.product_id
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE dp.deal_id = ?
	`

	rows, err := db.Query(query, dealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.DealProduct

	// Iterate through products and enrich with related data
	for rows.Next() {
		var p dtos.DealProduct

		// Scan base product information
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
			&p.CategoryName, &p.Tag, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
		)
		if err != nil {
			return nil, err
		}

		// Fetch product images (main image and additional images)
		images, err := fetchProductImages(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Images = images

		// Fetch warranty information
		warranties, err := FetchProductWarranties(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Warranty = &warranties

		// Fetch product variants (size, color, etc.)
		variants, err := getProductVariants(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.ProductVariants = variants

		// Fetch tax information
		tax, err := fetchProductTax(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Tax = &tax

		products = append(products, p)
	}

	// Check for any row iteration errors
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

// GetProductsByDealIDWithPagination retrieves paginated products for a deal.
//
// This function is similar to GetProductsByDealID but with pagination support,
// making it suitable for deals with large numbers of products.
//
// Parameters:
//   - dealID: The deal_id to fetch products for
//   - page: Page number (1-based)
//   - limit: Number of products per page
//
// Returns:
//   - []dtos.DealProduct: Array of products with full details for the requested page
//   - *dtos.PaginationMeta: Pagination metadata (page, size, total, has_prev, has_next)
//   - error: Database error if queries fail
//
// Product Details Include:
//   - Basic info: ID, name, description, SKU, price, stock, category
//   - Deal-specific: discount and discount_type from deal_products table
//   - Specifications: weight, dimensions, manufacturer, weight_limit
//   - Related data: images, variants, warranties, tax (fetched via helper functions)
//
// Performance:
//   - Uses LIMIT/OFFSET for efficient pagination
//   - Queries total count separately for pagination metadata
func GetProductsByDealIDWithPagination(db DBExecutor, dealID string, page, limit int) ([]dtos.DealProduct, *dtos.PaginationMeta, error) {
	// Get total product count for pagination
	countQuery := `SELECT COUNT(*) FROM deal_products WHERE deal_id = ?`
	var totalItems int
	err := db.QueryRow(countQuery, dealID).Scan(&totalItems)

	// Query products with pagination
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at,
			c.name AS category_name, p.tag, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM deal_products dp
		INNER JOIN products p ON p.product_id = dp.product_id
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE dp.deal_id = ?
		LIMIT ? OFFSET ?
	`

	// Calculate pagination offset
	offset := (page - 1) * limit
	rows, err := db.Query(query, dealID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var products []dtos.DealProduct

	// Iterate through products and enrich with related data
	for rows.Next() {
		var p dtos.DealProduct

		// Scan base product information
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
			&p.CategoryName, &p.Tag, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit,
		)
		if err != nil {
			return nil, nil, err
		}

		// Fetch product images (main image and additional images)
		images, err := fetchProductImages(db, p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Images = images

		// Fetch warranty information
		warranty, err := FetchProductWarranties(db, p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Warranty = &warranty

		// Fetch product variants (size, color, etc.)
		variants, err := getProductVariants(db, p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.ProductVariants = variants

		// Fetch tax information
		tax, err := fetchProductTax(db, p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Tax = &tax

		products = append(products, p)
	}

	// Check for any row iteration errors
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Build pagination metadata
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: (totalItems + limit - 1) / limit, // Ceiling division
		HasPrev:    page > 1,
		HasNext:    page*limit < totalItems,
	}

	return products, pagination, nil
}
