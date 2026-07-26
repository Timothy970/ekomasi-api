package routes

import (
	"github.com/gin-gonic/gin"
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"
)

// SetupAdminGinRoutes configures all admin-specific routes using native Gin router groups
func SetupAdminGinRoutes(api *gin.RouterGroup) {
	admin := api.Group("/admin")

	// Admin addresses
	admin.POST("/profile/addresses", middleware.GinAuthenticateToken(), handlers.AdminCreateAddress)
	admin.PATCH("/profile/addresses/:address_id", middleware.GinAuthenticateToken(), handlers.AdminUpdateAddress)
	admin.DELETE("/profile/addresses/:address_id", middleware.GinAuthenticateToken(), handlers.AdminDeleteAddress)

	// Admin product management
	admin.GET("/products/search", handlers.AdminSearchProductsHandler)
	admin.POST("/products", middleware.GinAuthenticateToken(), handlers.CreateProductHandler)
	admin.POST("/products/specifications", middleware.GinAuthenticateToken(), handlers.HandleProductSpecifications)
	admin.PATCH("/products/specifications", middleware.GinAuthenticateToken(), handlers.HandleProductSpecificationsUpdate)
	admin.PATCH("/products/:product_id", middleware.GinAuthenticateToken(), handlers.UpdateProductHandler)
	admin.DELETE("/products/:product_id", middleware.GinAuthenticateToken(), handlers.DeleteProductHandler)

	// Admin product variants
	admin.PATCH("/products/variants/:variant_id", middleware.GinAuthenticateToken(), handlers.UpdateVariant)
	admin.POST("/products/variants", middleware.GinAuthenticateToken(), handlers.CreateVariant)
	admin.DELETE("/products/variants/:variant_id", middleware.GinAuthenticateToken(), handlers.DeleteVariant)
	admin.POST("/add-products/variants/:variant_id", middleware.GinAuthenticateToken(), handlers.AddProductVariant)
	admin.DELETE("/remove-products/variants/:variant_id/:product_id", middleware.GinAuthenticateToken(), handlers.RemoveProductVariant)

	// Admin deals
	admin.PATCH("/deals/:deal_id", middleware.GinAuthenticateToken(), handlers.UpdateDealHandler)
	admin.DELETE("/deals/:deal_id", middleware.GinAuthenticateToken(), handlers.DeleteDealHandler)
	admin.POST("/add-flash-deal", middleware.GinAuthenticateToken(), handlers.CreateDealProductHandler)

	// Admin reviews
	admin.DELETE("/products/:product_id/reviews/:review_id", middleware.GinAuthenticateToken(), handlers.DeleteReview)

	// Admin coupons
	admin.POST("/add-coupon", handlers.AddCoupon)

	// Admin orders
	admin.GET("/view-orders", handlers.ViewOrderAdminHandler)
	admin.PATCH("/update-order", handlers.UpdateOrderStatusHandler)
	admin.GET("/all-orders", middleware.GinAuthenticateToken(), handlers.AdminListOrders)
	admin.GET("/orders/pdf/:order_id", middleware.GinAuthenticateToken(), handlers.DownloadOrderInvoicePDF)
	admin.GET("/all-orders/csv", middleware.GinAuthenticateToken(), handlers.StreamOrdersCSV)
	admin.GET("/all-orders/count", middleware.GinAuthenticateToken(), handlers.GetOrderCountsByStatus)

	// Admin banners
	admin.POST("/banners", middleware.GinAuthenticateToken(), handlers.AddBannerInfo)
	admin.GET("/banners", middleware.GinAuthenticateToken(), handlers.AdminGetSliderData)
	admin.PATCH("/banners/:banner_id", middleware.GinAuthenticateToken(), handlers.UpdateBannerInfo)
	admin.DELETE("/banners/:banner_id", middleware.GinAuthenticateToken(), handlers.DeleteBannerInfo)

	// Admin bundles
	admin.POST("/products/bundles", middleware.GinAuthenticateToken(), handlers.CreateBundleHandler)
	admin.PATCH("/products/bundles/:bundle_id", middleware.GinAuthenticateToken(), handlers.UpdateBundleHandler)
	admin.DELETE("/products/bundles/:bundle_id", middleware.GinAuthenticateToken(), handlers.DeleteBundleHandler)
	admin.DELETE("/products/bundles/products/:bundle_id", middleware.GinAuthenticateToken(), handlers.RemoveProductsFromBundleHandler)

	// Admin categories
	admin.PATCH("/products/categories/:category_id", middleware.GinAuthenticateToken(), handlers.UpdateCategoryHandler)
	admin.DELETE("/products/categories/:category_id", middleware.GinAuthenticateToken(), handlers.DeleteCategoryHandler)

	// Admin users
	admin.GET("/users", middleware.GinAuthenticateToken(), handlers.GetAllUsers)
	admin.POST("/users", middleware.GinAuthenticateToken(), handlers.AddUser)
	admin.GET("/users/csv", middleware.GinAuthenticateToken(), handlers.DownloadUsersCSVHandler)
	admin.GET("/users/:user_id", middleware.GinAuthenticateToken(), handlers.GetUserByID)
	admin.PATCH("/users/:user_id", middleware.GinAuthenticateToken(), handlers.UpdateUserByAdmin)
	admin.DELETE("/users/:user_id", middleware.GinAuthenticateToken(), handlers.DeleteUserByAdmin)
	admin.PATCH("/users/:user_id/activate", middleware.GinAuthenticateToken(), handlers.ActivateUserByAdmin)
	admin.DELETE("/users/:user_id/de-activate", middleware.GinAuthenticateToken(), handlers.DeactivateUserByAdmin)

	// Admin promotions
	admin.POST("/promotions", middleware.GinAuthenticateToken(), handlers.NewPromotionHandler)
	admin.DELETE("/promotions/:promotion_id", middleware.GinAuthenticateToken(), handlers.DeletePromotionHandler)
	admin.PATCH("/promotions/:promotion_id", middleware.GinAuthenticateToken(), handlers.EditPromotionHandler)
	admin.POST("/add-products/promotions", middleware.GinAuthenticateToken(), handlers.AttachProductToPromotionHandler)
	admin.DELETE("/remove-products/promotions", middleware.GinAuthenticateToken(), handlers.RemoveProductFromPromotionHandler)

	// Admin blogs
	admin.POST("/blogs", middleware.GinAuthenticateToken(), handlers.CreateBlogHandler)
	admin.PATCH("/blogs/:blog_id", middleware.GinAuthenticateToken(), handlers.UpdateBlogHandler)
	admin.DELETE("/blogs/:blog_id", middleware.GinAuthenticateToken(), handlers.DeleteBlogHandler)

	// Admin notifications
	admin.POST("/notifications", middleware.GinAuthenticateToken(), handlers.CreateNotificationHandler)
	admin.GET("/notifications", middleware.GinAuthenticateToken(), handlers.ListNotificationsHandler)
	admin.PATCH("/notifications/:notification_id", middleware.GinAuthenticateToken(), handlers.UpdateNotificationHandler)
	admin.DELETE("/notifications/:notification_id", middleware.GinAuthenticateToken(), handlers.DeleteNotificationHandler)

	// Admin menu links & socials
	admin.POST("/menu-links", middleware.GinAuthenticateToken(), handlers.CreateMenuLink)
	admin.PATCH("/menu-links/:menulink_id", middleware.GinAuthenticateToken(), handlers.UpdateMenuLink)
	admin.DELETE("/menu-links/:menulink_id", middleware.GinAuthenticateToken(), handlers.DeleteMenuLink)
	admin.POST("/socials", middleware.GinAuthenticateToken(), handlers.CreateSocialLinkHandler)
	admin.GET("/socials", middleware.GinAuthenticateToken(), handlers.ListSocialLinksHandler)
	admin.PATCH("/socials/:social_id", middleware.GinAuthenticateToken(), handlers.UpdateSocialLinkHandler)
	admin.DELETE("/socials/:social_id", middleware.GinAuthenticateToken(), handlers.DeleteSocialLinkHandler)

	// Admin payments
	admin.POST("/payments", middleware.GinAuthenticateToken(), handlers.CreatePaymentHandler)
	admin.GET("/payments", middleware.GinAuthenticateToken(), handlers.ListPaymentsHandler)
	admin.GET("/payments/:payment_id", middleware.GinAuthenticateToken(), handlers.GetPaymentByIDHandler)
	admin.PATCH("/payments/:payment_id", middleware.GinAuthenticateToken(), handlers.UpdatePaymentHandler)
	admin.DELETE("/payments/:payment_id", middleware.GinAuthenticateToken(), handlers.DeletePaymentHandler)
	admin.PATCH("/process-refund/:refund_id", middleware.GinAuthenticateToken(), handlers.ProcessRefund)

	// Admin vouchers
	admin.POST("/vouchers", middleware.GinAuthenticateToken(), handlers.CreateVoucherDesign)
	admin.PATCH("/vouchers/designs/:voucher_id", middleware.GinAuthenticateToken(), handlers.EditVoucherDesign)
	admin.GET("/vouchers", middleware.GinAuthenticateToken(), handlers.ListVouchersHandler)
	admin.GET("/vouchers/purchases", middleware.GinAuthenticateToken(), handlers.ListVoucherPurchasesHandler)
	admin.GET("/vouchers/purchases/:purchase_id", middleware.GinAuthenticateToken(), handlers.GetVoucherPurchasesHandler)
	admin.DELETE("/vouchers/designs/:voucher_id", middleware.GinAuthenticateToken(), handlers.DeleteVoucherDesign)
	admin.GET("/vouchers/:voucher_id", middleware.GinAuthenticateToken(), handlers.GetVoucherHandler)
	admin.POST("/vouchers/create", middleware.GinAuthenticateToken(), handlers.CreateVoucherHandlerTest)
	admin.PATCH("/vouchers/:voucher_id", middleware.GinAuthenticateToken(), handlers.UpdateVoucherHandler)
	admin.DELETE("/vouchers/:voucher_id", middleware.GinAuthenticateToken(), handlers.DeleteVoucherHandler)

	// Admin inventory & locations & deliveries
	admin.POST("/locations", middleware.GinAuthenticateToken(), handlers.StoreShippingRates)
	admin.PATCH("/locations/:location_id", middleware.GinAuthenticateToken(), handlers.UpdateLocation)
	admin.DELETE("/locations/:location_id", middleware.GinAuthenticateToken(), handlers.DeleteLocation)
	admin.POST("/deliveries", middleware.GinAuthenticateToken(), handlers.CreateDeliveryHandler)
	admin.PATCH("/deliveries/:delivery_id", middleware.GinAuthenticateToken(), handlers.UpdateDeliveryHandler)
	admin.DELETE("/deliveries/:delivery_id", middleware.GinAuthenticateToken(), handlers.DeleteDeliveryHandler)
	admin.POST("/inventory", middleware.GinAuthenticateToken(), handlers.StockEntry)
	admin.PATCH("/inventory/:inventory_id", middleware.GinAuthenticateToken(), handlers.UpdateInventory)
	admin.DELETE("/inventory/:inventory_id", middleware.GinAuthenticateToken(), handlers.DeleteInventory)

	// Admin purchase orders & warehouses & suppliers
	admin.POST("/purchase-orders", middleware.GinAuthenticateToken(), handlers.CreatePurchaseOrder)
	admin.PATCH("/purchase-orders/:po_id", middleware.GinAuthenticateToken(), handlers.UpdatePurchaseOrder)
	admin.DELETE("/purchase-orders/:po_id", middleware.GinAuthenticateToken(), handlers.DeletePurchaseOrder)
	admin.POST("/purchase-order-items", middleware.GinAuthenticateToken(), handlers.AddPurchaseOrderItem)
	admin.DELETE("/purchase-order-items/:po_item_id", middleware.GinAuthenticateToken(), handlers.RemovePurchaseOrderItem)
	admin.POST("/warehouses", middleware.GinAuthenticateToken(), handlers.CreateWarehouse)
	admin.PATCH("/warehouses/:warehouse_id", middleware.GinAuthenticateToken(), handlers.UpdateWarehouse)
	api.DELETE("/admin/warehouses/:warehouse_id", middleware.GinAuthenticateToken(), handlers.DeleteWarehouse)
	admin.POST("/stock_transfers", middleware.GinAuthenticateToken(), handlers.CreateStockTransfer)
	admin.PATCH("/stock_transfers/:transfer_id", middleware.GinAuthenticateToken(), handlers.UpdateStockTransfer)
	admin.POST("/suppliers", middleware.GinAuthenticateToken(), handlers.CreateSupplier)
	admin.PATCH("/suppliers/:supplier_id", middleware.GinAuthenticateToken(), handlers.UpdateSupplier)
	admin.DELETE("/suppliers/:supplier_id", middleware.GinAuthenticateToken(), handlers.DeleteSupplier)

	// Admin accounts & entries & features & promo codes
	admin.POST("/accounts", middleware.GinAuthenticateToken(), handlers.CreateAccount)
	admin.GET("/accounts/stats", middleware.GinAuthenticateToken(), handlers.GetAccountStats)
	admin.PATCH("/accounts/:account_id", middleware.GinAuthenticateToken(), handlers.UpdateAccount)
	admin.DELETE("/accounts/:account_id", middleware.GinAuthenticateToken(), handlers.DeleteAccount)
	admin.POST("/entries", middleware.GinAuthenticateToken(), handlers.CreateEntry)
	admin.PATCH("/entries/:entry_id", middleware.GinAuthenticateToken(), handlers.UpdateEntry)
	admin.DELETE("/entries/:entry_id", middleware.GinAuthenticateToken(), handlers.DeleteEntry)
	admin.POST("/features/:product_id", middleware.GinAuthenticateToken(), handlers.AddProductFeatures)
	admin.PATCH("/features/:feature_id", middleware.GinAuthenticateToken(), handlers.UpdateProductFeatureHandler)
	admin.PATCH("/features/product/:product_id", middleware.GinAuthenticateToken(), handlers.UpdateAllProductFeaturesHandler)
	admin.DELETE("/features/:feature_id", middleware.GinAuthenticateToken(), handlers.DeleteProductFeatureHandler)

	api.POST("/admin/charges", middleware.GinAuthenticateToken(), handlers.AddChargeHandler)
	api.PATCH("/admin/charges/:charge_id", middleware.GinAuthenticateToken(), handlers.UpdateChargeHandler)
	api.DELETE("/admin/charges/:charge_id", middleware.GinAuthenticateToken(), handlers.DeleteChargeHandler)

	api.POST("/admin/promo-codes", middleware.GinAuthenticateToken(), handlers.AddPromoCodeHandler)
	api.GET("/admin/promo-codes", middleware.GinAuthenticateToken(), handlers.GetAllPromoCodesHandler)
	api.GET("/admin/promo-codes/:promo_id", middleware.GinAuthenticateToken(), handlers.GetPromoCodeByIDHandler)
	api.PATCH("/admin/promo-codes/:promo_id", middleware.GinAuthenticateToken(), handlers.UpdatePromoCodeHandler)
	api.DELETE("/admin/promo-codes/:promo_id", middleware.GinAuthenticateToken(), handlers.DeletePromoCodeHandler)

	// Admin roles & permissions & logs & static pages
	admin.GET("/logs/:user_id", middleware.GinAuthenticateToken(), handlers.GetUserLogsByUserID)
	admin.GET("/logs", middleware.GinAuthenticateToken(), handlers.GetUserLogs)
	api.POST("/admin/warranties", middleware.GinAuthenticateToken(), handlers.CreateWarrantType)
	api.PATCH("/admin/warranties/:warranty_type_id", middleware.GinAuthenticateToken(), handlers.UpdateWarrantType)
	api.DELETE("/admin/warranties/:warranty_type_id", middleware.GinAuthenticateToken(), handlers.DeleteWarrantType)
	api.POST("/admin/warranties/product", middleware.GinAuthenticateToken(), handlers.AddProductWarranties)

	admin.POST("/roles", middleware.GinAuthenticateToken(), handlers.CreateRoleHandler)
	admin.GET("/roles", middleware.GinAuthenticateToken(), handlers.GetRolesHandler)
	admin.PATCH("/roles/:role_id", middleware.GinAuthenticateToken(), handlers.UpdateRoleHandler)
	admin.DELETE("/roles/:role_id", middleware.GinAuthenticateToken(), handlers.DeleteRoleHandler)

	admin.GET("/permissions", middleware.GinAuthenticateToken(), handlers.GetPermissionsHandler)
	admin.GET("/permissions/available", middleware.GinAuthenticateToken(), handlers.GetAvailablePermissions)
	admin.POST("/permissions/available", middleware.GinAuthenticateToken(), handlers.AddAvailablePermission)
	admin.PATCH("/permissions/available", middleware.GinAuthenticateToken(), handlers.UpdateAvailablePermission)
	admin.DELETE("/permissions/available", middleware.GinAuthenticateToken(), handlers.RemoveAvailablePermission)

	admin.POST("/static-pages", middleware.GinAuthenticateToken(), handlers.CreateStaticPage)
	admin.PATCH("/static-pages/:static_page_id", middleware.GinAuthenticateToken(), handlers.UpdateStaticPage)
	admin.DELETE("/static-pages/:static_page_id", middleware.GinAuthenticateToken(), handlers.DeleteStaticPage)

	admin.DELETE("/payment-options/:payment_option_id", middleware.GinAuthenticateToken(), handlers.DeletePaymentOptionHandler)
	admin.GET("/payment-options/:payment_option_id", middleware.GinAuthenticateToken(), handlers.GetPaymentOptionByIDHandler)
	admin.GET("/payment-options", middleware.GinAuthenticateToken(), handlers.ListPaymentOptionsHandler)
	admin.PATCH("/payment-options/:payment_option_id", middleware.GinAuthenticateToken(), handlers.UpdatePaymentOptionHandler)
	admin.POST("/payment-options", middleware.GinAuthenticateToken(), handlers.CreatePaymentOptionHandler)

	admin.POST("/promotions/types", middleware.GinAuthenticateToken(), handlers.CreatePromotionsTypesHandler)
	admin.DELETE("/promotions/types/:id", middleware.GinAuthenticateToken(), handlers.DeletePromotionsTypesHandler)
	admin.PATCH("/promotions/types/:id", middleware.GinAuthenticateToken(), handlers.UpdatePromotionsTypesHandler)
	admin.POST("/orders/assign-rider", middleware.GinAuthenticateToken(), handlers.RiderAssignOrder)

	admin.POST("/swagger-ips", middleware.GinAuthenticateToken(), handlers.AddAllowedIPHandler)
	admin.GET("/swagger-ips", middleware.GinAuthenticateToken(), handlers.ListAllowedIPsHandler)
	admin.DELETE("/swagger-ips/:ip", middleware.GinAuthenticateToken(), handlers.DeleteAllowedIPHandler)
	admin.POST("/partners", middleware.GinAuthenticateToken(), handlers.AddPartner)
	admin.DELETE("/partners/:partner_id", middleware.GinAuthenticateToken(), handlers.DeletePartner)
	admin.GET("/subscribers", middleware.GinAuthenticateToken(), handlers.GetAllSubscribersHandler)
	admin.GET("/subscribers/csv", middleware.GinAuthenticateToken(), handlers.DownloadSubscribersCSVHandler)

	// Admin tenants management
	admin.POST("/tenants", middleware.GinAuthenticateToken(), handlers.CreateTenantHandler)
	admin.GET("/tenants", middleware.GinAuthenticateToken(), handlers.GetAllTenantsHandler)
	admin.GET("/tenants/:id", middleware.GinAuthenticateToken(), handlers.GetTenantByIDHandler)
	admin.PATCH("/tenants/:id", middleware.GinAuthenticateToken(), handlers.UpdateTenantHandler)
	admin.DELETE("/tenants/:id", middleware.GinAuthenticateToken(), handlers.DeleteTenantHandler)
}

// SetupAdminRoutes configures all admin-specific routes
