// Package models provides data access layer for the Adenzo e-commerce platform.
//
// This file handles delivery management including:
//   - Delivery creation and tracking
//   - Delivery status updates (PENDING, IN_TRANSIT, DELIVERED, etc.)
//   - Delivery listing with pagination (all deliveries or user-specific)
//   - Delivery CRUD operations
//
// Deliveries are linked to orders and track:
//   - Delivery charges/fees
//   - Delivery status throughout fulfillment
//   - Courier details (carrier, tracking number)
//   - Delivery address
package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"

	"github.com/teris-io/shortid"
)

// CreateNewDelivery creates a new delivery record for an order.
//
// This function validates order existence before creating the delivery.
// If no status is provided, defaults to "PENDING".
//
// Parameters:
//   - delivery: dtos.Delivery containing:
//   - OrderID: The order_id to create delivery for (must exist)
//   - DeliveryCharge: Delivery fee/cost
//   - Status: Delivery status (defaults to "PENDING" if empty)
//   - CourierDetails: Courier/carrier information (e.g., tracking number, carrier name)
//   - DeliveryAddress: Shipping address for the delivery
//
// Returns:
//   - error: "order not found" if order doesn't exist, or database error
//
// Status Values:
//   - PENDING: Delivery created but not yet dispatched
//   - IN_TRANSIT: Package is in transit
//   - DELIVERED: Package delivered successfully
//   - FAILED: Delivery failed
//   - RETURNED: Package returned to sender
func CreateNewDelivery(delivery dtos.Delivery) error {
	// Validate order exists before creating delivery
	exists, err := RecordExists(DB, "orders", "order_id = ?", delivery.OrderID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("order not found")
	}

	// Generate unique delivery ID
	deliveryID, _ := shortid.Generate()

	// Default to PENDING status if not provided
	if delivery.Status == "" {
		delivery.Status = "PENDING"
	}

	// Insert new delivery record
	_, err = DB.Exec(`INSERT INTO deliveries (delivery_id, order_id, delivery_charge, status, courier_details, delivery_address)
		VALUES (?, ?, ?, ?, ?, ?)`,
		deliveryID, delivery.OrderID, delivery.DeliveryCharge, delivery.Status, delivery.CourierDetails, delivery.DeliveryAddress)

	if err != nil {
		return err
	}
	return nil
}

// ListDeliveries retrieves paginated list of all deliveries.
//
// This function fetches all deliveries across all orders with pagination support.
// Page and size parameters are validated and defaulted if invalid.
//
// Parameters:
//   - page: Page number (1-based, defaults to 1 if <= 0)
//   - size: Number of deliveries per page (defaults to 10 if <= 0)
//
// Returns:
//   - []dtos.Delivery: Array of deliveries with delivery and courier details
//   - *dtos.PaginationMeta: Pagination metadata (page, size, total, has_prev, has_next)
//   - error: Database error if queries fail
func ListDeliveries(page, size int) ([]dtos.Delivery, *dtos.PaginationMeta, error) {
	// Validate and default page parameter
	if page <= 0 {
		page = 1
	}
	// Validate and default size parameter
	if size <= 0 {
		size = 10
	}

	// Get total count for pagination
	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM deliveries`).Scan(&total); err != nil {
		return nil, nil, err
	}

	// Calculate pagination offset
	offset := (page - 1) * size

	// Query deliveries with pagination
	rows, err := DB.Query(`SELECT delivery_id, order_id, delivery_charge, status, courier_details, delivery_address
		FROM deliveries LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var deliveries []dtos.Delivery
	// Collect delivery records
	for rows.Next() {
		var d dtos.Delivery
		if err := rows.Scan(&d.DeliveryID, &d.OrderID, &d.DeliveryCharge, &d.Status, &d.CourierDetails, &d.DeliveryAddress); err != nil {
			return nil, nil, err
		}
		deliveries = append(deliveries, d)
	}

	// Build pagination metadata
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: (total + size - 1) / size, // Ceiling division
		HasPrev:    page > 1,
		HasNext:    page < (total+size-1)/size,
	}

	return deliveries, &meta, nil
}

// ListDeliveriesByUserID retrieves paginated deliveries for a specific user.
//
// This function fetches deliveries associated with a user's orders by joining
// the deliveries and orders tables. Returns structured response with data and metadata.
//
// Parameters:
//   - userID: The user_id to fetch deliveries for
//   - page: Page number (1-based, defaults to 1 if <= 0)
//   - size: Number of deliveries per page (defaults to 10 if <= 0)
//
// Returns:
//   - *dtos.PagedDeliveries: Structured response containing:
//   - Data: Array of deliveries for this user
//   - Meta: Pagination metadata
//   - error: Database error if queries fail
func ListDeliveriesByUserID(userID string, page, size int) (*dtos.PagedDeliveries, error) {
	// Validate and default page parameter
	if page <= 0 {
		page = 1
	}
	// Validate and default size parameter
	if size <= 0 {
		size = 10
	}

	// Get total count of user's deliveries
	var total int
	queryCount := `SELECT COUNT(*) FROM deliveries d
		JOIN orders o ON o.order_id = d.order_id
		WHERE o.user_id = ?`
	if err := DB.QueryRow(queryCount, userID).Scan(&total); err != nil {
		return nil, err
	}

	// Calculate pagination offset
	offset := (page - 1) * size

	// Query user's deliveries with pagination
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
	// Collect user's delivery records
	for rows.Next() {
		var d dtos.Delivery
		if err := rows.Scan(&d.DeliveryID, &d.OrderID, &d.DeliveryCharge, &d.Status, &d.CourierDetails); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}

	// Build pagination metadata
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: (total + size - 1) / size, // Ceiling division
		HasPrev:    page > 1,
		HasNext:    page < (total+size-1)/size,
	}

	// Build structured response
	response := &dtos.PagedDeliveries{
		Data: deliveries,
		Meta: meta,
	}

	return response, nil
}

// GetDeliveryByID retrieves a single delivery by its delivery_id.
//
// This function fetches complete delivery information including address.
//
// Parameters:
//   - deliveryID: The delivery_id to retrieve
//
// Returns:
//   - dtos.Delivery: Delivery object with all fields including delivery_address
//   - error: "delivery not found" if delivery doesn't exist, or database error
func GetDeliveryByID(deliveryID string) (dtos.Delivery, error) {

	var d dtos.Delivery
	// Query delivery by ID
	err := DB.QueryRow(`SELECT delivery_id, order_id, delivery_charge, status, courier_details, delivery_address
		FROM deliveries WHERE delivery_id = ?`, deliveryID).
		Scan(&d.DeliveryID, &d.OrderID, &d.DeliveryCharge, &d.Status, &d.CourierDetails, &d.DeliveryAddress)

	if err == sql.ErrNoRows {
		// Delivery not found
		return dtos.Delivery{}, fmt.Errorf("delivery not found")
	} else if err != nil {
		return dtos.Delivery{}, err
	}

	return d, nil
}

// UpdateDelivery updates the status of a delivery.
//
// This function validates delivery existence before updating the status.
// Used to track delivery progress (PENDING → IN_TRANSIT → DELIVERED).
//
// Parameters:
//   - status: New delivery status (e.g., "IN_TRANSIT", "DELIVERED", "FAILED", "RETURNED")
//   - deliveryID: The delivery_id to update
//
// Returns:
//   - error: "delivery doesn't exist" if delivery not found, or database error
//
// Common Status Flow:
//   - PENDING → IN_TRANSIT → DELIVERED (successful delivery)
//   - PENDING → IN_TRANSIT → FAILED (delivery failed)
//   - PENDING → IN_TRANSIT → RETURNED (package returned)
func UpdateDelivery(status, deliveryID string) error {
	// Validate delivery exists
	exists, err := RecordExists(DB, "deliveries", "delivery_id = ?", deliveryID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("delivery doesn't exist")
	}

	// Update delivery status
	_, err = DB.Exec(`UPDATE deliveries SET status = ?
		WHERE delivery_id = ?`,
		status, deliveryID)

	if err != nil {
		return err
	}
	return nil
}

// DeleteDelivery removes a delivery record from the database.
//
// This function validates delivery existence before deletion.
//
// Parameters:
//   - deliveryID: The delivery_id to delete
//
// Returns:
//   - error: "delivery not found" if delivery doesn't exist, or database error
//
// Important:
//   - Consider implementing soft delete for audit trail
//   - May want to restrict deletion of delivered items
//   - No cascading effects on orders (order remains)
func DeleteDelivery(deliveryID string) error {
	// Validate delivery exists
	exists, err := RecordExists(DB, "deliveries", "delivery_id = ?", deliveryID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("delivery not found")
	}

	// Delete delivery record
	_, err = DB.Exec(`DELETE FROM deliveries
		WHERE delivery_id = ?`,
		deliveryID)

	if err != nil {
		return err
	}
	return nil
}
