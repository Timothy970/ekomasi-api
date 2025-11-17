package models

import "adenzo_backend/dtos"

func HandleMpesaMoneyReturnRefunds(OrderID string) error {
	//change the order status to refunded
	err := UpdateOrderStatus(OrderID, dtos.UpdateOrderStatusRequest{
		Status: "refunded",
	})
	if err != nil {
		return err
	}
	//get the order items
	order, err := GetOrderByID(OrderID)
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
