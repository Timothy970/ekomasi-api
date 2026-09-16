package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"strings"

	"github.com/teris-io/shortid"
)

func buildProduct(r *productRow) dtos.Product {
	product := dtos.Product{
		ID:            r.productID.String,
		Name:          r.name.String,
		Description:   r.desc.String,
		SKU:           r.sku.String,
		Price:         r.price.Float64,
		CategoryID:    r.categoryID.String,
		CategoryName:  r.categoryName.String,
		StockQuantity: intOrZero(r.stockQuantity),
		SearchVector:  r.searchVector.String,
	}

	// Optional string fields
	if r.tag.Valid {
		product.Tag = ptr(strings.TrimSpace(r.tag.String))
	}
	if r.discountType.Valid {
		product.DiscountType = ptr(r.discountType.String)
	}
	if r.dimensions.Valid {
		product.Dimensions = ptr(r.dimensions.String)
	}
	if r.manufacturer.Valid {
		product.Manufacturer = ptr(r.manufacturer.String)
	}

	// Optional numeric fields
	if r.discount.Valid {
		product.Discount = ptr(r.discount.Float64)
	}
	if r.weight.Valid {
		product.Weight = ptr(r.weight.Float64)
	}
	if r.weightLimit.Valid {
		product.WeightLimit = ptr(r.weightLimit.Float64)
	}

	// Optional timestamps
	if r.createdAt.Valid {
		product.CreatedAt = r.createdAt.Time
	}
	if r.updatedAt.Valid {
		product.LastUpdated = r.updatedAt.Time
	}
	if r.isFeatured.Valid {
		product.IsProductFeatured = r.isFeatured.Bool
	}

	return product
}

// intOrZero returns 0 if the sql.NullInt64 is invalid
func intOrZero(v sql.NullInt64) int {
	if v.Valid {
		return int(v.Int64)
	}
	return 0
}

// enrichProductAdmin adds admin-specific fields to a product.
//
// This function fetches additional data only needed by administrators:
//   - CreatedBy: Full name of the user who created the product
//   - IsInTodaysDeals: Whether product is in today's deals
//   - MaxStockQuantity: Total inventory across all locations
//
// Parameters:
//   - product: *dtos.Product - Product to enrich (modified in place)
//
// Returns:
//   - error: Database error or nil on success
func enrichProductAdmin(db DBExecutor, product *dtos.Product) error {
	var err error

	// Get full name of user who created the product
	product.CreatedBy, err = getProductCreator(product.ID)
	if err != nil {
		return err
	}

	// Check if product is in today's deals
	product.IsInTodaysDeals, err = isProductInTodaysDeal(db, product.ID)
	if err != nil {
		return err
	}

	// Get maximum stock quantity from inventory table
	product.MaxStockQuantity, err = getMaxQuantity(db, product.ID)
	// Ensure max quantity is at least current stock quantity
	if product.MaxStockQuantity < product.StockQuantity {
		product.MaxStockQuantity = product.StockQuantity
	}
	return err
}

// getProductCreator fetches the full name of the user who created a product.
//
// Parameters:
//   - productID: string - The product_id to look up
//
// Returns:
//   - string: Full name (first + last) of creator, empty string if not found or NULL
//   - error: Database error or nil on success
func getProductCreator(productID string) (string, error) {
	var createdByID sql.NullString
	query := `SELECT created_by_id FROM products WHERE product_id = ?`

	err := DB.QueryRow(query, productID).Scan(&createdByID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Product not found → no creator
		}
		return "", fmt.Errorf("failed to get product creator ID: %v", err)
	}

	// Handle NULL or empty created_by_id
	if !createdByID.Valid || createdByID.String == "" {
		return "", nil
	}

	// Fetch user's name from users table
	var firstName, lastName sql.NullString
	userQuery := `SELECT first_name, last_name FROM users WHERE user_id = ?`
	err = DB.QueryRow(userQuery, createdByID.String).Scan(&firstName, &lastName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // User not found → return empty
		}
		return "", fmt.Errorf("failed to get user details: %v", err)
	}

	// Handle possible NULL first/last names
	fn := ""
	ln := ""
	if firstName.Valid {
		fn = firstName.String
	}
	if lastName.Valid {
		ln = lastName.String
	}

	// Combine names and trim whitespace
	fullName := strings.TrimSpace(fn + " " + ln)
	return fullName, nil
}

// get user full name by user id
func getUserNames(userID string) (string, error) {
	// Fetch user's name from users table
	var firstName, lastName sql.NullString
	userQuery := `SELECT first_name, last_name FROM users WHERE user_id = ?`
	err := DB.QueryRow(userQuery, userID).Scan(&firstName, &lastName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // User not found → return empty
		}
		return "", fmt.Errorf("failed to get user details: %v", err)
	}

	// Handle possible NULL first/last names
	fn := ""
	ln := ""
	if firstName.Valid {
		fn = firstName.String
	}
	if lastName.Valid {
		ln = lastName.String
	}

	// Combine names and trim whitespace
	fullName := strings.TrimSpace(fn + " " + ln)
	return fullName, nil
}

// isProductInTodaysDeal checks if a product is part of today's deals.
//
// This function looks for a deal with "today" in its name (case-insensitive regex)
// and checks if the product is linked to that deal.
//
// Parameters:
//   - productID: string - The product_id to check
//
// Returns:
//   - bool: true if product is in today's deals, false otherwise
//   - error: Database error or nil on success
func isProductInTodaysDeal(db DBExecutor, productID string) (bool, error) {
	var dealID string

	// Find deal_id for a deal whose name matches 'today' (case-insensitive)
	dealQuery := `SELECT deal_id FROM deals WHERE name REGEXP '(?i)today' LIMIT 1`
	err := db.QueryRow(dealQuery).Scan(&dealID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // No "today" deal exists
		}
		return false, fmt.Errorf("failed to get today's deal: %v", err)
	}

	// Check if this product is linked to that deal
	var exists bool
	checkQuery := `SELECT EXISTS(
		SELECT 1 FROM deal_products WHERE product_id = ? AND deal_id = ?
	)`
	err = db.QueryRow(checkQuery, productID, dealID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check product-deal link: %v", err)
	}

	return exists, nil
}

// getMaxQuantity retrieves the total inventory quantity for a product across all locations.
//
// Parameters:
//   - productID: string - The product_id to sum inventory for
//
// Returns:
//   - int: Total quantity across all inventory records, 0 if none found
//   - error: Database error or nil on success
func getMaxQuantity(db DBExecutor, productID string) (int, error) {
	var totalQuantity sql.NullInt64

	// Sum all inventory quantities for this product
	query := `
		SELECT SUM(quantity)
		FROM inventory
		WHERE product_id = ?
	`
	err := db.QueryRow(query, productID).Scan(&totalQuantity)
	if err != nil {
		return 0, fmt.Errorf("failed to get quantity for product %s: %v", productID, err)
	}

	// Handle NULL result (no inventory records found)
	if !totalQuantity.Valid {
		return 0, nil
	}

	return int(totalQuantity.Int64), nil
}

// InsertProductSpecs creates a new product specification record.
//
// Parameters:
//   - req: dtos.ProductSpecs - Specification data containing:
//   - ProductID: Product to attach specs to
//   - Weight: Product weight
//   - WeightLimit: Maximum weight for shipping
//   - Dimensions: Product dimensions (e.g., "10x5x3")
//   - Manufacturer: Manufacturer name
//
// Returns:
//   - error: "product not found", database error, or nil on success
func InsertProductSpecs(db DBExecutor, req dtos.ProductSpecs) error {
	// Validate product exists
	err := IsProductThere(db, req.ProductID)
	if err != nil {
		return err
	}

	insertQuery := `INSERT INTO product_specifications (specifications_id, product_id, weight, weight_limit, dimensions, manufacturer) VALUES (?, ?, ?, ?,?,?)`

	// Generate unique specifications ID
	specificationsID, _ := shortid.Generate()

	// Insert specification record
	if _, err := db.Exec(insertQuery, specificationsID, req.ProductID, req.Weight, req.WeightLimit, req.Dimensions, req.Manufacturer); err != nil {
		return err
	}

	return nil
}

// GetExpensiveAndCheapProducts fetches the most and least expensive products.
//
// Returns:
//   - *dtos.ExpensiveCheapProduct: Struct containing:
//   - CheapestProduct: Product with lowest price
//   - ExpensiveProduct: Product with highest price
//   - error: Database error or nil on success
func GetExpensiveAndCheapProducts(db DBExecutor) (*dtos.ExpensiveCheapProduct, error) {
	// Fetch cheapest product
	cheapestProduct, err := getProductByPriceType(db, "cheapest")
	if err != nil {
		return nil, err
	}

	// Fetch most expensive product
	expensiveProduct, err := getProductByPriceType(db, "expensive")
	if err != nil {
		return nil, err
	}

	// Combine into response struct
	combinedProducts := &dtos.ExpensiveCheapProduct{
		CheapestProduct:  *cheapestProduct,
		ExpensiveProduct: *expensiveProduct,
	}
	return combinedProducts, nil
}

// getProductByPriceType fetches a single product with either the lowest or highest price.
//
// This internal function is used by GetExpensiveAndCheapProducts to fetch extreme price points.
//
// Parameters:
//   - priceType: string - Either "cheapest" or "expensive"
//
// Returns:
//   - *dtos.Product: Product with complete details, nil if no products found
//   - error: Invalid priceType, database error, or nil on success
