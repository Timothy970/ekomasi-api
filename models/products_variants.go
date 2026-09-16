package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"fmt"
)

func calculatePagination(page, limit int, totalItems int64) dtos.PaginationMeta {
	// Ensure minimum valid page size
	if limit <= 0 {
		limit = 10
	}

	// Ensure minimum valid page number
	if page <= 0 {
		page = 1
	}

	// Calculate total pages using ceiling division
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))

	return dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: int(totalItems),
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
}

// getProductVariants fetches all variants for a given product.
//
// This function retrieves product variants (e.g., sizes, colors) with their pricing,
// stock levels, and visual attributes (hex color codes).
//
// Parameters:
//   - productID: string - The product_id to fetch variants for
//
// Returns:
//   - []dtos.ProductVariants: Array of variants containing:
//   - VariantID: Unique variant identifier
//   - VariantType: Type of variant (e.g., "size", "color")
//   - Name: Variant name (e.g., "Large", "Red")
//   - HexCode: Color hex code (for color variants, nullable)
//   - AdditionalPrice: Price adjustment for this variant
//   - StockQuantity: Stock level for this specific variant
//   - error: Database error or nil on success
func getProductVariants(db DBExecutor, productID string) ([]dtos.ProductVariants, error) {
	// Join product_variants with variants table to get complete variant details
	query := `
		SELECT 
			v.variant_id,
			v.variant_type,
			v.name,
			v.hex_code,
			pv.additional_price,
			pv.stock_quantity
		FROM product_variants pv
		INNER JOIN variants v ON pv.variant_id = v.variant_id
		WHERE pv.product_id = ?
	`

	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, fmt.Errorf("querying product variants: %w", err)
	}
	defer rows.Close()

	// Scan variant rows
	var variants []dtos.ProductVariants
	for rows.Next() {
		var pv dtos.ProductVariants
		if err := rows.Scan(
			&pv.VariantID,
			&pv.VariantType,
			&pv.Name,
			&pv.HexCode,
			&pv.AdditionalPrice,
			&pv.StockQuantity,
		); err != nil {
			return nil, fmt.Errorf("scanning product variant: %w", err)
		}
		variants = append(variants, pv)
	}

	// Check for row iteration errors
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating product variants: %w", err)
	}

	return variants, nil
}

// GetVariantSelection retrieves the variant selection for a product, including available options and their stock levels.
// This function is used to populate variant selection dropdowns on the frontend, showing which options are in stock.
// Parameters:
//   - productID: string - The product_id to fetch variant selection for
//
// Returns:
//   - []dtos.VariantSelection: Array of variant selection options containing:
//   - VariantIDs: Array of variant IDs that make up this selection (e.g., size and color combination)
//   - Name: Human-readable name for this variant selection (e.g., "Large Red")
//   - SKU: SKU for this specific variant selection
//   - StockQuantity: Stock level for this variant selection (calculated from product and variant stock)
//
// - AdditionalPrice: Price adjustment for this variant selection (sum of all variant adjustments)
//   - error: Database error or nil on success
func GetVariantSelection(db DBExecutor, productID string) ([]dtos.VariantSelection, error) {
	query := `
	SELECT 
		pvc.id,
		pvc.name,
		pvc.sku,
		pvc.additional_price,
		pvc.stock_quantity,
		pcvo.variant_id
	FROM product_variant_combinations pvc
	LEFT JOIN product_variant_combination_options pcvo 
		ON pvc.id = pcvo.combination_id
	WHERE pvc.product_id = ?
	ORDER BY pvc.id;
	`

	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, fmt.Errorf("querying variant selections: %w", err)
	}
	defer rows.Close()

	// map to group combinations
	combinationMap := make(map[string]*dtos.VariantSelection)

	for rows.Next() {
		var (
			combinationID string
			variantID     *string // pointer to handle NULL (LEFT JOIN)
			name          string
			sku           string
			price         float64
			stock         int
		)

		if err := rows.Scan(&combinationID, &name, &sku, &price, &stock, &variantID); err != nil {
			return nil, fmt.Errorf("scanning variant selection: %w", err)
		}

		// create if not exists
		if _, exists := combinationMap[combinationID]; !exists {
			combinationMap[combinationID] = &dtos.VariantSelection{
				VariantIDs:      make([]string, 0, 3), // small optimization
				Name:            name,
				SKU:             sku,
				AdditionalPrice: price,
				StockQuantity:   stock,
			}
		}

		// append only if not NULL
		if variantID != nil {
			combinationMap[combinationID].VariantIDs = append(
				combinationMap[combinationID].VariantIDs,
				*variantID,
			)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	// convert map → slice (preallocate)
	selections := make([]dtos.VariantSelection, 0, len(combinationMap))
	for _, v := range combinationMap {
		selections = append(selections, *v)
	}

	return selections, nil
}

// GetProductByID retrieves complete details for a single product by its ID.
//
// This function fetches a product with all associated data including images, warranties,
// features, variants, tax information, and JSON details.
//
// Parameters:
//   - productID: string - The unique product_id to retrieve
//
// Returns:
//   - *dtos.Product: Complete product details including:
//   - Basic info: ID, Name, Description, SKU, Price, CategoryID, CategoryName, Tag
//   - Stock: StockQuantity
//   - Specifications: Weight, Dimensions, Manufacturer, WeightLimit
//   - Deal info: Discount, DiscountType (from active deals)
//   - Details: Array of detail strings (unmarshaled from JSON)
//   - Images: Product image gallery
//   - Warranty: Warranty details
//   - Features: Product features list
//   - ProductVariants: Available variants
//   - Tax: Tax information
//   - Timestamps: CreatedAt, LastUpdated
//   - error: "product not found" if ID doesn't exist, database error, or nil on success
func GetProductByID(db DBExecutor, productID string) (*dtos.Product, error) {
	// Validate product exists
	err := IsProductThere(db, productID)
	if err != nil {
		return nil, err
	}

	// Query product with LEFT JOINs for optional data
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, c.name, p.tag, p.details, dp.discount, dp.discount_type, ps.weight, ps.dimensions, v.name, ps.weight_limit, p.product_type, p.low_stock_quantity_warning, p.barcode
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		LEFT JOIN variants v ON ps.manufacturer = v.variant_id
		WHERE p.product_id = ?
	`

	var (
		p            dtos.Product
		detailsData  []byte // JSON blob for product details
		categotyID   sql.NullString
		categoryName sql.NullString
		productType  sql.NullString
	)

	// Scan basic product data
	err = db.QueryRow(query, productID).Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &categotyID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		&categoryName, &p.Tag, &detailsData, &p.Discount, &p.DiscountType, &p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit, &productType, &p.LowStockAlert, &p.Barcode,
	)
	if err != nil {
		return nil, err
	}

	if categotyID.Valid {
		p.CategoryID = categotyID.String
	}

	if categoryName.Valid {
		p.CategoryName = categoryName.String
	}
	// Unmarshal JSON details if present
	if len(detailsData) > 0 {
		err := json.Unmarshal(detailsData, &p.Details)
		if err != nil {
			return nil, err
		}
	} else {
		p.Details = []string{} // Empty array for no details
	}

	// Fetch associated product images
	images, err := fetchProductImages(db, p.ID)
	if err != nil {
		return nil, err
	}
	p.Images = images

	// Fetch product warranties
	warranties, err := FetchProductWarranties(db, p.ID)
	if err != nil {
		return nil, err
	}
	p.Warranty = &warranties
	// Fetch product features
	features, err := fetchProductFeatures(db, p.ID)
	if err != nil {
		return nil, err
	}
	p.Features = features

	// Fetch product variants (sizes, colors, etc.)
	variants, err := getProductVariants(db, p.ID)
	if err != nil {
		return nil, err
	}
	p.ProductVariants = variants
	p.VariantSelection, err = GetVariantSelection(db, p.ID)
	if err != nil {
		return nil, err
	}
	// Fetch tax information
	// tax, err := fetchProductTax(db, p.ID)
	// if err != nil {
	// 	return nil, err
	// }
	// p.Tax = &tax

	// Fetch bundle products if this is a bundle
	if productType.Valid && productType.String == "bundle" {
		bundleProducts, err := getBundleProducts(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.BundleProducts = bundleProducts
	}

	return &p, nil
}
