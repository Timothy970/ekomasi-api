package handlers

import (
	"adenzo_backend/models"
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

	for _, u := range users {
		// 👇 call your notification methods
		// notify.SendEmail(u.Email, "Reminder: Items left in your cart!")
		// notify.SendWhatsApp(u.Phone, "Hey! You still have items in your cart.")
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

	for _, u := range users {
		// 👇 call your notification methods
		// notify.SendEmail(u.Email, "Reminder: Items waiting in your wishlist!")
		// notify.SendWhatsApp(u.Phone, "Your wishlist items are waiting for you 😍")
		// notify.SendAppNotification(u.ID, "Check your wishlist today!")

		log.Printf("Wishlist reminder sent to user %s", u.ID)
	}
}
