package models

import (
	"adenzo_backend/dtos"

	"github.com/teris-io/shortid"
)

func CreateReturns(req dtos.ReturnRequest) error {
	returnID, _ := shortid.Generate()
	//create the return record in the database
	query := `
		INSERT INTO returns (return_id, reason, status, order_id)
		VALUES (?, ?, ?, ?)
	`
	_, err := DB.Exec(query, returnID, req.Reason, "Pending", req.OrderID)
	if err != nil {
		return err
	}
	//insert products associated with the return
	err = insertIntoReturnProducts(returnID, req)
	if err != nil {
		return err
	}
	return nil
}

func insertIntoReturnProducts(returnID string, req dtos.ReturnRequest) error {
	query := `
		INSERT INTO return_products (return_product_id, return_id, product_id, quantity)
		VALUES (?, ?, ?, ?)
	`
	for _, productID := range req.Products {
		returnProductID, _ := shortid.Generate()
		_, err := DB.Exec(query, returnProductID, returnID, productID, req.Quantity)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateReturnStatus(returnID string, statusUpdate dtos.ReturnStatusUpdate) error {
	query := `
		UPDATE returns SET status = ? WHERE return_id = ?
	`
	_, err := DB.Exec(query, statusUpdate.Status, returnID)
	return err
}
func GetReturnByID(returnID string) (dtos.ReturnResponse, error) {
	var ret dtos.ReturnResponse
	query := `
		SELECT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		WHERE r.return_id = ?
	`
	row := DB.QueryRow(query, returnID)
	err := row.Scan(&ret.ReturnID, &ret.Reason, &ret.Status, &ret.CreatedAt, &ret.OrderID)
	if err != nil {
		return ret, err
	}
	//get product ids associated with the return
	productIDsQuery := `
		SELECT rp.product_id, rp.quantity
		FROM return_products rp
		WHERE rp.return_id = ?
	`
	rows, err := DB.Query(productIDsQuery, returnID)
	if err != nil {
		return ret, err
	}
	defer rows.Close()
	var productID string
	var quantity int
	var products []dtos.Product

	for rows.Next() {
		var product *dtos.Product
		err := rows.Scan(&productID, &quantity)
		if err != nil {
			return ret, err
		}
		product, err = GetProductByID(productID)
		if err != nil {
			return ret, err
		}
		refund, err := GetProductRefundAmount(productID, quantity, ret.OrderID)
		if err != nil {
			return ret, err
		}
		ret.TotalRefund += refund
		products = append(products, *product)
	}
	ret.Products = products
	return ret, nil
}
func GetProductRefundAmount(productID string, quantity int, orderID string) (float64, error) {
	var price float64
	//get unit price from order_items
	query := `
		SELECT oi.unit_price
		FROM order_items oi
		WHERE oi.product_id = ? AND oi.order_id = ?
		LIMIT 1
	`
	row := DB.QueryRow(query, productID, orderID)
	err := row.Scan(&price)
	if err != nil {
		return 0, err
	}
	return price * float64(quantity), nil
}

func GetAllReturns(page, size int) ([]dtos.ReturnResponse, *dtos.PaginationMeta, error) {
	var returns []dtos.ReturnResponse
	var totalRecords int

	// Get total count
	countQuery := `SELECT COUNT(*) FROM returns`
	row := DB.QueryRow(countQuery)
	if err := row.Scan(&totalRecords); err != nil {
		return nil, nil, err
	}

	query := `
		SELECT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		LIMIT ? OFFSET ?
	`
	rows, err := DB.Query(query, size, (page-1)*size)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ret dtos.ReturnResponse
		if err := rows.Scan(&ret.ReturnID, &ret.Reason, &ret.Status, &ret.CreatedAt, &ret.OrderID); err != nil {
			return nil, nil, err
		}
		products, totalRefund, err := fetchReturnProductsAndRefund(ret.ReturnID, ret.OrderID)
		if err != nil {
			return nil, nil, err
		}
		ret.Products = products
		ret.TotalRefund = totalRefund
		returns = append(returns, ret)
	}

	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalRecords,
		TotalPages: (totalRecords + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < totalRecords,
	}
	return returns, meta, nil
}

// fetchReturnProductsAndRefund fetches products and calculates total refund for a return
func fetchReturnProductsAndRefund(returnID, orderID string) ([]dtos.Product, float64, error) {
	productIDsQuery := `
		SELECT rp.product_id, rp.quantity
		FROM return_products rp
		WHERE rp.return_id = ?
	`
	productRows, err := DB.Query(productIDsQuery, returnID)
	if err != nil {
		return nil, 0, err
	}
	defer productRows.Close()

	var products []dtos.Product
	var totalRefund float64
	for productRows.Next() {
		var productID string
		var quantity int
		if err := productRows.Scan(&productID, &quantity); err != nil {
			return nil, 0, err
		}
		product, err := GetProductByID(productID)
		if err != nil {
			return nil, 0, err
		}
		refund, err := GetProductRefundAmount(productID, quantity, orderID)
		if err != nil {
			return nil, 0, err
		}
		totalRefund += refund
		products = append(products, *product)
	}
	return products, totalRefund, nil
}

func DeleteReturn(returnID string) error {
	query := `
		DELETE FROM returns WHERE return_id = ?
	`
	_, err := DB.Exec(query, returnID)
	return err
}
