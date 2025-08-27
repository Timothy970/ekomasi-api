package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/teris-io/shortid"
)

func CreateCoupon(promo dtos.PromoCode) error {
	couponID, _ := shortid.Generate()
	// Validate date format
	layout := "2006-01-02 15:04:05"
	_, err := time.Parse(layout, promo.ExpiryDate)
	if err != nil {
		return fmt.Errorf("invalid date format: %v", err)
	}

	query := `
		INSERT INTO coupons (coupon_id, code, discount_pct, expires_at)
		VALUES (?, ?, ?, ?)
	`
	_, err = DB.Exec(query, couponID, promo.Code, promo.DiscountPercentage, promo.ExpiryDate)
	if err != nil {
		return fmt.Errorf("failed to insert promo code: %v", err)
	}
	return nil
}

func ValidateCoupon(couponCode string) (float64, error) {
	var discount float64
	var expiry string

	err := DB.QueryRow(`
		SELECT discount_pct, expires_at 
		FROM coupons 
		WHERE code = ?
	`, couponCode).Scan(&discount, &expiry)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("invalid coupon code")
		}
		return 0, err
	}
	log.Printf("current time:%s", time.Now())
	// Parse expiry date
	expiryTime, err := time.Parse(time.RFC3339, expiry)
	if err != nil {
		log.Printf("Parse error: raw expiry: %s", expiry)
		return 0, fmt.Errorf("invalid expiry date format")
	}

	// Check if expired
	if time.Now().After(expiryTime) {
		return 0, fmt.Errorf("coupon has expired")
	}

	return discount, nil
}
