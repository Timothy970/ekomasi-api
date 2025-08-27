package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"

	"github.com/teris-io/shortid"
)

func InsertCartItem(userID string, productID string, quantity int) error {
	// Generate cart ID
	cartID, err := shortid.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate cart ID: %w", err)
	}

	// Insert the item
	_, err = DB.Exec(`
        INSERT INTO cart_items(id, user_id, product_id, quantity)
        VALUES (?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE quantity = quantity + VALUES(quantity)
    `, cartID, userID, productID, quantity)
	if err != nil {
		return fmt.Errorf("failed to insert cart item: %w", err)
	}

	return nil
}

func GetCartItems(userID string) ([]dtos.CartItem, error) {
	rows, err := DB.Query(`
		SELECT c.product_id, p.name, c.quantity, p.price
		FROM cart_items c
		JOIN products p ON c.product_id = p.product_id
		WHERE c.user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dtos.CartItem
	for rows.Next() {
		var item dtos.CartItem
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func UpdateCartItem(userID string, productID string, quantity int) error {
	_, err := DB.Exec(`
		UPDATE cart_items SET quantity = ?
		WHERE user_id = ? AND product_id = ?
	`, quantity, userID, productID)
	return err
}

func DeleteCartItem(userID, productID string) error {
	_, err := DB.Exec(`
		DELETE FROM cart_items WHERE user_id = ? AND product_id = ?
	`, userID, productID)
	return err
}

// get product promotion data
func GetProductPromotionData(productID string) (dtos.PromotionData, error) {
	query := `
		SELECT 
			pt.name AS promotion_type, 
			pt.value AS promotion_value
		FROM promotion_products pp
		INNER JOIN promotions p ON p.promotion_id = pp.promotion_id
		INNER JOIN promotion_types pt ON pt.id = p.promotion_type_id
		WHERE pp.product_id = ?
		LIMIT 1
	`

	var promotionType string
	var promotionValue float64 // change to float64 if that's the actual DB type

	err := DB.QueryRow(query, productID).Scan(&promotionType, &promotionValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return dtos.PromotionData{}, nil // No promotion for product
		}
		return dtos.PromotionData{}, err
	}

	return dtos.PromotionData{
		Type:  promotionType,
		Value: promotionValue,
	}, nil
}
