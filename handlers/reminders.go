package handlers

import (
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/utils"
	"fmt"
	"log"
	"time"
)

// Start cart reminder scheduler
func StartCartReminderScheduler(interval time.Duration, days int) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			sendCartReminders(days)
		}
	}()
}

// Start wishlist reminder scheduler
func StartWishlistReminderScheduler(interval time.Duration, days int) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			sendWishlistReminders(days)
		}
	}()
}

func sendCartReminders(days int) {
	users, err := models.GetUsersWithStaleCart(days)
	if err != nil {
		log.Printf("error fetching stale cart users: %v", err)
		return
	}
	log.Printf("Found cart users %v", users)
	for _, u := range users {
		// notify.SendEmail
		if u.Email != "" {
			log.Printf("Sending cart reminder to %s", u.Email)
			htmlBody := utils.CartReminderEmail("app.uat.adenzo.co.ke/login", "timothy.kimani@gmial.com", "254746166343")
			notification.SendEmail(u.Email, "Did you forget something?", htmlBody)
		}
		// notify.SendSms
		if u.Phone != "" {
			log.Printf("Sending cart reminder to %s", u.Phone)

			reminder := CartReminderSMS("app.uat.adenzo.co.ke", "gmail@gmial.com", "2324542")
			notification.SendSmsMessages(u.Phone, reminder)

		}
		// notify.SendAppNotification(u.ID, "Don't forget your cart items.")

		log.Printf("Cart reminder sent to user %s", u.ID)
	}
}

func sendWishlistReminders(days int) {
	users, err := models.GetUsersWithStaleWishlist(days)
	if err != nil {
		log.Printf("error fetching stale wishlist users: %v", err)
		return
	}
	log.Printf("Found wishlist users %v", users)
	for _, u := range users {
		// notify.SendEmail
		if u.Email != "" {
			log.Printf("Sending wishlist reminder to %s", u.Email)
			// htmlBody := utils.WishlistReminderEmail("app.uat.adenzo.co.ke/login", "timothy.kimani@gmial.com", "254746166343")
			// notification.SendEmail(u.Email, "Did you forget something?", htmlBody)
		}
		// notify.SendSms
		if u.Phone != "" {
			log.Printf("Sending wishlist reminder to %s", u.Phone)

			// reminder := WishlistReminderSMS("app.uat.adenzo.co.ke")
			// notification.SendSmsMessages(u.Phone, reminder)

		}
		// notify.SendAppNotification(u.ID, "Check your wishlist today!")

		log.Printf("Wishlist reminder sent to user %s", u.ID)
	}
}
func CartReminderSMS(cartLink, supportEmail, phone string) string {
	return fmt.Sprintf(
		"Hi, you left items in your Adenzo cart. Complete your order here %s. Need help? Contact us at %s or %s. – The Adenzo Team",
		cartLink, supportEmail, phone,
	)
}

// Wishlist Reminder SMS (plain text)
func WishlistReminderSMS(wishlistLink string) string {
	return fmt.Sprintf(
		"Hi, your wishlist is waiting . Don’t miss out on your favorite items! Check it here  %s. – The Adenzo Team", wishlistLink,
	)
}
