// Package models provides data access functions for the Adenzo backend application.
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
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

// isPromotionThere validates if a promotion exists in the database.
//
// This is an internal helper function used before promotion operations.
//
// Parameters:
//   - id: string - The promotion_id to validate
//
// Returns:
//   - error: "promotion not found" if ID doesn't exist, database error, or nil if exists
func isPromotionThere(db DBExecutor, id string) error {
	// Check promotion existence
	exists, err := RecordExists(db, "promotions", "promotion_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("promotion not found")
	}
	return nil
}

// GetPromotionDetails retrieves promotion date range and linked product IDs.
//
// This function is used to fetch promotion metadata for effectiveness analysis.
//
// Parameters:
//   - promotionID: string - The promotion_id to retrieve
//
// Returns:
//   - time.Time: Promotion start date
//   - time.Time: Promotion end date
//   - []string: Array of product IDs linked to this promotion
//   - error: "promotion not found", database error, or nil on success
func GetPromotionDetails(db DBExecutor, promotionID string) (time.Time, time.Time, []string, error) {
	// Validate promotion exists
	err := isPromotionThere(db, promotionID)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}

	var startDate, endDate time.Time

	// Get promotion date range
	err = db.QueryRow(`
		SELECT start_date, end_date
		FROM promotions
		WHERE promotion_id = ?`, promotionID).Scan(&startDate, &endDate)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}

	// Get linked products
	rows, err := db.Query(`
		SELECT product_id
		FROM promotion_products
		WHERE promotion_id = ?`, promotionID)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}
	defer rows.Close()

	// Scan product IDs
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

// GetPromotionEffectiveness calculates sales performance for promoted products during a time period.
//
// This function aggregates quantity sold and revenue for specific products
// within the given date range, typically used for promotion ROI analysis.
//
// Parameters:
//   - productIDs: []string - Product IDs to analyze (must not be empty)
//   - start: time.Time - Analysis period start date
//   - end: time.Time - Analysis period end date
//
// Returns:
//   - []dtos.PromotionEffectiveness: Array of effectiveness data per product:
//   - ProductID: Product identifier
//   - TotalSold: Total quantity sold
//   - Revenue: Total revenue generated
//   - error: Database error or nil on success
func GetPromotionEffectiveness(db DBExecutor, productIDs []string, start, end time.Time) ([]dtos.PromotionEffectiveness, error) {
	var query string
	var args []interface{}

	// Build dynamic IN clause with placeholders
	placeholders := "?" + strings.Repeat(",?", len(productIDs)-1)
	query = `
			SELECT oi.product_id, SUM(oi.quantity) AS total_sold, SUM(oi.quantity * oi.unit_price) AS revenue
			FROM order_items oi
			JOIN orders o ON oi.order_id = o.order_id
			WHERE oi.product_id IN (` + placeholders + `)
			  AND o.last_updated_at BETWEEN ? AND ?
			GROUP BY oi.product_id;
		`

	// Add product IDs to args
	for _, id := range productIDs {
		args = append(args, id)
	}
	// Add date range to args
	args = append(args, start, end)

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan results
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

// GetPromotionAggregate calculates total sales and revenue for specific products during a time period.
//
// This function returns aggregated metrics (not per-product breakdown) for promotional analysis.
//
// Parameters:
//   - productIDs: []string - Product IDs to aggregate (must not be empty)
//   - start: time.Time - Period start date
//   - end: time.Time - Period end date
//
// Returns:
//   - int64: Total quantity sold across all products
//   - float64: Total revenue across all products
//   - error: Database error or nil on success
func GetPromotionAggregate(db DBExecutor, productIDs []string, start, end time.Time) (int64, float64, error) {
	// Build dynamic IN clause with placeholders
	query := `
		SELECT COALESCE(SUM(oi.quantity),0), COALESCE(SUM(oi.quantity * oi.unit_price),0)
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE oi.product_id IN (?` + strings.Repeat(",?", len(productIDs)-1) + `)
		  AND o.last_updated_at BETWEEN ? AND ?
	`

	// Build args array with product IDs and date range
	args := make([]interface{}, len(productIDs)+2)
	for i, id := range productIDs {
		args[i] = id
	}
	args[len(productIDs)] = start
	args[len(productIDs)+1] = end

	// Execute aggregation query
	var qty int64
	var rev float64
	err := db.QueryRow(query, args...).Scan(&qty, &rev)
	return qty, rev, err
}

// GetNonPromotionAggregate calculates total sales and revenue for ALL products during a time period.
//
// This function is used as a baseline comparison for promotion effectiveness,
// showing overall store performance regardless of promotions.
//
// Parameters:
//   - start: time.Time - Period start date
//   - end: time.Time - Period end date
//
// Returns:
//   - int64: Total quantity sold (all products)
//   - float64: Total revenue (all products)
//   - error: Database error or nil on success
func GetNonPromotionAggregate(db DBExecutor, start, end time.Time) (int64, float64, error) {
	// Aggregate all orders in date range (no product filter)
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
	err := db.QueryRow(query, start, end).Scan(&qty, &rev)
	if err != nil {
		return 0, 0, err
	}
	return qty, rev, nil
}

// GetPromotionSummary retrieves performance metrics for all promotions within a date range.
//
// This function provides a comprehensive overview of promotion effectiveness,
// including promotions with no sales (NULL order handling).
//
// Parameters:
//   - start: time.Time - Report period start date
//   - end: time.Time - Report period end date
//
// Returns:
//
//   - []dtos.PromotionSummary: Array of promotion summaries containing:
//
//   - PromotionID: Unique promotion identifier
//
//   - PromotionType: Type of promotion (e.g., "Flash Sale", "BOGO")
//
//   - StartDate: Promotion start date
//
//   - EndDate: Promotion end date
//
//   - TotalSold: Total quantity sold (0 if no orders)
//
//   - Revenue: Total revenue generated (0 if no orders)
//
//   - error: Database error or nil on success
//
//     the analysis period (intersects start or end dates).
func GetPromotionSummary(db DBExecutor, start, end time.Time) ([]dtos.PromotionSummary, error) {
	// Query promotions with LEFT JOINs to include promotions with no sales
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
       OR o.order_id IS NULL)   -- Include promotions with no orders
GROUP BY p.promotion_id, pt.name, p.start_date, p.end_date;

	`
	rows, err := db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan promotion summary rows
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
//   - error: "promo code not found", nil if not found (sql.ErrNoRows), or database error
func GetPromoCodeByID(db DBExecutor, id string) (*dtos.PromoCodeResponse, error) {
	// Validate promo code exists
	err := isPromoThere(db, id)
	if err != nil {
		return nil, err
	}

	// Query promo code details
	row := db.QueryRow(`
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

// GetAllPromoCodes retrieves all promo codes with pagination metadata.
//
// Parameters:
//   - page: int - Page number for pagination metadata (not used for actual paging)
//   - size: int - Page size for pagination metadata (not used for actual paging)
//
// Returns:
//
//   - []dtos.PromoCodeResponse: Array of all promo codes
//
//   - *dtos.PaginationMeta: Pagination metadata
//
//   - error: Database error or nil on success
//
//     Consider implementing actual pagination if code count grows large.
func GetAllPromoCodes(db DBExecutor, page, size int) ([]dtos.PromoCodeResponse, *dtos.PaginationMeta, error) {
	// Count total promo codes
	var totalCount int
	err := db.QueryRow(`SELECT COUNT(*) FROM promocodes`).Scan(&totalCount)
	if err != nil {
		return nil, nil, err
	}

	// Query all promo codes (no LIMIT/OFFSET applied)
	rows, err := db.Query(`
		SELECT promo_code_id, code, description, discount_type, discount_value, expires_at, is_active, minimum_order_value, maximum_use, times_used FROM promocodes ORDER BY created_at DESC`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Scan promo code rows
	var promos []dtos.PromoCodeResponse
	for rows.Next() {
		var pc dtos.PromoCodeResponse
		if err := rows.Scan(&pc.ID, &pc.Code, &pc.Description, &pc.DiscountType, &pc.DiscountValue, &pc.ExpiresAt, &pc.IsActive, &pc.MinimumOrderValue, &pc.MaximumUse, &pc.TimesUsed); err != nil {
			return nil, nil, err
		}
		promos = append(promos, pc)
	}

	// Build pagination metadata
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

// DeletePromoCode permanently removes a promo code from the database.
//
// Warning: This is a hard delete. Consider soft delete (is_active = false)
//
//	to maintain usage history and audit trails.
//
// Parameters:
//   - id: string - The promo_code_id to delete
//
// Returns:
//   - error: "promo code not found", database error, or nil on success
func DeletePromoCode(db DBExecutor, id string) error {
	// Validate promo code exists
	err := isPromoThere(db, id)
	if err != nil {
		return err
	}

	// Hard delete promo code
	_, err = db.Exec(`DELETE FROM promocodes WHERE promo_code_id = ?`, id)
	return err
}

// GetActivePromoByCode retrieves an active promo code by its code string.
//
// This function is used during checkout to validate and apply promo codes.
// Only returns codes that are active and not expired.
//
// Parameters:
//   - code: string - The promo code string to look up
//   - now: time.Time - Current timestamp for expiry check
//
// Returns:
//   - *dtos.PromoCodeResponse: Active promo code details (nil if not found or expired)
//   - error: Database error or nil on success
func GetActivePromoByCode(db DBExecutor, code string, now time.Time) (*dtos.PromoCodeResponse, error) {
	// Query active and non-expired promo code
	row := db.QueryRow(`
		SELECT promo_code_id, code, description, discount_type, discount_value, expires_at, is_active
		FROM promocodes WHERE code = ? AND is_active = 1 AND expires_at > ?`, code, now,
	)

	var pc dtos.PromoCodeResponse
	if err := row.Scan(&pc.ID, &pc.Code, &pc.Description, &pc.DiscountType, &pc.DiscountValue, &pc.ExpiresAt, &pc.IsActive); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	return &pc, nil
}

// SetPromoCodeActiveStatus toggles a promo code's active status.
//
// Parameters:
//   - id: string - The promo_code_id to update
//   - isActive: bool - New active status (true = active, false = inactive)
//
// Returns:
//   - error: Database error or nil on success
func SetPromoCodeActiveStatus(db DBExecutor, id string, isActive bool) error {
	_, err := DB.Exec(`
		UPDATE promocodes
		SET is_active = ?
		WHERE promo_code_id = ?`, isActive, id,
	)
	return err
}

// AddPromotionToProduct creates an association between a product and a promotion type.
//
// This function validates both the product and promotion type exist before creating
// the association, and prevents duplicate associations.
//
// Parameters:
//   - req: dtos.AddPromotionToProductRequest containing:
//   - ProductID: The product to associate with promotion
//   - PromotionTypeID: The promotion type to apply to product
//
// Returns:
//   - error: "product not found", "promotion type does not exist",
//     database error, or nil on success (including if already exists)
//
// Workflow:
//  1. Validate product exists in products table
//  2. Validate promotion type exists in promotion_types table
//  3. Check if association already exists (return nil if duplicate)
//  4. Create new association if none exists
func AddPromotionToProduct(db DBExecutor, req dtos.AddPromotionToProductRequest) error {
	// Validate product exists
	err := IsProductThere(db, req.ProductID)
	if err != nil {
		return err
	}

	// Validate promotion type exists
	exist, err := RecordExists(db, "promotion_types", "id = ?", req.PromotionTypeID)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New("promotion type does not exist")
	}

	// Check if association already exists (prevent duplicates)
	exists, err := RecordExists(db, "product_discounts", "product_id = ? AND promotion_type_id = ?", req.ProductID, req.PromotionTypeID)
	if err != nil {
		return err
	}
	if exists {
		return nil // Already associated, not an error
	}

	// Generate unique discount ID
	productDiscountID, _ := shortid.Generate()

	// Create new product-promotion association
	_, err = db.Exec(`
		INSERT INTO product_discounts (product_discount_id, product_id, promotion_type_id)
		VALUES (?, ?, ?)`, productDiscountID, req.ProductID, req.PromotionTypeID,
	)
	return err
}

// RemoveHeldProductPromotions removes a product-promotion association.
//
// This is typically called when updating a product's promotion associations -
// the old associations are removed, then new ones are added.
//
// Parameters:
//   - productDiscountID: string - The product_discount_id to delete
//
// Returns:
//   - error: Database error or nil on success
func RemoveHeldProductPromotions(db DBExecutor, productDiscountID string) error {
	// Delete specific product-promotion association
	_, err := DB.Exec(`DELETE FROM product_discounts WHERE product_discount_id = ?`, productDiscountID)
	return err
}

// HoldProductPromotions retrieves all promotion associations for a given product.
//
// This is used during product updates to get the current promotion list before
// making changes. The term "Hold" suggests these are cached/retrieved for comparison.
//
// Parameters:
//   - productID: string - The product_id to retrieve associations for
//
// Returns:
//   - []string: Array of product_discount_id strings associated with the product
//   - error: Database error or nil on success
func HoldProductPromotions(db DBExecutor, productID string) ([]string, error) {
	// Query all promotion associations for this product
	rows, err := db.Query(`SELECT product_discount_id FROM product_discounts WHERE product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect product discount IDs
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
