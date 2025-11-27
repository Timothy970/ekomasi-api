package models

import (
	"adenzo_backend/dtos"
	"fmt"

	"github.com/teris-io/shortid"
)

func CreateReturns(req dtos.ReturnRequest, userID string) error {
	returnID, _ := shortid.Generate()
	//create the return record in the database
	query := `
		INSERT INTO returns (return_id, reason, status, order_id, user_id)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := DB.Exec(query, returnID, req.Reason, "Pending", req.OrderID, userID)
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
	for _, product := range req.ReturnProducts {
		returnProductID, _ := shortid.Generate()
		_, err := DB.Exec(query, returnProductID, returnID, product.ProductID, product.Quantity)
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
		product.StockQuantity = quantity
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

func GetAllReturns(page, size int, status, q string) (*dtos.ReturnListResponse, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	where := "WHERE 1=1"
	var args []interface{}
	//use lower case for status comparison
	if status != "" && status != "All" {
		where += " AND LOWER(r.status) LIKE ?"
		args = append(args, "%"+status+"%")
	}

	if q != "" {
		where += `
			AND (
				r.reason LIKE ? 
				OR r.order_id LIKE ?
				OR p.name LIKE ?
			)
		`
		args = append(args, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}

	countQuery := `
		SELECT COUNT(DISTINCT r.return_id)
		FROM returns r
		LEFT JOIN return_products rp ON r.return_id = rp.return_id
		LEFT JOIN products p ON rp.product_id = p.product_id
		` + where

	var totalRecords int
	if err := DB.QueryRow(countQuery, args...).Scan(&totalRecords); err != nil {
		return nil, nil, err
	}

	selectQuery := `
		SELECT DISTINCT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		LEFT JOIN return_products rp ON r.return_id = rp.return_id
		LEFT JOIN products p ON rp.product_id = p.product_id
		` + where + `
		ORDER BY r.created_at DESC
		LIMIT ? OFFSET ?
	`
	argsWithPagination := append(args, size, offset)

	rows, err := DB.Query(selectQuery, argsWithPagination...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var returns []dtos.ReturnResponse

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

	// First get counts grouped by status
	countsQuery := `
    SELECT status, COUNT(*)
    FROM returns
    GROUP BY status
`
	countRows, err := DB.Query(countsQuery)
	if err != nil {
		return nil, nil, err
	}
	defer countRows.Close()

	// Prepare a map to store DB results
	statusMap := map[string]int{
		"Pending":  0,
		"Approved": 0,
		"Rejected": 0,
	}

	// Fill with DB results
	for countRows.Next() {
		var s string
		var c int
		if err := countRows.Scan(&s, &c); err != nil {
			return nil, nil, err
		}
		if _, ok := statusMap[s]; ok {
			statusMap[s] = c
		}
	}

	// Now build the ordered result
	var counts []dtos.ReturnsCounts = []dtos.ReturnsCounts{
		{Status: "Pending", Count: statusMap["Pending"]},
		{Status: "Approved", Count: statusMap["Approved"]},
		{Status: "Rejected", Count: statusMap["Rejected"]},
		{Status: "Total Returns", Count: totalRecords}, // add All last
	}

	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalRecords,
		TotalPages: (totalRecords + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    offset+size < totalRecords,
	}

	return &dtos.ReturnListResponse{
		Returns:        returns,
		CountsByStatus: counts,
	}, meta, nil
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
		product.StockQuantity = quantity
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

func ValidateReturnRequest(req dtos.ReturnRequest) error {
	// check if order exists
	err := IsOrderThere(req.OrderID)
	if err != nil {
		return err
	}
	// check if products exist in the order
	for _, product := range req.ReturnProducts {
		err := IsProductInOrder(req.OrderID, product.ProductID)
		if err != nil {
			return err
		}
	}
	return nil
}

func IsProductInOrder(orderID, productID string) error {
	//first is even product there
	err := IsProductThere(productID)
	if err != nil {
		return err
	}
	//now is the product in the order
	query := `
		SELECT COUNT(*) FROM order_items
		WHERE order_id = ? AND product_id = ?
	`
	var count int
	err = DB.QueryRow(query, orderID, productID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("product with ID %s not found in order %s", productID, orderID)
	}
	return nil
}

func GetAllOwnerReturns(status, q, ownerID string) ([]dtos.ReturnResponse, error) {
	where := "WHERE 1=1"
	var args []interface{}
	//use lower case for status comparison
	if status != "" && status != "All" {
		where += " AND LOWER(r.status) LIKE ?"
		args = append(args, "%"+status+"%")
	}

	if q != "" {
		where += `
			AND (
				r.reason LIKE ? 
				OR r.order_id LIKE ?
				OR p.name LIKE ?
			)
		`
		args = append(args, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	where += " AND r.user_id = ?"
	args = append(args, ownerID)
	selectQuery := `
		SELECT DISTINCT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		LEFT JOIN return_products rp ON r.return_id = rp.return_id
		LEFT JOIN products p ON rp.product_id = p.product_id
		` + where + `
		ORDER BY r.created_at DESC
	`

	rows, err := DB.Query(selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var returns []dtos.ReturnResponse

	for rows.Next() {
		var ret dtos.ReturnResponse

		if err := rows.Scan(&ret.ReturnID, &ret.Reason, &ret.Status, &ret.CreatedAt, &ret.OrderID); err != nil {
			return nil, err
		}

		products, totalRefund, err := fetchReturnProductsAndRefund(ret.ReturnID, ret.OrderID)
		if err != nil {
			return nil, err
		}

		ret.Products = products
		ret.TotalRefund = totalRefund
		returns = append(returns, ret)
	}

	return returns, nil
}

func GetOwnerReturnByID(returnID, ownerID string) (dtos.ReturnResponse, error) {
	var ret dtos.ReturnResponse
	query := `
		SELECT r.return_id, r.reason, r.status, r.created_at, r.order_id
		FROM returns r
		WHERE r.return_id = ? AND r.user_id = ?
	`
	row := DB.QueryRow(query, returnID, ownerID)
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
		product.StockQuantity = quantity
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
