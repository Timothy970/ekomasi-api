package models

import (
	"context"
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"fmt"
	"strings"
	"time"
)

// GetOrderTrackingTimeline calculates and returns the visual milestone tracking timeline for an order
func GetOrderTrackingTimeline(db DBExecutor, orderID, trackingToken string) (*dtos.OrderTrackingTimelineResponse, error) {
	query := `
		SELECT o.id, COALESCE(o.guest_tracking_token, ''), o.order_status, COALESCE(d.delivery_status, 'PENDING'),
		       COALESCE(o.payment_status, 'PENDING'), o.created_at,
		       COALESCE(u.first_name, o.guest_email, 'Customer'), COALESCE(u.email, o.guest_email, ''),
		       COALESCE(ua.address_line1, 'Standard Address')
		FROM orders o
		LEFT JOIN deliveries d ON o.id = d.order_id
		LEFT JOIN users u ON o.user_id = u.id
		LEFT JOIN user_addresses ua ON u.id = ua.user_id
		WHERE o.id = ?
	`
	if trackingToken != "" {
		query += " AND o.guest_tracking_token = ?"
	}

	var resp dtos.OrderTrackingTimelineResponse
	var token string
	var orderStatus, deliveryStatus, paymentStatus, name, email, address string
	var createdAt time.Time

	var err error
	if trackingToken != "" {
		err = db.QueryRowContext(context.Background(), query, orderID, trackingToken).Scan(
			&resp.OrderID, &token, &orderStatus, &deliveryStatus, &paymentStatus,
			&createdAt, &name, &email, &address,
		)
	} else {
		err = db.QueryRowContext(context.Background(), query, orderID).Scan(
			&resp.OrderID, &token, &orderStatus, &deliveryStatus, &paymentStatus,
			&createdAt, &name, &email, &address,
		)
	}

	if err == sql.ErrNoRows {
		return nil, errors.New("order not found or invalid tracking token")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order tracking data: %w", err)
	}

	resp.TrackingToken = token
	resp.OrderStatus = orderStatus
	resp.DeliveryStatus = deliveryStatus
	resp.PaymentStatus = paymentStatus
	resp.CustomerName = name
	resp.CustomerEmail = email
	resp.DeliveryAddress = address
	resp.CreatedAt = createdAt

	estDelivery := createdAt.Add(72 * time.Hour) // Default 3-day delivery window
	resp.EstimatedDelivery = &estDelivery

	// Build the 5 Visual Tracking Steps
	normOrder := strings.ToLower(orderStatus)
	normDeliv := strings.ToLower(deliveryStatus)

	isPlaced := true
	isConfirmed := normOrder == "confirmed" || normOrder == "processing" || normOrder == "completed" || normOrder == "shipped" || normDeliv == "shipped" || normDeliv == "delivered"
	isShipped := normDeliv == "shipped" || normDeliv == "out_for_delivery" || normDeliv == "delivered" || normOrder == "shipped"
	isOutForDelivery := normDeliv == "out_for_delivery" || normDeliv == "delivered"
	isDelivered := normDeliv == "delivered" || normOrder == "completed"

	currentStep := "placed"
	if isDelivered {
		currentStep = "delivered"
	} else if isOutForDelivery {
		currentStep = "out_for_delivery"
	} else if isShipped {
		currentStep = "shipped"
	} else if isConfirmed {
		currentStep = "processing"
	}

	steps := []dtos.OrderTimelineStep{
		{
			Step:        "placed",
			Title:       "Order Placed",
			Description: "Your order has been received and logged into our system.",
			IsCompleted: isPlaced,
			IsCurrent:   currentStep == "placed",
			CompletedAt: &createdAt,
		},
		{
			Step:        "processing",
			Title:       "Order Confirmed & Processing",
			Description: "Merchant confirmed payment and is preparing items for dispatch.",
			IsCompleted: isConfirmed,
			IsCurrent:   currentStep == "processing",
		},
		{
			Step:        "shipped",
			Title:       "Package Shipped",
			Description: "Items handed over to delivery courier.",
			IsCompleted: isShipped,
			IsCurrent:   currentStep == "shipped",
		},
		{
			Step:        "out_for_delivery",
			Title:       "Out for Delivery",
			Description: "Rider is actively delivering package to your destination address.",
			IsCompleted: isOutForDelivery,
			IsCurrent:   currentStep == "out_for_delivery",
		},
		{
			Step:        "delivered",
			Title:       "Delivered",
			Description: "Package delivered successfully.",
			IsCompleted: isDelivered,
			IsCurrent:   currentStep == "delivered",
		},
	}

	resp.TimelineSteps = steps
	return &resp, nil
}
