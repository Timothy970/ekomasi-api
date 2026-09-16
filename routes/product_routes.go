package routes

import (
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupProductGinRoutes configures all product-related routes using native Gin router groups
func SetupProductGinRoutes(api *gin.RouterGroup) {
	product := api.Group("/product")
	products := api.Group("/products")

	// Product listings and search
	products.GET("", handlers.GetProductsHandler)
	products.GET("/search", handlers.SearchProductsHandler)
	products.GET("/subcategories/:subcategory_id", handlers.GetProductsHandlerBySubCategoryID)
	products.GET("/related", handlers.GetRelatedProductsHandler)
	product.GET("/:product_id", handlers.GetProductByIDHandler)

	// Featured products
	products.GET("/featured", handlers.GetFeatured)
	products.GET("/cheap/expensive", handlers.GetExpensiveAndCheapProducts)

	// Product images
	product.POST("/upload-images", middleware.GinAuthenticateToken(), handlers.UploadProductImageHandler)
	product.PATCH("/upload-images", middleware.GinAuthenticateToken(), handlers.UpdateProductImageHandler)

	// Product bundles
	product.POST("/add-products/bundle", middleware.GinAuthenticateToken(), handlers.AddProductsToBundleHandler)
	api.GET("/products/bundles", handlers.GetBundleProductsHandler)
	api.GET("/products/bundles/:bundle_id", handlers.GetBundleByIDProductsHandler)

	// Product reviews
	products.GET("/:product_id/reviews", handlers.GetProductReviews)
	products.GET("/:product_id/reviews/:review_id", handlers.GetReview)
	products.POST("/:product_id/reviews", middleware.GinAuthenticateToken(), handlers.CreateReview)
	api.PATCH("/products/:product_id/reviews/:review_id", middleware.GinAuthenticateToken(), handlers.UpdateReview)

	// Categories
	products.GET("/categories", handlers.GetCategoriesHandler)
	products.POST("/categories", middleware.GinAuthenticateToken(), handlers.CreateCategoryHandler)
	products.GET("/categories/:id", handlers.GetCategoryByIDHandler)
	products.GET("/categories-products", handlers.GetCategoryProductsHandler)
	products.GET("/categories-products/:category_id", handlers.GetCategoryProductsHandlerByCategoryID)
	products.GET("/admin/categories", middleware.GinAuthenticateToken(), handlers.AdminGetCategoriesHandler)

	// Product variants
	products.GET("/variants-products", handlers.GetVariantProductsHandler)
	products.GET("/variants/:variant_id", handlers.GetVariant)
	products.GET("/variants", handlers.ListVariants)
	products.GET("/:product_id/variants", handlers.ListProductVariants)

	// Deals
	products.GET("/deals", handlers.GetDealsHandler)
	products.GET("/deals/:deal_id", handlers.GetDealWithProductsHandler)

	// Product features
	product.GET("/features/:product_id", handlers.GetFeaturesByProductHandler)

	// Bulk upload
	api.POST("/products/bulk-upload", middleware.GinAuthenticateToken(), handlers.BulkUploadProductsHandler)
	api.POST("/products/bulk-upload/publish", middleware.GinAuthenticateToken(), handlers.PublishBulkUploadedProductsHandler)
	api.DELETE("/products/bulk-upload/:bulk_product_id", middleware.GinAuthenticateToken(), handlers.DeleteBulkUploadProductsHandler)
	api.GET("/products/bulk-upload", middleware.GinAuthenticateToken(), handlers.GetBulkUploadProductsHandler)
	api.GET("/products/sample-csv", handlers.DownloadSampleCSVHandler)
}

// SetupProductRoutes configures all product-related routes
