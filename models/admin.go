// Package models provides database models and operations for admin functionality.
// Implements coupon, voucher, and promo code validation and management.
// Also handles product feature management with JSON field support.
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/teris-io/shortid"
)

// ===== Coupon Management =====

// CreateCoupon creates a new discount coupon with expiration date.
// Used for promotional campaigns and customer discounts.
//
// Parameters:
//   - promo: PromoCode containing code, discount percentage, and expiry date (format: "2006-01-02 15:04:05")
//
// Returns:
//   - error: Error if date format invalid or database operation fails
func CreateCoupon(db DBExecutor, promo dtos.PromoCode) error {
	// Generate unique coupon ID
	couponID, _ := shortid.Generate()
	// Validate date format matches expected layout
	layout := "2006-01-02 15:04:05"
	_, err := time.Parse(layout, promo.ExpiryDate)
	if err != nil {
		// Date format validation failed
		return fmt.Errorf("invalid date format: %v", err)
	}

	// Insert coupon into database
	query := `
		INSERT INTO coupons (coupon_id, code, discount_pct, expires_at)
		VALUES (?, ?, ?, ?)
	`
	_, err = db.Exec(query, couponID, promo.Code, promo.DiscountPercentage, promo.ExpiryDate)
	if err != nil {
		// Database insert failed
		return fmt.Errorf("failed to insert promo code: %v", err)
	}
	return nil
}

// ValidateCoupon validates a coupon code and returns its discount percentage.
// Checks if coupon exists, is not expired, and returns the discount value.
//
// Parameters:
//   - couponCode: The coupon code to validate (e.g., "SAVE20")
//
// Returns:
//   - float64: Discount percentage if coupon is valid (e.g., 20.0 for 20% off)
//   - error: Error if coupon invalid, expired, or database error
func ValidateCoupon(db DBExecutor, couponCode string) (float64, error) {
	var discount float64
	var expiry string

	// Query coupon details from database
	err := db.QueryRow(`
		SELECT discount_pct, expires_at 
		FROM coupons 
		WHERE code = ?
	`, couponCode).Scan(&discount, &expiry)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Coupon code not found in database
			return 0, fmt.Errorf("invalid coupon code")
		}
		// Database query failed
		return 0, err
	}
	log.Printf("current time:%s", time.Now())
	// Parse expiry date string to time.Time
	expiryTime, err := time.Parse(time.RFC3339, expiry)
	if err != nil {
		// Log raw expiry for debugging
		log.Printf("Parse error: raw expiry: %s", expiry)
		return 0, fmt.Errorf("invalid expiry date format")
	}

	// Check if coupon has expired
	if time.Now().After(expiryTime) {
		// Coupon is past expiration date
		return 0, fmt.Errorf("coupon has expired")
	}

	// Coupon valid - return discount percentage
	return discount, nil
}

// ===== Voucher Management =====

// ValidateVoucher validates a voucher code and returns its remaining balance.
// Checks if voucher exists, is active, not expired, and has remaining balance.
//
// Parameters:
//   - voucherCode: The voucher code to validate
//
// Returns:
//   - float64: Remaining voucher balance if valid
//   - error: Error if voucher invalid, expired, inactive, or has no balance
func ValidateVoucher(db DBExecutor, voucherCode string) (float64, error) {
	var balance float64
	var expiry time.Time
	var isActive string

	// Query voucher details from database
	err := db.QueryRow(`
		SELECT balance, expiry_date, status
		FROM vouchers 
		WHERE code = ?
	`, voucherCode).Scan(&balance, &expiry, &isActive)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Voucher code not found in database
			return 0, fmt.Errorf("invalid voucher code")
		}
		// Database query failed
		return 0, err
	}

	// Check if voucher has expired
	if time.Now().After(expiry) {
		// Voucher is past expiration date
		return 0, fmt.Errorf("voucher has expired")
	}

	// Check if voucher is active
	if isActive != "active" {
		// Voucher has been deactivated
		return 0, fmt.Errorf("voucher is inactive")
	}

	// Check if voucher has remaining balance
	if balance <= 0 {
		// Voucher has been fully used
		return 0, fmt.Errorf("voucher has no remaining balance")
	}

	// Voucher valid - return remaining balance
	return balance, nil
}

// ===== Promo Code Management =====

// ValidatePromoCode validates a promotional code against order value and usage limits.
// More advanced than coupons - supports minimum order values, usage limits, and different discount types.
//
// Parameters:
//   - voucherCode: The promo code to validate
//   - orderValue: Current order total amount for minimum order validation
//
// Returns:
//   - dtos.PromoCodeData: Promo code details including discount type, value, and usage info
//   - error: Error if promo invalid, expired, inactive, used up, or order value too low
func ValidatePromoCode(db DBExecutor, voucherCode string, orderValue float64) (dtos.PromoCodeData, error) {
	var promoCode dtos.PromoCodeData
	var expiry time.Time
	var isActive bool
	// Query promo code details from database
	err := db.QueryRow(`
	SELECT discount_type, expires_at, is_active, discount_value, minimum_order_value, maximum_use, promo_type, brand_id
	FROM promocodes 
	WHERE code = ?
`, voucherCode).Scan(
		&promoCode.DiscountType,
		&expiry,
		&isActive,
		&promoCode.DiscountValue,
		&promoCode.MinimumOrderValue,
		&promoCode.MaximumUse,
		&promoCode.PromoType,
		&promoCode.BrandID,
	)

	if err != nil {
		log.Println("Error querying promo code:", err)
		if errors.Is(err, sql.ErrNoRows) {
			// Promo code not found in database
			return dtos.PromoCodeData{}, fmt.Errorf("invalid promo code")
		}
		// Database query failed
		return dtos.PromoCodeData{}, err
	}

	// Check if promo code has expired
	if time.Now().After(expiry) {
		// Promo code is past expiration date
		return dtos.PromoCodeData{}, fmt.Errorf("promo code has expired")
	}

	// Check if promo code is active
	if !isActive {
		// Promo code has been deactivated
		return dtos.PromoCodeData{}, fmt.Errorf("promo code is inactive")
	}
	// Check if promo code has remaining uses
	//Note if if maximum use is 0, this means unlimited uses, so we only check if it's less than 0 which indicates all uses have been consumed
	if promoCode.MaximumUse < 0 {
		// All available uses have been consumed
		return dtos.PromoCodeData{}, fmt.Errorf("promo code has been used up")
	}
	// Check if order meets minimum order value requirement
	if promoCode.MinimumOrderValue != nil && orderValue < *promoCode.MinimumOrderValue {
		// Order total is below minimum required
		return dtos.PromoCodeData{}, fmt.Errorf("minimum order value of %.2f not met", *promoCode.MinimumOrderValue)
	}
	// Promo code valid - return details
	return promoCode, nil
}

// IncrementPromoCodeUsage decrements remaining uses and increments usage count.
// Called after successful promo code application to order.
//
// Parameters:
//   - code: The promo code that was used
//
// Returns:
//   - error: Error if database operation fails
func IncrementPromoCodeUsage(db DBExecutor, code string) error {
	// Decrement maximum_use and increment times_used atomically
	// Only updates if maximum_use > 0 to prevent negative values
	_, err := db.Exec(`UPDATE promocodes SET maximum_use = maximum_use - 1, times_used = times_used + 1 WHERE code = ? AND maximum_use > 0`, code)
	return err
}

// UpdateVoucherBalance updates voucher balance and marks as redeemed.
// Called after voucher is applied to order to deduct used amount.
//
// Parameters:
//   - code: The voucher code to update
//   - newBalance: Updated balance after deduction
//
// Returns:
//   - error: Error if database operation fails
func UpdateVoucherBalance(db DBExecutor, code string, newBalance float64) error {
	// Mark voucher as redeemed when balance is updated
	isRedeemed := true
	_, err := db.Exec(`UPDATE vouchers SET balance = ?, is_redeemed = ? WHERE code = ?`, newBalance, isRedeemed, code)
	return err
}

// ===== Product Feature Management =====

// AddProductFeature adds a feature section to a product (e.g., specifications, images, descriptions).
// Supports flexible product presentation with JSON fields for complex data structures.
//
// Parameters:
//   - input: ProductFeature containing header, description, images, specifications, and design settings
//   - productID: ID of the product to add feature to
//
// Returns:
//   - *dtos.ProductFeature: Created feature with generated ID
//   - error: Error if product not found or database operation fails
func AddProductFeature(db DBExecutor, input dtos.ProductFeature, productID string) (*dtos.ProductFeature, error) {
	log.Println("Adding feature to product:", productID)
	log.Printf("Feature input: %+v", input)
	// Validate that product exists before adding feature
	err := IsProductThere(db, productID)
	if err != nil {
		// Product not found
		return nil, err
	}
	// Generate unique feature ID
	featureID, _ := shortid.Generate()
	// Marshal complex JSON fields for database storage
	jsonProductSpecifications, _ := json.Marshal(input.ProductSpecifications)
	jsonTopSection, _ := json.Marshal(input.TopSection)
	jsonImages, _ := json.Marshal(input.Images)
	// Insert product feature into database
	_, err = db.Exec(`
		INSERT INTO product_features (feature_id, product_id, header, description, image, image_position, product_specifications, top_section, design_type, images)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		featureID, productID, input.Header, input.Description, input.Image, input.ImagePosition, jsonProductSpecifications, jsonTopSection, input.DesignType, jsonImages,
	)
	if err != nil {
		// Database insert failed
		return nil, err
	}

	// Return created feature with generated ID
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
