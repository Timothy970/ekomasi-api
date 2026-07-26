package routes

import (
	"github.com/gin-gonic/gin"
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"
)

// SetupUserGinRoutes configures all user profile-related routes using native Gin router groups
func SetupUserGinRoutes(api *gin.RouterGroup) {
	user := api.Group("/user")

	// User profile
	user.GET("/me", middleware.GinAuthenticateToken(), handlers.GetUserDetails)
	user.PATCH("/me", middleware.GinAuthenticateToken(), handlers.UpdateUser)
	user.PATCH("/me/verify-update", middleware.GinAuthenticateToken(), handlers.VerifyUserUpdateHandler)
	user.DELETE("/me", middleware.GinAuthenticateToken(), handlers.DeleteUser)

	// User addresses
	user.POST("/profile/addresses", middleware.GinAuthenticateToken(), handlers.CreateAddress)
	user.GET("/profile/addresses", middleware.GinAuthenticateToken(), handlers.GetUserAddress)
	user.PATCH("/profile/addresses/:address_id", middleware.GinAuthenticateToken(), handlers.UpdateAddress)
	user.DELETE("/profile/addresses/:address_id", middleware.GinAuthenticateToken(), handlers.DeleteAddress)

	// Deactivate user account
	user.DELETE("/deactivate", middleware.GinAuthenticateToken(), handlers.DeactivateMyAccount)

	// User recommendations
	user.GET("/products/recommendations", middleware.GinAuthenticateToken(), handlers.GetUserBasedProductsRecommendations)
}

// SetupUserRoutes configures all user profile-related routes
