package routes

import (
	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
)

// SetupHomeRoutes configures all homepage-related routes
func SetupHomeRoutes(api *mux.Router) {
	home := api.PathPrefix("/home/").Subrouter()

	// Homepage data
	home.HandleFunc("/data", handlers.HomePageData).Methods("GET")
	home.HandleFunc("/sliders", handlers.GetSliderData).Methods("GET")
	home.HandleFunc("/banners", handlers.GetHomeBannersData).Methods("GET")
	home.HandleFunc("/promotions", handlers.GetPromotionsHandler).Methods("GET")
	home.HandleFunc("/promotions/types", handlers.GetPromotionsTypesHandler).Methods("GET")

	// Categories and subcategories
	api.HandleFunc("/admin/categories-subcategories", handlers.GetCategoriesWithSubCategoriesHandler).Methods("GET")
}
