package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"github.com/teris-io/shortid"
	"math"
	"strings"
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

func ListVariants() ([]dtos.VariantResponse, error) {
	rows, err := DB.Query(`
        SELECT variant_id, variant_type, name, hex_code
        FROM variants
        ORDER BY variant_type, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []dtos.VariantResponse
	for rows.Next() {
		var v dtos.VariantResponse
		if err := rows.Scan(&v.VariantID, &v.VariantType, &v.Name, &v.HexCode); err != nil {
			return nil, err
		}
		variants = append(variants, v)
	}
	return variants, nil
}
func variantexists(id string) error {
	exists, err := RecordExists("variants", "where variant_id = ?", id)
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
	_, err = DB.Exec(`
        INSERT INTO product_variants (product_variants_id, variant_id, product_id, additional_price, stock_quantity)
        VALUES (?, ?, ?, ?, ?)`,
		pvID, id, req.ProductID, req.AdditionalPrice, req.StockQuantity,
	)
	return err
}

func RemoveProductVariant(productID, variantID string) error {
	_, err := DB.Exec(`
        DELETE FROM product_variants
        WHERE product_id = ? AND variant_id = ?`, productID, variantID,
	)
	if err == sql.ErrNoRows {
		return errors.New("product not found")
	} else if err != nil {
		return err
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

func GetVariantWithProductsPaginated(variantID, name string, page, limit int) (*dtos.VariantWithProducts, *dtos.PaginationMeta, error) {
	var args []interface{}
	query := `
		SELECT v.variant_id, v.variant_type, v.name, v.hex_code,
		       pv.additional_price, pv.stock_quantity
		FROM variants v
		INNER JOIN product_variants pv ON v.variant_id = pv.variant_id
		WHERE 1=1
	`

	if variantID != "" {
		query += " AND v.variant_id = ?"
		args = append(args, variantID)
	}
	if name != "" {
		query += " AND LOWER(v.name) = ?"
		args = append(args, strings.ToLower(name))
	}

	query += " LIMIT 1"

	row := DB.QueryRow(query, args...)
	var variant dtos.VariantWithProducts
	err := row.Scan(&variant.VariantID, &variant.VariantType, &variant.Name, &variant.HexCode,
		&variant.AdditionalPrice, &variant.StockQuantity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	// Count total products for pagination
	countQuery := `
		SELECT COUNT(*)
		FROM products p
		INNER JOIN product_variants pv ON p.product_id = pv.product_id
		WHERE pv.variant_id = ?
	`
	var total int
	err = DB.QueryRow(countQuery, variant.VariantID).Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	offset := (page - 1) * limit

	// Fetch paginated products
	products, err := fetchProductsByVariantPaginated(variant.VariantID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	variant.Products = products

	// Pagination meta
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return &variant, pagination, nil
}

func fetchProductsByVariantPaginated(variantID string, limit, offset int) ([]dtos.Product, error) {
	rows, err := DB.Query(`
		SELECT p.product_id, p.name, p.description, p.price, p.category_id,
		       p.stock_quantity, p.search_vector, p.created_at, p.last_updated
		FROM products p
		INNER JOIN product_variants pv ON p.product_id = pv.product_id
		WHERE pv.variant_id = ?
		LIMIT ? OFFSET ?`, variantID, limit, offset)
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
