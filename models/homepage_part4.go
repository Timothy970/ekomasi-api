package models

import (
	"ekomasi_backend/dtos"
	"fmt"
	"strings"

	"github.com/teris-io/shortid"
)

func CreatePromotionType(db DBExecutor, req dtos.PromotionType) error {
	//validate promotion type with the same name does not exist
	err := isPromotionTypeNameUnique(req.Name)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO promotion_types(name, description, value)
		VALUES (?, ?, ?)
	`, req.Name, req.Description, req.Value)
	return err
}

// helper function to check if a promotion type name is unique
// parameters:
//   - name: The promotion type name to check for uniqueness
//
// returns:
//   - error: "promotion type name already exists" if name is not unique, or database error
func isPromotionTypeNameUnique(name string) error {
	exists, err := RecordExists(DB, "promotion_types", "LOWER(name) = LOWER(?)", strings.ToLower(name))
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("promotion type name already exists")
	}
	return nil
}

// UpdatePromotionType updates an existing promotion type.
//
// This function updates the name, description, and value of a promotion type.
func UpdatePromotionType(db DBExecutor, typeID string, req dtos.PromotionType) error {
	err := isPromotionTypeThere(typeID)
	if err != nil {
		return err
	}
	err = isPromotionTypeNameUnique(req.Name)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE promotion_types
		SET name = ?, description = ?, value = ?
		WHERE id = ?
	`, req.Name, req.Description, req.Value, typeID)
	return err
}

// DeletePromotionType removes a promotion type from the database.
//
// This function deletes a promotion type by its ID.
func DeletePromotionType(typeID string) error {
	err := isPromotionTypeThere(typeID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
		DELETE FROM promotion_types WHERE id = ?
	`, typeID)
	return err
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
	args := []any{}
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
