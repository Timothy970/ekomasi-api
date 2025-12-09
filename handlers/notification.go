package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/utils"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Delete Blog
// @Summary Create notification
// @Description Create notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications [POST]
func CreateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Notifications")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.Notification](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Notifications") {
		return
	}
	//check if user exists
	exists, err := models.RecordExists("users", "user_id = ?", req.RecipientID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to check if user exists with ID " + req.RecipientID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if !exists {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "User not found with ID " + req.RecipientID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "user not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//check channle
	if req.Channel != "email" && req.Channel != "sms" && req.Channel != "whatsapp" && req.Channel != "push" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Invalid notification type",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Invalid notification type specified",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	err = sendNotification(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to send notification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	req.SentAt = time.Now().Format("2006-01-02 15:04:05")
	if err := models.CreateNotification(*req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to create notification",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("notifications_")
	utils.DeleteCacheByPrefix("notifications_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notification created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Notification created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func sendNotification(req dtos.Notification) error {
	user, err := models.GetUserByUserID(req.RecipientID)
	if err != nil {
		return err
	}
	if (req.Channel == "whatsapp" || req.Channel == "sms") && user.Phone == "" {
		return errors.New("user doesn't have a phone number")
	}
	switch req.Channel {
	case "sms":
		notification.SendSmsMessages(user.Phone, req.Content)
	case "whatsapp":
		//to include template name
		// phoneInt, _ := strconv.Atoi(user.Phone)
		// notification.SendWhatsappMessages(phoneInt, req.Content, "Notification")
	case "email":
		notification.SendEmail(user.Email, "Notification", req.Content)
	case "push":
		//handle websocketing sending
		utils.SendToUser(req.RecipientID, "", "", map[string]interface{}{
			"event":   "Notification",
			"message": "Your payment was successful!",
		})
	}

	return nil
}

// @Summary List all Notifications
// @Description List all notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications [GET]
func ListNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Notifications")
	if !ok {
		return
	}
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyNotification := fmt.Sprintf("notifications_%d_size_%d", page, limit)
	cacheKeyPagination := fmt.Sprintf("notifications_pagination_%d_size_%d", page, limit)
	var notifications []dtos.Notification
	var cachedNotifications []dtos.Notification
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyNotification, &cachedNotifications)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedNotifications == nil {
		var err error
		notifications, pagination, err = models.ListNotifications(page, limit)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Notifications",
					Description: "Failed to list notifications",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	} else {
		notifications = cachedNotifications
		pagination = cachedPagination
	}
	resp := dtos.NotificationListResponse{
		Notifications: notifications,
		Meta:          *pagination,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notifications fetched successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Notifications fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Update Notification
// @Description Update notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications{notification_id} [PATCH]
func UpdateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Notifications")
	if !ok {
		return
	}
	id := mux.Vars(r)["notification_id"]
	req, ok := DecodeRequestBody[dtos.UpdateNotification](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Notifications") {
		return
	}
	if err := models.UpdateNotification(req.Status, id); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to update notification with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("notifications_")
	utils.DeleteCacheByPrefix("notifications_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notification with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Notification updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Delete Notification
// @Description Delete notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications{notification_id} [DELETE]
func DeleteNotificationHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Notifications")
	if !ok {
		return
	}
	id := mux.Vars(r)["notification_id"]
	if err := models.DeleteNotification(id); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Notifications",
				Description: "Failed to delete notification with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("notifications_")
	utils.DeleteCacheByPrefix("notifications_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Notifications",
			Description: "Notification with ID " + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Notification deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary List all Notifications
// @Description List all notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications [GET]
func ListLogsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Users")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	logs, meta, err := models.ListLogs(page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Logs",
				Description: "Failed to list logs",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.LogListResponse{
		Logs: logs,
		Meta: *meta,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Logs",
			Description: "Logs retrieved successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Logs",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func StartOrderNotificationScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			processPendingOrderNotifications()
		}
	}()
}

// processPendingOrderNotifications is the main entry point to process the batch.
func processPendingOrderNotifications() {
	pendingOrders, err := models.GetPendingOrderNotifications()
	if err != nil {
		// Log the error and stop. If we can't get the list, we can't process anything.
		log.Printf("CRITICAL: Failed to fetch pending order notifications: %v", err)
		return
	}

	log.Printf("Processing %d pending order notifications...", len(pendingOrders))

	// Loop through each order ID and process it individually.
	for _, orderID := range pendingOrders {
		// processSingleOrder encapsulates all logic for one order.
		// This makes the main loop clean and easy to read.
		if err := processSingleOrder(orderID, "order_confirmation"); err != nil {
			// This error means a *retryable* failure occurred (e.g., email service down).
			// We log it and continue to the next order, leaving this one 'pending'
			// to be picked up in the next run.
			log.Printf("ERROR: Failed to process notification for order %s (will retry): %v", orderID, err)
		}
	}
}

// processSingleOrder handles all logic for fetching, processing, and sending a notification for one order.
// It returns an error *only* if the operation failed in a way that should be retried (e.g., network error).
// Permanent failures (like "no email") are handled internally and return 'nil' to stop retries.
func processSingleOrder(orderID string, emailType string) error {
	order, err := models.GetOrderByID(orderID)
	if err != nil {
		// If we can't get the order, we can't process it.
		// Log it and 'continue' (by returning nil) so we don't block other orders.
		// This might be a permanent failure (e.g., bad ID), so retrying is not ideal.
		log.Printf("WARNING: Skipping order %s, cannot fetch details: %v", orderID, err)
		return nil
	}

	// 1. Get Customer Details
	userEmail, customerName, err := getCustomerDetails(order)
	if err != nil {
		// This likely means a user ID was present but the user record was missing/corrupt.
		log.Printf("WARNING: Could not resolve user details for order %s: %v", orderID, err)
		// We'll proceed, but userEmail will be ""
	}

	// 2. Validate Email
	if userEmail == "" {
		log.Printf("INFO: No email found for order %s. Marking as 'failed'.", orderID)
		// This is a permanent failure. Mark as 'failed' so we don't retry.
		// CRITICAL BUG FIX: This was 'return' in your code, which would stop the whole batch.
		// It should be 'continue', which in this refactor means we update the status
		// and return 'nil' to signify we are done with this order.
		if err := models.MarkOrderNotificationSent(orderID, "failed"); err != nil {
			// If we fail to even *mark* it as failed, we have a problem.
			// Return this error so it gets retried.
			log.Printf("ERROR: Failed to mark order %s as 'failed': %v", orderID, err)
			return fmt.Errorf("marking order %s as failed: %w", orderID, err)
		}
		return nil // Successfully marked as 'failed', do not retry.
	}

	// 3. Build and Send Email
	subject := "Your Order Update"
	orderDetails := buildEmailData(order, customerName)
	htmlBody := utils.GenerateOrderConfirmationHTML(orderDetails, emailType)

	// IMPORTANT: Check for an error from SendEmail.
	if err := notification.SendEmail(userEmail, subject, htmlBody); err != nil {
		// This is a temporary, retryable error (e.g., email gateway is down).
		// Return the error so the main loop logs it and *does not* update the status.
		// The order will remain 'pending' and be retried next time.
		return fmt.Errorf("sending email for order %s: %w", orderID, err)
	}

	// 4. Mark as Sent
	if err := models.MarkOrderNotificationSent(orderID, "sent"); err != nil {
		// The email *was* sent, but we failed to update our DB.
		// This is a tricky state. We should return an error to retry updating the status,
		// even if it risks a duplicate email later (which is better than losing track).
		log.Printf("ERROR: Email sent for order %s, but failed to mark as 'sent': %v", orderID, err)
		return fmt.Errorf("marking order %s as sent: %w", orderID, err)
	}

	log.Printf("Successfully sent order notification for order ID %s", orderID)
	return nil
}

// getCustomerDetails extracts the email and a display name from the order.
func getCustomerDetails(order *dtos.Order) (email string, name string, err error) {
	if order.UserID != nil {
		// Registered user
		user, err := models.GetUserByUserID(*order.UserID)
		if err != nil {
			// UserID was present, but we couldn't fetch the user. This is an error.
			return "", "", fmt.Errorf("getting user %s: %w", *order.UserID, err)
		}

		email = user.Email
		if user.FirstName != "" || user.LastName != "" {
			name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		} else {
			// Use a more generic fallback than the email address for the name.
			name = "Valued Customer"
		}
		return email, name, nil
	}

	if (order.GuestPersonalDetails != dtos.GuestPersonalDetails{}) {
		// Guest user
		email = order.GuestPersonalDetails.Email
		name = fmt.Sprintf("%s %s", order.GuestPersonalDetails.FirstName, order.GuestPersonalDetails.LastName)
		return email, name, nil
	}

	// No UserID and no Guest details. Email will be empty.
	return "", "Valued Customer", nil
}

// buildEmailData maps the order model to the email DTO, handling nil pointers.
func buildEmailData(order *dtos.Order, customerName string) dtos.OrderEmailData {
	// Map order items
	orderItems := make([]dtos.OrderNotificationItemRequest, len(order.Items))
	for i, product := range order.Items {
		// Cleaned up DTO creation (no need for pointer)
		orderItems[i] = dtos.OrderNotificationItemRequest{
			ProductName: product.Name,
			UnitPrice:   product.Price,
			Quantity:    product.StockQuantity,
		}
	}

	// --- Fix Potential Panics ---
	// Safely dereference pointers
	var shippingFee float64
	if order.DeliveryCharge != nil {
		shippingFee = *order.DeliveryCharge
	}

	var deliveryAddress string
	if order.DeliveryAddress != nil {
		deliveryAddress = *order.DeliveryAddress
	}
	// --- End Fix ---

	// Return the struct value directly
	return dtos.OrderEmailData{
		OrderID:         order.OrderID,
		CustomerName:    customerName,
		OrderDate:       order.CreatedAt.Format("2006-01-02 15:04:05"), // Assumes CreatedAt is a time.Time
		ShippingFee:     shippingFee,
		Discount:        order.TotalDiscount,
		TotalAmount:     order.TotalAmount,
		DeliveryAddress: deliveryAddress,
		OrderItems:      orderItems,
	}
}

func StartLowStockEmailScheduler(interval time.Duration, hourOfDay int) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			log.Println("Low stock email scheduler tick")
			currentHour := time.Now().Hour()
			if currentHour == hourOfDay {
				processLowStockEmails()
			}
		}
	}()
}

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

// processLowStockEmails checks for low stock products and sends email notifications.
func processLowStockEmails() {
	lowStockProducts, err := models.GetLowStockProducts()
	if err != nil {
		log.Printf("CRITICAL: Failed to fetch low stock products: %v", err)
		return
	}
	if len(lowStockProducts) == 0 {
		log.Printf("No low stock products found.")
		return
	}
	log.Printf("Preparing to send low stock alert emails for ...")
	storeName := "Adenzo Store"
	emailData := dtos.LowStockEmailData{
		StoreName: storeName,
		AlertDate: time.Now().Format("2006-01-02"),
		Products:  lowStockProducts,
	}
	htmlBody := utils.GenerateLowStockAlertHTML(emailData)
	subject := "Low Stock Alert"
	emails := []string{
		"timothy.kimani@roamtech.com",
		"mbithe.taabu@roamtech.com",
	}
	for _, email := range emails {
		if err := notification.SendEmail(email, subject, htmlBody); err != nil {
			log.Printf("ERROR: Failed to send low stock email to %s: %v", email, err)
		} else {
			log.Printf("Low stock email sent successfully to %s", email)
		}
	}
}
