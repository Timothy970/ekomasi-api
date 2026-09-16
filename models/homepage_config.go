package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"fmt"
)

func DeleteMenuLink(menuLinkID int) error {
	// Verify menu link exists
	exists, err := RecordExists(DB, "menu_links", whereID, menuLinkID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("menu link not found")
	}

	// Delete menu link record
	query := `DELETE FROM menu_links WHERE id=?`
	_, err = DB.Exec(query, menuLinkID)
	return err
}

// CreateSocialLink creates a new social media link with icon and display order.
//
// This function creates social media links for the website footer/header.
// Platform names must be unique.
//
// Parameters:
//   - link: *dtos.SocialLinkRequest containing:
//   - Platform: Social media platform name (must be unique, e.g., "facebook", "twitter")
//   - URL: Link to social media profile
//   - IconClass: CSS class for icon display (e.g., "fa-facebook")
//   - DisplayOrder: Position ordering
//
// Returns:
//   - error: "platform already exists" if platform is duplicate,
//     or database error
func CreateSocialLink(link *dtos.SocialLinkRequest) error {
	// Validate platform uniqueness
	exists, err := RecordExists(DB, "social_links", "platform = ?", link.Platform)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("platform already exists")
	}

	// Insert social link record
	query := `INSERT INTO social_links (platform, url, icon_class, display_order) VALUES (?, ?, ?, ?)`
	_, err = DB.Exec(query, link.Platform, link.URL, link.IconClass, link.DisplayOrder)
	if err != nil {
		return err
	}

	return nil
}

// UpdateSocialLink updates an existing social media link's properties.
//
// Parameters:
//   - socialID: The social link ID to update
//   - link: dtos.SocialLinkRequest with updated values:
//   - Platform: Updated platform name
//   - URL: Updated profile URL
//   - IconClass: Updated icon CSS class
//   - DisplayOrder: Updated position
//
// Returns:
//   - error: "social not found" if link doesn't exist or database error
func UpdateSocialLink(socialID int, link dtos.SocialLinkRequest) error {
	// Verify social link exists
	exists, err := RecordExists(DB, "social_links", whereID, socialID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("social not found")
	}

	// Update social link properties
	query := `UPDATE social_links SET platform = ?, url = ?, icon_class = ?, display_order = ? WHERE id = ?`
	_, err = DB.Exec(query, link.Platform, link.URL, link.IconClass, link.DisplayOrder, socialID)
	return err
}

// DeleteSocialLink permanently removes a social media link.
//
// Parameters:
//   - id: The social link ID to delete
//
// Returns:
//   - error: "social not found" if link doesn't exist or database error
func DeleteSocialLink(id int) error {
	// Verify social link exists
	exists, err := RecordExists(DB, "social_links", whereID, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("social not found")
	}

	// Delete social link record
	query := `DELETE FROM social_links WHERE id = ?`
	_, err = DB.Exec(query, id)
	return err
}

// AddFeaturedProduct marks a product as featured for homepage display.
//
// This function adds a product to the featured products list, typically displayed
// prominently on the homepage or landing pages.
//
// Parameters:
//   - productID: The product_id to feature
//
// Returns:
//   - error: "product not found" if product doesn't exist,
//     or database error (including duplicate if already featured)
func AddFeaturedProduct(productID string) error {
	// Verify product exists
	exists, err := RecordExists(DB, "products", "product_id =? ", productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not found")
	}

	// Insert featured product record
	query := `INSERT INTO featured_products (product_id) VALUES (?)`
	_, err = DB.Exec(query, productID)
	return err
}

// RemoveFeaturedProduct removes a product from the featured list.
//
// Parameters:
//   - productID: The product_id to unfeature
//
// Returns:
//   - error: "product not found" if product doesn't exist or database error
func RemoveFeaturedProduct(productID string) error {
	// Verify product exists
	exists, err := RecordExists(DB, "products", "product_id =? ", productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not found")
	}

	// Delete featured product record
	query := `DELETE FROM featured_products WHERE product_id = ?`
	_, err = DB.Exec(query, productID)
	return err
}

// GetFeaturedProducts retrieves all featured products with full enrichment.
//
// This function fetches featured products with complete details including images,
// warranties, tax info, specifications, and active deal discounts. Products are
// ordered by featured date (most recent first).
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.Product: Array of enriched products with:
//   - Basic product info (name, price, description, SKU)
//   - Images: All product images with primary flag
//   - Warranty: Most recent warranty with type and dates
//   - Tax: First tax/charge associated with product
//   - Specifications: Weight, dimensions, manufacturer, weight limit
//   - Discount: Active deal discount if applicable (percentage or fixed)
//   - error: Database error if query fails
func GetFeaturedProducts() ([]dtos.Product, error) {
	// Query featured products with JOIN to get product details and deals
	rows, err := DB.Query(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, dp.discount, dp.discount_type, ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit
		FROM featured_products fp
		JOIN products p ON fp.product_id = p.product_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		ORDER BY fp.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var featured []dtos.Product

	// Iterate through featured product rows
	for rows.Next() {
		// Scan product basic info and specifications
		product, err := scanFeaturedProductRow(rows)
		if err != nil {
			return nil, err
		}

		// Enrich product with images
		images, err := fetchProductImages(DB, product.ID)
		if err != nil {
			return nil, err
		}
		product.Images = images

		// Enrich product with warranty info
		warranty, err := FetchProductWarranties(DB, product.ID)
		if err != nil {
			return nil, err
		}
		product.Warranty = &warranty

		// Enrich product with tax/charge info
		tax, err := fetchProductTax(DB, product.ID)
		if err != nil {
			return nil, err
		}
		product.Tax = &tax

		featured = append(featured, product)
	}

	return featured, nil
}

// scanFeaturedProductRow deserializes a featured product result row.
//
// This is an internal helper function that scans product data including
// specifications and deal discounts from a JOIN query result.
//
// Parameters:
//   - rows: Result set from GetFeaturedProducts query
//
// Returns:
//   - dtos.Product: Product with basic info, specs, and discount
//   - error: Scan error if column types mismatch
//
// Scanned Fields:
//   - Product: ID, Name, Description, SKU, Price, CategoryID, StockQuantity
//   - Metadata: SearchVector, CreatedAt, LastUpdated
//   - Deal: Discount, DiscountType (nullable)
//   - Specs: Weight, Dimensions, Manufacturer, WeightLimit (nullable)
func scanFeaturedProductRow(rows *sql.Rows) (dtos.Product, error) {
	var p dtos.Product

	// Scan all product fields from JOIN query
	err := rows.Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&p.Discount, &p.DiscountType, // From deal_products LEFT JOIN
		&p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit, // From product_specifications LEFT JOIN
	)
	return p, err
}

func ListSocialLinks() ([]dtos.SocialLinkRequest, error) {
	rows, err := DB.Query(`SELECT id, platform, url, icon_class, display_order FROM social_links ORDER BY display_order ASC`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No social links found, return empty slice
		}
		return nil, fmt.Errorf("failed to query social links: %w", err)
	}
	defer rows.Close()

	var links []dtos.SocialLinkRequest
	for rows.Next() {
		var link dtos.SocialLinkRequest
		if err := rows.Scan(&link.ID, &link.Platform, &link.URL, &link.IconClass, &link.DisplayOrder); err != nil {
			return nil, fmt.Errorf("failed to scan social link: %w", err)
		}
		links = append(links, link)
	}
	return links, nil
}
