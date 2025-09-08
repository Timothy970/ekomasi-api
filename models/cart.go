package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/teris-io/shortid"
)

func isCartThere(id string) error {
	exists, err := RecordExists("cart", "cart_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("cart not found")
	}
	return nil
}
func UpdateCartTimestamp(cartID string) error {
	_, err := DB.Exec(`
        UPDATE cart 
        SET updated_at = NOW() 
        WHERE cart_id = ?`, cartID)
	return err
}

func InsertCartItem(cartID string, productID string, quantity int) error {
	err := isCartThere(cartID)
	if err != nil {
		return err
	}
	err = isProductThere(productID)
	if err != nil {
		return err
	}
	// Generate cart ID
	ID, _ := shortid.Generate()

	// Insert the item
	_, err = DB.Exec(`
        INSERT INTO cart_items(id, cart_id, product_id, quantity)
        VALUES (?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE quantity = quantity + VALUES(quantity)
    `, ID, cartID, productID, quantity)
	if err != nil {
		return fmt.Errorf("failed to insert cart item: %w", err)
	}
	err = UpdateCartTimestamp(cartID)
	if err != nil {
		return err
	}
	return nil
}
func CreateCart(req dtos.CreateCartRequest) (string, error) {
	// Generate cart ID
	cartID, _ := shortid.Generate()
	if req.UserID != nil {
		cartID, err := GetUserCart(*req.UserID)
		if err != nil {
			return "", err
		}
		if cartID != "" {
			return cartID, nil
		}
	}
	// Insert the item
	_, err := DB.Exec(`
        INSERT INTO cart(cart_id, user_id)
        VALUES (?, ?)
    `, cartID, req.UserID)
	return cartID, err
}
func GetUserCart(userID string) (string, error) {
	var cartID string
	err := DB.QueryRow(`
        SELECT cart_id FROM cart WHERE user_id = ?
    `, userID).Scan(&cartID)

	if err == sql.ErrNoRows {
		// No cart yet for this user
		return "", nil
	}

	return cartID, err
}

func GetCartItems(cartID string) ([]dtos.CartItem, error) {
	err := isCartThere(cartID)
	if err != nil {
		return nil, err
	}
	rows, err := DB.Query(`
		SELECT c.product_id, p.name, c.quantity, p.price
		FROM cart_items c
		JOIN products p ON c.product_id = p.product_id
		WHERE c.cart_id = ?
	`, cartID)
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

func UpdateCartItem(cartID string, productID string, quantity int) error {
	err := isCartThere(cartID)
	if err != nil {
		return err
	}
	err = isProductThere(productID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
		UPDATE cart_items SET quantity = ?
		WHERE cart_id = ? AND product_id = ?
	`, quantity, cartID, productID)
	err = UpdateCartTimestamp(cartID)
	return err
}

func DeleteCartItem(cartID, productID string) error {
	err := isCartThere(cartID)
	if err != nil {
		return err
	}
	err = isProductThere(productID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
		DELETE FROM cart_items WHERE cart_id = ? AND product_id = ?
	`, cartID, productID)
	err = UpdateCartTimestamp(cartID)
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

// GET /analytics/cart-abandonment?period=month&start=2025-08-01&end=2025-08-31

func GetCartAbandonmentRate(start, end time.Time) (*AbandonmentStats, error) {
	var cartsCreated, completedPurchases, abandonedCarts int

	// carts created in period
	err := DB.QueryRow(`
		SELECT COUNT(*) 
		FROM cart 
		WHERE created_at BETWEEN ? AND ?
	`, start, end).Scan(&cartsCreated)
	if err != nil {
		return nil, err
	}

	// completed purchases = carts updated within last 7 days OR empty carts
	err = DB.QueryRow(`
    SELECT COUNT(*)
    FROM (
        SELECT c.cart_id
        FROM cart c
        LEFT JOIN cart_items ci ON c.cart_id = ci.cart_id
        WHERE c.updated_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
           OR c.cart_id NOT IN (SELECT cart_id FROM cart_items)
        GROUP BY c.cart_id
    ) AS completed
`).Scan(&completedPurchases)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// abandoned carts = carts with items that haven't been updated in >7 days
	threshold := time.Now().AddDate(0, 0, -7)
	err = DB.QueryRow(`
		SELECT COUNT(DISTINCT c.cart_id)
		FROM cart c
		JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE c.created_at BETWEEN ? AND ?
		  AND c.updated_at < ?
	`, start, end, threshold).Scan(&abandonedCarts)
	if err != nil {
		return nil, err
	}

	// calculate abandonment rate
	rate := 0.0
	if cartsCreated > 0 {
		rate = (float64(abandonedCarts) / float64(cartsCreated)) * 100
	}

	return &AbandonmentStats{
		TotalCarts: cartsCreated,
		// CompletedPurchases: completedPurchases,
		AbandonedCarts: abandonedCarts,
		Rate:           rate,
	}, nil
}

type AbandonmentTrend struct {
	Date         string `json:"date"`
	CartsCreated int    `json:"carts_created"`
	// CompletedPurchases int     `json:"completed_purchases"`
	AbandonmentRate float64 `json:"abandonment_rate"`
}
type AbandonmentStats struct {
	// CompletedPurchases int     `json:"active_carts"`
	Rate           float64 `json:"rate"`
	AbandonedCarts int     `json:"abandoned_carts"`
	TotalCarts     int     `json:"total_carts"`
}

func GetCartAbandonmentTrend(start, end time.Time, period string) ([]AbandonmentTrend, error) {
	var groupBy, periodSelect string

	switch period {
	case "daily":
		groupBy = "DATE(c.created_at)"
		periodSelect = "DATE(c.created_at)"
	case "weekly":
		groupBy = "YEARWEEK(c.created_at)"
		periodSelect = "YEARWEEK(c.created_at)"
	case "monthly":
		groupBy = "DATE_FORMAT(c.created_at, '%Y-%m')"
		periodSelect = "DATE_FORMAT(c.created_at, '%Y-%m')"
	case "quarterly":
		groupBy = "CONCAT(YEAR(c.created_at), '-Q', QUARTER(c.created_at))"
		periodSelect = "CONCAT(YEAR(c.created_at), '-Q', QUARTER(c.created_at))"
	case "yearly":
		groupBy = "YEAR(c.created_at)"
		periodSelect = "YEAR(c.created_at)"
		// default:
		// 	groupBy = "DATE(c.created_at)" /
		// 	periodSelect = "DATE(c.created_at)"
	}

	query := fmt.Sprintf(`
		SELECT %s AS period_date,
			   COUNT(DISTINCT c.cart_id) AS carts_created,
			   SUM(CASE 
					   WHEN c.updated_at >= DATE_SUB(NOW(), INTERVAL 7 DAY) 
							OR NOT EXISTS (SELECT 1 FROM cart_items ci WHERE ci.cart_id = c.cart_id) 
					   THEN 1 ELSE 0 
				   END) AS completed_purchases
		FROM cart c
		WHERE c.created_at BETWEEN ? AND ?
		GROUP BY %s
		ORDER BY %s ASC
	`, periodSelect, groupBy, groupBy)

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AbandonmentTrend
	threshold := time.Now().AddDate(0, 0, -7)

	for rows.Next() {
		var periodDate string
		var cartsCreated, completedPurchases int

		if err := rows.Scan(&periodDate, &cartsCreated, &completedPurchases); err != nil {
			return nil, err
		}

		// abandoned carts = carts with items not updated in >7 days
		var abandonedCarts int
		abandonedQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT c.cart_id)
			FROM cart c
			JOIN cart_items ci ON c.cart_id = ci.cart_id
			WHERE %s = ?
			  AND c.updated_at < ?
		`, periodSelect)

		err = DB.QueryRow(abandonedQuery, periodDate, threshold).Scan(&abandonedCarts)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}

		rate := 0.0
		if cartsCreated > 0 {
			rate = (float64(abandonedCarts) / float64(cartsCreated)) * 100
		}

		results = append(results, AbandonmentTrend{
			Date:         periodDate,
			CartsCreated: cartsCreated,
			// CompletedPurchases: completedPurchases,
			AbandonmentRate: rate,
		})
	}

	return results, nil
}

type CartItem struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}

type Cart struct {
	CartID    string     `json:"cart_id"`
	UserID    *string    `json:"user_id,omitempty"`
	Items     []CartItem `json:"items"`
	CreatedAt time.Time  `json:"created_at"`
}

type PaginationMeta struct {
	Page       int  `json:"page"`
	Size       int  `json:"size"`
	TotalItems int  `json:"total_items"`
	TotalPages int  `json:"total_pages"`
	HasPrev    bool `json:"has_prev"`
	HasNext    bool `json:"has_next"`
}

type AbandonedCartsResponse struct {
	Carts []Cart         `json:"carts"`
	Meta  PaginationMeta `json:"meta"`
}

type AnalyticsRepository struct {
	DB *sql.DB
}

// GetAbandonedCarts returns carts with items older than 7 days (considered abandoned)
func (r *AnalyticsRepository) GetAbandonedCarts(page, size int) (*AbandonedCartsResponse, error) {
	offset := (page - 1) * size

	// Step 1: Count total abandoned carts
	var totalItems int
	countQuery := `
		SELECT COUNT(DISTINCT c.cart_id)
		FROM cart c
		JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE ci.created_at < NOW() - INTERVAL 7 DAY
	`
	if err := r.DB.QueryRow(countQuery).Scan(&totalItems); err != nil {
		return nil, fmt.Errorf("count abandoned carts: %w", err)
	}

	// Step 2: Get paginated abandoned carts
	query := `
		SELECT c.cart_id, c.user_id, c.created_at
		FROM cart c
		JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE ci.created_at < NOW() - INTERVAL 7 DAY
		GROUP BY c.cart_id
		ORDER BY c.created_at ASC
		LIMIT ? OFFSET ?
	`

	rows, err := r.DB.Query(query, size, offset)
	if err != nil {
		return nil, fmt.Errorf("query abandoned carts: %w", err)
	}
	defer rows.Close()

	var carts []Cart
	for rows.Next() {
		var cart Cart
		if err := rows.Scan(&cart.CartID, &cart.UserID, &cart.CreatedAt); err != nil {
			return nil, err
		}

		// Step 3: Get items for this cart
		itemQuery := `
			SELECT id, product_id, quantity, created_at
			FROM cart_items
			WHERE cart_id = ? AND created_at < NOW() - INTERVAL 7 DAY
		`
		itemRows, err := r.DB.Query(itemQuery, cart.CartID)
		if err != nil {
			return nil, err
		}

		for itemRows.Next() {
			var item CartItem
			if err := itemRows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.CreatedAt); err != nil {
				itemRows.Close()
				return nil, err
			}
			cart.Items = append(cart.Items, item)
		}
		itemRows.Close()

		carts = append(carts, cart)
	}

	// Step 4: Build pagination meta
	totalPages := (totalItems + size - 1) / size
	meta := PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return &AbandonedCartsResponse{
		Carts: carts,
		Meta:  meta,
	}, nil
}
