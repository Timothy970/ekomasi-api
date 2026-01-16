package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
)

// SetupWishlistRoutes configures all wishlist-related routes
func SetupWishlistRoutes(api *mux.Router) {
	wishlist := api.PathPrefix("/wishlist").Subrouter()

	const productPath = "/product"

	// Wishlist operations
	wishlist.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetAllUserWishList))).Methods("GET")
	wishlist.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetMyWishList))).Methods("GET")
	wishlist.Handle("delete/{wishlist_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteWishList))).Methods("DELETE")

	// Wishlist items
	wishlist.Handle(productPath, middleware.AuthenticateToken(http.HandlerFunc(handlers.AddToWishList))).Methods("POST")
	wishlist.Handle("/product/{product_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.RemoveFromWishList))).Methods("DELETE")

	// Wishlist sharing
	wishlist.Handle("/share", middleware.AuthenticateToken(http.HandlerFunc(handlers.SendWishlistToShare))).Methods("POST")
	wishlist.HandleFunc("/share/{wishlist_id}", handlers.ReceiceWishlistShared).Methods("GET")
}
