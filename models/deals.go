// Package models provides data access layer for the Adenzo e-commerce platform.
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
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"
	"os"

	"github.com/teris-io/shortid"
)

// CreateDeal creates a new promotional deal in the database.
//
// This function validates deal name uniqueness before creating the deal.
// Deals are time-limited promotional campaigns that can contain multiple products.
//
// Parameters:
//   - deal: dtos.CreateDeal containing:
//   - Name: Unique deal name (e.g., "Black Friday 2026")
//   - Description: Deal description
//   - Discount: Default discount for the deal
//   - StartDate: When the deal becomes active
//   - EndDate: When the deal expires
//   - Image: Deal banner/promotional image URL
//
// Returns:
//   - string: Generated deal_id (shortid format)
//   - error: "deal with name {name} already exists" if name exists, or database error
func CreateDeal(deal dtos.CreateDeal) (string, error) {
	// Validate deal name is unique
	exists, err := RecordExists("deals", "name = ?", deal.Name)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("deal with name %s already exists", deal.Name)
	}

	// Generate unique deal ID
	dealID, _ := shortid.Generate()

	// Insert new deal into database
	_, err = DB.Exec(`INSERT INTO deals (deal_id, name, description, discount, start_date, end_date, image) VALUES (?, ?, ?, ?, ?, ?, ?)`, dealID, deal.Name, deal.Description, deal.Discount, deal.StartDate, deal.EndDate, deal.Image)
	if err != nil {
		return "", err
	}
	return dealID, nil
}

// GetAllDeals retrieves paginated list of all deals with their products.
//
// This function fetches deals with basic information and associated products.
// Each deal includes a deep link for marketing campaigns.
//
// Parameters:
//   - page: Page number (1-based)
//   - size: Number of deals per page
//
// Returns:
//   - []dtos.Deal: Array of deals with products
//   - *dtos.PaginationMeta: Pagination metadata (page, size, total, has_prev, has_next)
//   - error: Database error if queries fail
//
// Deal Link Format:
//   - {BASE_URL}/products/deals/{deal_id}
//   - Used for promotional campaigns and marketing
func GetAllDeals(page, size int) ([]dtos.Deal, *dtos.PaginationMeta, error) {
	// Get total count for pagination calculation
	var countTotal int
	err := DB.QueryRow(`SELECT COUNT(*) FROM deals`).Scan(&countTotal)

	// Query deals with pagination
	rows, err := DB.Query(`
	SELECT deal_id, name, start_date, end_date, is_active, image
	FROM deals d
	LIMIT ? OFFSET ?
	`, size, (page-1)*size)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var deals []dtos.Deal
	// Iterate through deals and fetch associated products
	for rows.Next() {
		var d dtos.Deal
		if err := rows.Scan(&d.DealID, &d.Name, &d.StartDate, &d.EndDate, &d.IsActive, &d.Image); err != nil {
			return nil, nil, err
		}

		// Fetch all products associated with this deal
		products, err := GetProductsByDealID(d.DealID)
		if err != nil {
			return nil, nil, err
		}

		// Generate deep link for this deal
		baseUrl := os.Getenv("BASE_URL")
		d.Link = fmt.Sprintf("%s/products/deals/%s", baseUrl, d.DealID)
		d.Products = products
		deals = append(deals, d)
	}

	// Build pagination metadata
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: countTotal,
		TotalPages: (countTotal + size - 1) / size, // Ceiling division
		HasPrev:    page > 1,
		HasNext:    page*size < countTotal,
	}
	return deals, pagination, nil
}

// isDealThere validates that a deal exists by deal_id.
//
// This is an internal validation helper used by other deal functions.
// Returns an error (rather than bool) for easier use in validation chains.
//
// Parameters:
//   - dealID: The deal_id to validate
//
// Returns:
//   - error: nil if deal exists, "deal with ID {id} does not exist" error if not found,
//     or database error if query fails
func isDealThere(dealID string) error {
	// Check if deal record exists in database
	if exists, err := RecordExists("deals", "deal_id = ?", dealID); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("deal with ID %s does not exist", dealID)
	}
	return nil
}

// UpdateDeal updates an existing deal's information.
//
// This function validates deal existence before updating. It handles optional
// image updates - if image is nil, the existing image is preserved.
//
// Parameters:
//   - dealID: The deal_id to update
//   - deal: dtos.Deal containing:
//   - Name: Updated deal name
//   - StartDate: Updated start date
//   - EndDate: Updated end date
//   - IsActive: Pointer to is_active status (activate/deactivate deal)
//   - Image: Optional pointer to new image URL (nil preserves existing image)
//
// Returns:
//   - error: "deal with ID {id} does not exist" if deal not found, or database error
func UpdateDeal(dealID string, deal dtos.Deal) error {
	// Validate deal exists
	if err := isDealThere(dealID); err != nil {
		return err
	}

	// If image is provided, update including image field
	if deal.Image != nil {
		_, err := DB.Exec(`UPDATE deals SET name = ?, start_date = ?, end_date = ?, is_active = ?, image = ? WHERE deal_id = ?`,
			deal.Name, deal.StartDate, deal.EndDate, *deal.IsActive, deal.Image, dealID)
		return err
	}

	// Otherwise, update without changing image
	_, err := DB.Exec(`UPDATE deals SET name = ?, start_date = ?, end_date = ?, is_active = ? WHERE deal_id = ?`,
		deal.Name, deal.StartDate, deal.EndDate, *deal.IsActive, dealID)
	return err
}

// DeleteDeal removes a deal from the database.
//
// This function validates deal existence before deletion.
//
// Parameters:
//   - dealID: The deal_id to delete
//
// Returns:
//   - error: "deal with ID {id} does not exist" if deal not found, or database error
//
// Important:
//   - No check for associated deal_products - deletion may fail if foreign key constraints exist
//   - Consider implementing cascade delete or removing products first
//   - May want to implement soft delete for audit trail
func DeleteDeal(dealID string) error {
	// Validate deal exists
	if err := isDealThere(dealID); err != nil {
		return err
	}

	// Delete deal from database
	_, err := DB.Exec(`DELETE FROM deals WHERE deal_id = ?`, dealID)
	return err
}

// AddProductToDeal associates a product with a deal or updates existing association.
//
// This function creates or updates a product-deal relationship with custom discount.
// The discount overrides the deal's default discount for this specific product.
//
// Parameters:
//   - dealID: The deal_id to associate with
//   - productID: The product_id to add to the deal
//   - discountType: Pointer to discount type ("percentage" or "fixed")
//   - discount: Pointer to discount value (e.g., 25.0 for 25% or $25)
//
// Returns:
//   - error: Validation error if deal or product doesn't exist, or database error
//
// Behavior:
//   - If product-deal association exists, updates the discount and discount_type (idempotent)
//   - If association doesn't exist, creates new entry with generated product_deal_id
//   - Validates both deal and product existence before operation
func AddProductToDeal(dealID, productID string, discountType *string, discount *float64) error {
	// Validate deal exists
	if err := isDealThere(dealID); err != nil {
		return err
	}

	// Validate product exists
	if err := IsProductThere(productID); err != nil {
		return err
	}

	// Check if product is already associated with this deal
	exists, err := RecordExists("deal_products", "deal_id = ? AND product_id = ?", dealID, productID)
	if err != nil {
		return err
	}

	if exists {
		// Update existing product-deal association (idempotent operation)
		_, err := DB.Exec(`UPDATE deal_products SET discount = ?, discount_type = ? WHERE deal_id = ? AND product_id = ?`,
			discount, discountType, dealID, productID)
		return err
	}

	// Create new product-deal association
	productDealID, _ := shortid.Generate()
	_, err = DB.Exec(`INSERT INTO deal_products (product_deal_id, deal_id, product_id, discount, discount_type) VALUES (?, ?, ?, ?, ?)`,
		productDealID, dealID, productID, discount, discountType)
	return err
}

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
func RemoveProductFromDeal(dealID, productID string) error {
	// Validate deal exists
	if err := isDealThere(dealID); err != nil {
		return err
	}

	// Validate product exists
	if err := IsProductThere(productID); err != nil {
		return err
	}

	// Delete product-deal association
	_, err := DB.Exec(`DELETE FROM deal_products WHERE deal_id = ? AND product_id = ?`, dealID, productID)
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
func GetDealWithProducts(dealID string, page, limit int) (*dtos.DealWithProducts, dtos.PaginationMeta, error) {
	// Validate deal exists
	if err := isDealThere(dealID); err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Query deal details
	query := `
		SELECT 
			d.deal_id, d.name, d.start_date, d.end_date, d.is_active, d.image
		FROM deals d
		WHERE d.deal_id = ?
	`

	row := DB.QueryRow(query, dealID)

	var deal dtos.DealWithProducts
	var (
		name      sql.NullString
		startDate sql.NullTime
		endDate   sql.NullTime
	)

	// Scan deal fields, handling nullable columns
	if err := row.Scan(&deal.DealID, &name, &startDate, &endDate, &deal.IsActive, &deal.Image); err != nil {
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
	products, pagination, err := GetProductsByDealIDWithPagination(dealID, page, limit)
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
func GetProductsByDealID(dealID string) ([]dtos.DealProduct, error) {
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

	rows, err := DB.Query(query, dealID)
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
		images, err := fetchProductImages(p.ID)
		if err != nil {
			return nil, err
		}
		p.Images = images

		// Fetch warranty information
		warranties, err := FetchProductWarranties(p.ID)
		if err != nil {
			return nil, err
		}
		p.Warranty = &warranties

		// Fetch product variants (size, color, etc.)
		variants, err := getProductVariants(p.ID)
		if err != nil {
			return nil, err
		}
		p.ProductVariants = variants

		// Fetch tax information
		tax, err := fetchProductTax(p.ID)
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
func GetProductsByDealIDWithPagination(dealID string, page, limit int) ([]dtos.DealProduct, *dtos.PaginationMeta, error) {
	// Get total product count for pagination
	countQuery := `SELECT COUNT(*) FROM deal_products WHERE deal_id = ?`
	var totalItems int
	err := DB.QueryRow(countQuery, dealID).Scan(&totalItems)

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
	rows, err := DB.Query(query, dealID, limit, offset)
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
		images, err := fetchProductImages(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Images = images

		// Fetch warranty information
		warranty, err := FetchProductWarranties(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Warranty = &warranty

		// Fetch product variants (size, color, etc.)
		variants, err := getProductVariants(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.ProductVariants = variants

		// Fetch tax information
		tax, err := fetchProductTax(p.ID)
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
