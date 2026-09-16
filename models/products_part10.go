package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"strings"

	"github.com/teris-io/shortid"
)

func getProductByPriceType(db DBExecutor, priceType string) (*dtos.Product, error) {
	// Determine sort order based on price type
	var orderClause string
	switch strings.ToLower(priceType) {
	case "cheapest":
		orderClause = "ASC" // Ascending = lowest price first
	case "expensive":
		orderClause = "DESC" // Descending = highest price first
	default:
		return nil, fmt.Errorf("invalid price type: %s", priceType)
	}

	// Query to fetch product with extreme price
	query := fmt.Sprintf(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, 
			p.category_id, p.stock_quantity, p.search_vector, 
			p.created_at, p.last_updated_at, c.name AS category_name, p.tag
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE p.product_type = 'single'
		ORDER BY p.price %s
		LIMIT 1
	`, orderClause)

	var p dtos.Product
	err := db.QueryRow(query).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&p.CategoryName, &p.Tag,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No products found
		}
		return nil, err
	}

	// Fetch product images
	if p.Images, err = fetchProductImages(db, p.ID); err != nil {
		return nil, err
	}

	// Fetch product warranties
	warranty, err := FetchProductWarranties(db, p.ID)
	if err != nil {
		return nil, err
	}
	p.Warranty = &warranty

	// Fetch product features
	if p.Features, err = fetchProductFeatures(db, p.ID); err != nil {
		return nil, err
	}

	// Fetch product variants
	if p.ProductVariants, err = getProductVariants(db, p.ID); err != nil {
		return nil, err
	}

	// Fetch tax information
	// tax, err := fetchProductTax(db, p.ID)
	// if err != nil {
	// 	return nil, err
	// }
	// p.Tax = &tax
	p.VariantSelection, err = GetVariantSelection(db, p.ID)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// IsProductInUserWishlist checks if a product is in a user's wishlist.
//
// Parameters:
//   - userID: string - The user_id to check
//   - productID: string - The product_id to check
//
// Returns:
//   - bool: true if product is in user's wishlist, false otherwise
//   - error: Database error or nil on success
func IsProductInUserWishlist(db DBExecutor, userID, productID string) (bool, error) {
	var exists bool

	// Check if product exists in user's wishlist via wishlist_items join
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM wishlists w
			JOIN wishlist_items witems ON w.wishlist_id = witems.wishlist_id
			WHERE w.user_id = ? AND witems.product_id = ?
		)
	`

	err := db.QueryRow(query, userID, productID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check wishlist: %w", err)
	}

	return exists, nil
}

// RemoveHeldProductSpecs deletes a product specification record.
//
// Parameters:
//   - specID: string - The specifications_id to delete
//
// Returns:
//   - error: Database error or nil on success
func RemoveHeldProductSpecs(db DBExecutor, specID string) error {
	query := `DELETE FROM product_specifications WHERE specifications_id = ?`
	_, err := db.Exec(query, specID)
	return err
}

// HoldProductSpecs retrieves all specification IDs for a product.
//
// This function is used to get specification IDs before deletion operations
// to ensure proper cleanup of related records.
//
// Parameters:
//   - productID: string - The product_id to fetch specification IDs for
//
// Returns:
//   - []string: Array of specifications_id values
//   - error: Database error or nil on success
func HoldProductSpecs(db DBExecutor, productID string) ([]string, error) {
	// Query all specification IDs for this product
	rows, err := db.Query(`SELECT specifications_id FROM product_specifications WHERE product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect specification IDs
	var specIDs []string
	for rows.Next() {
		var specID string
		if err := rows.Scan(&specID); err != nil {
			return nil, err
		}
		specIDs = append(specIDs, specID)
	}
	return specIDs, nil
}

// RemoveAllProductVariants deletes all variant associations for a product.
//
// This function removes records from product_variants table (the join table),
// but does NOT delete the variant definitions from the variants table.
//
// Parameters:
//   - productID: string - The product_id to remove variants for
//
// Returns:
//   - error: Database error or nil on success
func RemoveAllProductVariants(productID string) error {
	_, err := DB.Exec(`DELETE FROM product_variants WHERE product_id = ?`, productID)
	return err
}

// InsertCombination inserts a product combination into the database and returns the new combination ID.
// param db: Database executor for performing the query
// param combination: Data transfer object containing combination details (product ID, name, SKU, additional price)
// returns: The ID of the newly created combination and any error encountered
func InsertCombination(db DBExecutor, combination dtos.Combination) (string, error) {
	combinationID, _ := shortid.Generate()
	query := `INSERT INTO product_variant_combinations (id, product_id, name, sku, additional_price) VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, combinationID, combination.ProductID, combination.Name, combination.SKU, combination.AdditionalPrice)
	return combinationID, err
}

// InserCombinationOption inserts options for a product combination into the database.
// param db: Database executor for performing the query
// param combinationID: The ID of the combination to associate options with
// param variantID: The ID of the variant that defines the option (e.g., size, color)
// returns: Any error encountered during the insertion process
func InsertCombinationOption(db DBExecutor, combinationID, variantID string, productID string) error {
	combinationOptionsID, _ := shortid.Generate()
	query := `INSERT INTO product_variant_combination_options (id, combination_id, variant_id) VALUES (?, ?, ?)`
	_, err := db.Exec(query, combinationOptionsID, combinationID, variantID)
	if err != nil {
		return err
	}
	// Check if product-variant association already exists
	exists, err := isProductWithVariant(db, variantID, productID)
	if err != nil {
		return err
	}

	// Handle optional additional price
	additionalPrice := 0.0
	// Skip insertion if association already exists (idempotent)
	if exists {
		return nil

	} else {
		pvID, _ := shortid.Generate()
		// Insert new product-variant record
		_, err = db.Exec(`
			INSERT INTO product_variants (product_variants_id, variant_id, product_id, additional_price, stock_quantity)
			VALUES (?, ?, ?, ?, ?)`,
			pvID, variantID, productID, additionalPrice, 0,
		)
	}

	return err
}
