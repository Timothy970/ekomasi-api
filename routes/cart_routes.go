package routes

import (
	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
	"net/http"
)

// SetupCartRoutes configures all cart-related routes
func SetupCartRoutes(api *mux.Router) {
	cart := api.PathPrefix("/cart").Subrouter()

	// Cart operations
	cart.HandleFunc("", handlers.CreateCartHandler).Methods("POST")
	cart.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserCartHandler))).Methods("GET")
	cart.HandleFunc("/add", handlers.AddToCartHandler).Methods("POST")
	cart.HandleFunc("/view/{cart_id}", handlers.ViewCartHandler).Methods("GET")
	cart.HandleFunc("/update/{cart_id}", handlers.UpdateCartItemHandler).Methods("PATCH")
	cart.HandleFunc("/remove/{cart_id}", handlers.RemoveFromCartHandler).Methods("DELETE")

	// Cart discount
	api.HandleFunc("/cart/apply-discount", handlers.ApplyDiscountHandler).Methods("POST")
}
