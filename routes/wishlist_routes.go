package routes

import (
	"github.com/gin-gonic/gin"
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"
)

// SetupWishlistGinRoutes configures all wishlist-related routes using native Gin router groups
func SetupWishlistGinRoutes(api *gin.RouterGroup) {
	wishlist := api.Group("/wishlist")
	const productPath = "/product"

	// Wishlist operations
	wishlist.GET("", middleware.GinAuthenticateToken(), handlers.GetAllUserWishList)
	wishlist.GET("/me", middleware.GinAuthenticateToken(), handlers.GetMyWishList)
	wishlist.DELETE("/delete/:wishlist_id", middleware.GinAuthenticateToken(), handlers.DeleteWishList)

	// Wishlist items
	wishlist.POST(productPath, middleware.GinAuthenticateToken(), handlers.AddToWishList)
	wishlist.DELETE("/product/:product_id", middleware.GinAuthenticateToken(), handlers.RemoveFromWishList)

	// Wishlist sharing
	wishlist.POST("/share", middleware.GinAuthenticateToken(), handlers.SendWishlistToShare)
	wishlist.GET("/share/:wishlist_id", handlers.ReceiceWishlistShared)
}

// SetupWishlistRoutes configures all wishlist-related routes
