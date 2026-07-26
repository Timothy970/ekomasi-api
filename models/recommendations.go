package models

import (
	"context"
	"ekomasi_backend/dtos"
	"fmt"
)

// GetFrequentlyBoughtTogetherProducts returns products co-ordered with the given product_id
func GetFrequentlyBoughtTogetherProducts(db DBExecutor, productID, tenantID string, limit int) ([]dtos.Product, error) {
	if limit <= 0 {
		limit = 4
	}

	query := `
		SELECT p.id, p.name, p.price, p.description, p.sku, p.created_at
		FROM order_items oi1
		JOIN order_items oi2 ON oi1.order_id = oi2.order_id AND oi1.product_id != oi2.product_id
		JOIN products p ON oi2.product_id = p.id
		WHERE oi1.product_id = ? AND p.tenant_id = ? AND p.is_active = 1
		GROUP BY p.id
		ORDER BY COUNT(oi2.order_id) DESC
		LIMIT ?
	`

	rows, err := db.QueryContext(context.Background(), query, productID, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recommendations: %w", err)
	}
	defer rows.Close()

	var products []dtos.Product
	for rows.Next() {
		var p dtos.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Description, &p.SKU, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan recommended product: %w", err)
		}
		products = append(products, p)
	}
	return products, nil
}
