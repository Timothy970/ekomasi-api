package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/teris-io/shortid"
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
func isPromoCodeTaken(code string) error {
	var exists bool
	err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM promocodes WHERE code = ?)`, code).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("promo code " + code + " already exists")
	}
	return nil
}
func AddPromoCode(input dtos.PromoCodeRequest) (*dtos.PromoCodeRequest, error) {
	err := isPromoCodeTaken(input.Discount_Code)
	if err != nil {
		return nil, err
	}
	id, _ := shortid.Generate()
	code, err := secureRandomString(8)
	if input.Discount_Code != "" {
		code = input.Discount_Code
	}
	if err != nil {
		return nil, err
	}
	expiryTime := StringToTime(input.ExpiresAt)
	if time.Now().After(expiryTime) {
		return nil, errors.New("expiry time must be in the future")
	}
	_, err = DB.Exec(`
		INSERT INTO promocodes (promo_code_id, code, description, discount_type, discount_value, expires_at, is_active, minimum_order_value, maximum_use)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, code, input.Description, input.DiscountType, input.DiscountValue, expiryTime, input.IsActive, input.MinimumOrderValue, input.MaximumUse,
	)
	if err != nil {
		return nil, err
	}

	input.ID = id
	return &input, nil
}
func isPromoThere(id string) error {
	exists, err := RecordExists("promocodes", "promo_code_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("promo code not found")
	}
	return nil
}
func UpdatePromoCode(id string, input dtos.PromoCodeRequest) (*dtos.PromoCodeRequest, error) {
	err := isPromoThere(id)
	if err != nil {
		return nil, err
	}
	_, err = DB.Exec(`
		UPDATE promocodes
		SET description = ?, discount_type = ?, discount_value = ?, expires_at = ?, is_active = ?, minimum_order_value = ?, maximum_use = ?
		WHERE promo_code_id = ?`,
		input.Description, input.DiscountType, input.DiscountValue, input.ExpiresAt, input.IsActive, input.MinimumOrderValue, input.MaximumUse, id,
	)
	if err != nil {
		return nil, err
	}
	input.ID = id
	return &input, nil
}

func GetPromoCodeByID(id string) (*dtos.PromoCodeResponse, error) {
	err := isPromoThere(id)
	if err != nil {
		return nil, err
	}
	row := DB.QueryRow(`
		SELECT promo_code_id, code, description, discount_type, discount_value, expires_at, is_active, minimum_order_value, maximum_use, times_used
		FROM promocodes WHERE promo_code_id = ?`, id,
	)

	var pc dtos.PromoCodeResponse
	if err := row.Scan(&pc.ID, &pc.Code, &pc.Description, &pc.DiscountType, &pc.DiscountValue, &pc.ExpiresAt, &pc.IsActive, &pc.MinimumOrderValue, &pc.MaximumUse, &pc.TimesUsed); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &pc, nil
}

func GetAllPromoCodes(page, size int) ([]dtos.PromoCodeResponse, *dtos.PaginationMeta, error) {
	var totalCount int
	err := DB.QueryRow(`SELECT COUNT(*) FROM promocodes`).Scan(&totalCount)
	if err != nil {
		return nil, nil, err
	}
	rows, err := DB.Query(`
		SELECT promo_code_id, code, description, discount_type, discount_value, expires_at, is_active, minimum_order_value, maximum_use, times_used FROM promocodes`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var promos []dtos.PromoCodeResponse
	for rows.Next() {
		var pc dtos.PromoCodeResponse
		if err := rows.Scan(&pc.ID, &pc.Code, &pc.Description, &pc.DiscountType, &pc.DiscountValue, &pc.ExpiresAt, &pc.IsActive, &pc.MinimumOrderValue, &pc.MaximumUse, &pc.TimesUsed); err != nil {
			return nil, nil, err
		}
		promos = append(promos, pc)
	}
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

func DeletePromoCode(id string) error {
	err := isPromoThere(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM promocodes WHERE promo_code_id = ?`, id)
	return err
}

func GetActivePromoByCode(code string, now time.Time) (*dtos.PromoCodeResponse, error) {
	row := DB.QueryRow(`
		SELECT promo_code_id, code, description, discount_type, discount_value, expires_at, is_active
		FROM promocodes WHERE code = ? AND is_active = 1 AND expires_at > ?`, code, now,
	)

	var pc dtos.PromoCodeResponse
	if err := row.Scan(&pc.ID, &pc.Code, &pc.Description, &pc.DiscountType, &pc.DiscountValue, &pc.ExpiresAt, &pc.IsActive); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &pc, nil
}
func SetPromoCodeActiveStatus(id string, isActive bool) error {
	_, err := DB.Exec(`
		UPDATE promocodes
		SET is_active = ?
		WHERE promo_code_id = ?`, isActive, id,
	)
	return err
}

func AddPromotionToProduct(req dtos.AddPromotionToProductRequest) error {
	// Check if product exists
	err := IsProductThere(req.ProductID)
	if err != nil {
		return err
	}
	// Check if promotion type exists
	exist, err := RecordExists("promotion_types", "id = ?", req.PromotionTypeID)
	if err != nil {
		return err
	}

	if !exist {
		return errors.New("promotion type does not exist")
	}
	// Check if the association already exists
	exists, err := RecordExists("product_discounts", "product_id = ? AND promotion_type_id = ?", req.ProductID, req.PromotionTypeID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	productDiscountID, _ := shortid.Generate()
	// Insert the new association
	_, err = DB.Exec(`
		INSERT INTO product_discounts (product_discount_id, product_id, promotion_type_id)
		VALUES (?, ?,?)`, productDiscountID, req.ProductID, req.PromotionTypeID,
	)
	return err
}

func RemoveHeldProductPromotions(productDiscountID string) error {
	_, err := DB.Exec(`DELETE FROM product_discounts WHERE product_discount_id = ?`, productDiscountID)
	return err
}

func HoldProductPromotions(productID string) ([]string, error) {
	rows, err := DB.Query(`SELECT product_discount_id FROM product_discounts WHERE product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
