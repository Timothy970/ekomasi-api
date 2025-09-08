package models

import (
	"adenzo_backend/dtos"
	"errors"
	"log"
	"strings"
	"time"
)

func isPromotionThere(id string) error {
	exists, err := RecordExists("promotions", "promotion_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("promotion not found")
	}
	return nil
}
func GetPromotionDetails(promotionID string) (time.Time, time.Time, []string, error) {
	err := isPromotionThere(promotionID)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}
	var startDate, endDate time.Time
	// Get promotion info
	err = DB.QueryRow(`
		SELECT start_date, end_date
		FROM promotions
		WHERE promotion_id = ?`, promotionID).Scan(&startDate, &endDate)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}

	// Get linked products
	rows, err := DB.Query(`
		SELECT product_id
		FROM promotion_products
		WHERE promotion_id = ?`, promotionID)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}
	defer rows.Close()

	var products []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return time.Time{}, time.Time{}, nil, err
		}
		products = append(products, pid)
	}
	return startDate, endDate, products, nil
}

func GetPromotionEffectiveness(productIDs []string, start, end time.Time) ([]dtos.PromotionEffectiveness, error) {
	var query string
	var args []interface{}

	placeholders := "?" + strings.Repeat(",?", len(productIDs)-1)
	query = `
			SELECT oi.product_id, SUM(oi.quantity) AS total_sold, SUM(oi.quantity * oi.unit_price) AS revenue
			FROM order_items oi
			JOIN orders o ON oi.order_id = o.order_id
			WHERE oi.product_id IN (` + placeholders + `)
			  AND o.last_updated_at BETWEEN ? AND ?
			GROUP BY oi.product_id;
		`

	for _, id := range productIDs {
		args = append(args, id)
	}
	args = append(args, start, end)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dtos.PromotionEffectiveness
	for rows.Next() {
		var pe dtos.PromotionEffectiveness
		if err := rows.Scan(&pe.ProductID, &pe.TotalSold, &pe.Revenue); err != nil {
			return nil, err
		}
		results = append(results, pe)
	}
	return results, nil
}

func GetPromotionAggregate(productIDs []string, start, end time.Time) (int64, float64, error) {
	query := `
		SELECT COALESCE(SUM(oi.quantity),0), COALESCE(SUM(oi.quantity * oi.unit_price),0)
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE oi.product_id IN (?` + strings.Repeat(",?", len(productIDs)-1) + `)
		  AND o.last_updated_at BETWEEN ? AND ?
	`
	args := make([]interface{}, len(productIDs)+2)
	for i, id := range productIDs {
		args[i] = id
	}
	args[len(productIDs)] = start
	args[len(productIDs)+1] = end

	var qty int64
	var rev float64
	err := DB.QueryRow(query, args...).Scan(&qty, &rev)
	return qty, rev, err
}
func GetNonPromotionAggregate(start, end time.Time) (int64, float64, error) {
	query := `
		SELECT 
			COALESCE(SUM(oi.quantity), 0) AS total_qty,
			COALESCE(SUM(oi.quantity * oi.unit_price), 0) AS total_rev
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE o.last_updated_at BETWEEN ? AND ?;
	`

	var qty int64
	var rev float64
	err := DB.QueryRow(query, start, end).Scan(&qty, &rev)
	if err != nil {
		return 0, 0, err
	}
	return qty, rev, nil
}

func GetPromotionSummary(start, end time.Time) ([]dtos.PromotionSummary, error) {
	query := `
SELECT 
    p.promotion_id, 
    pt.name, 
    p.start_date, 
    p.end_date,
    COALESCE(SUM(oi.quantity),0) AS total_sold, 
    COALESCE(SUM(oi.quantity * oi.unit_price),0) AS revenue
FROM promotions p
JOIN promotion_types pt ON p.promotion_type_id = pt.id
LEFT JOIN promotion_products pp ON p.promotion_id = pp.promotion_id
LEFT JOIN order_items oi ON pp.product_id = oi.product_id
LEFT JOIN orders o ON oi.order_id = o.order_id
WHERE (o.last_updated_at BETWEEN GREATEST(p.start_date, ?) 
                             AND LEAST(p.end_date, ?)
       OR o.order_id IS NULL)   -- keep promos with no orders
GROUP BY p.promotion_id, pt.name, p.start_date, p.end_date;

	`
	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []dtos.PromotionSummary
	for rows.Next() {
		var ps dtos.PromotionSummary
		if err := rows.Scan(&ps.PromotionID, &ps.PromotionType, &ps.StartDate, &ps.EndDate, &ps.TotalSold, &ps.Revenue); err != nil {
			return nil, err
		}
		summaries = append(summaries, ps)
	}
	return summaries, nil
}

func GetEffectiveness(promotionID string) ([]dtos.PromotionEffectiveness, error) {
	start, end, products, err := GetPromotionDetails(promotionID)
	if err != nil {
		return nil, err
	}
	log.Printf("after get promotion details***********%v", products)
	if len(products) > 0 {
		return GetPromotionEffectiveness(products, start, end)
	}

	return nil, errors.New("no product sold for the promotion")

}

func GetComparison(promotionID string, baselineStart, baselineEnd time.Time) (*dtos.PromotionComparison, error) {
	start, end, products, err := GetPromotionDetails(promotionID)
	if err != nil {
		return nil, err
	}
	promoQty := int64(0)
	promoRev := 0.0
	// Promotion period sales
	log.Printf("product ids*****%v", products)
	if len(products) > 0 {
		log.Printf("get promotion shoul be hit if product ids are there")
		promoQty, promoRev, err = GetPromotionAggregate(products, start, end)
		if err != nil {
			return nil, err
		}
	}
	// Baseline period sales
	baseQty, baseRev, err := GetNonPromotionAggregate(baselineStart, baselineEnd)
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
