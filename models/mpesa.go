package models

import "adenzo_backend/dtos"

func HandleMpesaMoneyReturnRefunds(order dtos.Order) error {
	//change the order status to refunded
	status := "Refunded"
	err := UpdateOrderStatus(order.OrderID, dtos.UpdateOrderStatusRequest{
		Status: &status,
	})
	if err != nil {
		return err
	}

	for _, item := range order.Items {
		//restock the items
		err = UpdateProductStock(item.ID, item.StockQuantity)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateProductStock(productID string, quantity int) error {
	_, err := DB.Exec(`
		UPDATE products SET stock_quantity = stock_quantity + ? WHERE product_id = ?`,
		quantity, productID,
	)
	return err
}
