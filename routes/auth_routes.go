package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
)

// SetupAuthRoutes configures all authentication-related routes
func SetupAuthRoutes(api *mux.Router) {
	auth := api.PathPrefix("/auth/").Subrouter()

	// User authentication
	auth.HandleFunc("/signup", handlers.RegisterHandler).Methods("POST")
	auth.HandleFunc("/whatsapp/signup", handlers.WhatsAppLoginHandler).Methods("POST")
	auth.HandleFunc("/whatsappwehbook/signup", handlers.WhatsAppWebhookHandler).Methods("POST")
	api.HandleFunc("/verify-whatsapp", handlers.VerifyWhatsAppHandler).Methods("GET")
	auth.HandleFunc("/signin", handlers.LoginHandler).Methods("POST")
	auth.HandleFunc("/admin/signin", handlers.AdminLoginHandler).Methods("POST")

	// OTP management
	auth.HandleFunc("/resend-otp", handlers.ResendOptHandler).Methods("POST")
	auth.HandleFunc("/verify-otp", handlers.VerifySignupOTPHandler).Methods("POST")

	// Token management
	auth.Handle("/refresh-token", middleware.AuthenticateRefreshToken(http.HandlerFunc(handlers.RefreshTokenHandler))).Methods("POST")
	auth.HandleFunc("/decode-token", handlers.DecodeTokenHandler).Methods("GET")
	auth.Handle("/logout", middleware.AuthenticateToken(http.HandlerFunc(handlers.LogoutHandler))).Methods("POST")
}
