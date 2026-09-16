package utils

import (
	"fmt"
	"regexp"
)

func IsValidKenyanPhone(phone string) bool {
	// Regex for Kenyan phone numbers (07/01 local, +2547/+2541 international)
	re := regexp.MustCompile(`^(?:\+?2547\d{8}|\+?2541\d{8}|07\d{8}|01\d{8})$`)
	return re.MatchString(phone)
}

// StringPtr returns pointer to string, or nil if string is empty.
//
// Useful for optional database fields that need nil for empty values.
//
// Parameters:
//   - s: string - String value
//
// Returns:
//   - *string: Pointer to string, or nil if empty
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// IntPtr returns pointer to int.
//
// Useful for optional integer fields in structs.
//
// Parameters:
//   - i: int - Integer value
//
// Returns:
//   - *int: Pointer to integer
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns pointer to bool.
//
// Useful for optional boolean fields in structs.
//
// Parameters:
//   - b: bool - Boolean value
//
// Returns:
//   - *bool: Pointer to boolean
func BoolPtr(b bool) *bool {
	return &b
}

// CartReminderEmail generates HTML email for cart abandonment reminders.
//
// Creates responsive email template with:
//   - Purple gradient theme matching brand colors
//   - Prominent call-to-action button
//   - Cart link for completing purchase
//   - Support contact information
//   - Mobile-responsive design
//
// Parameters:
//   - cartLink: string - URL to user's shopping cart
//   - supportEmail: string - Support email address
//   - phone: string - Support phone number
//
// Returns:
//   - string: Complete HTML email body
func CartReminderEmail(cartLink, supportEmail, phone string) string {
	return `
	<!DOCTYPE html>
	<html lang="en">
	<head>
	  <meta charset="UTF-8">
	  <title>Cart Reminder</title>
	  <style>
	    body {
	      font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
	      background-color: #f8f6fb;
	      margin: 0;
	      padding: 0;
	      color: #333;
	    }
	    .container {
	      max-width: 600px;
	      margin: 30px auto;
	      background: #ffffff;
	      border-radius: 16px;
	      box-shadow: 0 4px 12px rgba(150, 120, 200, 0.15);
	      overflow: hidden;
	    }
	    .header {
	      background-color: #d9b3ff;
	      padding: 20px;
	      text-align: center;
	      color: #4a148c;
	      font-size: 24px;
	      font-weight: bold;
	    }
	    .content {
	      padding: 25px;
	      font-size: 16px;
	      line-height: 1.6;
	    }
	    .cta {
	      display: inline-block;
	      background: #b388ff;
	      color: #fff !important;
	      padding: 14px 28px;
	      margin: 20px 0;
	      border-radius: 8px;
	      font-size: 16px;
	      text-decoration: none;
	      font-weight: bold;
	      box-shadow: 0 3px 8px rgba(150, 120, 200, 0.25);
	      transition: background 0.3s ease;
	    }
	    .cta:hover {
	      background: #9c6cff;
	    }
	    .footer {
	      background: #f3e8ff;
	      padding: 15px;
	      text-align: center;
	      font-size: 14px;
	      color: #555;
	      border-top: 1px solid #e1c4ff;
	    }
	  </style>
	</head>
	<body>
	  <div class="container">
	    <div class="header">
	      🛒 You Forgot Something!
	    </div>
	    <div class="content">
	      <p>Hi ,</p>
	      <p>It looks like you left a few things in your shopping cart. We've saved them for you in case you'd like to come back and complete your purchase.</p>
	      <p>Ready to make them yours?</p>
	      <a href="` + cartLink + `" class="cta">Complete Your Order</a>
	      <p>If you have any questions or ran into an issue, don't hesitate to contact our support team at <a href="mailto:` + supportEmail + `">` + supportEmail + `</a> or call us at <strong>` + phone + `</strong>.</p>
	      <p>Thanks,<br>The Ekomasi Team</p>
	    </div>
	    <div class="footer">
	      &copy; 2025 Ekomasi. All rights reserved.
	    </div>
	  </div>
	</body>
	</html>
	`
}

// WishlistReminderEmail generates HTML email for wishlist reminders.
//
// Creates responsive email template with:
//   - Purple gradient theme
//   - Personalized greeting with customer name
//   - Wishlist link with call-to-action
//   - Support contact information
//   - Urgency messaging (items may not be available forever)
//
// Parameters:
//   - customerName: string - Customer's name for personalization
//   - wishlistLink: string - URL to customer's wishlist
//   - supportEmail: string - Support email address
//   - phone: string - Support phone number
//
// Returns:
//   - string: Complete HTML email body
func WishlistReminderEmail(customerName, wishlistLink, supportEmail, phone string) string {
	return `
	<!DOCTYPE html>
	<html lang="en">
	<head>
	  <meta charset="UTF-8">
	  <title>Wishlist Reminder</title>
	  <style>
	    body {
	      font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
	      background-color: #f8f6fb;
	      margin: 0;
	      padding: 0;
	      color: #333;
	    }
	    .container {
	      max-width: 600px;
	      margin: 30px auto;
	      background: #ffffff;
	      border-radius: 16px;
	      box-shadow: 0 4px 12px rgba(150, 120, 200, 0.15);
	      overflow: hidden;
	    }
	    .header {
	      background-color: #d9b3ff;
	      padding: 20px;
	      text-align: center;
	      color: #4a148c;
	      font-size: 24px;
	      font-weight: bold;
	    }
	    .content {
	      padding: 25px;
	      font-size: 16px;
	      line-height: 1.6;
	    }
	    .cta {
	      display: inline-block;
	      background: #b388ff;
	      color: #fff !important;
	      padding: 14px 28px;
	      margin: 20px 0;
	      border-radius: 8px;
	      font-size: 16px;
	      text-decoration: none;
	      font-weight: bold;
	      box-shadow: 0 3px 8px rgba(150, 120, 200, 0.25);
	      transition: background 0.3s ease;
	    }
	    .cta:hover {
	      background: #9c6cff;
	    }
	    .footer {
	      background: #f3e8ff;
	      padding: 15px;
	      text-align: center;
	      font-size: 14px;
	      color: #555;
	      border-top: 1px solid #e1c4ff;
	    }
	  </style>
	</head>
	<body>
	  <div class="container">
	    <div class="header">
	      💜 Your Wishlist Awaits!
	    </div>
	    <div class="content">
	      <p>Hi ` + customerName + `,</p>
	      <p>We noticed you saved some items in your wishlist. They're still waiting for you — and they might not be available forever!</p>
	      <p>Why not treat yourself today?</p>
	      <a href="` + wishlistLink + `" class="cta">View My Wishlist</a>
	      <p>If you need any assistance, reach out to our support team at <a href="mailto:` + supportEmail + `">` + supportEmail + `</a> or call us at <strong>` + phone + `</strong>.</p>
	      <p>Happy Shopping,<br>The Ekomasi Team</p>
	    </div>
	    <div class="footer">
	      &copy; 2025 Ekomasi. All rights reserved.
	    </div>
	  </div>
	</body>
	</html>`
}

// WishlistReminderSMS generates plain text SMS for wishlist reminders.
//
// Creates short, friendly SMS message with:
//   - Personalized greeting
//   - Wishlist link
//   - Emojis for visual appeal
//   - Brand signature
//
// Parameters:
//   - customerName: string - Customer's name
//   - wishlistLink: string - URL to wishlist (shortened recommended for SMS)
//
// Returns:
//   - string: Plain text SMS message
func WishlistReminderSMS(customerName, wishlistLink string) string {
	return fmt.Sprintf(
		"Hi %s, your wishlist is waiting 💜. Don’t miss out on your favorite items! Check it here 👉 %s. – The Ekomasi Team",
		customerName, wishlistLink,
	)
}

// SendVoucherEmail sends an e-voucher email with HTML and plain-text fallback
