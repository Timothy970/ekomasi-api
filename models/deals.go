package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"

	"github.com/teris-io/shortid"
)

func CreateDeal(deal dtos.CreateDeal) (string, error) {
	//check id name exists
	exists, err := RecordExists("deals", "name = ?", deal.Name)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("deal with name %s already exists", deal.Name)
	}
	dealID, _ := shortid.Generate()
	_, err = DB.Exec(`INSERT INTO deals (deal_id, name, description, discount, start_date, end_date) VALUES (?, ?, ?, ?, ?, ?)`, dealID, deal.Name, deal.Description, deal.Discount, deal.StartDate, deal.EndDate)
	if err != nil {
		return "", err
	}
	return dealID, nil
}
func GetAllDeals(page, size int) ([]dtos.Deal, *dtos.PaginationMeta, error) {
	var countTotal int
	err := DB.QueryRow(`SELECT COUNT(*) FROM deals`).Scan(&countTotal)
	rows, err := DB.Query(`
	SELECT deal_id, name, start_date, end_date
	FROM deals d
	LIMIT ? OFFSET ?
	`, size, (page-1)*size)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var deals []dtos.Deal
	for rows.Next() {
		var d dtos.Deal
		if err := rows.Scan(&d.DealID, &d.Name, &d.StartDate, &d.EndDate); err != nil {
			return nil, nil, err
		}
		// get the deals products
		products, err := GetProductsByDealID(d.DealID)
		if err != nil {
			return nil, nil, err
		}
		d.Products = products
		deals = append(deals, d)
	}
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: countTotal,
		TotalPages: (countTotal + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < countTotal,
	}
	return deals, pagination, nil
}

func isDealThere(dealID string) error {
	if exists, err := RecordExists("deals", "deal_id = ?", dealID); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("deal with ID %s does not exist", dealID)
	}
	return nil
}
func UpdateDeal(dealID string, deal dtos.Deal) error {
	if err := isDealThere(dealID); err != nil {
		return err
	}
	_, err := DB.Exec(`UPDATE deals SET name = ?, start_date = ?, end_date = ? WHERE deal_id = ?`,
		deal.Name, deal.StartDate, deal.EndDate, dealID)
	return err
}
func DeleteDeal(dealID string) error {
	if err := isDealThere(dealID); err != nil {
		return err
	}
	_, err := DB.Exec(`DELETE FROM deals WHERE deal_id = ?`, dealID)
	return err
}
func AddProductToDeal(dealID, productID string, discountType *string, discount *float64) error {
	if err := isDealThere(dealID); err != nil {
		return err
	}
	if err := IsProductThere(productID); err != nil {
		return err
	}
	//check if product is already in deal
	exists, err := RecordExists("deal_products", "deal_id = ? AND product_id = ?", dealID, productID)
	if err != nil {
		return err
	}
	if exists {
		// Update existing entry
		_, err := DB.Exec(`UPDATE deal_products SET discount = ?, discount_type = ? WHERE deal_id = ? AND product_id = ?`,
			discount, discountType, dealID, productID)
		return err
	}
	productDealID, _ := shortid.Generate()
	_, err = DB.Exec(`INSERT INTO deal_products (product_deal_id, deal_id, product_id, discount, discount_type) VALUES (?, ?, ?, ?, ?)`,
		productDealID, dealID, productID, discount, discountType)
	return err
}
func RemoveProductFromDeal(dealID, productID string) error {
	if err := isDealThere(dealID); err != nil {
		return err
	}
	if err := IsProductThere(productID); err != nil {
		return err
	}
	_, err := DB.Exec(`DELETE FROM deal_products WHERE deal_id = ? AND product_id = ?`, dealID, productID)
	return err
}
func GetDealWithProducts(dealID string, page, limit int) (*dtos.DealWithProducts, dtos.PaginationMeta, error) {
	// Check if the deal exists
	if err := isDealThere(dealID); err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	query := `
		SELECT 
			d.deal_id, d.name, d.start_date, d.end_date
		FROM deals d
		WHERE d.deal_id = ?
	`

	row := DB.QueryRow(query, dealID)

	var deal dtos.DealWithProducts
	var (
		name      sql.NullString
		startDate sql.NullTime
		endDate   sql.NullTime
	)

	if err := row.Scan(&deal.DealID, &name, &startDate, &endDate); err != nil {
		if err == sql.ErrNoRows {
			return nil, dtos.PaginationMeta{}, fmt.Errorf("deal not found")
		}
		return nil, dtos.PaginationMeta{}, err
	}

	deal.Name = name.String
	deal.StartDate = startDate.Time
	deal.EndDate = endDate.Time

	// Fetch paginated deal products
	products, pagination, err := GetProductsByDealIDWithPagination(dealID, page, limit)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	deal.Products = products

	return &deal, *pagination, nil
}

func GetProductsByDealID(dealID string) ([]dtos.DealProduct, error) {
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at,
			c.name AS category_name, p.tag, dp.discount, dp.discount_type
		FROM deal_products dp
		INNER JOIN products p ON p.product_id = dp.product_id
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE dp.deal_id = ?
	`

	rows, err := DB.Query(query, dealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.DealProduct

	for rows.Next() {
		var p dtos.DealProduct
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
			&p.CategoryName, &p.Tag, &p.Discount, &p.DiscountType,
		)
		if err != nil {
			return nil, err
		}

		// Fetch product images
		images, err := fetchProductImages(p.ID)
		if err != nil {
			return nil, err
		}
		p.Images = images

		// Fetch product variants
		variants, err := getProductVariants(p.ID)
		if err != nil {
			return nil, err
		}
		p.ProductVariants = variants

		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func GetProductsByDealIDWithPagination(dealID string, page, limit int) ([]dtos.DealProduct, *dtos.PaginationMeta, error) {
	countQuery := `SELECT COUNT(*) FROM deal_products WHERE deal_id = ?`
	var totalItems int
	err := DB.QueryRow(countQuery, dealID).Scan(&totalItems)
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at,
			c.name AS category_name, p.tag, dp.discount, dp.discount_type
		FROM deal_products dp
		INNER JOIN products p ON p.product_id = dp.product_id
		LEFT JOIN categories c ON p.category_id = c.category_id
		WHERE dp.deal_id = ?
		LIMIT ? OFFSET ?
	`
	offset := (page - 1) * limit
	rows, err := DB.Query(query, dealID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var products []dtos.DealProduct

	for rows.Next() {
		var p dtos.DealProduct
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
			&p.CategoryName, &p.Tag, &p.Discount, &p.DiscountType,
		)
		if err != nil {
			return nil, nil, err
		}

		// Fetch product images
		images, err := fetchProductImages(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.Images = images

		// Fetch product variants
		variants, err := getProductVariants(p.ID)
		if err != nil {
			return nil, nil, err
		}
		p.ProductVariants = variants

		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: (totalItems + limit - 1) / limit,
		HasPrev:    page > 1,
		HasNext:    page*limit < totalItems,
	}

	return products, pagination, nil
}
