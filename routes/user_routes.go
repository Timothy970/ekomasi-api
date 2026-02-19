package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
)

// SetupUserRoutes configures all user profile-related routes
func SetupUserRoutes(api *mux.Router) {
	user := api.PathPrefix("/user/").Subrouter()

	// User profile
	user.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserDetails))).Methods("GET")
	user.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateUser))).Methods("PATCH")
	user.Handle("/me/verify-update", middleware.AuthenticateToken(http.HandlerFunc(handlers.VerifyUserUpdateHandler))).Methods("PATCH")
	user.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteUser))).Methods("DELETE")

	// User addresses
	user.Handle("/profile/addresses", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateAddress))).Methods("POST")
	user.Handle("/profile/addresses", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserAddress))).Methods("GET")
	user.Handle("/profile/addresses/{address_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateAddress))).Methods("PATCH")
	user.Handle("/profile/addresses/{address_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteAddress))).Methods("DELETE")
	//deactivate user account
	user.Handle("/deactivate", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeactivateMyAccount))).Methods("DELETE")

	// User recommendations
	user.Handle("/products/recommendations", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserBasedProductsRecommendations))).Methods("GET")
}
