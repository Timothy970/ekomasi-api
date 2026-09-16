// Package models provides database models and operations for admin functionality.
// Implements coupon, voucher, and promo code validation and management.
// Also handles product feature management with JSON field support.
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"errors"
	"log"

	"github.com/teris-io/shortid"
)

// isFeatureThere validates that a product feature exists.
// Used for validation before updating or deleting features.
//
// Parameters:
//   - id: Feature ID to validate
//
// Returns:
//   - error: Error if feature not found or database error
func isFeatureThere(db DBExecutor, id string) error {
	// Check if feature exists in product_features table
	exists, err := RecordExists(db, "product_features", "feature_id = ?", id)
	if err != nil {
		// Database query failed
		return err
	}
	if !exists {
		// Feature not found
		return errors.New("feature not found")
	}
	return nil
}

// UpdateProductFeature updates an existing product feature with partial or full field updates.
// Supports dynamic query building to update only provided fields.
//
// Parameters:
//   - input: ProductFeature with fields to update (nil fields are skipped)
//   - featureID: ID of the feature to update
//
// Returns:
//   - *dtos.ProductFeature: Updated feature with all fields
//   - error: Error if feature not found or database operation fails
func UpdateProductFeature(db DBExecutor, input dtos.ProductFeature, featureID string) (*dtos.ProductFeature, error) {
	// Validate that feature exists before updating
	err := isFeatureThere(db, featureID)
	if err != nil {
		// Feature not found
		return nil, err
	}
	// Marshal complex JSON fields for database storage
	jsonProductSpecifications, _ := json.Marshal(input.ProductSpecifications)
	jsonTopSection, _ := json.Marshal(input.TopSection)
	jsonImages, _ := json.Marshal(input.Images)

	// Build dynamic update query based on which fields are provided
	query := "UPDATE product_features SET "
	args := []any{}
	if input.Image != nil {
		// Update image if provided
		query += "image = ?, "
		args = append(args, input.Image)
	}
	if input.ProductSpecifications != nil {
		// Update specifications if provided
		query += "product_specifications = ?, "
		args = append(args, jsonProductSpecifications)
	}
	if input.TopSection != nil {
		// Update top section if provided
		query += "top_section = ?, "
		args = append(args, jsonTopSection)
	}
	if input.Images != nil {
		// Update images array if provided
		query += "images = ?, "
		args = append(args, jsonImages)
	}
	// Always update these standard fields
	query += "header = ?, description = ?, image_position = ?, design_type = ? WHERE feature_id = ?"
	args = append(args, input.Header, input.Description, input.ImagePosition, input.DesignType, featureID)

	// Execute update query
	_, err = db.Exec(query, args...)
	if err != nil {
		// Database update failed
		return nil, err
	}

	// Fetch and return updated feature
	return GetProductFeatureByID(db, featureID)
}

// GetProductFeaturesByProductID retrieves all features for a specific product.
// Used for displaying product detail pages with all feature sections.
//
// Parameters:
//   - productID: ID of the product to retrieve features for
//
// Returns:
//   - []dtos.ProductFeature: List of all features for the product
//   - error: Error if product not found or database operation fails
func GetProductFeaturesByProductID(db DBExecutor, productID string) ([]dtos.ProductFeature, error) {
	// Validate that product exists
	err := IsProductThere(db, productID)
	if err != nil {
		// Product not found
		return nil, err
	}
	// Prepare nullable string variables for JSON fields
	var topSectionStr, productSpecificationsStr, imagesStr sql.NullString
	// Query all features for the product
	rows, err := db.Query(`
		SELECT feature_id, product_id, header, description, image, image_position, product_specifications, top_section, design_type, images
		FROM product_features
		WHERE product_id = ?
	`, productID)
	if err != nil {
		// Database query failed
		return nil, err
	}
	defer rows.Close()

	// Scan all features into slice
	var features []dtos.ProductFeature
	for rows.Next() {
		var f dtos.ProductFeature
		if err := rows.Scan(&f.ID, &f.ProductID, &f.Header, &f.Description, &f.Image, &f.ImagePosition, &productSpecificationsStr, &topSectionStr, &f.DesignType, &imagesStr); err != nil {
			// Row scan failed
			return nil, err
		}
		// Unmarshal JSON fields if they exist
		if productSpecificationsStr.Valid {
			json.Unmarshal([]byte(productSpecificationsStr.String), &f.ProductSpecifications)
		}
		if topSectionStr.Valid {
			json.Unmarshal([]byte(topSectionStr.String), &f.TopSection)
		}
		if imagesStr.Valid {
			json.Unmarshal([]byte(imagesStr.String), &f.Images)
		}
		features = append(features, f)
	}
	return features, nil
}

// GetProductFeatureByID retrieves a single product feature by its ID.
//
// Parameters:
//   - featureID: ID of the feature to retrieve
//
// Returns:
//   - *dtos.ProductFeature: Feature details with all fields including JSON data
//   - error: Error if feature not found or database operation fails
func GetProductFeatureByID(db DBExecutor, featureID string) (*dtos.ProductFeature, error) {
	// Validate that feature exists
	err := isFeatureThere(db, featureID)
	if err != nil {
		// Feature not found
		return nil, err
	}
	var f dtos.ProductFeature
	// Prepare nullable string variables for JSON fields
	var topSectionStr, productSpecificationsStr, imagesStr sql.NullString
	// Query feature details
	err = db.QueryRow(`
		SELECT feature_id, product_id, header, description, image, image_position, product_specifications, top_section, design_type, images
		FROM product_features
		WHERE feature_id = ?
	`, featureID).Scan(&f.ID, &f.ProductID, &f.Header, &f.Description, &f.Image, &f.ImagePosition, &productSpecificationsStr, &topSectionStr, &f.DesignType, &imagesStr)

	// Unmarshal JSON fields if they exist
	if productSpecificationsStr.Valid {
		json.Unmarshal([]byte(productSpecificationsStr.String), &f.ProductSpecifications)
	}
	if topSectionStr.Valid {
		json.Unmarshal([]byte(topSectionStr.String), &f.TopSection)
	}
	if imagesStr.Valid {
		json.Unmarshal([]byte(imagesStr.String), &f.Images)
	}

	if err != nil {
		// Database query failed
		return nil, err
	}
	return &f, nil
}

// DeleteProductFeature removes a product feature from the database.
//
// Parameters:
//   - featureID: ID of the feature to delete
//
// Returns:
//   - error: Error if feature not found or database operation fails
func DeleteProductFeature(db DBExecutor, featureID string) error {
	// Validate that feature exists before deletion
	err := isFeatureThere(db, featureID)
	if err != nil {
		// Feature not found
		return err
	}
	// Delete feature from database
	_, err = db.Exec(`DELETE FROM product_features WHERE feature_id = ?`, featureID)
	return err
}

// UpdateProductFeatures replaces all features for a product with a single new feature.
// Warning: Destructive operation - deletes all existing features before creating new one.
//
// Parameters:
//   - input: ProductFeature to create (replaces all existing features)
//   - productID: ID of the product to update features for
//
// Returns:
//   - *dtos.ProductFeature: Newly created feature
//   - error: Error if product not found or database operation fails
func UpdateProductFeatures(db DBExecutor, input dtos.ProductFeature, productID string) (*dtos.ProductFeature, error) {
	log.Println("Adding feature to product:", productID)
	// Validate that product exists
	err := IsProductThere(db, productID)
	if err != nil {
		// Product not found
		return nil, err
	}
	// Delete all existing features for the product
	_, err = db.Exec(`DELETE FROM product_features WHERE product_id = ?`, productID)
	if err != nil {
		// Failed to delete existing features
		return nil, err
	}
	// Generate unique feature ID for new feature
	featureID, _ := shortid.Generate()
	// Marshal complex JSON fields for database storage
	jsonProductSpecifications, _ := json.Marshal(input.ProductSpecifications)
	jsonTopSection, _ := json.Marshal(input.TopSection)
	jsonImages, _ := json.Marshal(input.Images)
	// Insert new product feature
	_, err = db.Exec(`
		INSERT INTO product_features (feature_id, product_id, header, description, image, image_position, product_specifications, top_section, design_type, images)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		featureID, productID, input.Header, input.Description, input.Image, input.ImagePosition, jsonProductSpecifications, jsonTopSection, input.DesignType, jsonImages,
	)
	if err != nil {
		// Database insert failed
		return nil, err
	}

	// Return created feature
	return &dtos.ProductFeature{
		ID:                    featureID,
		ProductID:             productID,
		Header:                input.Header,
		ImagePosition:         input.ImagePosition,
		Description:           input.Description,
		Image:                 input.Image,
		ProductSpecifications: input.ProductSpecifications,
		TopSection:            input.TopSection,
		DesignType:            input.DesignType,
		Images:                input.Images,
	}, err
}
