package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"
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
	order := []string{} // preserve insertion order

	for rows.Next() {
		var v dtos.VariantResponse
		if err := rows.Scan(&v.VariantID, &v.VariantType, &v.Name, &v.HexCode); err != nil {
			return nil, err
		}
		// first time we see this type → track it
		if _, exists := groupMap[v.VariantType]; !exists {
			order = append(order, v.VariantType)
		}
		groupMap[v.VariantType] = append(groupMap[v.VariantType], v)
	}

	// build grouped slice in deterministic order
	var grouped []dtos.GroupedVariants
	for _, t := range order {
		grouped = append(grouped, dtos.GroupedVariants{
			VariantType: t,
			Variants:    groupMap[t],
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

	// Extract unique filters from variants
	variantIDs, variantNames := extractVariantFilters(variants)

	// Build and execute base query
	variantResults, variantMap, err := executeVariantQuery(variantIDs, variantNames)
	if err != nil {
		return nil, nil, err
	}
	if len(variantResults) == 0 {
		return nil, nil, nil
	}

	// Get pagination info
	resultVariantIDs := extractVariantIDs(variantResults)
	total, err := countTotalProducts(resultVariantIDs)
	if err != nil {
		return nil, nil, err
	}

	// Fetch paginated products
	products, productVariantMap, err := fetchProductsByVariantsPaginated(resultVariantIDs, limit, (page-1)*limit)
	if err != nil {
		return nil, nil, err
	}

	// Associate products with variants
	associateProductsWithVariants(variantMap, products, productVariantMap)

	// Handle "All" variants
	if err := handleAllVariants(variants, variantMap); err != nil {
		return nil, nil, err
	}

	pagination := createPaginationMeta(page, limit, total)
	return variantResults, pagination, nil
}

// Helper functions
func extractVariantFilters(variants []dtos.Variant) ([]string, []string) {
	variantIDMap := make(map[string]bool)
	variantNameMap := make(map[string]bool)
	var variantIDs, variantNames []string

	for _, variant := range variants {
		if variant.VariantID != "" && !variantIDMap[variant.VariantID] {
			variantIDs = append(variantIDs, variant.VariantID)
			variantIDMap[variant.VariantID] = true
		}
		if isVariantNameValid(variant.Name) && !variantNameMap[variant.Name] {
			variantNames = append(variantNames, variant.Name)
			variantNameMap[variant.Name] = true
		}
	}

	return variantIDs, variantNames
}

func isVariantNameValid(name string) bool {
	return name != "" && name != "All"
}

func executeVariantQuery(variantIDs, variantNames []string) ([]*dtos.VariantWithProducts, map[string]*dtos.VariantWithProducts, error) {
	query, args := buildVariantQuery(variantIDs, variantNames)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	return scanVariantRows(rows)
}

func buildVariantQuery(variantIDs, variantNames []string) (string, []interface{}) {
	query := `
        SELECT v.variant_id, v.variant_type, v.name, v.hex_code,
               pv.additional_price, pv.stock_quantity
        FROM variants v
        INNER JOIN product_variants pv ON v.variant_id = pv.variant_id
        WHERE 1=1
    `
	var args []interface{}

	query, args = addInClause(query, args, "v.variant_id", variantIDs)
	query, args = addInClause(query, args, "LOWER(v.name)", transformToLower(variantNames))

	return query, args
}

func addInClause(query string, args []interface{}, field string, values []string) (string, []interface{}) {
	if len(values) == 0 {
		return query, args
	}

	placeholders := "?" + strings.Repeat(",?", len(values)-1)
	query += fmt.Sprintf(" AND %s IN (%s)", field, placeholders)

	for _, value := range values {
		args = append(args, value)
	}

	return query, args
}

func transformToLower(names []string) []string {
	result := make([]string, len(names))
	for i, name := range names {
		result[i] = strings.ToLower(name)
	}
	return result
}

func scanVariantRows(rows *sql.Rows) ([]*dtos.VariantWithProducts, map[string]*dtos.VariantWithProducts, error) {
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

	return variantResults, variantMap, nil
}

func extractVariantIDs(variants []*dtos.VariantWithProducts) []string {
	ids := make([]string, len(variants))
	for i, v := range variants {
		ids[i] = v.VariantID
	}
	return ids
}

func countTotalProducts(variantIDs []string) (int, error) {
	if len(variantIDs) == 0 {
		return 0, nil
	}

	query := "SELECT COUNT(DISTINCT p.product_id) FROM products p " +
		"INNER JOIN product_variants pv ON p.product_id = pv.product_id " +
		"WHERE pv.variant_id IN (?" + strings.Repeat(",?", len(variantIDs)-1) + ")"

	args := makeInterfaceSlice(variantIDs)

	var total int
	err := DB.QueryRow(query, args...).Scan(&total)
	return total, err
}

func associateProductsWithVariants(variantMap map[string]*dtos.VariantWithProducts, products []dtos.Product, productVariantMap map[string][]string) {
	productMap := createProductMap(products)

	for variantID, productIDs := range productVariantMap {
		if variant, exists := variantMap[variantID]; exists {
			variant.Products = findProductsByIDs(productMap, productIDs)
		}
	}
}

func createProductMap(products []dtos.Product) map[string]dtos.Product {
	productMap := make(map[string]dtos.Product)
	for _, product := range products {
		productMap[product.ID] = product
	}
	return productMap
}

func findProductsByIDs(productMap map[string]dtos.Product, productIDs []string) []dtos.Product {
	var result []dtos.Product
	for _, productID := range productIDs {
		if product, exists := productMap[productID]; exists {
			result = append(result, product)
		}
	}
	return result
}

func handleAllVariants(variants []dtos.Variant, variantMap map[string]*dtos.VariantWithProducts) error {
	for _, variant := range variants {
		if variant.Name == "All" && variant.VariantID != "" {
			if existingVariant, exists := variantMap[variant.VariantID]; exists {
				allProducts, err := fetchAllProductsByVariant(variant.VariantID)
				if err != nil {
					return err
				}
				existingVariant.Products = allProducts
			}
		}
	}
	return nil
}

func createPaginationMeta(page, limit, total int) *dtos.PaginationMeta {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
}

func makeInterfaceSlice(strings []string) []interface{} {
	args := make([]interface{}, len(strings))
	for i, s := range strings {
		args[i] = s
	}
	return args
}

// Refactored fetchProductsByVariantsPaginated function
func fetchProductsByVariantsPaginated(variantIDs []string, limit, offset int) ([]dtos.Product, map[string][]string, error) {
	if len(variantIDs) == 0 {
		return nil, nil, errors.New("no variant IDs provided")
	}

	productVariantMap, productIDs, err := fetchProductVariantMappings(variantIDs)
	if err != nil {
		return nil, nil, err
	}

	if len(productIDs) == 0 {
		return nil, productVariantMap, nil
	}

	products, err := fetchPaginatedProducts(productIDs, limit, offset)
	if err != nil {
		return nil, nil, err
	}

	return products, productVariantMap, nil
}

func fetchProductVariantMappings(variantIDs []string) (map[string][]string, []string, error) {
	query := buildMappingQuery(variantIDs)
	args := makeInterfaceSlice(variantIDs)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	return scanProductVariantMappings(rows)
}

func buildMappingQuery(variantIDs []string) string {
	return `
        SELECT pv.variant_id, pv.product_id
        FROM product_variants pv
        WHERE pv.variant_id IN (?` + strings.Repeat(",?", len(variantIDs)-1) + `)
        ORDER BY pv.product_id`
}

func scanProductVariantMappings(rows *sql.Rows) (map[string][]string, []string, error) {
	productVariantMap := make(map[string][]string)
	allProductIDs := make(map[string]bool)
	var productIDs []string

	for rows.Next() {
		var variantID, productID string
		if err := rows.Scan(&variantID, &productID); err != nil {
			return nil, nil, err
		}

		productVariantMap[variantID] = append(productVariantMap[variantID], productID)
		if !allProductIDs[productID] {
			allProductIDs[productID] = true
			productIDs = append(productIDs, productID)
		}
	}

	return productVariantMap, productIDs, nil
}

func fetchPaginatedProducts(productIDs []string, limit, offset int) ([]dtos.Product, error) {
	query := buildProductQueryVariants(productIDs)
	args := makeInterfaceSlice(productIDs)
	args = append(args, limit, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products, err := scanProducts(rows)
	if err != nil {
		return nil, err
	}

	return fetchProductImagesBatch(products)
}

func buildProductQueryVariants(productIDs []string) string {
	return `
        SELECT p.product_id, p.name, p.description, p.price, p.category_id,
               p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
        FROM products p
        WHERE p.product_id IN (?` + strings.Repeat(",?", len(productIDs)-1) + `)
        ORDER BY p.product_id
        LIMIT ? OFFSET ?`
}

func scanProducts(rows *sql.Rows) ([]dtos.Product, error) {
	var products []dtos.Product

	for rows.Next() {
		var p dtos.Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}

func fetchProductImagesBatch(products []dtos.Product) ([]dtos.Product, error) {
	for i := range products {
		images, err := fetchProductImages(products[i].ID)
		if err != nil {
			return nil, err
		}
		products[i].Images = images
	}
	return products, nil
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
