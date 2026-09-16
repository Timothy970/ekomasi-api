package routes

import (
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupCartGinRoutes configures all cart-related routes using native Gin router groups
func SetupCartGinRoutes(api *gin.RouterGroup) {
	cart := api.Group("/cart")

	// Cart operations
	cart.POST("", handlers.CreateCartHandler)
	cart.GET("", middleware.GinAuthenticateToken(), handlers.GetUserCartHandler)
	cart.POST("/add", handlers.AddToCartHandler)
	cart.GET("/view/:cart_id", handlers.ViewCartHandler)
	cart.PATCH("/update/:cart_id", handlers.UpdateCartItemHandler)
	cart.DELETE("/remove/:cart_id", handlers.RemoveFromCartHandler)

	// Cart discount
	api.POST("/cart/apply-discount", handlers.ApplyDiscountHandler)
}

// SetupCartRoutes configures all cart-related routes
