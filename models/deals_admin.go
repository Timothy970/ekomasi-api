// Package models provides data access layer for the Ekomasi e-commerce platform.
//
// This file handles deals/promotions management including:
//   - Deal CRUD operations (time-limited promotional campaigns)
//   - Product-deal associations (linking products to deals with specific discounts)
//   - Deal retrieval with product listings
//   - Pagination support for deal browsing
//
// Deals represent promotional campaigns with:
//   - Name uniqueness validation
//   - Start and end dates for time-limited offers
//   - Active/inactive status control
//   - Per-product discount overrides (percentage or fixed amount)
//   - Deep linking for marketing campaigns
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"strconv"
)

// Helper function to fetch product discount for a specific product in a deal
func GetProductDiscount(db DBExecutor, productID string) (float64, string, error) {
	var discount float64
	var discountType string
	err := db.QueryRow(`
		SELECT dp.discount, dp.discount_type
		FROM deal_products dp
		INNER JOIN deals d ON dp.deal_id = d.deal_id
		WHERE dp.product_id = ? AND d.end_date > NOW()
		LIMIT 1
	`, productID).Scan(&discount, &discountType)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", nil // No discount for this product
		}
		return 0, "", err
	}
	return discount, discountType, nil
}

//helper funtion to return product in a deal
// params: brandID, dealType, brandDiscount, brandDiscountType
//returns: []dtos.ProductsDeal, error

func GetProductIDsByBrandID(db DBExecutor, brandID, brandDiscount, brandDiscountType string) ([]dtos.ProductsDeal, error) {
	var products []dtos.ProductsDeal
	var productIds []string
	discountValue, err := strconv.ParseFloat(brandDiscount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid brand discount %q", brandDiscount)
	}
	parsedDiscount := dtos.FloatOrString(discountValue)

	rows, err := db.Query(`SELECT product_id FROM product_variants WHERE variant_id = ?`, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var productID string
		if err := rows.Scan(&productID); err != nil {
			return nil, err
		}
		productIds = append(productIds, productID)
	}
	if len(productIds) == 0 {
		return nil, fmt.Errorf("no products found for brand ID %s", brandID)
	}
	for _, productID := range productIds {
		products = append(products, dtos.ProductsDeal{
			ProductID:    productID,
			Discount:     parsedDiscount,
			DiscountType: brandDiscountType,
		})
	}
	return products, nil
}
