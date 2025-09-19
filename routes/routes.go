package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
	"adenzo_backend/payments"
	"adenzo_backend/utils"
)

func SetupRoutes(router *mux.Router) {
	// Authentication routes
	//handles userDecodeTokenHandler
	api := router.PathPrefix("/api/").Subrouter()
	auth := api.PathPrefix("/auth/").Subrouter()
	auth.HandleFunc("/signup", handlers.RegisterHandler).Methods("POST")
	auth.HandleFunc("/whatsapp/signup", handlers.WhatsAppLoginHandler).Methods("POST")
	auth.HandleFunc("/whatsappwehbook/signup", handlers.WhatsAppWebhookHandler).Methods("POST")
	api.HandleFunc("/verify-whatsapp", handlers.VerifyWhatsAppHandler).Methods("GET")
	auth.HandleFunc("/signin", handlers.LoginHandler).Methods("POST")
	auth.HandleFunc("/resend-otp", handlers.ResendOptHandler).Methods("POST")
	auth.Handle("/refresh-token", middleware.AuthenticateRefreshToken(http.HandlerFunc(handlers.RefreshTokenHandler))).Methods("POST")
	auth.HandleFunc("/decode-token", handlers.DecodeTokenHandler).Methods("GET")
	auth.HandleFunc("/verify-otp", handlers.VerifySignupOTPHandler).Methods("POST")
	auth.Handle("/logout", middleware.AuthenticateToken(http.HandlerFunc(handlers.LogoutHandler))).Methods("POST")

	//Home details enpoints
	home := api.PathPrefix("/home/").Subrouter()
	home.HandleFunc("/data", handlers.HomePageData).Methods("GET")
	home.HandleFunc("/sliders", handlers.GetSliderData).Methods("GET")
	home.HandleFunc("/banners", handlers.GetHomeBannersData).Methods("GET")
	// home.HandleFunc("/categories", handlers.GetCategories).Methods("GET")
	home.HandleFunc("/promotions", handlers.GetPromotionsHandler).Methods("GET")
	home.HandleFunc("/promotions/types", handlers.GetPromotionsTypesHandler).Methods("GET")

	// User profile routes (require authentication)
	user := api.PathPrefix("/user/").Subrouter()
	user.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserDetails))).Methods("GET")
	user.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateUser))).Methods("PATCH")
	//add address
	user.Handle("/profile/addresses", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateAddress))).Methods("POST")
	user.Handle("/profile/addresses", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserAddress))).Methods("GET")
	user.Handle("/profile/addresses/{address_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateAddress))).Methods("PATCH")
	user.Handle("/profile/addresses/{address_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteAddress))).Methods("DELETE")

	// Wishlist routes (Customer Endpoints)
	wishlist := api.PathPrefix("/wishlist").Subrouter()

	wishlist.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetAllUserWishList))).Methods("GET")
	// wishlist.Handle("/{wishlist_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetAllUserWishList))).Methods("GET")
	wishlist.Handle("/{wishlist_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteWishList))).Methods("DELETE")
	// wishlist.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateWishList))).Methods("POST")
	wishlist.Handle("/product", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddToWishList))).Methods("POST")
	wishlist.Handle("/product/{wishlist_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.RemoveFromWishList))).Methods("DELETE")
	wishlist.Handle("/share", middleware.AuthenticateToken(http.HandlerFunc(handlers.SendWishlistToShare))).Methods("POST")
	wishlist.Handle("/share/{wishlist_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.ReceiceWishlistShared))).Methods("GET")

	//products api
	product := api.PathPrefix("/product").Subrouter()
	products := api.PathPrefix("/products").Subrouter()
	//Get product all products with their categories
	products.HandleFunc("", handlers.GetProductsHandler).Methods("GET")
	products.HandleFunc("/search", handlers.SearchProductsHandler).Methods("GET")
	products.HandleFunc("/subcategories/{subcategory_id}", handlers.GetProductsHandlerBySubCategoryID).Methods("GET")
	products.HandleFunc("/related", handlers.GetRelatedProductsHandler).Methods("GET")
	product.HandleFunc("/{product_id}", handlers.GetProductByIDHandler).Methods("GET")
	product.HandleFunc("/upload-images", handlers.UploadProductImageHandler).Methods("POST")
	product.Handle("/update/bundle", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateBundleHandler))).Methods("PATCH")
	product.Handle("/delete/bundle", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteBundleHandler))).Methods("DELETE")
	product.Handle("/add-products/bundle", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddProductsToBundleHandler))).Methods("POST")
	// featured products
	products.HandleFunc("/featured", handlers.GetFeatured).Methods("GET")

	//reviews endpoints
	products.HandleFunc("/{product_id}/reviews", handlers.GetReviews).Methods("GET")
	products.HandleFunc("/{product_id}/reviews/{review_id}", handlers.GetReviews).Methods("GET")
	//add a new product review
	products.Handle("/{product_id}/reviews", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateReview))).Methods("POST")

	//categories endpoints
	// categories := api.PathPrefix("/categories").Subrouter()
	//Get a list of categories
	products.HandleFunc("/categories", handlers.GetCategoriesHandler).Methods("GET")
	//Create a category
	products.Handle("/categories", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateCategoryHandler))).Methods("POST")
	products.HandleFunc("/categories/{id}", handlers.GetCategoryByIDHandler).Methods("GET")
	//Get category products
	products.HandleFunc("/categories-products", handlers.GetCategoryProductsHandler).Methods("GET")
	products.HandleFunc("/categories-products/{category_id}", handlers.GetCategoryProductsHandlerByCategoryID).Methods("GET")

	// Cart routes
	cart := api.PathPrefix("/cart").Subrouter()

	cart.HandleFunc("", handlers.CreateCartHandler).Methods("POST")
	cart.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserCartHandler))).Methods("GET")
	cart.HandleFunc("/add", handlers.AddToCartHandler).Methods("POST")
	cart.HandleFunc("/view/{cart_id}", handlers.ViewCartHandler).Methods("GET")
	cart.HandleFunc("/update/{cart_id}", handlers.UpdateCartItemHandler).Methods("PATCH")
	cart.HandleFunc("/remove/{cart_id}", handlers.RemoveFromCartHandler).Methods("DELETE")
	cart.Handle("/apply-coupon", middleware.AuthenticateToken(http.HandlerFunc(handlers.ApplyCouponHandler))).Methods("POST")

	//get shipping fee
	shipping := api.PathPrefix("/shipping").Subrouter()
	shipping.HandleFunc("/fee", handlers.GetShippingCostHandler).Methods("POST")

	//admin routes
	admin := api.PathPrefix("/admin").Subrouter()
	//create a new product
	admin.Handle("/products", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateProductHandler))).Methods("POST")
	//Get product variants
	products.HandleFunc("/variants-products", handlers.GetVariantProductsHandler).Methods("GET")
	//Get variants by ID
	products.HandleFunc("/variants/{variant_id}", handlers.GetVariant).Methods("GET")
	//List variants
	products.HandleFunc("/variants", handlers.ListVariants).Methods("GET")
	//patch variant
	admin.HandleFunc("/products/variants/{variant_id}", handlers.UpdateVariant).Methods("PATCH")
	// **Create Product Variant**
	admin.Handle("/products/variants", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateVariant))).Methods("POST")
	//Delete a variant
	admin.Handle("/products/variants/{variant_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteVariant))).Methods("DELETE")
	// Add a product variant
	admin.Handle("/add-products/variants/{variant_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddProductVariant))).Methods("POST")
	// Remove a product variant
	admin.Handle("/remove-products/variants/{variant_id}/{product_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.RemoveProductVariant))).Methods("DELETE")
	//List Product variants
	products.HandleFunc("/variants/{product_id}", handlers.ListProductVariants).Methods("GET")
	//Moderate a review
	admin.Handle("/products/{product_id}/reviews/{review_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateReview))).Methods("PATCH")
	//Delete a reviews
	admin.Handle("/products/{product_id}/reviews/{review_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteReview))).Methods("DELETE")
	//update a product by id
	admin.Handle("/products/{product_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateProductHandler))).Methods("PATCH")
	admin.Handle("/products/{product_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteProductHandler))).Methods("DELETE")
	admin.HandleFunc("/add-coupon", handlers.AddCoupon).Methods("POST")
	admin.HandleFunc("/view-orders", handlers.ViewOrderAdminHandler).Methods("GET")
	admin.HandleFunc("/update-order", handlers.UpdateOrderStatusHandler).Methods("PATCH")
	admin.Handle("/banners", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddBannerInfo))).Methods("POST")
	admin.Handle("/banners/{banner_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateBannerInfo))).Methods("PATCH")
	admin.Handle("/banners/{banner_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteBannerInfo))).Methods("DELETE")
	//handle bundle CRUD operationsad
	// adminbundles := admin.PathPrefix("/products/bundles/{bundle_id}").Subrouter()
	admin.Handle("/products/bundles", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateBundleHandler))).Methods("POST")
	api.HandleFunc("/products/bundles", handlers.GetBundleProductsHandler).Methods("GET")
	// adminbundles.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetBundleProductsHandler))).Methods("GET")
	// adminbundles.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateBundleHandler))).Methods("PATCH")
	// adminbundles.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteBundleHandler))).Methods("DELETE")
	//add products to a bundle
	// admin.Handle("/products/bundles/products/{bundle_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddProductsToBundleHandler))).Methods("POST")
	admin.Handle("/products/bundles/products/{bundle_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.RemoveProductsFromBundleHandler))).Methods("DELETE")
	//Admin moderate categories
	admin.Handle("/products/categories/{category_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateCategoryHandler))).Methods("PATCH")
	admin.Handle("/products/categories/{category_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteCategoryHandler))).Methods("DELETE")
	//admin user endpoints
	admin.Handle("/users", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddUser))).Methods("POST")
	admin.Handle("/users", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetAllUsers))).Methods("GET")
	adminUser := admin.PathPrefix("/users/{user_id}").Subrouter()
	adminUser.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserByID))).Methods("GET")
	adminUser.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateUserByAdmin))).Methods("PATCH")
	adminUser.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteUserByAdmin))).Methods("DELETE")
	adminUser.Handle("/activate", middleware.AuthenticateToken(http.HandlerFunc(handlers.ActivateUserByAdmin))).Methods("PATCH")
	adminUser.Handle("/de-activate", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeactivateUserByAdmin))).Methods("DELETE")
	//handle promotions
	admin.Handle("/promotions", middleware.AuthenticateToken(http.HandlerFunc(handlers.NewPromotionHandler))).Methods("POST")
	admin.Handle("/promotions/{promotion_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeletePromotionHandler))).Methods("DELETE")
	admin.Handle("/promotions/{promotion_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.EditPromotionHandler))).Methods("PATCH")
	admin.Handle("/add-products/promotions", middleware.AuthenticateToken(http.HandlerFunc(handlers.AttachProductToPromotionHandler))).Methods("POST")
	admin.Handle("/remove-products/promotions", middleware.AuthenticateToken(http.HandlerFunc(handlers.RemoveProductFromPromotionHandler))).Methods("DELETE")
	//create blog
	adminBlog := admin.PathPrefix("/blogs/{blog_id}").Subrouter()
	admin.Handle("/blogs", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateBlogHandler))).Methods("POST")
	adminBlog.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateBlogHandler))).Methods("PATCH")
	adminBlog.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteBlogHandler))).Methods("DELETE")
	// notification routes
	adminNotifications := admin.PathPrefix("/notifications/{notification_id}").Subrouter()
	admin.Handle("/notifications", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateNotificationHandler))).Methods("POST")
	admin.Handle("/notifications", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListNotificationsHandler))).Methods("GET")
	// adminNotifications.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetNotificationHandler))).Methods("GET")
	adminNotifications.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateNotificationHandler))).Methods("PATCH")
	adminNotifications.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteNotificationHandler))).Methods("DELETE")
	//admin handle menu links
	admin.Handle("/menu-links", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateMenuLink))).Methods("POST")
	admin.Handle("/menu-links/{menulink_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateMenuLink))).Methods("PATCH")
	admin.Handle("/menu-links/{menulink_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteMenuLink))).Methods("DELETE")
	//admin handle socials
	admin.Handle("/socials", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateSocialLinkHandler))).Methods("POST")
	admin.Handle("/socials/{social_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateSocialLinkHandler))).Methods("PATCH")
	admin.Handle("/socials/{social_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteSocialLinkHandler))).Methods("DELETE")
	//admin payments
	admin.Handle("/payments", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreatePaymentHandler))).Methods("POST")
	admin.Handle("/payments", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListPaymentsHandler))).Methods("GET")
	adminPayments := admin.PathPrefix("/payments/{payment_id}").Subrouter()
	adminPayments.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetPaymentByIDHandler))).Methods("GET")
	adminPayments.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdatePaymentHandler))).Methods("PATCH")
	adminPayments.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeletePaymentHandler))).Methods("DELETE")
	//refund
	api.Handle("/request-refund", middleware.AuthenticateToken(http.HandlerFunc(handlers.RequestRefund))).Methods("POST")
	admin.Handle("/process-refund/{refund_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.ProcessRefund))).Methods("PATCH")
	api.HandleFunc("/refunds", handlers.ListRefundsHandler).Methods("GET")
	api.HandleFunc("/refunds/{refund_id}", handlers.GetRefundByIDHandler).Methods("GET")
	api.Handle("user/refunds/{user_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetRefundByUserIDHandler))).Methods("GET")
	//handle vouchers
	vouchers := api.PathPrefix("/vouchers").Subrouter()
	admin.Handle("/vouchers", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateVoucherHandler))).Methods("POST")
	vouchers.HandleFunc("", handlers.ListVouchersHandler).Methods("GET")
	vouchers.HandleFunc("/{voucher_id}", handlers.GetVoucherHandler).Methods("GET")
	adminvoucher := admin.PathPrefix("/vouchers/{voucher_id}").Subrouter()
	adminvoucher.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateVoucherHandler))).Methods("PATCH")
	adminvoucher.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteVoucherHandler))).Methods("DELETE")
	//location rates
	admin.Handle("/locations", middleware.AuthenticateToken(http.HandlerFunc(handlers.StoreShippingRates))).Methods("POST")
	api.HandleFunc("/locations", handlers.ListLocations).Methods("GET")
	api.HandleFunc("/locations/{location_id:[0-9]+}", handlers.GetLocation).Methods("GET")
	adminlocations := admin.PathPrefix("/locations/{location_id:[0-9]+}").Subrouter()
	adminlocations.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateLocation))).Methods("PATCH")
	adminlocations.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteLocation))).Methods("DELETE")
	// Delivery Routes
	admin.Handle("/deliveries", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateDeliveryHandler))).Methods("POST")
	deliveries := api.PathPrefix("/deliveries").Subrouter()
	deliveries.HandleFunc("", handlers.ListDeliveriesHandler).Methods("GET")
	deliveries.Handle("/user/{user_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListUserDeliveriesHandler))).Methods("GET")
	deliveries.HandleFunc("/{delivery_id}", handlers.GetDeliveryHandler).Methods("GET")
	admin.Handle("/deliveries/{delivery_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateDeliveryHandler))).Methods("PATCH")
	admin.Handle("/deliveries/{delivery_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteDeliveryHandler))).Methods("DELETE")
	//delivery feedback
	deliveries.Handle("/feedback", middleware.AuthenticateToken(http.HandlerFunc(handlers.SubmitFeedbackHandler))).Methods("POST")
	deliveries.HandleFunc("/feedback/{delivery_id}", handlers.GetDeliveryFeedbacks).Methods("GET")
	deliveries.Handle("/feedback/{feedback_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteFeedbackHandler))).Methods("DELETE")
	// api.HandleFunc("/feedbacks", handlers.GetDeliveryFeedbacks).Methods("GET")
	deliveries.HandleFunc("/feedback/user/{user_id}", handlers.GetUserDeliveryFeedbacks).Methods("GET")
	//restock notifications
	restockNotifications := api.PathPrefix("/restock-notifications").Subrouter()
	restockNotifications.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.RequestRestockNotification))).Methods("POST")
	restockNotifications.HandleFunc("/{user_id}", handlers.ListUserRestockNotifications).Methods("GET")
	restockNotifications.Handle("/{user_id}/{notification_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.CancelRestockNotification))).Methods("DELETE")
	restockNotifications.Handle("/trigger", middleware.AuthenticateToken(http.HandlerFunc(handlers.TriggerRestockNotifications))).Methods("POST")
	//inventory ebdpoints
	api.Handle("/inventory", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListInventory))).Methods("GET")
	admin.Handle("/inventory", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateInventory))).Methods("POST")
	api.Handle("/inventory/{inventory_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetInventory))).Methods("GET")
	adminInventory := admin.PathPrefix("/inventory/{inventory_id}").Subrouter()
	adminInventory.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateInventory))).Methods("PATCH")
	adminInventory.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteInventory))).Methods("DELETE")
	//purchase orders end points
	admin.Handle("/purchase-orders", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreatePurchaseOrder))).Methods("POST")
	api.HandleFunc("/purchase-orders", handlers.ListPurchaseOrders).Methods("GET")
	api.HandleFunc("/purchase-orders/{po_id}", handlers.GetPurchaseOrder).Methods("GET")
	poAdmin := admin.PathPrefix("/purchase-orders/{po_id}").Subrouter()
	poAdmin.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdatePurchaseOrder))).Methods("PATCH")
	poAdmin.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeletePurchaseOrder))).Methods("DELETE")
	poItems := admin.PathPrefix("/purchase-order-items").Subrouter()
	poItems.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddPurchaseOrderItem))).Methods("POST")
	poItems.Handle("/{po_item_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.RemovePurchaseOrderItem))).Methods("DELETE")
	//warehouses endpoints
	adminWarehouses := admin.PathPrefix("/warehouses").Subrouter()
	warehouses := api.PathPrefix("/warehouses").Subrouter()
	warehouses.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListWarehouses))).Methods("GET")
	adminWarehouses.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateWarehouse))).Methods("POST")
	warehouses.Handle("/{warehouse_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetWarehouse))).Methods("GET")
	adminWarehouses.Handle("/{warehouse_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateWarehouse))).Methods("PATCH")
	api.Handle("/admin/warehouses/{warehouse_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteWarehouse))).Methods("DELETE")
	//
	// Stock Transfers
	adminstockTransfers := admin.PathPrefix("/stock_transfers").Subrouter()
	stockTransfers := api.PathPrefix("/stock_transfers").Subrouter()
	stockTransfers.HandleFunc("", handlers.ListStockTransfers).Methods("GET")
	adminstockTransfers.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateStockTransfer))).Methods("POST")
	stockTransfers.HandleFunc("/{transfer_id}", handlers.GetStockTransfer).Methods("GET")
	adminstockTransfers.Handle("/{transfer_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateStockTransfer))).Methods("PATCH")
	// suppliers endpoints
	suppliers := admin.PathPrefix("/suppliers").Subrouter()
	suppliers.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateSupplier))).Methods("POST")                   // Create supplier
	api.Handle("/suppliers", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListSuppliers))).Methods("GET")                 // List suppliers with pagination
	api.Handle("/suppliers/{supplier_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetSupplierByID))).Methods("GET") // Get supplier details
	suppliers.Handle("/{supplier_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateSupplier))).Methods("PATCH")    // Update supplier
	suppliers.Handle("/{supplier_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteSupplier))).Methods("DELETE")   // Delete supplier
	//featured admin products
	admin.Handle("/products/featured/{product_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddFeaturedProduct))).Methods("POST")
	admin.Handle("/products/featured/{product_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.RemoveFeatured))).Methods("DELETE")
	//accounts and journals endpoints
	accounts := admin.PathPrefix("/accounts").Subrouter()
	entries := admin.PathPrefix("/entries").Subrouter()
	accounts.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateAccount))).Methods("POST")   // Create chart of account
	api.Handle("/accounts", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListAccounts))).Methods("GET") // List charts of accounts with pagination
	api.HandleFunc("/accounts/{account_id}", handlers.GetAccount).Methods("GET")                                  // Get chart of account details
	accounts.Handle("/{account_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateAccount))).Methods("PATCH")
	accounts.Handle("/{account_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteAccount))).Methods("DELETE")
	//create account
	entries.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateEntry))).Methods("POST")
	api.Handle("/entries", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListEntries))).Methods("GET")
	api.HandleFunc("/entries/{entry_id}", handlers.GetEntry).Methods("GET")
	entries.Handle("/{entry_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateEntry))).Methods("PATCH")
	entries.Handle("/{entry_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteEntry))).Methods("DELETE")
	// blogs routes
	api.HandleFunc("/blogs", handlers.ListBlogsHandler).Methods("GET")
	api.HandleFunc("/blogs/{blog_id}", handlers.GetBlogHandler).Methods("GET")
	//payments routes
	payment := api.PathPrefix("/payment").Subrouter()
	payment.HandleFunc("/mpesa/callback", payments.HandleMpesaCallback).Methods("POST")
	payment.HandleFunc("/pay", payments.HandleMpesaPayment).Methods("POST")

	//orders endpointsPro
	order := api.PathPrefix("/order").Subrouter()
	order.HandleFunc("/create", handlers.CreateOrderHandler).Methods("POST")
	order.Handle("/view", middleware.AuthenticateToken(http.HandlerFunc(handlers.ViewOrder))).Methods("GET")
	order.Handle("/list-orders", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListOrders))).Methods("GET")
	order.HandleFunc("/guest-orders/{order_id}/{email}/{phone_number}", handlers.ListGuestOrders).Methods("GET")
	//reports
	// reportsRepo := &repositories.ReportsRepository{DB: db}
	// reportsHandler := &handlers.ReportsHandler{Repo: reportsRepo}

	api.Handle("/reports/balance-sheet", middleware.AuthenticateToken(http.HandlerFunc(handlers.BalanceSheet))).Methods("GET")
	api.Handle("/reports/income-statement", middleware.AuthenticateToken(http.HandlerFunc(handlers.IncomeStatement))).Methods("GET")
	api.Handle("/reports/cash-flow", middleware.AuthenticateToken(http.HandlerFunc(handlers.CashFlow))).Methods("GET")
	api.Handle("/reports/ledger/{account_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.Ledger))).Methods("GET")
	// r.GET("/reports/ledger/:account_id", reportsHandler.Ledger)
	// # Export only asset accounts
	// GET /accounts/export/csv?account_type=Asset

	// # Export accounts starting with 100
	// GET /accounts/export/csv?code_prefix=100

	// # Export journal entries for January 2025 for account 123
	// GET /journal-entries/export/csv?start_date=2025-01-01&end_date=2025-01-31&account_id=123
	// Reports

	//websocket
	websocket := api.PathPrefix("/ws").Subrouter()
	websocket.HandleFunc("", utils.HandleWebSocket)

	// Telemetry example routes
	api.HandleFunc("/telemetry/example", handlers.ExampleHandler).Methods("GET")
	api.HandleFunc("/health", handlers.HealthCheckHandler).Methods("GET")

	// Reports
	reports := api.PathPrefix("/reports").Subrouter()
	reports.HandleFunc("/analytics/cart-abandonment", handlers.CartAbandonmentReport).Methods("GET")
	reports.HandleFunc("/analytics/cart-abandonment/trend", handlers.CartAbandonmentTrendReport).Methods("GET")
	reports.HandleFunc("/analytics/product-performance", handlers.GetProductPerformanceSummary).Methods("GET")
	reports.HandleFunc("/analytics/product-performance/{product_id}", handlers.GetIndividualProductPerformanceSummary).Methods("GET")
	reports.HandleFunc("/analytics/sales-trends", handlers.GetSalesTrendsSummary).Methods("GET")
	reports.HandleFunc("/analytics/sales-trends/trend", handlers.GetSalesTrendsOverTime).Methods("GET")

	reports.HandleFunc("/promotions/effectiveness/{promotion_id}", handlers.GetEffectiveness).Methods("GET")
	reports.HandleFunc("/promotions/comparison/{promotion_id}", handlers.GetComparison).Methods("GET")
	reports.HandleFunc("/promotions/summary", handlers.GetSummary).Methods("GET")

	reports.HandleFunc("/customers/retention", handlers.GetCustomerRetention).Methods("GET")
	reports.HandleFunc("/customers/retention/trend", handlers.GetCustomerRetentionTrends).Methods("GET")
	reports.HandleFunc("/customers/retention/summary", handlers.GetCustomerRetentionSummary).Methods("GET")
	//user based recommended products
	user.Handle("/products/recommendations", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserBasedProductsRecommendations))).Methods("GET")

	reports.HandleFunc("/inventory/turnover", handlers.GetInventoryTurnover).Methods("GET")
	reports.HandleFunc("/inventory/turnover/{product_id}", handlers.GetInventoryTurnoverByProduct).Methods("GET")

	reports.HandleFunc("/customer/segmentation", handlers.GetCustomerSegmentation).Methods("GET")
	reports.HandleFunc("/sales/segmentation", handlers.GetSalesByRegion).Methods("GET")
	api.HandleFunc("/updateimages", handlers.MigrateImageURLs).Methods("GET")
	//add subscription
	api.HandleFunc("/subscribe", handlers.AddSubscriber).Methods("POST")
}
