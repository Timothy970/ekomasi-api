// Package models provides data access layer for the Adenzo e-commerce platform.
//
// This file handles homepage and content management including:
//   - Footer data (copyright, contact info)
//   - Social media links
//   - Menu/navigation links
//   - Banners (hero images, promotional banners)
//   - Promotions (time-limited offers with product associations)
//   - Categories with products
//   - Blog management (CRUD operations with draft/published status)
//   - Featured products
//
// Key Features:
//   - Dynamic banner management with display order
//   - Active promotions filtered by date range
//   - JSON storage for flexible blog content structure
//   - Product associations with warranties, images, tax, and specifications
package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

// Package-level constants for reusable query clauses
var fetchblog = "blog_id = ?"
var noblog = "blog not found"
var whereID = "id = ?"

// GetFooterData retrieves footer contact information for the website.
//
// This function fetches company contact details displayed in the website footer.
// Limited to 1 record (single footer configuration).
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.Footer: Array of footer data (typically contains 1 element)
//   - error: Database error if query fails
//
// Footer Information:
//   - CopyrightText: Copyright notice
//   - CompanyAddress: Physical address
//   - ContactEmail: Contact email address
//   - PhoneNumber: Contact phone number
func GetFooterData() ([]dtos.Footer, error) {
	// Query footer/contact information (limited to 1 record)
	rows, err := DB.Query("SELECT copyright_text, company_address, contact_email, phone_number FROM contact_info LIMIT 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []dtos.Footer
	// Collect footer data
	for rows.Next() {
		var f dtos.Footer
		if err := rows.Scan(&f.CopyrightText, &f.CompanyAddress, &f.ContactEmail, &f.PhoneNumber); err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, nil
}

// GetSocialsData retrieves social media links for the website.
//
// This function fetches social media platform links (Facebook, Twitter, Instagram, etc.)
// displayed in the website footer or header. Results are ordered by display_order.
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.SocialLink: Array of social links with platform, URL, and icon class
//   - error: Database error if query fails
//
// Social Link Fields:
//   - Platform: Platform name (e.g., "Facebook", "Twitter")
//   - URL: Link to social media profile
//   - IconClass: CSS class for icon (e.g., "fa-facebook", "fa-twitter")
func GetSocialsData() ([]dtos.SocialLink, error) {
	// Query social links ordered by display preference
	rows, err := DB.Query("SELECT platform, url, icon_class FROM social_links ORDER BY display_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []dtos.SocialLink
	// Collect social media links
	for rows.Next() {
		var l dtos.SocialLink
		if err := rows.Scan(&l.Platform, &l.URL, &l.IconClass); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

// GetMenuData retrieves navigation menu links from static pages.
//
// This function fetches menu links for the website navigation, pulling from
// static pages ordered by most recently updated.
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.MenuLink: Array of menu links with title and path
//   - error: Database error if query fails
//
// Menu Link Fields:
//   - Title: Display text for the menu item
//   - HREF: URL path for the menu item
func GetMenuData() ([]dtos.MenuLink, error) {
	// Query static pages for menu navigation (most recent first)
	rows, err := DB.Query("SELECT  title, path FROM static_pages ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []dtos.MenuLink
	// Collect menu links
	for rows.Next() {
		var l dtos.MenuLink
		if err := rows.Scan(&l.Title, &l.HREF); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

// GetBannersData retrieves active banners filtered by type.
//
// This function fetches banners for specific sections of the website (hero, sidebar, etc.).
// Only active banners are returned, ordered by display_order for consistent positioning.
//
// Parameters:
//   - value: Banner type filter (e.g., "hero", "sidebar", "footer")
//
// Returns:
//   - []dtos.Banner: Array of active banners of the specified type
//   - error: Database error if query fails
//
// Banner Fields:
//   - ID, ImageURL, Text, Heading: Visual content
//   - ButtonText, ButtonURL: Call-to-action button
//   - DisplayOrder: Position ordering
//   - IsActive, Type: Filtering criteria
func GetBannersData(value string) ([]dtos.Banner, error) {
	// Query active banners of specific type, ordered by display preference
	rows, err := DB.Query(`
		SELECT id, image_url, text, heading, button_text, button_url, display_order, is_active, type
		FROM banners WHERE is_active = true AND type = ? ORDER BY display_order ASC`, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var banners []dtos.Banner
	// Collect banner data
	for rows.Next() {
		var b dtos.Banner
		err := rows.Scan(&b.ID, &b.ImageURL, &b.Text, &b.Heading, &b.ButtonText, &b.ButtonURL, &b.DisplayOrder, &b.IsActive, &b.Type)
		if err != nil {
			return nil, err
		}
		banners = append(banners, b)
	}
	return banners, nil
}

// GetPromotions retrieves currently active promotions with associated products.
//
// This function fetches promotions that are active and within their date range.
// Each promotion includes its type, description, and grouped product categories.
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.Promotion: Array of active promotions with product groups
//   - error: Database error if queries fail
//
// Filtering Logic:
//   - is_active = TRUE
//   - start_date <= NOW
//   - end_date >= NOW
//
// Promotion Fields:
//   - Basic: ID, Name, StartDate, EndDate, IsActive
//   - Type Info: PromotionType, PromotionDescription, Amount
//   - Products: PromotionProducts (grouped by category)
func GetPromotions() ([]dtos.Promotion, error) {
	// Get current timestamp for date range filtering
	now := time.Now()

	// Query active promotions within date range with type details
	promotionQuery := `
	SELECT 
		p.promotion_id, 
		p.name, 
		p.start_date, 
		p.end_date, 
		p.is_active, 
		pt.name AS promotion_type, 
		pt.description AS promotion_description,
		pt.value AS amount
	FROM promotions p
	LEFT JOIN promotion_types pt ON pt.id = p.promotion_type_id
	WHERE p.is_active = TRUE 
	  AND p.start_date <= ? 
	  AND p.end_date >= ?
`

	rows, err := DB.Query(promotionQuery, now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []dtos.Promotion

	// Iterate through active promotions
	for rows.Next() {
		var promo dtos.Promotion
		if err := rows.Scan(&promo.ID, &promo.Name, &promo.StartDate, &promo.EndDate, &promo.IsActive, &promo.PromotionType, &promo.PromotionDescription, &promo.Amount); err != nil {
			return nil, err
		}
		log.Printf("Got promotion ::::::::%v", promo)

		// Fetch associated products grouped by category
		products, err := getPromotionProductGroups(promo.ID)
		if err != nil {
			return nil, err
		}

		promo.PromotionProducts = products
		promotions = append(promotions, promo)
	}
	log.Printf("Got promotion ::::::::%v", promotions)

	return promotions, nil
}

// getPromotionProductGroups retrieves products associated with a promotion, grouped by category.
//
// This is an internal helper function that fetches products for a specific promotion
// and organizes them by their categories.
//
// Parameters:
//   - promotionID: The promotion_id to fetch products for
//
// Returns:
//   - []dtos.PromotionProductGroup: Array of product groups with category information
//   - error: Database error if queries fail
//
// Product Group Structure:
//   - ID, PromotionID, ProductID: Relationship identifiers
//   - Categories: Array of CategoryGroup with products
func getPromotionProductGroups(promotionID string) ([]dtos.PromotionProductGroup, error) {
	// Query promotion-product associations
	query := `
		SELECT promotion_product_id, promotion_id, product_id 
		FROM promotion_products 
		WHERE promotion_id = ?
	`

	rows, err := DB.Query(query, promotionID)
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
		product, category, err := getProductWithCategory(pp.ProductID)
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
func getProductWithCategory(productID string) (dtos.Product, dtos.CategoryGroup, error) {
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
	row := DB.QueryRow(query, productID)
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
	imageRows, err := DB.Query(`
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
func GetCategoriesWithProducts() ([]dtos.CategoryWithProducts, error) {
	// Query categories with products, deals, and specifications
	rows, err := DB.Query(`
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
			enrichedProduct, err := enrichProductDetails(product)
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
func enrichProductDetails(product dtos.Product) (dtos.Product, error) {
	// Fetch product images
	images, err := fetchProductImages(product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Images = images

	// Fetch warranty information
	warranty, err := FetchProductWarranties(product.ID)
	if err != nil {
		return dtos.Product{}, err
	}
	product.Warranty = &warranty

	// Fetch tax/charge information
	tax, err := fetchProductTax(product.ID)
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
func fetchProductImages(productID string) ([]dtos.Image, error) {
	rows, err := DB.Query(`
		SELECT image_id, url, is_primary, type
		FROM product_images 
		WHERE product_id = ?`, productID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []dtos.Image
	for rows.Next() {
		var img dtos.Image
		if err := rows.Scan(&img.ImageID, &img.URL, &img.IsPrimary, &img.Type); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}

// fetchProductTax retrieves tax/charge information for a product.
//
// This is an internal helper function that fetches the first tax or charge
// associated with a product (e.g., VAT, sales tax, environmental charges).
//
// Parameters:
//   - productID: The product_id to fetch tax for
//
// Returns:
//   - dtos.ProductTax: Tax information (ID, Name, Value) or empty struct if none
//   - error: Database error if query fails (sql.ErrNoRows returns empty struct, not error)
func fetchProductTax(productID string) (dtos.ProductTax, error) {
	query := `SELECT c.charge_id, c.charge_name, c.charge_value
		FROM product_charges pc
		JOIN charges c ON pc.charge_id = c.charge_id
		WHERE pc.product_id = ? LIMIT 1`
	var tax dtos.ProductTax
	err := DB.QueryRow(query, productID).Scan(&tax.ID, &tax.Name, &tax.Value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dtos.ProductTax{}, nil
		}
		return dtos.ProductTax{}, err
	}
	return tax, nil
}

// FetchProductWarranties retrieves warranty information for a product.
//
// This function fetches the most recent warranty details for a product,
// including warranty period, dates, and warranty type.
//
// Parameters:
//   - productID: The product_id to fetch warranty for
//
// Returns:
//   - dtos.ProductWarranty: Warranty information or empty struct if none
//   - error: Database error if query fails (sql.ErrNoRows returns empty struct, not error)
//
// Warranty Fields:
//   - WarrantyPeriod: Duration of warranty coverage
//   - ManufacturingDate, ExpiryDate: Warranty validity dates
//   - WarrantyType: Type description (e.g., "Manufacturer", "Extended")
//   - WarrantyID: Warranty type identifier
func FetchProductWarranties(productID string) (dtos.ProductWarranty, error) {
	query := `
		SELECT pw.warranty_period, pw.manufacturing_date, pw.expiry_date, wt.name, wt.warranty_type_id
		FROM product_warranties pw
		JOIN warranty_types wt ON pw.warranty_type_id = wt.warranty_type_id
		WHERE pw.product_id = ?
		ORDER BY pw.created_at DESC
		LIMIT 1
	`

	var warranty dtos.ProductWarranty

	err := DB.QueryRow(query, productID).
		Scan(&warranty.WarrantyPeriod, &warranty.ManufacturingDate, &warranty.ExpiryDate, &warranty.WarrantyType, &warranty.WarrantyID)

	if err == sql.ErrNoRows {
		return dtos.ProductWarranty{}, nil
	}

	if err != nil {
		return dtos.ProductWarranty{}, err
	}

	return warranty, nil
}

// fetchProductFeatures retrieves feature highlights for a product.
//
// This is an internal helper function that fetches product feature descriptions
// with images and positioning for product detail pages.
//
// Parameters:
//   - productID: The product_id to fetch features for
//
// Returns:
//   - []dtos.ProductFeature: Array of features ordered by creation
//   - error: Database error if query fails
//
// Feature Fields:
//   - ID, ProductID: Identifiers
//   - Header: Feature title/heading
//   - Image: Feature illustration image URL
//   - Description: Feature details
//   - ImagePosition: Layout position ("left", "right", etc.)
func fetchProductFeatures(productID string) ([]dtos.ProductFeature, error) {
	query := `
		SELECT feature_id, product_id, header, image, description, image_position, product_specifications, top_section, design_type, images
		FROM product_features
		WHERE product_id = ?
		ORDER BY created_at ASC`
	rows, err := DB.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []dtos.ProductFeature
	for rows.Next() {
		var (
			feature               dtos.ProductFeature
			productSpecifications sql.NullString
			topSection            sql.NullString
			designType            sql.NullString
			images                sql.NullString
		)
		if err := rows.Scan(&feature.ID, &feature.ProductID, &feature.Header, &feature.Image, &feature.Description, &feature.ImagePosition, &productSpecifications, &topSection, &designType, &images); err != nil {
			return nil, err
		}
		// Unmarshal JSON fields if valid
		if productSpecifications.Valid {
			json.Unmarshal([]byte(productSpecifications.String), &feature.ProductSpecifications)
		}
		if topSection.Valid {
			json.Unmarshal([]byte(topSection.String), &feature.TopSection)
		}
		if designType.Valid {
			json.Unmarshal([]byte(designType.String), &feature.DesignType)
		}
		if images.Valid {
			json.Unmarshal([]byte(images.String), &feature.Images)
		}
		features = append(features, feature)
	}
	return features, nil
}

// InsertBannerDetails creates a new banner in the database.
//
// This function inserts a banner with all configuration details including
// image, text content, and call-to-action button.
//
// Parameters:
//   - url: Banner image URL
//   - req: dtos.BannerInfo containing:
//   - Text: Banner text content
//   - Heading: Banner heading/title
//   - ButtonText: CTA button label
//   - ButtonURL: CTA button destination
//   - DisplayOrder: Position ordering
//   - IsActive: Active/inactive status
//   - Type: Banner type ("hero", "sidebar", etc.)
//
// Returns:
//   - error: Database error if insertion fails
func InsertBannerDetails(url string, req dtos.BannerInfo) error {
	query := `INSERT INTO banners (image_url, text, heading, button_text, button_url, display_order, is_active, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := DB.Exec(query, url, req.Text, req.Heading, req.ButtonText, req.ButtonURL, req.DisplayOrder, req.IsActive, req.Type)
	return err
}

// UpdateBannerDetails updates an existing banner with partial field updates.
//
// This function validates banner existence and dynamically builds UPDATE query
// for only the provided fields (partial updates supported).
//
// Parameters:
//   - req: dtos.UpdateBannerInfo with optional fields:
//   - Text: Banner text (empty string skipped)
//   - Heading: Banner heading (empty string skipped)
//   - ButtonText: Button label (empty string skipped)
//   - ButtonURL: Button destination (empty string skipped)
//   - DisplayOrder: Position (0 skipped)
//   - IsActive: Active status (nil skipped)
//   - bannerID: The banner ID to update
//
// Returns:
//   - error: "banner not found" if banner doesn't exist,
//     "request cannot be empty" if no fields provided,
//     or database error
func UpdateBannerDetails(req dtos.UpdateBannerInfo, bannerID string) error {
	// Validate banner exists
	exists, err := RecordExists("banners", whereID, bannerID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("banner not found")
	}

	// Build dynamic UPDATE query with only provided fields
	query := "UPDATE banners SET"
	args := []interface{}{}
	updates := []string{}

	// Add Text field if provided
	if req.Text != "" {
		updates = append(updates, "text = ?")
		args = append(args, req.Text)
	}
	// Add Heading field if provided
	if req.Heading != "" {
		updates = append(updates, "heading = ?")
		args = append(args, req.Heading)
	}
	// Add ButtonText field if provided
	if req.ButtonText != "" {
		updates = append(updates, "button_text = ?")
		args = append(args, req.ButtonText)
	}
	// Add ButtonURL field if provided
	if req.ButtonURL != "" {
		updates = append(updates, "button_url = ?")
		args = append(args, req.ButtonURL)
	}
	// Add DisplayOrder field if non-zero
	if req.DisplayOrder != 0 {
		updates = append(updates, "display_order = ?")
		args = append(args, req.DisplayOrder)
	}
	// Add IsActive field if provided
	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		args = append(args, req.IsActive)
	}

	// Validate at least one field is being updated
	if len(updates) == 0 {
		return fmt.Errorf("request cannot be empty") // Nothing to update
	}

	// Build final query and execute
	query += " " + strings.Join(updates, ", ") + " WHERE id = ?"
	args = append(args, bannerID)

	if _, err := DB.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update review: %v", err)
	}

	return nil
}

// GetPromotionsTypes retrieves all available promotion types.
//
// This function fetches promotion type definitions used for categorizing
// and describing different types of promotional offers.
//
// Parameters:
//   - None
//
// Returns:
//   - []dtos.PromotionType: Array of promotion types with descriptions and values
//   - error: Database error if query fails
//
// Promotion Type Fields:
//   - ID: Unique type identifier
//   - Name: Type name (e.g., "Percentage Off", "Buy One Get One")
//   - Description: Type description
//   - Value: Associated value or calculation
func GetPromotionsTypes() ([]dtos.PromotionType, error) {
	// Query all promotion type definitions
	rows, err := DB.Query(`
		SELECT id, name, description, value 
		FROM promotion_types 
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []dtos.PromotionType
	// Collect promotion types
	for rows.Next() {
		var typesingle dtos.PromotionType
		if err := rows.Scan(&typesingle.ID, &typesingle.Name, &typesingle.Description, &typesingle.Value); err != nil {
			return nil, err
		}
		types = append(types, typesingle)
	}
	return types, nil
}

// CreateNewPromotion creates a new promotion in the database.
//
// This function creates a time-limited promotional campaign with a specific type.
// Products can be associated after creation using AttachProductsToPromotion.
//
// Parameters:
//   - req: dtos.NewPromotion containing:
//   - Name: Promotion name
//   - PromotionIdType: Promotion type ID (links to promotion_types table)
//   - StartDate: When promotion becomes active
//   - EndDate: When promotion expires
//
// Returns:
//   - string: Generated promotion_id
//   - error: Database error if insertion fails
func CreateNewPromotion(req dtos.NewPromotion) (string, error) {
	// Generate unique promotion ID
	promotionID, _ := shortid.Generate()

	// Insert new promotion
	_, err := DB.Exec(`
        INSERT INTO promotions(promotion_id, name, promotion_type_id, start_date, end_date)
        VALUES (?, ?, ?, ?, ?)
    `, promotionID, req.Name, req.PromotionIdType, req.StartDate, req.EndDate)
	if err != nil {
		return "", fmt.Errorf("failed to promotion: %w", err)
	}

	return promotionID, nil
}

// DeletePromotion removes a promotion from the database.
//
// This function deletes a promotion by its ID. Associated products in
// promotion_products should be handled by database constraints or deleted separately.
//
// Parameters:
//   - promotionID: The promotion_id to delete
//
// Returns:
//   - error: Database error if deletion fails
func DeletePromotion(promotionID string) error {
	// Delete promotion record
	_, err := DB.Exec(`
		DELETE FROM promotions WHERE promotion_id = ?
	`, promotionID)
	return err
}

// EditPromotion updates an existing promotion with partial field updates.
//
// This function dynamically builds UPDATE query for only the provided fields.
// Supports partial updates - only non-zero/non-nil fields are updated.
//
// Parameters:
//   - req: dtos.EditPromotion with optional fields:
//   - PromotionID: Promotion to update (required for WHERE clause)
//   - Name: Updated promotion name (empty string skipped)
//   - PromotionIdType: Updated type ID (0 skipped)
//   - StartDate: Updated start date (zero time skipped)
//   - EndDate: Updated end date (zero time skipped)
//   - IsActive: Active status (nil skipped)
//
// Returns:
//   - error: Database error if update fails, nil if nothing to update
func EditPromotion(req dtos.EditPromotion) error {
	// Build dynamic UPDATE query with only provided fields
	query := "UPDATE promotions SET"
	args := []interface{}{}
	updates := []string{}

	// Add Name field if provided
	if req.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, req.Name)
	}
	// Add PromotionIdType field if non-zero
	if req.PromotionIdType != 0 {
		updates = append(updates, "promotion_type_id = ?")
		args = append(args, req.PromotionIdType)
	}
	// Add StartDate field if not zero time
	if !req.StartDate.IsZero() {
		updates = append(updates, "start_date = ?")
		args = append(args, req.StartDate)
	}
	// Add EndDate field if not zero time (BUG: uses StartDate instead of EndDate)
	if !req.EndDate.IsZero() {
		updates = append(updates, "end_date = ?")
		args = append(args, req.StartDate) // BUG: Should be req.EndDate
	}
	// Add IsActive field if provided
	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		args = append(args, req.IsActive)
	}

	// If no fields to update, return nil
	if len(updates) == 0 {
		return nil // Nothing to update
	}

	// Build final query and execute
	query += " " + strings.Join(updates, ", ") + " WHERE promotion_id = ?"
	args = append(args, req.PromotionID)

	if _, err := DB.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update promotion: %v", err)
	}

	return nil
}

// AttachProductsToPromotion associates multiple products with a promotion.
//
// This function creates product-promotion relationships for one or more products.
// It's idempotent - skips products already attached to the promotion.
//
// Parameters:
//   - req: dtos.AttachProductToPromotion containing:
//   - PromotionID: The promotion to attach products to
//   - ProductIDs: Array of product_id values to attach
//
// Returns:
//   - error: "no product IDs provided" if array empty,
//     or database error if insertion fails
//
// Behavior:
//   - Checks existence before insertion (idempotent)
//   - Skips products already associated
//   - Processes all products in array
func AttachProductsToPromotion(req dtos.AttachProductToPromotion) error {
	// Validate at least one product ID provided
	if len(req.ProductIDs) == 0 {
		return fmt.Errorf("no product IDs provided")
	}

	// Prepare queries for existence check and insertion
	checkQuery := `SELECT COUNT(1) FROM promotion_products WHERE promotion_id = ? AND product_id = ?`
	insertQuery := `INSERT INTO promotion_products (promotion_product_id, promotion_id, product_id) VALUES (?, ?, ?)`

	// Iterate through products and attach to promotion
	for _, productID := range req.ProductIDs {
		// Generate unique ID for association
		promotionProductID, _ := shortid.Generate()

		// Check if product already attached to this promotion
		var count int
		err := DB.QueryRow(checkQuery, req.PromotionID, productID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check existence for product %s: %v", productID, err)
		}

		// Skip if association already exists (idempotent)
		if count > 0 {
			continue // skip if already exists
		}

		// Insert new product-promotion association
		if _, err := DB.Exec(insertQuery, promotionProductID, req.PromotionID, productID); err != nil {
			return fmt.Errorf("failed to insert product %s: %v", productID, err)
		}
	}
	return nil
}

// CheckPromotionExists validates that a promotion exists by promotion_id.
//
// This function checks for promotion existence without returning promotion data.
//
// Parameters:
//   - promotionID: The promotion_id to validate
//
// Returns:
//   - bool: true if promotion exists, false otherwise
//   - error: Database error if query fails
func CheckPromotionExists(promotionID string) (bool, error) {
	var exists bool
	// Check promotion existence
	query := `SELECT EXISTS(SELECT 1 FROM promotions WHERE promotion_id = ?)`
	err := DB.QueryRow(query, promotionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check promotion existence: %v", err)
	}
	return exists, nil
}

// RemoveProductFromPromotion removes products from a promotion.
//
// This function deletes product-promotion associations for one or more products.
// Returns error if no products were removed.
//
// Parameters:
//   - req: dtos.AttachProductToPromotion containing:
//   - PromotionID: The promotion to remove products from
//   - ProductIDs: Array of product_id values to remove
//
// Returns:
//   - error: "no matching products found" if no products were removed,
//     or database error if deletion fails
func RemoveProductFromPromotion(req dtos.AttachProductToPromotion) error {
	// Prepare delete query
	query := `DELETE FROM promotion_products WHERE promotion_id = ? AND product_id = ?`

	var anyDeleted bool

	// Iterate through products and remove from promotion
	for _, productID := range req.ProductIDs {
		// Delete product-promotion association
		result, err := DB.Exec(query, req.PromotionID, productID)
		if err != nil {
			return fmt.Errorf("failed to remove product %s from promotion %s: %v", productID, req.PromotionID, err)
		}

		// Track if any rows were deleted
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			anyDeleted = true
		}
	}

	// Error if no products were actually removed
	if !anyDeleted {
		return fmt.Errorf("no matching products found to remove for promotion %s", req.PromotionID)
	}

	return nil
}

// CreateBlog creates a new blog post with flexible JSON content structure.
//
// This function creates a blog with draft or published status. Published blogs
// get a published_at timestamp, while drafts remain unpublished.
//
// Parameters:
//   - blog: dtos.BlogRequest containing:
//   - Title: Blog post title
//   - Sections: Array of content sections (marshaled to JSON)
//   - Author: Author information object (marshaled to JSON)
//   - Tags: Array of tags (marshaled to JSON)
//   - Description: Blog summary/excerpt
//   - ReadTimeMinutes: Estimated reading time
//   - Status: "draft" or "published" (defaults to "published" if nil)
//   - BannerImageUrl: Header/banner image
//   - authorID: User ID of the blog author
//
// Returns:
//   - error: JSON marshaling error or database error
//
// Status Handling:
//   - Draft: is_published=false, published_at=NULL
//   - Published: is_published=true, published_at=NOW()
func CreateBlog(blog dtos.BlogRequest, authorID string) error {
	// Generate unique blog ID
	blogID, _ := shortid.Generate()

	var publishedAt *string
	isPublished := true

	// Default status to "published" if not provided
	if blog.Status == nil {
		blog.Status = new(string)
		*blog.Status = "published"
	}

	// Handle draft vs published status
	if strings.ToLower(*blog.Status) == "draft" {
		// Draft: unpublished with no published_at date
		isPublished = false
		publishedAt = nil // ← represents NULL in DB
	} else {
		// Published: set published_at to current timestamp
		now := time.Now().Format("2006-01-02 15:04:05")
		publishedAt = &now

	}

	// Marshal content sections to JSON
	contentData, err := json.Marshal(blog.Sections)
	if err != nil {
		fmt.Println("Error converting to JSON:", err)
		return err
	}

	// Marshal author information to JSON
	authorData, err := json.Marshal(blog.Author)
	if err != nil {
		fmt.Println("Error converting author to JSON:", err)
		return err
	}

	// Marshal tags to JSON
	tagData, err := json.Marshal(blog.Tags)
	if err != nil {
		fmt.Println("Error converting tags to JSON:", err)
		return err
	}

	// Insert blog record with JSON fields
	query := `
		INSERT INTO blogs (blog_id, title, content, author_id, published_at, is_published, author, tags, description, read_time, status, banner_image_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = DB.Exec(query, blogID, blog.Title, contentData, authorID, publishedAt, isPublished, authorData, tagData, blog.Description, blog.ReadTimeMinutes, blog.Status, blog.BannerImageUrl)
	return err
}

// GetBlogByID retrieves a single blog post by its blog_id.
//
// This function fetches complete blog data including JSON fields (content sections,
// author info, tags) and deserializes them into structured objects.
//
// Parameters:
//   - blogID: The blog_id to retrieve
//
// Returns:
//   - *dtos.BlogRequest: Pointer to blog with deserialized JSON fields
//   - error: "blog not found" if blog doesn't exist,
//     JSON unmarshal error if JSON invalid,
//     or database error
//
// JSON Fields:
//   - Sections: Array of content sections
//   - Author: Author information object
//   - Tags: Array of tag strings
func GetBlogByID(blogID string) (*dtos.BlogRequest, error) {
	exists, err := RecordExists("blogs", fetchblog, blogID)
	if err != nil {
		return nil, fmt.Errorf("failed to check blog existence: %w", err)
	}
	if !exists {
		return nil, errors.New(noblog)
	}

	query := `
		SELECT blog_id, title, content, author_id, published_at, is_published, 
		       author, tags, description, read_time, status, created_at, updated_at, banner_image_url
		FROM blogs
		WHERE blog_id = ?
	`

	var (
		blog        dtos.BlogRequest
		contentJSON sql.NullString
		authorJSON  sql.NullString
		tagsJSON    sql.NullString
		publishedAt sql.NullTime
		readTime    sql.NullInt64
	)

	row := DB.QueryRow(query, blogID)
	if err := row.Scan(
		&blog.BlogID, &blog.Title, &contentJSON, &blog.AuthorID,
		&publishedAt, &blog.IsPublished, &authorJSON, &tagsJSON,
		&blog.Description, &readTime, &blog.Status,
		&blog.CreatedAt, &blog.UpdatedAt, &blog.BannerImageUrl,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New(noblog)
		}
		return nil, fmt.Errorf("failed to scan blog: %w", err)
	}

	unmarshalJSONField := func(data sql.NullString, target interface{}, field string) error {
		if data.Valid {
			if err := json.Unmarshal([]byte(data.String), target); err != nil {
				return fmt.Errorf("invalid %s JSON: %w", field, err)
			}
		}
		return nil
	}

	if err := unmarshalJSONField(contentJSON, &blog.Sections, "content"); err != nil {
		return nil, err
	}
	if err := unmarshalJSONField(authorJSON, &blog.Author, "author"); err != nil {
		return nil, err
	}
	if err := unmarshalJSONField(tagsJSON, &blog.Tags, "tags"); err != nil {
		return nil, err
	}

	if publishedAt.Valid {
		blog.PublishedAt = publishedAt.Time
	}
	if readTime.Valid {
		blog.ReadTimeMinutes = int(readTime.Int64)
	}

	return &blog, nil
}

// UpdateBlog updates an existing blog post with full field replacement.
//
// This function validates blog existence and updates all fields including JSON
// content. It handles status transitions and published_at accordingly.
//
// Parameters:
//   - blog: dtos.BlogRequest with fields to update:
//   - Title: Updated title
//   - Sections: Updated content sections (marshaled to JSON)
//   - Author: Updated author info (marshaled to JSON)
//   - Tags: Updated tags (marshaled to JSON)
//   - Description: Updated summary
//   - ReadTimeMinutes: Updated read time
//   - Status: "draft" or "published" (defaults to "published" if nil)
//   - BannerImageUrl: Updated banner image
//   - blogID: The blog_id to update
//
// Returns:
//   - error: "blog not found" if blog doesn't exist,
//     JSON marshaling error,
//     or database error
//
// Status Transitions:
//   - draft → published: Sets published_at to NOW(), is_published=true
//   - published → draft: Sets published_at=NULL, is_published=false
func UpdateBlog(blog dtos.BlogRequest, blogID string) error {
	// Verify blog exists
	exists, err := RecordExists("blogs", fetchblog, blogID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noblog)
	}

	var publishedAt *string
	isPublished := true

	// Default status to "published" if not provided
	if blog.Status == nil {
		blog.Status = new(string)
		*blog.Status = "published"
	}

	// Handle draft vs published status
	if strings.ToLower(*blog.Status) == "draft" {
		// Draft: unpublished with no published_at date
		isPublished = false
		publishedAt = nil // ← represents NULL in DB
	} else {
		// Published: set published_at to current timestamp
		now := time.Now().Format("2006-01-02 15:04:05")
		publishedAt = &now

	}

	// Marshal content sections to JSON
	contentData, err := json.Marshal(blog.Sections)
	if err != nil {
		fmt.Println("Error converting to JSON:", err)
		return err
	}

	// Marshal author information to JSON
	authorData, err := json.Marshal(blog.Author)
	if err != nil {
		fmt.Println("Error converting author to JSON:", err)
		return err
	}

	// Marshal tags to JSON
	tagData, err := json.Marshal(blog.Tags)
	if err != nil {
		fmt.Println("Error converting tags to JSON:", err)
		return err
	}

	// Update all blog fields
	query := `
		UPDATE blogs SET title = ?, content = ?, published_at = ?, is_published = ?, author = ?, tags = ?, description = ?, read_time = ?, status = ?, banner_image_url = ?
		WHERE blog_id = ?`

	_, err = DB.Exec(query, blog.Title, contentData, publishedAt, isPublished, authorData, tagData, blog.Description, blog.ReadTimeMinutes, blog.Status, blog.BannerImageUrl, blogID)
	return err
}

// DeleteBlog permanently removes a blog post and returns error if not found.
//
// Parameters:
//   - blogID: The blog_id to delete
//
// Returns:
//   - error: "blog not found" if blog doesn't exist or database error
func DeleteBlog(blogID string) error {
	// Verify blog exists
	exists, err := RecordExists("blogs", fetchblog, blogID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noblog)
	}

	// Delete blog record
	query := `DELETE FROM blogs WHERE blog_id = ?`
	_, err = DB.Exec(query, blogID)
	return err
}

// ListBlogs retrieves paginated list of blogs with optional status filtering.
//
// This function fetches blogs with pagination metadata and supports filtering
// by blog status (draft/published). Results are ordered by publish date descending.
//
// Parameters:
//   - page: Page number (1-indexed)
//   - limit: Items per page
//   - status: Filter by status ("draft", "published", or "" for all)
//
// Returns:
//   - []dtos.BlogRequest: Array of blogs with deserialized JSON fields
//   - *dtos.PaginationMeta: Pagination info (total, pages, current page)
//   - error: Database error or JSON unmarshal error
func ListBlogs(page, limit int, status string) ([]dtos.BlogRequest, *dtos.PaginationMeta, error) {
	// Calculate offset for pagination
	offset := (page - 1) * limit

	// Get total count for pagination metadata
	totalItems, err := countBlogs(status)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count blogs: %w", err)
	}

	// Fetch blog records
	rows, err := fetchBlogs(status, limit, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query blogs: %w", err)
	}
	defer rows.Close()

	// Deserialize blog rows
	blogs, err := scanBlogs(rows)
	if err != nil {
		return nil, nil, err
	}

	// Build pagination metadata
	meta := buildPagination(limit, offset, totalItems)
	return blogs, meta, nil
}

// countBlogs returns total blog count with optional status filtering.
//
// This is an internal helper function used for pagination calculations.
//
// Parameters:
//   - status: Filter by status ("draft", "published", or "" for all)
//
// Returns:
//   - int: Total number of blogs matching filter
//   - error: Database error if query fails
func countBlogs(status string) (int, error) {
	// Build count query with optional status filter
	query := "SELECT COUNT(*) FROM blogs"
	var args []interface{}

	// Add status filter if provided
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	// Execute count query
	var total int
	if err := DB.QueryRow(query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// fetchBlogs retrieves blog rows with pagination and optional status filtering.
//
// This is an internal helper function that returns sql.Rows for scanning.
//
// Parameters:
//   - status: Filter by status ("draft", "published", or "" for all)
//   - limit: Maximum number of rows to return
//   - offset: Number of rows to skip
//
// Returns:
//   - *sql.Rows: Result set with blog records (caller must close)
//   - error: Database error if query fails
func fetchBlogs(status string, limit, offset int) (*sql.Rows, error) {
	// Query all blog fields with JSON columns
	query := `
		SELECT blog_id, title, content, author_id, published_at, is_published, 
		       author, tags, description, read_time, status, created_at, updated_at, banner_image_url
		FROM blogs
	`
	var args []interface{}

	// Add status filter if provided
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	// Add ordering and pagination
	query += " ORDER BY published_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	return DB.Query(query, args...)
}

// scanBlogs deserializes result rows into blog objects with JSON unmarshaling.
//
// This is an internal helper function that handles row scanning and JSON
// deserialization for blog list queries.
//
// Parameters:
//   - rows: Result set from fetchBlogs (or similar query)
//
// Returns:
//   - []dtos.BlogRequest: Array of blogs with deserialized JSON fields
//   - error: Scan error or JSON unmarshal error
//
// JSON Fields Deserialized:
//   - content: Sections array
//   - author: Author object
//   - tags: Tags array
func scanBlogs(rows *sql.Rows) ([]dtos.BlogRequest, error) {
	var blogs []dtos.BlogRequest

	// Iterate through result rows
	for rows.Next() {
		var (
			blog        dtos.BlogRequest
			contentJSON sql.NullString
			authorJSON  sql.NullString
			tagsJSON    sql.NullString
			publishedAt sql.NullTime
			readTime    sql.NullInt64
		)

		// Scan row into blog struct and JSON fields
		if err := rows.Scan(
			&blog.BlogID, &blog.Title, &contentJSON, &blog.AuthorID,
			&publishedAt, &blog.IsPublished, &authorJSON, &tagsJSON,
			&blog.Description, &readTime, &blog.Status,
			&blog.CreatedAt, &blog.UpdatedAt, &blog.BannerImageUrl,
		); err != nil {
			return nil, fmt.Errorf("failed to scan blog: %w", err)
		}

		// Parse JSON fields and nullable fields
		if err := parseBlogFields(&blog, contentJSON, authorJSON, tagsJSON, publishedAt, readTime); err != nil {
			return nil, err
		}

		blogs = append(blogs, blog)
	}
	return blogs, nil
}

// parseBlogFields deserializes JSON fields and nullable fields into blog struct.
//
// This is an internal helper function that handles JSON unmarshaling and
// nullable field conversion for blog objects.
//
// Parameters:
//   - blog: Pointer to blog struct to populate
//   - contentJSON: JSON content sections (nullable)
//   - authorJSON: JSON author info (nullable)
//   - tagsJSON: JSON tags array (nullable)
//   - publishedAt: Published timestamp (nullable)
//   - readTime: Reading time in minutes (nullable)
//
// Returns:
//   - error: JSON unmarshal error if any JSON field is invalid
//
// JSON Handling:
//   - Valid JSON is unmarshaled into corresponding struct fields
//   - NULL values are handled gracefully (fields remain zero-valued)
func parseBlogFields(
	blog *dtos.BlogRequest,
	contentJSON, authorJSON, tagsJSON sql.NullString,
	publishedAt sql.NullTime,
	readTime sql.NullInt64,
) error {
	// Generic JSON unmarshal helper
	unmarshal := func(data sql.NullString, target interface{}, field string) error {
		if data.Valid {
			if err := json.Unmarshal([]byte(data.String), target); err != nil {
				return fmt.Errorf("invalid %s JSON: %w", field, err)
			}
		}
		return nil
	}

	// Unmarshal JSON fields
	if err := unmarshal(contentJSON, &blog.Sections, "content"); err != nil {
		return err
	}
	if err := unmarshal(authorJSON, &blog.Author, "author"); err != nil {
		return err
	}
	if err := unmarshal(tagsJSON, &blog.Tags, "tags"); err != nil {
		return err
	}

	// Convert nullable timestamp and int fields
	if publishedAt.Valid {
		blog.PublishedAt = publishedAt.Time
	}
	if readTime.Valid {
		blog.ReadTimeMinutes = int(readTime.Int64)
	}
	return nil
}

// DeleteBanner permanently removes a banner and returns error if not found.
//
// Parameters:
//   - bannerID: The banner ID to delete
//
// Returns:
//   - error: "banner not found" if banner doesn't exist or database error
func DeleteBanner(bannerID string) error {
	// Verify banner exists
	exists, err := RecordExists("banners", whereID, bannerID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("banner not found")
	}

	// Delete banner record
	query := `DELETE FROM banners WHERE id = ?`
	_, err = DB.Exec(query, bannerID)
	return err
}

// CreateMenuLink creates a new menu link with optional parent (for hierarchy).
//
// This function creates navigation menu links with optional parent-child relationships.
// Menu titles must be unique across all menu links.
//
// Parameters:
//   - req: dtos.MenuLinkRequest containing:
//   - Title: Menu link title (must be unique)
//   - URL: Link destination URL
//   - DisplayOrder: Position ordering
//   - ParentID: Optional parent menu ID for nested menus (nil for top-level)
//
// Returns:
//   - error: "menu title already exists" if title is duplicate,
//     or database error
//
// Parent Handling:
//   - ParentID=nil: Creates top-level menu item
//   - ParentID set: Creates submenu under specified parent
func CreateMenuLink(req dtos.MenuLinkRequest) error {
	// Validate menu title uniqueness
	exists, err := RecordExists("menu_links", "title = ?", req.Title)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("menu title already exists")
	}

	// Handle parent-child relationship
	if req.ParentID == nil {
		// Insert top-level menu (no parent)
		query := `INSERT INTO menu_links (title, url, display_order) VALUES (?, ?, ?)`
		_, err := DB.Exec(query, req.Title, req.URL, req.DisplayOrder)
		if err != nil {
			return err
		}

		// // Get inserted ID
		// id, err := result.LastInsertId()
		// if err != nil {
		// 	return err
		// }

		// // Update parent_id to self
		// _, err = DB.Exec(`UPDATE menu_links SET parent_id = ? WHERE id = ?`, id, id)
		// if err != nil {
		// 	return err
		// }

	} else {
		// Insert submenu with parent reference
		query := `INSERT INTO menu_links (title, url, display_order, parent_id) VALUES (?, ?, ?, ?)`
		_, err = DB.Exec(query, req.Title, req.URL, req.DisplayOrder, req.ParentID)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateMenuLink updates an existing menu link's properties.
//
// Parameters:
//   - menu: dtos.MenuLinkRequest with updated values:
//   - Title: Updated menu title
//   - URL: Updated link destination
//   - DisplayOrder: Updated position
//   - menuLinkID: The menu link ID to update
//
// Returns:
//   - error: "menu link not found" if menu doesn't exist or database error
func UpdateMenuLink(menu dtos.MenuLinkRequest, menuLinkID int) error {
	// Verify menu link exists
	exists, err := RecordExists("menu_links", whereID, menuLinkID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("menu link not found")
	}

	// Update menu link properties
	query := `UPDATE menu_links SET title=?, url=?, display_order=? WHERE id=?`
	_, err = DB.Exec(query, menu.Title, menu.URL, menu.DisplayOrder, menuLinkID)
	if err != nil {
		return err
	}
	return nil
}

// DeleteMenuLink permanently removes a menu link.
//
// Parameters:
//   - menuLinkID: The menu link ID to delete
//
// Returns:
//   - error: "menu link not found" if menu doesn't exist or database error
func DeleteMenuLink(menuLinkID int) error {
	// Verify menu link exists
	exists, err := RecordExists("menu_links", whereID, menuLinkID)
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
	exists, err := RecordExists("social_links", "platform = ?", link.Platform)
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
	exists, err := RecordExists("social_links", whereID, socialID)
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
	exists, err := RecordExists("social_links", whereID, id)
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
	exists, err := RecordExists("products", "product_id =? ", productID)
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
	exists, err := RecordExists("products", "product_id =? ", productID)
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
		images, err := fetchProductImages(product.ID)
		if err != nil {
			return nil, err
		}
		product.Images = images

		// Enrich product with warranty info
		warranty, err := FetchProductWarranties(product.ID)
		if err != nil {
			return nil, err
		}
		product.Warranty = &warranty

		// Enrich product with tax/charge info
		tax, err := fetchProductTax(product.ID)
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
