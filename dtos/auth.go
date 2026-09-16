package dtos

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID           string     `json:"user_id"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone,omitempty"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	Status       string     `json:"status"`
	Permissions  []string   `json:"permissions,omitempty"`
}

type Users struct {
	ID          string        `json:"user_id"`
	FirstName   string        `json:"first_name"`
	LastName    string        `json:"last_name"`
	Email       string        `json:"email"`
	Phone       string        `json:"phone"`
	Role        string        `json:"role"`
	LastLogin   string        `json:"last_login"`
	DateJoined  string        `json:"date_joined"`
	Status      string        `json:"status"`
	UserAddress []UserAddress `json:"user_address"`
	Permissions []string      `json:"permissions,omitempty"`
}
type UserInput struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Phonenumber string `json:"phone_number"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type VerifyOTP struct {
	Email string `json:"email"`
	Phone string `json:"phone_number"`
	OTP   string `json:"otp" validate:"required"`
}
type ResendOTP struct {
	Email string `json:"email"`
	Phone string `json:"phone_number"`
}

type VerifyUserUpdate struct {
	OTP string `json:"otp" validate:"required"`
}

// CustomClaims can be extended as needed
type CustomClaims struct {
	UserID      string   `json:"id"`
	Email       string   `json:"email"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Role        string   `json:"role"`
	Phone       string   `json:"phone_number"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}
type WhatsappLogin struct {
	Phone string `json:"phone_number" validate:"required"`
}

// WhatsAppWebhook represents the incoming webhook structure
type WhatsAppWebhook struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

// Entry represents a single entry in the webhook
type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

// Change represents a change notification
type Change struct {
	Field string `json:"field"`
	Value Value  `json:"value"`
}

// Value contains the actual message data
type Value struct {
	MessagingProduct string    `json:"messaging_product"`
	Metadata         Metadata  `json:"metadata"`
	Contacts         []Contact `json:"contacts"`
	Messages         []Message `json:"messages"`
}

// Metadata contains phone number information
type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

// Contact contains sender information
type Contact struct {
	Profile ProfileName `json:"profile"`
	WaID    string      `json:"wa_id"`
}

// Profile contains user profile data
type ProfileName struct {
	Name string `json:"name"`
}

// Message represents an individual message
type Message struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      Text   `json:"text,omitempty"`
	// Add other message type fields (image, video, etc.) as needed
}

// Text contains the message text content
type Text struct {
	Body string `json:"body"`
}
