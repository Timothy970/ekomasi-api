package models

import (
	"ekomasi_backend/dtos"
	"log"
	"time"
)

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
func GetFooterData(db DBExecutor) ([]dtos.Footer, error) {
	// Query footer/contact information (limited to 1 record)
	rows, err := db.Query("SELECT copyright_text, company_address, contact_email, phone_number FROM contact_info LIMIT 1")
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
	if len(result) == 0 {
		return nil, nil
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
func GetSocialsData(db DBExecutor) ([]dtos.SocialLink, error) {
	// Query social links ordered by display preference
	rows, err := db.Query("SELECT platform, url, icon_class FROM social_links ORDER BY display_order")
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
func GetMenuData(db DBExecutor) ([]dtos.MenuLink, error) {
	// Query static pages for menu navigation (most recent first)
	rows, err := db.Query("SELECT  title, path FROM static_pages ORDER BY updated_at DESC")
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
func GetBannersData(db DBExecutor, value string) ([]dtos.Banner, error) {
	// Query active banners of specific type, ordered by display preference
	rows, err := db.Query(`
		SELECT id, image_url, text, heading, button_text, button_url, display_order, is_active, type
		FROM banners WHERE is_active = true AND type = ?`, value)
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
func AdminGetBannersData(db DBExecutor, value string) ([]dtos.Banner, error) {
	// Query active banners of specific type, ordered by display preference
	rows, err := db.Query(`
		SELECT id, image_url, text, heading, button_text, button_url, display_order, is_active, type
		FROM banners WHERE type = ? `, value)
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
func GetPromotions(db DBExecutor) ([]dtos.Promotion, error) {
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

	rows, err := db.Query(promotionQuery, now, now)
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
		products, err := getPromotionProductGroups(db, promo.ID)
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
