package routes

import (
	"github.com/gin-gonic/gin"
	"ekomasi_backend/handlers"
)

// SetupHomeGinRoutes configures all homepage-related routes using native Gin router groups
func SetupHomeGinRoutes(api *gin.RouterGroup) {
	home := api.Group("/home")

	// Homepage data
	home.GET("/data", handlers.HomePageData)
	home.GET("/sliders", handlers.GetSliderData)
	home.GET("/banners", handlers.GetHomeBannersData)
	home.GET("/promotions", handlers.GetPromotionsHandler)
	home.GET("/promotions/types", handlers.GetPromotionsTypesHandler)

	// Categories and subcategories
	api.GET("/admin/categories-subcategories", handlers.GetCategoriesWithSubCategoriesHandler)
}

// SetupHomeRoutes configures all homepage-related routes
