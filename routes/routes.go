package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
	"adenzo_backend/payments"
	"adenzo_backend/utils"
)

const vouchersPath = "/vouchers"
const productPath = "/product"

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
	api.HandleFunc("/admin/categories-subcategories", handlers.GetCategoriesWithSubCategoriesHandler).Methods("GET")
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
	wishlist.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetMyWishList))).Methods("GET")
	// wishlist.Handle("/{wishlist_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetAllUserWishList))).Methods("GET")
	wishlist.Handle("delete/{wishlist_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteWishList))).Methods("DELETE")
	wishlist.Handle(productPath, middleware.AuthenticateToken(http.HandlerFunc(handlers.AddToWishList))).Methods("POST")
	wishlist.Handle("/product/{product_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.RemoveFromWishList))).Methods("DELETE")
	wishlist.Handle("/share", middleware.AuthenticateToken(http.HandlerFunc(handlers.SendWishlistToShare))).Methods("POST")
	wishlist.HandleFunc("/share/{wishlist_id}", handlers.ReceiceWishlistShared).Methods("GET")

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
	//get expensive aand cheapest products
	products.HandleFunc("/cheap/expensive", handlers.GetExpensiveAndCheapProducts).Methods("GET")
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
	//admin get categories
	products.Handle("/admin/categories", middleware.AuthenticateToken(http.HandlerFunc(handlers.AdminGetCategoriesHandler))).Methods("GET")
	// Cart routes
	cart := api.PathPrefix("/cart").Subrouter()

	cart.HandleFunc("", handlers.CreateCartHandler).Methods("POST")
	cart.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserCartHandler))).Methods("GET")
	cart.HandleFunc("/add", handlers.AddToCartHandler).Methods("POST")
	cart.HandleFunc("/view/{cart_id}", handlers.ViewCartHandler).Methods("GET")
	cart.HandleFunc("/update/{cart_id}", handlers.UpdateCartItemHandler).Methods("PATCH")
	cart.HandleFunc("/remove/{cart_id}", handlers.RemoveFromCartHandler).Methods("DELETE")
	cart.HandleFunc("/apply-coupon", handlers.ApplyCouponHandler).Methods("POST")

	//get shipping fee
	shipping := api.PathPrefix("/shipping").Subrouter()
	shipping.HandleFunc("/fee", handlers.GetShippingCostHandler).Methods("POST")

	//admin routes
	admin := api.PathPrefix("/admin").Subrouter()
	admin.HandleFunc("/products/search", handlers.AdminSearchProductsHandler).Methods("GET")
	//create a new product
	admin.Handle("/products", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateProductHandler))).Methods("POST")
	//add product specifications
	admin.Handle("/products/specifications", middleware.AuthenticateToken(http.HandlerFunc(handlers.HandleProductSpecifications))).Methods("POST")
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
	//List Deals
	dealWithID := "/deals/{deal_id}"
	products.HandleFunc("/deals", handlers.GetDealsHandler).Methods("GET")
	products.HandleFunc(dealWithID, handlers.GetDealWithProductsHandler).Methods("GET")
	//update deal
	admin.Handle(dealWithID, middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateDealHandler))).Methods("PATCH")
	//delete deal
	admin.Handle(dealWithID, middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteDealHandler))).Methods("DELETE")
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
	admin.Handle("/add-flash-deal", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateDealProductHandler))).Methods("POST")
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
	vouchers := api.PathPrefix(vouchersPath).Subrouter()
	voucherWithID := "/vouchers/{voucher_id}"
	admin.Handle(vouchersPath, middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateVoucherDesign))).Methods("POST")
	admin.Handle(voucherWithID, middleware.AuthenticateToken(http.HandlerFunc(handlers.EditVoucherDesign))).Methods("PATCH")
	admin.Handle(vouchersPath, middleware.AuthenticateToken(http.HandlerFunc(handlers.ListVouchersHandler))).Methods("GET")
	admin.Handle(voucherWithID, middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteVoucherDesign))).Methods("DELETE")
	admin.Handle(voucherWithID, middleware.AuthenticateToken(http.HandlerFunc(handlers.GetVoucherHandler))).Methods("GET")
	api.HandleFunc("/vouchers/designs", handlers.GetAllVoucherDesigns).Methods("GET")
	//user vouchers
	vouchers.Handle("/me/{voucher_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserVoucherHandler))).Methods("GET")
	vouchers.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListUserVoucherHandler))).Methods("GET")
	adminvoucher := admin.PathPrefix(vouchersPath + "/{voucher_id}").Subrouter()
	vouchers.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListUserVoucherHandler))).Methods("GET")
	adminvoucher.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateVoucherHandler))).Methods("PATCH")
	adminvoucher.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteVoucherHandler))).Methods("DELETE")
	// endpoint for users to buy a voucher
	vouchers.Handle("/buy-voucher", middleware.AuthenticateToken(http.HandlerFunc(handlers.BuyVoucherHandler))).Methods("POST")
	// adminvoucher.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetVoucherHandler))).Methods("GET")
	//endpoint for users to redeem a voucher
	vouchers.Handle("/redeem", middleware.AuthenticateToken(http.HandlerFunc(handlers.RedeemVoucherHandler))).Methods("POST")
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
	admin.Handle("/inventory", middleware.AuthenticateToken(http.HandlerFunc(handlers.StockEntry))).Methods("POST")
	api.Handle("/inventory/{inventory_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetInventory))).Methods("GET")
	api.Handle("/inventory/stock-summary/{inventory_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetInventoryStockSummary))).Methods("GET")
	api.Handle("/inventory/stock-history/{inventory_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetInventoryStockHistory))).Methods("GET")
	adminInventory := admin.PathPrefix("/inventory/{inventory_id}").Subrouter()
	adminInventory.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateInventory))).Methods("PATCH")
	adminInventory.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteInventory))).Methods("DELETE")
	//purchase orders end points
	admin.Handle("/purchase-orders", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreatePurchaseOrder))).Methods("POST")
	api.Handle("/purchase-orders", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListPurchaseOrders))).Methods("GET")
	api.Handle("/purchase-orders/{po_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetPurchaseOrder))).Methods("GET")
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
	// pos view order
	order.HandleFunc("/pos/view", handlers.ViewOrderPOS).Methods("GET")
	order.Handle("/list-orders", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListOrders))).Methods("GET")
	order.HandleFunc("/guest-orders/{order_id}/{email}/{phone_number}", handlers.ListGuestOrders).Methods("GET")
	//admin get all orders
	admin.Handle("/all-orders", middleware.AuthenticateToken(http.HandlerFunc(handlers.AdminListOrders))).Methods("GET")
	//csv export
	admin.Handle("/all-orders/csv", middleware.AuthenticateToken(http.HandlerFunc(handlers.StreamOrdersCSV))).Methods("GET")
	//order count
	admin.Handle("/all-orders/count", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetOrderCountsByStatus))).Methods("GET")
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
	//autocomplete
	api.HandleFunc("/autocomplete", handlers.AutoCompleteHandler).Methods("GET")
	//product features endpoints
	admin.Handle("/features/{product_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddProductFeatures))).Methods("POST")
	product.HandleFunc("/features/{product_id}", handlers.GetFeaturesByProductHandler).Methods("GET")
	admin.Handle("/features/{feature_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateProductFeatureHandler))).Methods("PATCH")
	admin.Handle("/features/{feature_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteProductFeatureHandler))).Methods("DELETE")
	//charges endpoints
	charges := api.PathPrefix("/admin/charges").Subrouter()
	var chargeID = "/{charge_id}"
	charges.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddChargeHandler))).Methods("POST")
	charges.HandleFunc("", handlers.GetAllChargesHandler).Methods("GET")
	charges.HandleFunc(chargeID, handlers.GetChargeByIDHandler).Methods("GET")
	charges.Handle(chargeID, middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateChargeHandler))).Methods("PATCH")
	charges.Handle(chargeID, middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteChargeHandler))).Methods("DELETE")
	// promo codes endpoints
	promocodes := api.PathPrefix("/admin/promo-codes").Subrouter()
	promocodeID := "/{promo_id}"
	promocodes.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.AddPromoCodeHandler))).Methods("POST")
	promocodes.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetAllPromoCodesHandler))).Methods("GET")
	promocodes.Handle(promocodeID, middleware.AuthenticateToken(http.HandlerFunc(handlers.GetPromoCodeByIDHandler))).Methods("GET")
	promocodes.Handle(promocodeID, middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdatePromoCodeHandler))).Methods("PATCH")
	// promocodes.Handle("ativate/{promo_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdatePromoCodeHandler))).Methods("PATCH")
	promocodes.Handle(promocodeID, middleware.AuthenticateToken(http.HandlerFunc(handlers.DeletePromoCodeHandler))).Methods("DELETE")
	//get user logs
	admin.Handle("/logs/{user_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserLogsByUserID))).Methods("GET")
	admin.Handle("/logs", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserLogs))).Methods("GET")

	//warranty endpoints
	warranty := api.PathPrefix("/admin/warranties").Subrouter()
	warranty.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateWarrantType))).Methods("POST")
	warranty.HandleFunc("", handlers.GetAllWarrantyTypes).Methods("GET")
	warranty.Handle("/{warranty_type_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateWarrantType))).Methods("PATCH")
	warranty.Handle("/{warranty_type_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteWarrantType))).Methods("DELETE")
	warranty.Handle(productPath, middleware.AuthenticateToken(http.HandlerFunc(handlers.AddProductWarranties))).Methods("POST")

	// roles endpints
	admin.Handle("/roles", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateRoleHandler))).Methods("POST")
	admin.HandleFunc("/roles", handlers.GetRolesHandler).Methods("GET")
	admin.Handle("/roles/{role_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateRoleHandler))).Methods("PATCH")
	admin.Handle("roles/{role_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteRoleHandler))).Methods("DELETE")
	//permissions endpoints
	admin.Handle("/permissions", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreatePermissionHandler))).Methods("POST")
	admin.HandleFunc("/permissions", handlers.GetPermissionsHandler).Methods("GET")
	admin.Handle("/permissions/{permission_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdatePermissionHandler))).Methods("PATCH")
	admin.Handle("/permissions/{permission_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeletePermissionHandler))).Methods("DELETE")
	var permissionsAvailable = "/permissions/available"
	admin.Handle(permissionsAvailable, middleware.AuthenticateToken(http.HandlerFunc(handlers.GetAvailablePermissions))).Methods("GET")
	admin.Handle(permissionsAvailable, middleware.AuthenticateToken(http.HandlerFunc(handlers.AddAvailablePermission))).Methods("POST")
	admin.Handle(permissionsAvailable, middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateAvailablePermission))).Methods("PATCH")
	admin.Handle(permissionsAvailable, middleware.AuthenticateToken(http.HandlerFunc(handlers.RemoveAvailablePermission))).Methods("DELETE")
	//static pages endpoints
	var staticPageID = "/static-pages/{page_id}"
	admin.Handle("/static-pages", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateStaticPage))).Methods("POST")
	admin.Handle(staticPageID, middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateStaticPage))).Methods("PATCH")
	admin.Handle(staticPageID, middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteStaticPage))).Methods("DELETE")
	api.HandleFunc("/static-pages", handlers.GetStaticPages).Methods("GET")
	api.HandleFunc(staticPageID, handlers.GetStaticPageByID).Methods("GET")
	//endpoint to upload an image
	api.Handle("/upload-image", middleware.AuthenticateToken(http.HandlerFunc(handlers.UploadImageHandler))).Methods("POST")
}
