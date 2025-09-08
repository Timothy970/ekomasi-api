package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"math"
	"strings"

	"github.com/teris-io/shortid"
)

func CreateVariant(req dtos.VariantRequest) (string, error) {
	id, _ := shortid.Generate()
	_, err := DB.Exec(`
        INSERT INTO variants (variant_id, variant_type, name, hex_code)
        VALUES (?, ?, ?, ?)`,
		id, req.VariantType, req.Name, req.HexCode,
	)
	return id, err
}

func GetVariant(id string) (*dtos.VariantResponse, error) {
	err := variantexists(id)
	if err != nil {
		return nil, err
	}
	var v dtos.VariantResponse
	err = DB.QueryRow(`
        SELECT variant_id, variant_type, name, hex_code
        FROM variants
        WHERE variant_id = ?`, id,
	).Scan(&v.VariantID, &v.VariantType, &v.Name, &v.HexCode)
	if err == sql.ErrNoRows {
		return nil, errors.New("variant not found")
	}
	return &v, err
}

func ListVariants() ([]dtos.GroupedVariants, error) {
	rows, err := DB.Query(`
        SELECT variant_id, variant_type, name, hex_code
        FROM variants
        ORDER BY variant_type, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groupMap := make(map[string][]dtos.VariantResponse)

	for rows.Next() {
		var v dtos.VariantResponse
		if err := rows.Scan(&v.VariantID, &v.VariantType, &v.Name, &v.HexCode); err != nil {
			return nil, err
		}
		groupMap[v.VariantType] = append(groupMap[v.VariantType], v)
	}

	// convert map → slice
	var grouped []dtos.GroupedVariants
	for t, vs := range groupMap {
		grouped = append(grouped, dtos.GroupedVariants{
			VariantType: t,
			Variants:    vs,
		})
	}

	return grouped, nil
}

func variantexists(id string) error {
	exists, err := RecordExists("variants", "variant_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("variant not found")
	}
	return nil
}
func UpdateVariantByID(id string, req dtos.VariantRequest) error {
	err := variantexists(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
        UPDATE variants
        SET variant_type = ?, name = ?, hex_code = ?
        WHERE variant_id = ?`,
		req.VariantType, req.Name, req.HexCode, id,
	)
	return err
}

func DeleteVariantByID(id string) error {
	err := variantexists(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM variants WHERE variant_id = ?`, id)
	return err
}

// Product Variants
func AddProductVariant(id string, req dtos.ProductVariantRequest) error {
	pvID, _ := shortid.Generate()
	err := variantexists(id)
	if err != nil {
		return err
	}
	err = isProductThere(req.ProductID)
	if err != nil {
		return err
	}
	additonalPrice := 0.0
	if req.AdditionalPrice != nil {
		additonalPrice = *req.AdditionalPrice
	}
	_, err = DB.Exec(`
        INSERT INTO product_variants (product_variants_id, variant_id, product_id, additional_price, stock_quantity)
        VALUES (?, ?, ?, ?, ?)`,
		pvID, id, req.ProductID, additonalPrice, req.StockQuantity,
	)
	return err
}

func RemoveProductVariant(productID, variantID string) error {
	err := isProductThere(productID)
	if err != nil {
		return err
	}
	err = variantexists(variantID)
	if err != nil {
		return err
	}
	result, err := DB.Exec(`
		DELETE FROM product_variants
		WHERE product_id = ? AND variant_id = ?`, productID, variantID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no such product variant mapping found")
	}
	return nil
}

func ListProductVariants(productID string) ([]dtos.ProductVariantResponse, error) {
	rows, err := DB.Query(`
        SELECT variant_id, product_id, additional_price, stock_quantity
        FROM product_variants
        WHERE product_id = ?`, productID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pv []dtos.ProductVariantResponse
	for rows.Next() {
		var item dtos.ProductVariantResponse
		if err := rows.Scan(&item.VariantID, &item.ProductID, &item.AdditionalPrice, &item.StockQuantity); err != nil {
			return nil, err
		}
		pv = append(pv, item)
	}
	return pv, nil
}

func GetVariantsWithProductsPaginated(variants []dtos.Variant, page, limit int) ([]*dtos.VariantWithProducts, *dtos.PaginationMeta, error) {
	if len(variants) == 0 {
		return nil, nil, errors.New("no variants provided")
	}

	// Build variant filters
	var variantIDs []string
	var variantNames []string
	var variantIDMap = make(map[string]bool)
	var variantNameMap = make(map[string]bool)

	for _, variant := range variants {
		if variant.VariantID != "" {
			if !variantIDMap[variant.VariantID] {
				variantIDs = append(variantIDs, variant.VariantID)
				variantIDMap[variant.VariantID] = true
			}
		}
		if variant.Name != "" && variant.Name != "All" {
			if !variantNameMap[variant.Name] {
				variantNames = append(variantNames, variant.Name)
				variantNameMap[variant.Name] = true
			}
		}
	}

	// Build base query
	var args []interface{}
	query := `
        SELECT v.variant_id, v.variant_type, v.name, v.hex_code,
               pv.additional_price, pv.stock_quantity
        FROM variants v
        INNER JOIN product_variants pv ON v.variant_id = pv.variant_id
        WHERE 1=1
    `

	if len(variantIDs) > 0 {
		query += " AND v.variant_id IN (?" + strings.Repeat(",?", len(variantIDs)-1) + ")"
		for _, id := range variantIDs {
			args = append(args, id)
		}
	}

	if len(variantNames) > 0 {
		query += " AND LOWER(v.name) IN (?" + strings.Repeat(",?", len(variantNames)-1) + ")"
		for _, name := range variantNames {
			args = append(args, strings.ToLower(name))
		}
	}

	// Get all matching variants
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var variantResults []*dtos.VariantWithProducts
	variantMap := make(map[string]*dtos.VariantWithProducts)

	for rows.Next() {
		var variant dtos.VariantWithProducts
		err := rows.Scan(&variant.VariantID, &variant.VariantType, &variant.Name, &variant.HexCode,
			&variant.AdditionalPrice, &variant.StockQuantity)
		if err != nil {
			return nil, nil, err
		}
		variantMap[variant.VariantID] = &variant
		variantResults = append(variantResults, &variant)
	}

	if len(variantResults) == 0 {
		return nil, nil, nil
	}

	// Get variant IDs for product counting
	var resultVariantIDs []string
	for _, v := range variantResults {
		resultVariantIDs = append(resultVariantIDs, v.VariantID)
	}

	// Count total products across all variants for pagination
	countQuery := `
        SELECT COUNT(DISTINCT p.product_id)
        FROM products p
        INNER JOIN product_variants pv ON p.product_id = pv.product_id
        WHERE pv.variant_id IN (?
    ` + strings.Repeat(",?", len(resultVariantIDs)-1) + ")"

	countArgs := make([]interface{}, len(resultVariantIDs))
	for i, id := range resultVariantIDs {
		countArgs[i] = id
	}

	var total int
	err = DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	offset := (page - 1) * limit

	// Fetch paginated products for all variants
	products, productVariantMap, err := fetchProductsByVariantsPaginated(resultVariantIDs, limit, offset)
	if err != nil {
		return nil, nil, err
	}

	// Distribute products to their respective variants using the join table mapping
	for variantID, productIDs := range productVariantMap {
		if variant, exists := variantMap[variantID]; exists {
			for _, product := range products {
				for _, productID := range productIDs {
					if product.ID == productID {
						variant.Products = append(variant.Products, product)
						break
					}
				}
			}
		}
	}

	// Handle "All" variant names - return all products for those variant IDs
	for _, variant := range variants {
		if variant.Name == "All" && variant.VariantID != "" {
			if existingVariant, exists := variantMap[variant.VariantID]; exists {
				// This variant should include all products, not just paginated ones
				allProducts, err := fetchAllProductsByVariant(variant.VariantID)
				if err != nil {
					return nil, nil, err
				}
				existingVariant.Products = allProducts
			}
		}
	}

	// Pagination meta
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return variantResults, pagination, nil
}

func fetchProductsByVariantsPaginated(variantIDs []string, limit, offset int) ([]dtos.Product, map[string][]string, error) {
	if len(variantIDs) == 0 {
		return nil, nil, errors.New("no variant IDs provided")
	}

	// First, get the product IDs and their variant mappings
	mappingQuery := `
        SELECT pv.variant_id, pv.product_id
        FROM product_variants pv
        WHERE pv.variant_id IN (?
    ` + strings.Repeat(",?", len(variantIDs)-1) + `)
    ORDER BY pv.product_id`

	mappingArgs := make([]interface{}, len(variantIDs))
	for i, id := range variantIDs {
		mappingArgs[i] = id
	}

	mappingRows, err := DB.Query(mappingQuery, mappingArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer mappingRows.Close()

	productVariantMap := make(map[string][]string) // variantID -> []productIDs
	allProductIDs := make(map[string]bool)
	var productIDs []string

	for mappingRows.Next() {
		var variantID, productID string
		err := mappingRows.Scan(&variantID, &productID)
		if err != nil {
			return nil, nil, err
		}

		productVariantMap[variantID] = append(productVariantMap[variantID], productID)
		if !allProductIDs[productID] {
			allProductIDs[productID] = true
			productIDs = append(productIDs, productID)
		}
	}

	if len(productIDs) == 0 {
		return nil, productVariantMap, nil
	}

	// Now fetch the actual product data with pagination
	productQuery := `
        SELECT p.product_id, p.name, p.description, p.price, p.category_id,
               p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
        FROM products p
        WHERE p.product_id IN (?
    ` + strings.Repeat(",?", len(productIDs)-1) + `)
    ORDER BY p.product_id
    LIMIT ? OFFSET ?`

	productArgs := make([]interface{}, len(productIDs)+2)
	for i, id := range productIDs {
		productArgs[i] = id
	}
	productArgs[len(productIDs)] = limit
	productArgs[len(productIDs)+1] = offset

	productRows, err := DB.Query(productQuery, productArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer productRows.Close()

	var products []dtos.Product
	productMap := make(map[string]dtos.Product)

	for productRows.Next() {
		var p dtos.Product
		err := productRows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated)
		if err != nil {
			return nil, nil, err
		}
		productMap[p.ID] = p
		products = append(products, p)
	}

	// Fetch images for all products
	for i := range products {
		images, err := fetchProductImages(products[i].ID)
		if err != nil {
			return nil, nil, err
		}
		products[i].Images = images
	}

	return products, productVariantMap, nil
}

func fetchAllProductsByVariant(variantID string) ([]dtos.Product, error) {
	rows, err := DB.Query(`
        SELECT p.product_id, p.name, p.description, p.price, p.category_id,
               p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
        FROM products p
        INNER JOIN product_variants pv ON p.product_id = pv.product_id
        WHERE pv.variant_id = ?`, variantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.Product
	for rows.Next() {
		var p dtos.Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated)
		if err != nil {
			return nil, err
		}

		images, err := fetchProductImages(p.ID)
		if err != nil {
			return nil, err
		}
		p.Images = images

		products = append(products, p)
	}

	return products, nil
}
