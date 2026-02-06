package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
)

// SetupOrderRoutes configures all order-related routes
func SetupOrderRoutes(api *mux.Router) {
	order := api.PathPrefix("/order").Subrouter()

	// Order creation and viewing
	order.HandleFunc("/create", handlers.NewCreateOrderHandler).Methods("POST")
	order.Handle("/view", middleware.AuthenticateToken(http.HandlerFunc(handlers.ViewOrder))).Methods("GET")
	order.HandleFunc("/pos/view", handlers.ViewOrderPOS).Methods("GET")
	order.Handle("/list-orders", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListOrders))).Methods("GET")
	order.HandleFunc("/guest-orders/{order_id}/{email}/{phone_number}", handlers.ListGuestOrders).Methods("GET")
	//get rider orders
	order.Handle("/rider", middleware.AuthenticateToken(http.HandlerFunc(handlers.RiderListOrders))).Methods("GET")
	// Update order status by rider
	order.Handle("/rider/status", middleware.AuthenticateToken(http.HandlerFunc(handlers.RiderUpdateOrderStatus))).Methods("PATCH")
}
