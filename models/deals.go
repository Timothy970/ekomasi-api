package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"

	"github.com/teris-io/shortid"
)

func CreateDeal(deal dtos.CreateDeal) error {
	//check id name exists
	exists, err := RecordExists("deals", "name = ?", deal.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("deal with name %s already exists", deal.Name)
	}
	dealID, _ := shortid.Generate()
	_, err = DB.Exec(`INSERT INTO deals (deal_id, name, description, discount) VALUES (?, ?, ?)`, dealID, deal.Name, deal.Description, deal.Discount)
	return err
}
func GetAllDeals() ([]dtos.Deal, error) {
	rows, err := DB.Query(`SELECT deal_id, name, description, discount FROM deals`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var deals []dtos.Deal
	for rows.Next() {
		var d dtos.Deal
		if err := rows.Scan(&d.DealID, &d.Name, &d.Description, &d.Discount); err != nil {
			return nil, err
		}
		deals = append(deals, d)
	}
	return deals, nil
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
	_, err := DB.Exec(`UPDATE deals SET name = ?, description = ?, discount = ? WHERE deal_id = ?`,
		deal.Name, deal.Description, deal.Discount, dealID)
	return err
}
func DeleteDeal(dealID string) error {
	if err := isDealThere(dealID); err != nil {
		return err
	}
	_, err := DB.Exec(`DELETE FROM deals WHERE deal_id = ?`, dealID)
	return err
}
func AddProductToDeal(dealID, productID string) error {
	if err := isDealThere(dealID); err != nil {
		return err
	}
	if err := isProductThere(productID); err != nil {
		return err
	}
	//check if product is already in deal
	exists, err := RecordExists("deal_products", "deal_id = ? AND product_id = ?", dealID, productID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("product with ID %s is already in deal %s", productID, dealID)
	}
	productDealID, _ := shortid.Generate()
	_, err = DB.Exec(`INSERT INTO deal_products (product_deal_id, deal_id, product_id) VALUES (?, ?, ?)`, productDealID, dealID, productID)
	return err
}
func RemoveProductFromDeal(dealID, productID string) error {
	if err := isDealThere(dealID); err != nil {
		return err
	}
	if err := isProductThere(productID); err != nil {
		return err
	}
	_, err := DB.Exec(`DELETE FROM deal_products WHERE deal_id = ? AND product_id = ?`, dealID, productID)
	return err
}
func GetDealWithProducts(dealID string, page, limit int) (*dtos.DealWithProducts, dtos.PaginationMeta, error) {
	// Check if deal exists
	if err := isDealThere(dealID); err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Count total products in the deal
	var total int
	countQuery := `SELECT COUNT(*) FROM deal_products dp WHERE dp.deal_id = ?`
	if err := DB.QueryRow(countQuery, dealID).Scan(&total); err != nil {
		return nil, dtos.PaginationMeta{}, err
	}

	// Pagination setup
	offset := (page - 1) * limit
	totalPages := (total + limit - 1) / limit
	pagination := dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	// Query deal details + paginated products
	query := `
		SELECT d.deal_id, d.name, d.description, d.discount, dp.product_id
		FROM deals d
		LEFT JOIN deal_products dp ON d.deal_id = dp.deal_id
		WHERE d.deal_id = ?
		LIMIT ? OFFSET ?
	`
	rows, err := DB.Query(query, dealID, limit, offset)
	if err != nil {
		return nil, dtos.PaginationMeta{}, err
	}
	defer rows.Close()

	var deal *dtos.DealWithProducts

	for rows.Next() {
		var (
			dealIDVal, name, desc sql.NullString
			discount              sql.NullFloat64
			productID             sql.NullString
		)

		if err := rows.Scan(&dealIDVal, &name, &desc, &discount, &productID); err != nil {
			return nil, dtos.PaginationMeta{}, err
		}

		// Initialize deal once
		if deal == nil {
			deal = &dtos.DealWithProducts{
				DealID:      dealIDVal.String,
				Name:        name.String,
				Description: desc.String,
				Discount:    nullableFloat64(discount),
				Products:    []dtos.Product{},
			}
		}

		// If product exists, fetch details
		if productID.Valid {
			product, err := GetProductByID(productID.String)
			if err != nil {
				return nil, dtos.PaginationMeta{}, err
			}
			deal.Products = append(deal.Products, *product)
		}
	}

	if deal == nil {
		// In case deal exists but has no products
		deal = &dtos.DealWithProducts{
			DealID:   dealID,
			Products: []dtos.Product{},
		}
	}

	return deal, pagination, nil
}

// helper for nullable float64
func nullableFloat64(f sql.NullFloat64) *float64 {
	if f.Valid {
		return &f.Float64
	}
	return nil
}
