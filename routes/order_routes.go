package routes

import (
	"github.com/gin-gonic/gin"
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"
)

// SetupOrderGinRoutes configures all order-related routes using native Gin router groups
func SetupOrderGinRoutes(api *gin.RouterGroup) {
	order := api.Group("/order")

	// Order creation and viewing
	order.POST("/create", handlers.NewCreateOrderHandler)
	order.GET("/view", middleware.GinAuthenticateToken(), handlers.ViewOrder)
	order.GET("/pos/view", handlers.ViewOrderPOS)
	order.GET("/list-orders", middleware.GinAuthenticateToken(), handlers.ListOrders)
	order.GET("/guest-orders/:order_id/:email/:phone_number", handlers.ListGuestOrders)

	// Rider orders
	order.GET("/rider", middleware.GinAuthenticateToken(), handlers.RiderListOrders)
	order.PATCH("/rider/status", middleware.GinAuthenticateToken(), handlers.RiderUpdateOrderStatus)

	// Visual Order Tracking Timeline
	order.GET("/:order_id/tracking", handlers.GetOrderTrackingTimelineHandler)
}
