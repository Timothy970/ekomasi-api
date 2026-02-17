package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
)

// SetupProductRoutes configures all product-related routes
func SetupProductRoutes(api *mux.Router) {
	product := api.PathPrefix("/product").Subrouter()
	products := api.PathPrefix("/products").Subrouter()

	// Product listings and search
	products.HandleFunc("", handlers.GetProductsHandler).Methods("GET")
	products.HandleFunc("/search", handlers.SearchProductsHandler).Methods("GET")
	products.HandleFunc("/subcategories/{subcategory_id}", handlers.GetProductsHandlerBySubCategoryID).Methods("GET")
	products.HandleFunc("/related", handlers.GetRelatedProductsHandler).Methods("GET")
	product.HandleFunc("/{product_id}", handlers.GetProductByIDHandler).Methods("GET")

	// Featured products
	products.HandleFunc("/featured", handlers.GetFeatured).Methods("GET")
	products.HandleFunc("/cheap/expensive", handlers.GetExpensiveAndCheapProducts).Methods("GET")

	// Product images
	product.Handle("/upload-images", middleware.AuthenticateToken(http.HandlerFunc(handlers.UploadProductImageHandler))).Methods("POST")
	product.Handle("/upload-images", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateProductImageHandler))).Methods("PATCH")

	// Product bundles
	product.Handle("/add-products/bundle", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddProductsToBundleHandler))).Methods("POST")
	api.HandleFunc("/products/bundles", handlers.GetBundleProductsHandler).Methods("GET")
	api.HandleFunc("/products/bundles/{bundle_id}", handlers.GetBundleByIDProductsHandler).Methods("GET")

	// Product reviews
	products.HandleFunc("/{product_id}/reviews", handlers.GetProductReviews).Methods("GET")
	products.HandleFunc("/{product_id}/reviews/{review_id}", handlers.GetReview).Methods("GET")
	products.Handle("/{product_id}/reviews", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateReview))).Methods("POST")
	api.Handle("/products/{product_id}/reviews/{review_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateReview))).Methods("PATCH")

	// Categories
	products.HandleFunc("/categories", handlers.GetCategoriesHandler).Methods("GET")
	products.Handle("/categories", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateCategoryHandler))).Methods("POST")
	products.HandleFunc("/categories/{id}", handlers.GetCategoryByIDHandler).Methods("GET")
	products.HandleFunc("/categories-products", handlers.GetCategoryProductsHandler).Methods("GET")
	products.HandleFunc("/categories-products/{category_id}", handlers.GetCategoryProductsHandlerByCategoryID).Methods("GET")
	products.Handle("/admin/categories", middleware.AuthenticateToken(http.HandlerFunc(handlers.AdminGetCategoriesHandler))).Methods("GET")

	// Product variants
	products.HandleFunc("/variants-products", handlers.GetVariantProductsHandler).Methods("GET")
	products.HandleFunc("/variants/{variant_id}", handlers.GetVariant).Methods("GET")
	products.HandleFunc("/variants", handlers.ListVariants).Methods("GET")
	products.HandleFunc("/variants/{product_id}", handlers.ListProductVariants).Methods("GET")

	// Deals
	products.HandleFunc("/deals", handlers.GetDealsHandler).Methods("GET")
	products.HandleFunc("/deals/{deal_id}", handlers.GetDealWithProductsHandler).Methods("GET")

	// Product features
	product.HandleFunc("/features/{product_id}", handlers.GetFeaturesByProductHandler).Methods("GET")

	// Bulk upload
	api.Handle("/products/bulk-upload", middleware.AuthenticateToken(http.HandlerFunc(handlers.BulkUploadProductsHandler))).Methods("POST")
	api.Handle("/products/bulk-upload/publish", middleware.AuthenticateToken(http.HandlerFunc(handlers.PublishBulkUploadedProductsHandler))).Methods("POST")
	api.Handle("/products/bulk-upload/{bulk_product_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteBulkUploadProductsHandler))).Methods("DELETE")
	api.Handle("/products/bulk-upload", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetBulkUploadProductsHandler))).Methods("GET")
	api.HandleFunc("/products/sample-csv", handlers.DownloadSampleCSVHandler).Methods("GET")
}
