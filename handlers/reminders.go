// Package handlers provides HTTP request handlers and automated reminder services.
// This file contains schedulers and notification functions for sending automated reminders
// to users about abandoned shopping carts and wishlist items.
// It handles email, SMS, and potentially push notifications to improve conversion rates.
package handlers

import (
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/utils"
	"fmt"
	"log"
	"time"
)

// StartCartReminderScheduler initializes and starts an automated scheduler for cart abandonment reminders.
// It runs in the background as a goroutine, periodically checking for users with stale carts
// and sending reminder notifications via email and SMS.
//
// Parameters:
//   - interval: The time interval between reminder checks (e.g., 24 hours)
//   - days: The number of days a cart must be inactive before sending a reminder
//
// The scheduler continues running until the application stops.
func StartCartReminderScheduler(interval time.Duration, days int) {
	// Create a ticker that fires at the specified interval
	ticker := time.NewTicker(interval)
	// Launch a goroutine to handle periodic reminder checks
	go func() {
		// Listen for ticker events and send reminders on each tick
		for range ticker.C {
			sendCartReminders(days)
		}
	}()
}

// StartWishlistReminderScheduler initializes and starts an automated scheduler for wishlist reminders.
// It runs in the background as a goroutine, periodically checking for users with stale wishlists
// and sending reminder notifications to re-engage customers.
//
// Parameters:
//   - interval: The time interval between reminder checks (e.g., 7 days)
//   - days: The number of days a wishlist must be inactive before sending a reminder
//
// The scheduler continues running until the application stops.
func StartWishlistReminderScheduler(interval time.Duration, days int) {
	// Create a ticker that fires at the specified interval
	ticker := time.NewTicker(interval)
	// Launch a goroutine to handle periodic reminder checks
	go func() {
		// Listen for ticker events and send reminders on each tick
		for range ticker.C {
			sendWishlistReminders(days)
		}
	}()
}

// sendCartReminders identifies users with abandoned carts and sends reminder notifications.
// It retrieves users whose carts have been inactive for the specified number of days
// and sends personalized reminders via email and SMS to encourage order completion.
//
// Parameters:
//   - days: The minimum number of days a cart must be inactive to trigger a reminder
//
// The function logs all operations and handles errors gracefully without crashing.
func sendCartReminders(days int) {
	// Fetch users with carts that have been inactive for the specified days
	users, err := models.GetUsersWithStaleCart(days)
	if err != nil {
		// Log error and return gracefully without crashing the scheduler
		log.Printf("error fetching stale cart users: %v", err)
		return
	}
	// Log the number of users found for monitoring
	log.Printf("Found cart users %v", users)

	// Iterate through each user and send appropriate reminders
	for _, u := range users {
		// Send email reminder if user has a valid email address
		if u.Email != "" {
			log.Printf("Sending cart reminder to %s", u.Email)
			// Generate HTML email body with cart recovery link
			htmlBody := utils.CartReminderEmail("app.uat.adenzo.co.ke/login", "timothy.kimani@gmial.com", "254746166343")
			// Send email notification
			notification.SendEmail(u.Email, "Did you forget something?", htmlBody)
		}
		// Send SMS reminder if user has a valid phone number
		if u.Phone != "" {
			log.Printf("Sending cart reminder to %s", u.Phone)
			// Generate SMS message with cart link and support contact
			reminder := CartReminderSMS("app.uat.adenzo.co.ke", "gmail@gmial.com", "2324542")
			// Send SMS notification
			notification.SendSmsMessages(u.Phone, reminder)
		}
		// TODO1: Implement in-app push notification
		// notify.SendAppNotification(u.ID, "Don't forget your cart items.")

		// Log successful reminder delivery
		log.Printf("Cart reminder sent to user %s", u.ID)
	}
}

// sendWishlistReminders identifies users with inactive wishlists and sends reminder notifications.
// It retrieves users whose wishlists have been inactive for the specified number of days
// and sends personalized reminders to re-engage them with their saved items.
//
// Parameters:
//   - days: The minimum number of days a wishlist must be inactive to trigger a reminder
//
// Note: Email and SMS sending are currently commented out and need to be enabled when ready.
func sendWishlistReminders(days int) {
	// Fetch users with wishlists that have been inactive for the specified days
	users, err := models.GetUsersWithStaleWishlist(days)
	if err != nil {
		// Log error and return gracefully without crashing the scheduler
		log.Printf("error fetching stale wishlist users: %v", err)
		return
	}
	// Log the number of users found for monitoring
	log.Printf("Found wishlist users %v", users)

	// Iterate through each user and send appropriate reminders
	for _, u := range users {
		// Send email reminder if user has a valid email address
		if u.Email != "" {
			log.Printf("Sending wishlist reminder to %s", u.Email)
			// TODO1: Uncomment when wishlist email template is ready
			// htmlBody := utils.WishlistReminderEmail("app.uat.adenzo.co.ke/login", "timothy.kimani@gmial.com", "254746166343")
			// notification.SendEmail(u.Email, "Did you forget something?", htmlBody)
		}
		// Send SMS reminder if user has a valid phone number
		if u.Phone != "" {
			log.Printf("Sending wishlist reminder to %s", u.Phone)
			// TODO1: Uncomment when wishlist SMS is ready
			// reminder := WishlistReminderSMS("app.uat.adenzo.co.ke")
			// notification.SendSmsMessages(u.Phone, reminder)
		}
		// TODO1: Implement in-app push notification
		// notify.SendAppNotification(u.ID, "Check your wishlist today!")

		// Log successful reminder delivery
		log.Printf("Wishlist reminder sent to user %s", u.ID)
	}
}

// CartReminderSMS generates a personalized SMS message for cart abandonment reminders.
// The message includes a direct link to the cart and contact information for customer support.
//
// Parameters:
//   - cartLink: URL link to the user's shopping cart
//   - supportEmail: Customer support email address
//   - phone: Customer support phone number
//
// Returns:
//   - A formatted SMS message string ready to be sent
func CartReminderSMS(cartLink, supportEmail, phone string) string {
	// Format and return personalized SMS message with cart link and support contact
	return fmt.Sprintf(
		"Hi, you left items in your Adenzo cart. Complete your order here %s. Need help? Contact us at %s or %s. – The Adenzo Team",
		cartLink, supportEmail, phone,
	)
}

// WishlistReminderSMS generates a personalized SMS message for wishlist reminders.
// The message encourages users to revisit their saved items with a direct link to their wishlist.
//
// Parameters:
//   - wishlistLink: URL link to the user's wishlist page
//
// Returns:
//   - A formatted SMS message string ready to be sent
func WishlistReminderSMS(wishlistLink string) string {
	// Format and return personalized SMS message with wishlist link
	return fmt.Sprintf(
		"Hi, your wishlist is waiting . Don’t miss out on your favorite items! Check it here  %s. – The Adenzo Team", wishlistLink,
	)
}
