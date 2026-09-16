// Package models provides data access functions for the Ekomasi backend application.
//
// This file contains promotion and promo code management functionality including:
//   - Promotion effectiveness tracking and analytics
//   - Promo code creation, validation, and redemption
//   - Promotion-to-product associations
//   - Promotion comparison with baseline periods
//   - Sales and revenue aggregation for promotional periods
//   - Active promo code lookup and validation
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"time"

	"github.com/teris-io/shortid"
)

// GetAllPromoCodes retrieves all promo codes with pagination metadata.
//
// Parameters:
//   - page: int - Page number for pagination metadata (not used for actual paging)
//   - size: int - Page size for pagination metadata (not used for actual paging)
//
// Returns:
//
//   - []dtos.PromoCodeResponse: Array of all promo codes
//
//   - *dtos.PaginationMeta: Pagination metadata
//
//   - error: Database error or nil on success
//
//     Consider implementing actual pagination if code count grows large.
func GetAllPromoCodes(db DBExecutor, page, size int) ([]dtos.PromoCodeResponse, *dtos.PaginationMeta, error) {
	// Count total promo codes
	var totalCount int
	err := db.QueryRow(`SELECT COUNT(*) FROM promocodes`).Scan(&totalCount)
	if err != nil {
		return nil, nil, err
	}

	// Query all promo codes (no LIMIT/OFFSET applied)
	rows, err := db.Query(`
		SELECT promo_code_id, code, description, discount_type, discount_value, expires_at, is_active, minimum_order_value, maximum_use, times_used, promo_type, brand_id FROM promocodes ORDER BY created_at DESC`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan promo code rows
	var promos []dtos.PromoCodeResponse
	for rows.Next() {
		var pc dtos.PromoCodeResponse
		if err := rows.Scan(&pc.ID, &pc.Code, &pc.Description, &pc.DiscountType, &pc.DiscountValue, &pc.ExpiresAt, &pc.IsActive, &pc.MinimumOrderValue, &pc.MaximumUse, &pc.TimesUsed, &pc.PromoType, &pc.BrandID); err != nil {
			return nil, nil, err
		}
		promos = append(promos, pc)
	}

	// Build pagination metadata
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalCount,
		TotalPages: (totalCount + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < totalCount,
	}

	return promos, pagination, nil
}

// DeletePromoCode permanently removes a promo code from the database.
//
// Warning: This is a hard delete. Consider soft delete (is_active = false)
//
//	to maintain usage history and audit trails.
//
// Parameters:
//   - id: string - The promo_code_id to delete
//
// Returns:
//   - error: "promo code not found", database error, or nil on success
func DeletePromoCode(db DBExecutor, id string) error {
	// Validate promo code exists
	err := isPromoThere(db, id)
	if err != nil {
		return err
	}

	// Hard delete promo code
	_, err = db.Exec(`DELETE FROM promocodes WHERE promo_code_id = ?`, id)
	return err
}

// GetActivePromoByCode retrieves an active promo code by its code string.
//
// This function is used during checkout to validate and apply promo codes.
// Only returns codes that are active and not expired.
//
// Parameters:
//   - code: string - The promo code string to look up
//   - now: time.Time - Current timestamp for expiry check
//
// Returns:
//   - *dtos.PromoCodeResponse: Active promo code details (nil if not found or expired)
//   - error: Database error or nil on success
func GetActivePromoByCode(db DBExecutor, code string, now time.Time) (*dtos.PromoCodeResponse, error) {
	// Query active and non-expired promo code
	row := db.QueryRow(`
		SELECT promo_code_id, code, description, discount_type, discount_value, expires_at, is_active
		FROM promocodes WHERE code = ? AND is_active = 1 AND expires_at > ?`, code, now,
	)

	var pc dtos.PromoCodeResponse
	if err := row.Scan(&pc.ID, &pc.Code, &pc.Description, &pc.DiscountType, &pc.DiscountValue, &pc.ExpiresAt, &pc.IsActive); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	return &pc, nil
}

// SetPromoCodeActiveStatus toggles a promo code's active status.
//
// Parameters:
//   - id: string - The promo_code_id to update
//   - isActive: bool - New active status (true = active, false = inactive)
//
// Returns:
//   - error: Database error or nil on success
func SetPromoCodeActiveStatus(db DBExecutor, id string, isActive bool) error {
	_, err := DB.Exec(`
		UPDATE promocodes
		SET is_active = ?
		WHERE promo_code_id = ?`, isActive, id,
	)
	return err
}

// AddPromotionToProduct creates an association between a product and a promotion type.
//
// This function validates both the product and promotion type exist before creating
// the association, and prevents duplicate associations.
//
// Parameters:
//   - req: dtos.AddPromotionToProductRequest containing:
//   - ProductID: The product to associate with promotion
//   - PromotionTypeID: The promotion type to apply to product
//
// Returns:
//   - error: "product not found", "promotion type does not exist",
//     database error, or nil on success (including if already exists)
//
// Workflow:
//  1. Validate product exists in products table
//  2. Validate promotion type exists in promotion_types table
//  3. Check if association already exists (return nil if duplicate)
//  4. Create new association if none exists
func AddPromotionToProduct(db DBExecutor, req dtos.AddPromotionToProductRequest) error {
	// Validate product exists
	err := IsProductThere(db, req.ProductID)
	if err != nil {
		return err
	}

	// Validate promotion type exists
	exist, err := RecordExists(db, "promotion_types", "id = ?", req.PromotionTypeID)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New("promotion type does not exist")
	}

	// Check if association already exists (prevent duplicates)
	exists, err := RecordExists(db, "product_discounts", "product_id = ? AND promotion_type_id = ?", req.ProductID, req.PromotionTypeID)
	if err != nil {
		return err
	}
	if exists {
		return nil // Already associated, not an error
	}

	// Generate unique discount ID
	productDiscountID, _ := shortid.Generate()

	// Create new product-promotion association
	_, err = db.Exec(`
		INSERT INTO product_discounts (product_discount_id, product_id, promotion_type_id)
		VALUES (?, ?, ?)`, productDiscountID, req.ProductID, req.PromotionTypeID,
	)
	return err
}

// RemoveHeldProductPromotions removes a product-promotion association.
//
// This is typically called when updating a product's promotion associations -
// the old associations are removed, then new ones are added.
//
// Parameters:
//   - productDiscountID: string - The product_discount_id to delete
//
// Returns:
//   - error: Database error or nil on success
func RemoveHeldProductPromotions(db DBExecutor, productDiscountID string) error {
	// Delete specific product-promotion association
	_, err := DB.Exec(`DELETE FROM product_discounts WHERE product_discount_id = ?`, productDiscountID)
	return err
}

// HoldProductPromotions retrieves all promotion associations for a given product.
//
// This is used during product updates to get the current promotion list before
// making changes. The term "Hold" suggests these are cached/retrieved for comparison.
//
// Parameters:
//   - productID: string - The product_id to retrieve associations for
//
// Returns:
//   - []string: Array of product_discount_id strings associated with the product
//   - error: Database error or nil on success
func HoldProductPromotions(db DBExecutor, productID string) ([]string, error) {
	// Query all promotion associations for this product
	rows, err := db.Query(`SELECT product_discount_id FROM product_discounts WHERE product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect product discount IDs
	var heldPromotions []string
	for rows.Next() {
		var pdID string
		if err := rows.Scan(&pdID); err != nil {
			return nil, err
		}
		heldPromotions = append(heldPromotions, pdID)
	}
	return heldPromotions, nil
}
