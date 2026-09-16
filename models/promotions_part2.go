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
	"log"
	"time"

	"github.com/teris-io/shortid"
)

// GetEffectiveness retrieves effectiveness metrics for a specific promotion.
//
// This is a convenience function that combines GetPromotionDetails and
// GetPromotionEffectiveness for a complete promotion analysis.
//
// Parameters:
//   - promotionID: string - The promotion_id to analyze
//
// Returns:
//   - []dtos.PromotionEffectiveness: Per-product effectiveness data
//   - error: "promotion not found", "no product sold for the promotion", or database error
func GetEffectiveness(db DBExecutor, promotionID string) ([]dtos.PromotionEffectiveness, error) {
	// Get promotion details (date range and products)
	start, end, products, err := GetPromotionDetails(db, promotionID)
	if err != nil {
		return nil, err
	}

	log.Printf("after get promotion details***********%v", products)

	// Calculate effectiveness if products exist
	if len(products) > 0 {
		return GetPromotionEffectiveness(db, products, start, end)
	}

	return nil, errors.New("no product sold for the promotion")

}

// GetComparison compares promotion performance against a baseline period.
//
// This function calculates the lift/impact of a promotion by comparing
// sales during the promotion period vs. a baseline (non-promotional) period.
//
// Parameters:
//   - promotionID: string - The promotion_id to analyze
//   - baselineStart: time.Time - Baseline comparison period start date
//   - baselineEnd: time.Time - Baseline comparison period end date
//
// Returns:
//   - *dtos.PromotionComparison: Comparison data containing:
//   - PromotionSales: Quantity sold during promotion
//   - PromotionRev: Revenue during promotion
//   - BaselineSales: Quantity sold during baseline period (all products)
//   - BaselineRev: Revenue during baseline period (all products)
//   - error: "promotion not found" or database error
func GetComparison(db DBExecutor, promotionID string, baselineStart, baselineEnd time.Time) (*dtos.PromotionComparison, error) {
	// Get promotion details (date range and products)
	start, end, products, err := GetPromotionDetails(db, promotionID)
	if err != nil {
		return nil, err
	}

	// Initialize promotion metrics
	promoQty := int64(0)
	promoRev := 0.0

	// Calculate promotion period sales (only if products are linked)
	log.Printf("product ids*****%v", products)
	if len(products) > 0 {
		log.Printf("get promotion shoul be hit if product ids are there")
		promoQty, promoRev, err = GetPromotionAggregate(db, products, start, end)
		if err != nil {
			return nil, err
		}
	}

	// Calculate baseline period sales (all products)
	baseQty, baseRev, err := GetNonPromotionAggregate(db, baselineStart, baselineEnd)
	if err != nil {
		return nil, err
	}

	return &dtos.PromotionComparison{
		PromotionSales: promoQty,
		PromotionRev:   promoRev,
		BaselineSales:  baseQty,
		BaselineRev:    baseRev,
	}, nil
}

// isPromoCodeTaken validates that a promo code is not already in use.
//
// This is an internal helper function used before promo code creation.
//
// Parameters:
//   - code: string - The promo code to check
//
// Returns:
//   - error: "promo code <code> already exists" if taken, database error, or nil if available
func isPromoCodeTaken(db DBExecutor, code string) error {
	var exists bool

	// Check if code already exists
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM promocodes WHERE code = ?)`, code).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("promo code " + code + " already exists")
	}
	return nil
}

// AddPromoCode creates a new promo code with validation and auto-generation.
//
// This function creates a promo code with either a user-specified code or
// an auto-generated 8-character secure random code. It validates expiry dates
// and code uniqueness before creation.
//
// Parameters:
//   - input: dtos.PromoCodeRequest containing:
//   - Discount_Code: Optional custom code (if empty, auto-generates 8-char code)
//   - Description: Promo code description
//   - DiscountType: Type of discount (e.g., "percentage", "fixed")
//   - DiscountValue: Discount amount or percentage
//   - ExpiresAt: Expiry timestamp string (must be future date)
//   - IsActive: Whether promo is immediately active
//   - MinimumOrderValue: Minimum order amount to apply promo
//   - MaximumUse: Maximum number of redemptions allowed
//
// Returns:
//   - *dtos.PromoCodeRequest: Created promo code with generated ID
//   - error: "promo code already exists", "expiry time must be in the future",
//     database error, or nil on success
func AddPromoCode(db DBExecutor, input dtos.PromoCodeRequest) (*dtos.PromoCodeRequest, error) {
	// Validate code uniqueness if provided
	err := isPromoCodeTaken(db, input.Discount_Code)
	if err != nil {
		return nil, err
	}

	// Generate unique promo code ID
	id, _ := shortid.Generate()

	// Generate or use provided promo code
	code, err := secureRandomString(8) // Auto-generate 8-char code
	if input.Discount_Code != "" {
		code = input.Discount_Code // Use custom code if provided
	}
	if err != nil {
		return nil, err
	}

	// Parse and validate expiry date
	expiryTime := StringToTime(input.ExpiresAt)
	if time.Now().After(expiryTime) {
		return nil, errors.New("expiry time must be in the future")
	}

	// Insert promo code
	_, err = db.Exec(`
		INSERT INTO promocodes (promo_code_id, code, description, discount_type, discount_value, expires_at, is_active, minimum_order_value, maximum_use, promo_type, brand_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, code, input.Description, input.DiscountType, input.DiscountValue, expiryTime, input.IsActive, input.MinimumOrderValue, input.MaximumUse, input.PromoType, input.BrandID,
	)
	if err != nil {
		return nil, err
	}

	input.ID = id
	return &input, nil
}

// isPromoThere validates if a promo code exists in the database.
//
// This is an internal helper function used before promo code operations.
//
// Parameters:
//   - id: string - The promo_code_id to validate
//
// Returns:
//   - error: "promo code not found" if ID doesn't exist, database error, or nil if exists
func isPromoThere(db DBExecutor, id string) error {
	// Check promo code existence
	exists, err := RecordExists(db, "promocodes", "promo_code_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("promo code not found")
	}
	return nil
}

// UpdatePromoCode updates an existing promo code's details.
//
// Parameters:
//   - id: string - The promo_code_id to update
//   - input: dtos.PromoCodeRequest - Updated promo code details:
//   - Description, DiscountType, DiscountValue
//   - ExpiresAt: New expiry timestamp
//   - IsActive: Active status
//   - MinimumOrderValue, MaximumUse
//
// Returns:
//   - *dtos.PromoCodeRequest: Updated promo code
//   - error: "promo code not found", database error, or nil on success
func UpdatePromoCode(db DBExecutor, id string, input dtos.PromoCodeRequest) (*dtos.PromoCodeRequest, error) {
	// Validate promo code exists
	err := isPromoThere(db, id)
	if err != nil {
		return nil, err
	}

	// Update promo code attributes (code itself is not updated)
	_, err = db.Exec(`
		UPDATE promocodes
		SET code = ?, description = ?, discount_type = ?, discount_value = ?, expires_at = ?, is_active = ?, minimum_order_value = ?, maximum_use = ?, promo_type = ?, brand_id = ?
		WHERE promo_code_id = ?`,
		input.Discount_Code, input.Description, input.DiscountType, input.DiscountValue, input.ExpiresAt, input.IsActive, input.MinimumOrderValue, input.MaximumUse, input.PromoType, input.BrandID, id,
	)
	if err != nil {
		return nil, err
	}

	input.ID = id
	return &input, nil
}

// GetPromoCodeByID retrieves a single promo code by its ID.
//
// Parameters:
//   - id: string - The promo_code_id to retrieve
//
// Returns:
//   - *dtos.PromoCodeResponse: Promo code details including:
//   - ID, Code, Description
//   - DiscountType, DiscountValue
//   - ExpiresAt, IsActive
//   - MinimumOrderValue, MaximumUse, TimesUsed
//   - PromoType, BrandID
//   - error: "promo code not found", nil if not found (sql.ErrNoRows), or database error
func GetPromoCodeByID(db DBExecutor, id string) (*dtos.PromoCodeResponse, error) {
	// Validate promo code exists
	err := isPromoThere(db, id)
	if err != nil {
		return nil, err
	}

	// Query promo code details
	row := db.QueryRow(`
		SELECT promo_code_id, code, description, discount_type, discount_value, expires_at, is_active, minimum_order_value, maximum_use, times_used, promo_type, brand_id
		FROM promocodes WHERE promo_code_id = ?`, id,
	)

	var pc dtos.PromoCodeResponse
	if err := row.Scan(&pc.ID, &pc.Code, &pc.Description, &pc.DiscountType, &pc.DiscountValue, &pc.ExpiresAt, &pc.IsActive, &pc.MinimumOrderValue, &pc.MaximumUse, &pc.TimesUsed, &pc.PromoType, &pc.BrandID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &pc, nil
}
