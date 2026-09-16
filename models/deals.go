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
	"ekomasi_backend/dtos"
	"fmt"
	"os"

	"github.com/teris-io/shortid"
)

// CreateDeal creates a new promotional deal in the database.
//
// This function validates deal name uniqueness before creating the deal.
// Deals are time-limited promotional campaigns that can contain multiple products.
//
// Parameters:
//   - deal: dtos.CreateDeal containing:
//   - Name: Unique deal name (e.g., "Black Friday 2026")
//   - Description: Deal description
//   - Discount: Default discount for the deal
//   - StartDate: When the deal becomes active
//   - EndDate: When the deal expires
//   - Image: Deal banner/promotional image URL
//
// Returns:
//   - string: Generated deal_id (shortid format)
//   - error: "deal with name {name} already exists" if name exists, or database error
func CreateDeal(db DBExecutor, deal dtos.CreateDeal) (string, error) {
	// Validate deal name is unique
	exists, err := RecordExists(db, "deals", "name = ?", deal.Name)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("deal with name %s already exists", deal.Name)
	}

	// Generate unique deal ID
	dealID, _ := shortid.Generate()

	// Insert new deal into database
	_, err = db.Exec(`INSERT INTO deals (deal_id, name, description, discount, start_date, end_date, image, deal_type, brand_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, dealID, deal.Name, deal.Description, deal.Discount, deal.StartDate, deal.EndDate, deal.Image, deal.DealType, deal.BrandID)
	if err != nil {
		return "", err
	}
	return dealID, nil
}

// GetAllDeals retrieves paginated list of all deals with their products.
//
// This function fetches deals with basic information and associated products.
// Each deal includes a deep link for marketing campaigns.
//
// Parameters:
//   - page: Page number (1-based)
//   - size: Number of deals per page
//
// Returns:
//   - []dtos.Deal: Array of deals with products
//   - *dtos.PaginationMeta: Pagination metadata (page, size, total, has_prev, has_next)
//   - error: Database error if queries fail
//
// Deal Link Format:
//   - {BASE_URL}/products/deals/{deal_id}
//   - Used for promotional campaigns and marketing
func GetAllDeals(db DBExecutor, page, size int, isAdmin bool) ([]dtos.Deal, *dtos.PaginationMeta, error) {
	var (
		countTotal  int
		whereClause string
		args        []any
	)

	// Apply filters only if NOT admin
	if !isAdmin {
		whereClause = "WHERE is_active = TRUE AND end_date >= NOW()"
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM deals %s", whereClause)

	err := db.QueryRow(countQuery).Scan(&countTotal)
	if err != nil {
		return nil, nil, err
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT deal_id, name, start_date, end_date, is_active, image, deal_type, brand_id
		FROM deals
		%s
		ORDER BY created_at desc
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, size, (page-1)*size)

	rows, err := db.Query(dataQuery, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var deals []dtos.Deal
	// Iterate through deals and fetch associated products
	for rows.Next() {
		var d dtos.Deal
		if err := rows.Scan(&d.DealID, &d.Name, &d.StartDate, &d.EndDate, &d.IsActive, &d.Image, &d.DealType, &d.BrandID); err != nil {
			return nil, nil, err
		}

		// Fetch all products associated with this deal
		products, err := GetProductsByDealID(db, d.DealID)
		if err != nil {
			return nil, nil, err
		}

		// Generate deep link for this deal
		baseUrl := os.Getenv("BASE_URL")
		d.Link = fmt.Sprintf("%s/products/deals/%s", baseUrl, d.DealID)
		d.Products = products
		deals = append(deals, d)
	}

	// Build pagination metadata
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: countTotal,
		TotalPages: (countTotal + size - 1) / size, // Ceiling division
		HasPrev:    page > 1,
		HasNext:    page*size < countTotal,
	}
	return deals, pagination, nil
}

// isDealThere validates that a deal exists by deal_id.
//
// This is an internal validation helper used by other deal functions.
// Returns an error (rather than bool) for easier use in validation chains.
//
// Parameters:
//   - dealID: The deal_id to validate
//
// Returns:
//   - error: nil if deal exists, "deal with ID {id} does not exist" error if not found,
//     or database error if query fails
func isDealThere(db DBExecutor, dealID string) error {
	// Check if deal record exists in database
	if exists, err := RecordExists(db, "deals", "deal_id = ?", dealID); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("deal with ID %s does not exist", dealID)
	}
	return nil
}

// UpdateDeal updates an existing deal's information.
//
// This function validates deal existence before updating. It handles optional
// image updates - if image is nil, the existing image is preserved.
//
// Parameters:
//   - dealID: The deal_id to update
//   - deal: dtos.Deal containing:
//   - Name: Updated deal name
//   - StartDate: Updated start date
//   - EndDate: Updated end date
//   - IsActive: Pointer to is_active status (activate/deactivate deal)
//   - Image: Optional pointer to new image URL (nil preserves existing image)
//
// Returns:
//   - error: "deal with ID {id} does not exist" if deal not found, or database error
func UpdateDeal(db DBExecutor, dealID string, deal dtos.UpdateDeal) error {
	// Validate deal exists
	if err := isDealThere(db, dealID); err != nil {
		return err
	}

	// If image is provided, update including image field
	if deal.Image != nil {
		_, err := db.Exec(`UPDATE deals SET name = ?, start_date = ?, end_date = ?, is_active = ?, image = ?, deal_type = ?, brand_id = ? WHERE deal_id = ?`,
			deal.Name, deal.StartDate, deal.EndDate, *deal.IsActive, deal.Image, deal.DealType, deal.BrandID, dealID)
		return err
	}

	// Otherwise, update without changing image
	_, err := db.Exec(`UPDATE deals SET name = ?, start_date = ?, end_date = ?, is_active = ?, deal_type = ?, brand_id = ? WHERE deal_id = ?`,
		deal.Name, deal.StartDate, deal.EndDate, *deal.IsActive, deal.DealType, deal.BrandID, dealID)
	if err != nil {
		return err
	}

	for _, p := range deal.Products {
		discount := float64(p.Discount)
		if err := AddProductToDeal(db, dealID, p.ProductID, &p.DiscountType, &discount); err != nil {
			return err
		}
	}
	return nil
}

// DeleteDeal removes a deal from the database.
//
// This function validates deal existence before deletion.
//
// Parameters:
//   - dealID: The deal_id to delete
//
// Returns:
//   - error: "deal with ID {id} does not exist" if deal not found, or database error
//
// Important:
//   - No check for associated deal_products - deletion may fail if foreign key constraints exist
//   - Consider implementing cascade delete or removing products first
//   - May want to implement soft delete for audit trail
func DeleteDeal(db DBExecutor, dealID string) error {
	// Validate deal exists
	if err := isDealThere(db, dealID); err != nil {
		return err
	}

	// Delete deal from database
	_, err := db.Exec(`DELETE FROM deals WHERE deal_id = ?`, dealID)
	return err
}

// AddProductToDeal associates a product with a deal or updates existing association.
//
// This function creates or updates a product-deal relationship with custom discount.
// The discount overrides the deal's default discount for this specific product.
//
// Parameters:
//   - dealID: The deal_id to associate with
//   - productID: The product_id to add to the deal
//   - discountType: Pointer to discount type ("percentage" or "fixed")
//   - discount: Pointer to discount value (e.g., 25.0 for 25% or $25)
//
// Returns:
//   - error: Validation error if deal or product doesn't exist, or database error
//
// Behavior:
//   - If product-deal association exists, updates the discount and discount_type (idempotent)
//   - If association doesn't exist, creates new entry with generated product_deal_id
//   - Validates both deal and product existence before operation
func AddProductToDeal(db DBExecutor, dealID, productID string, discountType *string, discount *float64) error {
	// Validate deal exists
	if err := isDealThere(db, dealID); err != nil {
		return err
	}

	// Validate product exists
	if err := IsProductThere(db, productID); err != nil {
		return err
	}

	// Update discount and discount_type for this product in all other active or future deals
	// to ensure it takes the current discount and value everywhere
	_, err := db.Exec(`
		UPDATE deal_products 
		SET discount = ?, discount_type = ?
		WHERE product_id = ? 
		  AND deal_id != ? 
		  AND deal_id IN (SELECT deal_id FROM deals WHERE end_date > NOW())
	`, discount, discountType, productID, dealID)
	if err != nil {
		return err
	}

	// Check if product is already associated with this deal
	exists, err := RecordExists(db, "deal_products", "deal_id = ? AND product_id = ?", dealID, productID)
	if err != nil {
		return err
	}

	if exists {
		// Update existing product-deal association (idempotent operation)
		_, err := db.Exec(`UPDATE deal_products SET discount = ?, discount_type = ? WHERE deal_id = ? AND product_id = ?`,
			discount, discountType, dealID, productID)
		return err
	}

	// Create new product-deal association
	productDealID, _ := shortid.Generate()
	_, err = db.Exec(`INSERT INTO deal_products (product_deal_id, deal_id, product_id, discount, discount_type) VALUES (?, ?, ?, ?, ?)`,
		productDealID, dealID, productID, discount, discountType)
	return err
}
