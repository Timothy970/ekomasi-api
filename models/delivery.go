package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"

	"github.com/teris-io/shortid"
)

func CreateNewDelivery(delivery dtos.Delivery) error {
	exists, err := RecordExists("orders", "order_id = ?", delivery.OrderID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("order not found")
	}
	deliveryID, _ := shortid.Generate()

	if delivery.Status == "" {
		delivery.Status = "PENDING"
	}

	_, err = DB.Exec(`INSERT INTO deliveries (delivery_id, order_id, delivery_charge, status, courier_details, delivery_address)
		VALUES (?, ?, ?, ?, ?, ?)`,
		deliveryID, delivery.OrderID, delivery.DeliveryCharge, delivery.Status, delivery.CourierDetails, delivery.DeliveryAddress)

	if err != nil {
		return err
	}
	return nil
}

func ListDeliveries(page, size int) ([]dtos.Delivery, *dtos.PaginationMeta, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM deliveries`).Scan(&total); err != nil {
		return nil, nil, err
	}

	offset := (page - 1) * size
	rows, err := DB.Query(`SELECT delivery_id, order_id, delivery_charge, status, courier_details, delivery_address
		FROM deliveries LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var deliveries []dtos.Delivery
	for rows.Next() {
		var d dtos.Delivery
		if err := rows.Scan(&d.DeliveryID, &d.OrderID, &d.DeliveryCharge, &d.Status, &d.CourierDetails, &d.DeliveryAddress); err != nil {
			return nil, nil, err
		}
		deliveries = append(deliveries, d)
	}

	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: (total + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page < (total+size-1)/size,
	}

	return deliveries, &meta, nil
}

func ListDeliveriesByUserID(userID string, page, size int) (*dtos.PagedDeliveries, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	var total int
	queryCount := `SELECT COUNT(*) FROM deliveries d
		JOIN orders o ON o.order_id = d.order_id
		WHERE o.user_id = ?`
	if err := DB.QueryRow(queryCount, userID).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * size
	query := `SELECT d.delivery_id, d.order_id, d.delivery_charge, d.status, d.courier_details
		FROM deliveries d
		JOIN orders o ON o.order_id = d.order_id
		WHERE o.user_id = ?
		LIMIT ? OFFSET ?`
	rows, err := DB.Query(query, userID, size, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []dtos.Delivery
	for rows.Next() {
		var d dtos.Delivery
		if err := rows.Scan(&d.DeliveryID, &d.OrderID, &d.DeliveryCharge, &d.Status, &d.CourierDetails); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}

	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: (total + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page < (total+size-1)/size,
	}

	response := &dtos.PagedDeliveries{
		Data: deliveries,
		Meta: meta,
	}

	return response, nil
}

func GetDeliveryByID(deliveryID string) (dtos.Delivery, error) {

	var d dtos.Delivery
	err := DB.QueryRow(`SELECT delivery_id, order_id, delivery_charge, status, courier_details, delivery_address
		FROM deliveries WHERE delivery_id = ?`, deliveryID).
		Scan(&d.DeliveryID, &d.OrderID, &d.DeliveryCharge, &d.Status, &d.CourierDetails, &d.DeliveryAddress)

	if err == sql.ErrNoRows {
		return dtos.Delivery{}, fmt.Errorf("delivery not found")
	} else if err != nil {
		return dtos.Delivery{}, err
	}

	return d, nil
}
func UpdateDelivery(status, deliveryID string) error {
	exists, err := RecordExists("deliveries", "delivery_id = ?", deliveryID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("delivery doesn't exist")
	}
	_, err = DB.Exec(`UPDATE deliveries SET status = ?
		WHERE delivery_id = ?`,
		status, deliveryID)

	if err != nil {
		return err
	}
	return nil
}

func DeleteDelivery(deliveryID string) error {
	exists, err := RecordExists("deliveries", "delivery_id = ?", deliveryID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("delivery not found")
	}
	_, err = DB.Exec(`DELETE FROM deliveries
		WHERE delivery_id = ?`,
		deliveryID)

	if err != nil {
		return err
	}
	return nil
}
