// Package handlers provides HTTP request handlers for notification management.
// This file contains handlers for creating, listing, updating, and deleting notifications,
// as well as background schedulers for automated notifications (order confirmations, low stock alerts).
// Supports multi-channel delivery: email, SMS, WhatsApp, and push notifications via WebSocket.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"time"
)

// Returns error only for retryable failures (e.g., network issues).
// Permanent failures (e.g., no email) are handled internally and marked as failed.
func processSingleOrder(orderID string, emailType string) error {
	// Fetch order details from database
	order, err := models.GetOrderByID(models.DB, orderID)
	if err != nil {
		// Cannot fetch order - likely permanent failure (bad ID)
		// Skip and don't retry to avoid blocking other orders
		log.Printf("WARNING: Skipping order %s, cannot fetch details: %v", orderID, err)
		return nil
	}

	// Step 1: Get customer email and name
	userEmail, customerName, err := getCustomerDetails(order)
	if err != nil {
		// User ID present but user record missing/corrupt
		log.Printf("WARNING: Could not resolve user details for order %s: %v", orderID, err)
		// Continue processing, userEmail will be empty string
	}

	// Step 2: Validate email exists
	if userEmail == "" {
		log.Printf("INFO: No email found for order %s. Marking as 'failed'.", orderID)
		// Permanent failure - mark as failed so we don't retry
		if err := models.MarkOrderNotificationSent(models.DB, orderID, "failed"); err != nil {
			// Failed to mark as failed - return error to retry status update
			log.Printf("ERROR: Failed to mark order %s as 'failed': %v", orderID, err)
			return fmt.Errorf("marking order %s as failed: %w", orderID, err)
		}
		return nil // Successfully marked as failed, do not retry
	}

	// Step 3: Build email content and send
	subject := "Your Order Update"
	orderDetails := buildEmailData(order, customerName)
	htmlBody := utils.GenerateOrderConfirmationHTML(orderDetails, emailType)

	// Send email to customer
	if err := notification.SendEmail(userEmail, subject, htmlBody); err != nil {
		// Temporary retryable error (e.g., email gateway down)
		// Return error so order remains pending for next retry
		return fmt.Errorf("sending email for order %s: %w", orderID, err)
	}

	// Step 4: Mark notification as sent in database
	if err := models.MarkOrderNotificationSent(models.DB, orderID, "sent"); err != nil {
		// Email sent but failed to update status - return error to retry update
		// Risk of duplicate email, but better than losing track of sent status
		log.Printf("ERROR: Email sent for order %s, but failed to mark as 'sent': %v", orderID, err)
		return fmt.Errorf("marking order %s as sent: %w", orderID, err)
	}

	log.Printf("Successfully sent order notification for order ID %s", orderID)
	return nil
}

// getCustomerDetails extracts customer email and display name from order.
// Handles both registered users and guest checkouts.
// Returns empty email if neither user nor guest details available.
func getCustomerDetails(order *dtos.Order) (email string, name string, err error) {
	if order.UserID != nil {
		// Registered user - fetch from users table
		user, err := models.GetUserByUserID(models.DB, *order.UserID)
		if err != nil {
			// User ID present but user record not found
			return "", "", fmt.Errorf("getting user %s: %w", *order.UserID, err)
		}

		email = user.Email
		if user.FirstName != "" || user.LastName != "" {
			// Construct full name from first and last name
			name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		} else {
			// Fallback if name not available
			name = "Valued Customer"
		}
		return email, name, nil
	}

	if (order.GuestPersonalDetails != dtos.GuestPersonalDetails{}) {
		var email, name string
		// Guest user - use guest checkout details with nil checks
		if order.GuestPersonalDetails.Email != nil {
			email = *order.GuestPersonalDetails.Email
		} else {
			email = ""
		}
		var firstName, lastName string
		if order.GuestPersonalDetails.FirstName != nil {
			firstName = *order.GuestPersonalDetails.FirstName
		} else {
			firstName = ""
		}
		if order.GuestPersonalDetails.LastName != nil {
			lastName = *order.GuestPersonalDetails.LastName
		} else {
			lastName = ""
		}
		name = fmt.Sprintf("%s %s", firstName, lastName)
		return email, name, nil
	}

	// No UserID and no guest details - email will be empty
	return "", "Valued Customer", nil
}

// buildEmailData transforms order model to email DTO with safe pointer handling.
// Maps order items, handles nil pointers for delivery charge and address.
// Returns complete email data structure for template generation.
func buildEmailData(order *dtos.Order, customerName string) dtos.OrderEmailData {
	// Map order items to email notification format
	orderItems := make([]dtos.OrderNotificationItemRequest, len(order.Items))
	for i, product := range order.Items {
		// Create item DTO with product details
		orderItems[i] = dtos.OrderNotificationItemRequest{
			ProductName: product.Name,
			UnitPrice:   product.Price,
			Quantity:    product.StockQuantity,
		}
	}

	// Safely dereference delivery charge pointer to avoid nil panic
	var shippingFee float64
	if order.DeliveryCharge != nil {
		shippingFee = *order.DeliveryCharge
	}

	// Safely dereference delivery address pointer to avoid nil panic
	var deliveryAddress string
	if order.DeliveryAddress != nil {
		deliveryAddress = *order.DeliveryAddress
	}

	// Return complete email data structure
	return dtos.OrderEmailData{
		OrderID:         order.OrderID,
		CustomerName:    customerName,
		OrderDate:       order.CreatedAt.Format("2006-01-02 15:04:05"),
		ShippingFee:     shippingFee,
		Discount:        order.TotalDiscount,
		TotalAmount:     order.TotalAmount,
		DeliveryAddress: deliveryAddress,
		OrderItems:      orderItems,
	}
}

// StartLowStockEmailScheduler initializes background scheduler for low stock alerts.
// Runs at specified interval but only sends emails at configured hour of day.
// Essential for proactive inventory management.
func StartLowStockEmailScheduler(interval time.Duration, hourOfDay int) {
	// Create ticker that checks at specified interval
	ticker := time.NewTicker(interval)
	// Run scheduler in background goroutine
	go func() {
		for range ticker.C {
			log.Println("Low stock email scheduler tick")
			// Only process emails at specified hour (e.g., 9 AM daily)
			currentHour := time.Now().Hour()
			if currentHour == hourOfDay {
				// Time to send low stock alerts
				processLowStockEmails()
			}
		}
	}()
}

// Alternative implementation (commented out): runs on every tick without hour check
// func StartLowStockEmailScheduler(interval time.Duration) {
// 	log.Printf("Low stock scheduler working at interval %v....", interval)
// 	ticker := time.NewTicker(interval)
// 	go func() {
// 		for range ticker.C {
// 			log.Println("Low stock email scheduler tick - processing low stock emails")
// 			processLowStockEmails()
// 		}
// 	}()
// }

// processLowStockEmails checks inventory and sends alerts for low stock products.
// Fetches products below reorder threshold and emails admin team.
// Critical for preventing stockouts and lost sales.
func processLowStockEmails() {
	// Fetch all products with stock below reorder point
	lowStockProducts, err := models.GetLowStockProducts()
	if err != nil {
		// Critical error: cannot fetch inventory data
		log.Printf("CRITICAL: Failed to fetch low stock products: %v", err)
		return
	}
	if len(lowStockProducts) == 0 {
		// No low stock products - no alerts needed
		log.Printf("No low stock products found.")
		return
	}
	log.Printf("Preparing to send low stock alert emails for ...")
	// Prepare email data with store info and low stock products
	storeName := "Ekomasi Store"
	emailData := dtos.LowStockEmailData{
		StoreName: storeName,
		AlertDate: time.Now().Format("2006-01-02"),
		Products:  lowStockProducts,
	}
	// Generate HTML email body from template
	htmlBody := utils.GenerateLowStockAlertHTML(emailData)
	subject := "Low Stock Alert"
	// Send to admin/inventory management team
	emails := []string{
		"timothy.kimani@roamtech.com",
		"mbithe.taabu@roamtech.com",
	}
	for _, email := range emails {
		// Send alert to each admin email
		if err := notification.SendEmail(email, subject, htmlBody); err != nil {
			log.Printf("ERROR: Failed to send low stock email to %s: %v", email, err)
		} else {
			log.Printf("Low stock email sent successfully to %s", email)
		}
	}
}
