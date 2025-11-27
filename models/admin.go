package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
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

func ValidateVoucher(voucherCode string) (float64, error) {
	var balance float64
	var expiry time.Time
	var isActive string

	err := DB.QueryRow(`
		SELECT balance, expiry_date, status
		FROM vouchers 
		WHERE code = ?
	`, voucherCode).Scan(&balance, &expiry, &isActive)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("invalid voucher code")
		}
		return 0, err
	}

	// Check if expired
	if time.Now().After(expiry) {
		return 0, fmt.Errorf("voucher has expired")
	}

	// Check if active
	if isActive != "active" {
		return 0, fmt.Errorf("voucher is inactive")
	}

	if balance <= 0 {
		return 0, fmt.Errorf("voucher has no remaining balance")
	}

	return balance, nil
}

func ValidatePromoCode(voucherCode string, orderValue float64) (dtos.PromoCodeData, error) {
	var promoCode dtos.PromoCodeData
	var expiry time.Time
	var isActive bool
	err := DB.QueryRow(`
	SELECT discount_type, expires_at, is_active, discount_value, minimum_order_value, maximum_use
	FROM promocodes 
	WHERE code = ?
`, voucherCode).Scan(
		&promoCode.DiscountType,
		&expiry,
		&isActive,
		&promoCode.DiscountValue,
		&promoCode.MinimumOrderValue,
		&promoCode.MaximumUse,
	)

	if err != nil {
		log.Println("Error querying promo code:", err)
		if errors.Is(err, sql.ErrNoRows) {
			return dtos.PromoCodeData{}, fmt.Errorf("invalid promo code")
		}
		return dtos.PromoCodeData{}, err
	}

	// Check if expired
	if time.Now().After(expiry) {
		return dtos.PromoCodeData{}, fmt.Errorf("promo code has expired")
	}

	// Check if active
	if !isActive {
		return dtos.PromoCodeData{}, fmt.Errorf("promo code is inactive")
	}
	//check if used up
	if promoCode.MaximumUse <= 0 {
		return dtos.PromoCodeData{}, fmt.Errorf("promo code has been used up")
	}
	//check if minimum order value is met
	if promoCode.MinimumOrderValue != nil && orderValue < *promoCode.MinimumOrderValue {
		return dtos.PromoCodeData{}, fmt.Errorf("minimum order value of %.2f not met", *promoCode.MinimumOrderValue)
	}
	return promoCode, nil
}
func IncrementPromoCodeUsage(code string) error {
	//also increment number of times used
	_, err := DB.Exec(`UPDATE promocodes SET maximum_use = maximum_use - 1, times_used = times_used + 1 WHERE code = ? AND maximum_use > 0`, code)
	return err
}
func UpdateVoucherBalance(code string, newBalance float64) error {
	isRedeemed := true
	_, err := DB.Exec(`UPDATE vouchers SET balance = ?, is_redeemed = ? WHERE code = ?`, newBalance, isRedeemed, code)
	return err
}

func AddProductFeature(input dtos.ProductFeature, productID string) (*dtos.ProductFeature, error) {
	log.Println("Adding feature to product:", productID)
	err := IsProductThere(productID)
	if err != nil {
		return nil, err
	}
	featureID, _ := shortid.Generate()
	jsonProductSpecifications, _ := json.Marshal(input.ProductSpecifications)
	jsonTopSection, _ := json.Marshal(input.TopSection)
	jsonImages, _ := json.Marshal(input.Images)
	_, err = DB.Exec(`
		INSERT INTO product_features (feature_id, product_id, header, description, image, image_position, product_specifications, top_section, design_type, images)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		featureID, productID, input.Header, input.Description, input.Image, input.ImagePosition, jsonProductSpecifications, jsonTopSection, input.DesignType, jsonImages,
	)
	if err != nil {
		return nil, err
	}

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
func isFeatureThere(id string) error {
	exists, err := RecordExists("product_features", "feature_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("feature not found")
	}
	return nil
}
func UpdateProductFeature(input dtos.ProductFeature, featureID string) (*dtos.ProductFeature, error) {
	err := isFeatureThere(featureID)
	if err != nil {
		return nil, err
	}
	jsonProductSpecifications, _ := json.Marshal(input.ProductSpecifications)
	jsonTopSection, _ := json.Marshal(input.TopSection)
	jsonImages, _ := json.Marshal(input.Images)

	// Build dynamic update query
	query := "UPDATE product_features SET "
	args := []interface{}{}
	if input.Image != "" {
		query += "image = ?, "
		args = append(args, input.Image)
	}
	query += "header = ?, description = ?, image_position = ?, product_specifications = ?, top_section = ?, design_type = ?, images = ? WHERE feature_id = ?"
	args = append(args, input.Header, input.Description, input.ImagePosition, jsonProductSpecifications, jsonTopSection, input.DesignType, jsonImages, featureID)

	_, err = DB.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	// Return updated feature
	return GetProductFeatureByID(featureID)
}
func GetProductFeaturesByProductID(productID string) ([]dtos.ProductFeature, error) {
	err := IsProductThere(productID)
	if err != nil {
		return nil, err
	}
	var topSectionStr, productSpecificationsStr, imagesStr sql.NullString
	rows, err := DB.Query(`
		SELECT feature_id, product_id, header, description, image, image_position, product_specifications, top_section, design_type, images
		FROM product_features
		WHERE product_id = ?
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []dtos.ProductFeature
	for rows.Next() {
		var f dtos.ProductFeature
		if err := rows.Scan(&f.ID, &f.ProductID, &f.Header, &f.Description, &f.Image, &f.ImagePosition, &productSpecificationsStr, &topSectionStr, &f.DesignType, &imagesStr); err != nil {
			return nil, err
		}
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
func GetProductFeatureByID(featureID string) (*dtos.ProductFeature, error) {
	err := isFeatureThere(featureID)
	if err != nil {
		return nil, err
	}
	var f dtos.ProductFeature
	var topSectionStr, productSpecificationsStr, imagesStr sql.NullString
	err = DB.QueryRow(`
		SELECT feature_id, product_id, header, description, image, image_position, product_specifications, top_section, design_type, images
		FROM product_features
		WHERE feature_id = ?
	`, featureID).Scan(&f.ID, &f.ProductID, &f.Header, &f.Description, &f.Image, &f.ImagePosition, &productSpecificationsStr, &topSectionStr, &f.DesignType, &imagesStr)

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
		return nil, err
	}
	return &f, nil
}
func DeleteProductFeature(featureID string) error {
	err := isFeatureThere(featureID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM product_features WHERE feature_id = ?`, featureID)
	return err
}
