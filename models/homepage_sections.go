package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func fetchProductImages(db DBExecutor, productID string) ([]dtos.Image, error) {
	rows, err := db.Query(`
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
func fetchProductTax(db DBExecutor, productID string) (dtos.ProductTax, error) {
	query := `SELECT c.charge_id, c.charge_name, c.charge_value
		FROM product_charges pc
		JOIN charges c ON pc.charge_id = c.charge_id
		WHERE pc.product_id = ? LIMIT 1`
	var tax dtos.ProductTax
	err := db.QueryRow(query, productID).Scan(&tax.ID, &tax.Name, &tax.Value)
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
func FetchProductWarranties(db DBExecutor, productID string) (dtos.ProductWarranty, error) {
	query := `
		SELECT pw.warranty_period, pw.manufacturing_date, pw.expiry_date, wt.name, wt.warranty_type_id
		FROM product_warranties pw
		JOIN warranty_types wt ON pw.warranty_type_id = wt.warranty_type_id
		WHERE pw.product_id = ?
		ORDER BY pw.created_at DESC
		LIMIT 1
	`

	var warranty dtos.ProductWarranty

	err := db.QueryRow(query, productID).
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
func fetchProductFeatures(db DBExecutor, productID string) ([]dtos.ProductFeature, error) {
	query := `
		SELECT feature_id, product_id, header, image, description, image_position, product_specifications, top_section, design_type, images
		FROM product_features
		WHERE product_id = ?
		ORDER BY created_at ASC`
	rows, err := db.Query(query, productID)
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
			images                sql.NullString
		)
		if err := rows.Scan(&feature.ID, &feature.ProductID, &feature.Header, &feature.Image, &feature.Description, &feature.ImagePosition, &productSpecifications, &topSection, &feature.DesignType, &images); err != nil {
			return nil, err
		}
		// Unmarshal JSON fields if valid
		if productSpecifications.Valid {
			json.Unmarshal([]byte(productSpecifications.String), &feature.ProductSpecifications)
		}
		if topSection.Valid {
			json.Unmarshal([]byte(topSection.String), &feature.TopSection)
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
func InsertBannerDetails(db DBExecutor, url string, req dtos.BannerInfo) error {
	query := `INSERT INTO banners (image_url, text, heading, button_text, button_url, display_order, is_active, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(query, url, req.Text, req.Heading, req.ButtonText, req.ButtonURL, req.DisplayOrder, req.IsActive, "banner")
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
func UpdateBannerDetails(db DBExecutor, req dtos.BannerInfo, bannerID string) error {
	// Validate banner exists
	exists, err := RecordExists(db, "banners", whereID, bannerID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("banner not found")
	}

	// Build dynamic UPDATE query with only provided fields
	query := "UPDATE banners SET"
	args := []any{}
	updates := []string{}

	// Add Text field if provided
	if req.Text != nil && *req.Text != "" {
		updates = append(updates, "text = ?")
		args = append(args, *req.Text)
	}
	// Add Heading field if provided
	if req.Heading != nil && *req.Heading != "" {
		updates = append(updates, "heading = ?")
		args = append(args, *req.Heading)
	}
	// Add ButtonText field if provided
	if req.ButtonText != nil && *req.ButtonText != "" {
		updates = append(updates, "button_text = ?")
		args = append(args, *req.ButtonText)
	}
	// Add ButtonURL field if provided
	if req.ButtonURL != nil && *req.ButtonURL != "" {
		updates = append(updates, "button_url = ?")
		args = append(args, *req.ButtonURL)
	}
	// Add DisplayOrder field if non-zero
	if req.DisplayOrder != nil && *req.DisplayOrder != 0 {
		updates = append(updates, "display_order = ?")
		args = append(args, *req.DisplayOrder)
	}
	// Add IsActive field if provided
	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		args = append(args, *req.IsActive)
	}
	if req.Image != nil && *req.Image != "" {
		updates = append(updates, "image_url = ?")
		args = append(args, *req.Image)
	}

	// Validate at least one field is being updated
	if len(updates) == 0 {
		return fmt.Errorf("request cannot be empty") // Nothing to update
	}

	// Build final query and execute
	query += " " + strings.Join(updates, ", ") + " WHERE id = ?"
	args = append(args, bannerID)

	if _, err := DB.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update banner: %v", err)
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

// helper function to check if a record exists
func isPromotionTypeThere(promotionTypeID string) error {
	exists, err := RecordExists(DB, "promotion_types", whereID, promotionTypeID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("promotion type does not exist")
	}
	return nil
}

//Create new Promotion type
// Parameters:
//   - req: dtos.NewPromotionType containing:
//   - Name: Promotion type name

// Returns:
//   - error: Database error if insertion fails
