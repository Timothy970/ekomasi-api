package routes

import (
	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
)

// SetupPaymentRoutes configures all payment-related routes
func SetupPaymentRoutes(api *mux.Router) {
	payment := api.PathPrefix("/payment").Subrouter()

	// M-Pesa payment routes
	payment.HandleFunc("/callback", handlers.HandleMpesaCallback).Methods("POST")
	payment.HandleFunc("/balance", handlers.HandleMpesaBalance).Methods("GET")
	payment.HandleFunc("/return", handlers.HandleMpesaMoneyReturn).Methods("GET")
	payment.HandleFunc("/return/callback", handlers.HandleMpesaReturnCallback).Methods("GET")
	payment.HandleFunc("/balance/callback", handlers.HandleMpesaBalanceCallback).Methods("POST")
	payment.HandleFunc("/pay", handlers.HandleMpesaPayment).Methods("POST")
	payment.HandleFunc("/mpesa/register-url", handlers.RegisterMpesaRoutesHandler).Methods("POST")
}
