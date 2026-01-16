package routes

import (
	"github.com/gorilla/mux"
)

// SetupRoutes configures all application routes by delegating to modular route setup functions
func SetupRoutes(router *mux.Router) {
	// Create the main API subrouter
	api := router.PathPrefix("/api/").Subrouter()

	// Setup all route groups
	SetupAuthRoutes(api)
	SetupHomeRoutes(api)
	SetupUserRoutes(api)
	SetupWishlistRoutes(api)
	SetupProductRoutes(api)
	SetupCartRoutes(api)
	SetupOrderRoutes(api)
	SetupPaymentRoutes(api)
	SetupAdminRoutes(api)
	SetupReportsRoutes(api)
	SetupMiscRoutes(api)

}
