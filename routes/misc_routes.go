package routes

import (
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupMiscGinRoutes configures miscellaneous routes using native Gin router groups
func SetupMiscGinRoutes(api *gin.RouterGroup) {
	// Shipping
	shipping := api.Group("/shipping")
	shipping.POST("/fee", handlers.GetShippingCostHandler)

	// Refunds
	api.POST("/request-refund", middleware.GinAuthenticateToken(), handlers.RequestRefund)
	api.GET("/refunds", handlers.ListRefundsHandler)
	api.GET("/refunds/:refund_id", handlers.GetRefundByIDHandler)

	// Vouchers (public)
	vouchers := api.Group("/vouchers")
	api.GET("/vouchers/designs", handlers.GetAllVoucherDesigns)
	api.GET("/vouchers/designs/:voucher_id", handlers.GetVoucherDesignByID)
	vouchers.GET("/me/:voucher_id", middleware.GinAuthenticateToken(), handlers.GetUserVoucherHandler)
	vouchers.GET("/me", middleware.GinAuthenticateToken(), handlers.ListUserVoucherHandler)
	vouchers.POST("/buy-voucher", middleware.GinAuthenticateToken(), handlers.BuyVoucherHandler)
	vouchers.PATCH("/buy-voucher/:voucher_id", middleware.GinAuthenticateToken(), handlers.BuyVoucherUpdateHandler)
	vouchers.POST("/redeem", middleware.GinAuthenticateToken(), handlers.RedeemVoucherHandler)

	// Locations
	api.GET("/locations", handlers.ListLocations)
	api.GET("/locations/:location_id", handlers.GetLocation)

	// Deliveries
	deliveries := api.Group("/deliveries")
	deliveries.GET("", handlers.ListDeliveriesHandler)
	deliveries.GET("/user/:user_id", middleware.GinAuthenticateToken(), handlers.ListUserDeliveriesHandler)
	deliveries.GET("/:delivery_id", handlers.GetDeliveryHandler)
	deliveries.POST("/feedback", middleware.GinAuthenticateToken(), handlers.SubmitFeedbackHandler)
	deliveries.GET("/feedback/:delivery_id", handlers.GetDeliveryFeedbacks)
	deliveries.DELETE("/feedback/:feedback_id", middleware.GinAuthenticateToken(), handlers.DeleteFeedbackHandler)
	deliveries.GET("/feedback/user/:user_id", handlers.GetUserDeliveryFeedbacks)

	// Restock notifications
	restockNotifications := api.Group("/restock-notifications")
	restockNotifications.POST("", middleware.GinAuthenticateToken(), handlers.RequestRestockNotification)
	restockNotifications.GET("/:user_id", handlers.ListUserRestockNotifications)
	restockNotifications.DELETE("/:user_id/:notification_id", middleware.GinAuthenticateToken(), handlers.CancelRestockNotification)
	restockNotifications.POST("/trigger", middleware.GinAuthenticateToken(), handlers.TriggerRestockNotifications)

	// Inventory
	api.GET("/inventory", middleware.GinAuthenticateToken(), handlers.ListInventory)
	api.GET("/inventory/:inventory_id", middleware.GinAuthenticateToken(), handlers.GetInventory)
	api.GET("/inventory/csv/:inventory_id", middleware.GinAuthenticateToken(), handlers.DownloadInventoryCSV)
	api.GET("/inventory/pdf/:inventory_id", middleware.GinAuthenticateToken(), handlers.DownloadInventoryPDF)
	api.GET("/inventory/stock-summary/:inventory_id", middleware.GinAuthenticateToken(), handlers.GetInventoryStockSummary)
	api.GET("/inventory/stock-history/:inventory_id", middleware.GinAuthenticateToken(), handlers.GetInventoryStockHistory)

	// Purchase orders & Warehouses & Stock transfers & Suppliers & Accounts & Entries
	api.GET("/purchase-orders", middleware.GinAuthenticateToken(), handlers.ListPurchaseOrders)
	api.GET("/purchase-orders/:po_id", middleware.GinAuthenticateToken(), handlers.GetPurchaseOrder)
	warehouses := api.Group("/warehouses")
	warehouses.GET("", handlers.ListWarehouses)
	warehouses.GET("/:warehouse_id", handlers.GetWarehouse)
	stockTransfers := api.Group("/stock_transfers")
	stockTransfers.GET("", handlers.ListStockTransfers)
	stockTransfers.GET("/:transfer_id", handlers.GetStockTransfer)
	api.GET("/suppliers", middleware.GinAuthenticateToken(), handlers.ListSuppliers)
	api.GET("/suppliers/:supplier_id", middleware.GinAuthenticateToken(), handlers.GetSupplierByID)

	api.GET("/accounts", middleware.GinAuthenticateToken(), handlers.ListAccounts)
	api.GET("/accounts/csv", middleware.GinAuthenticateToken(), handlers.ExportAccountsCSVHandler)
	api.GET("/accounts/next-code", middleware.GinAuthenticateToken(), handlers.GetNextAccountCode)
	api.GET("/accounts/:account_id", handlers.GetAccount)
	api.GET("/entries", middleware.GinAuthenticateToken(), handlers.ListEntries)
	api.GET("/entries/csv", middleware.GinAuthenticateToken(), handlers.ExportJournalEntriesCSVHandler)
	api.GET("/entries/:entry_id", handlers.GetEntry)

	// Blogs & Telemetry & Health & Utilities & POS & Returns & Transactions & Charges & Static Pages & Partners & Vouchers
	api.GET("/blogs", handlers.ListBlogsHandler)
	api.GET("/blogs/:blog_id", handlers.GetBlogHandler)
	api.GET("/telemetry/example", handlers.ExampleHandler)
	api.GET("/health", handlers.HealthCheckHandler)
	api.GET("/updateimages", handlers.MigrateImageURLs)
	api.POST("/subscribe", handlers.AddSubscriber)
	api.GET("/autocomplete", handlers.AutoCompleteHandler)
	api.POST("/upload-image", middleware.GinAuthenticateToken(), handlers.UploadImageHandler2)
	api.POST("/upload-image2", middleware.GinAuthenticateToken(), handlers.UploadImageHandler)

	pos := api.Group("/pos")
	pos.GET("/scan/product", middleware.GinAuthenticateToken(), handlers.ScanProductsHandler)
	pos.POST("/cash/payment", middleware.GinAuthenticateToken(), handlers.ProcessCashPaymentHandler)
	pos.POST("/credit/payment", middleware.GinAuthenticateToken(), handlers.ProcessCreditPaymentHandler)
	pos.POST("/split/payment", middleware.GinAuthenticateToken(), handlers.ProcessSplitPaymentHandler)
	pos.POST("/voucher/payment", middleware.GinAuthenticateToken(), handlers.ProcessVoucherPaymentHandler)
	pos.POST("/print/:order_id", middleware.GinAuthenticateToken(), handlers.PrintReceiptHandler)
	pos.POST("/download/receipt/:order_id", middleware.GinAuthenticateToken(), handlers.DownloadReceiptHandler)
	pos.POST("/hold/order/:order_id", middleware.GinAuthenticateToken(), handlers.HoldOrderHandler)
	pos.GET("/hold/order/:order_id", middleware.GinAuthenticateToken(), handlers.ReleaseOrderHandler)

	returnsPath := api.Group("/returns")
	returnsPath.POST("", middleware.GinAuthenticateToken(), handlers.CreateReturnsHandler)
	returnsPath.GET("/:return_id", middleware.GinAuthenticateToken(), handlers.GetReturnByIDHandler)
	returnsPath.PATCH("/:return_id", middleware.GinAuthenticateToken(), handlers.UpdateReturnStatusHandler)
	returnsPath.GET("", middleware.GinAuthenticateToken(), handlers.ListAllReturnsHandler)
	api.GET("/returns/owner/me", middleware.GinAuthenticateToken(), handlers.ListOwnerReturnsHandler)
	api.GET("/returns/owner/me/:return_id", middleware.GinAuthenticateToken(), handlers.GetOwnerReturnsHandler)

	api.GET("/transactions", middleware.GinAuthenticateToken(), handlers.GetAllTransactionHandler)
	api.GET("/transactions/:transaction_id", middleware.GinAuthenticateToken(), handlers.GetTransactionByIDHandler)
	api.PATCH("/transactions/:transaction_id", middleware.GinAuthenticateToken(), handlers.UpdateTransactionStatusHandler)
	api.POST("/transactions/mpesa/status", middleware.GinAuthenticateToken(), handlers.HandleMpesaTransactionStatus)
	api.POST("/transactions/status/callback", handlers.HandleTransactionStatusCallback)

	charges := api.Group("/admin/charges")
	charges.GET("", handlers.GetAllChargesHandler)
	charges.GET("/:charge_id", handlers.GetChargeByIDHandler)

	warranty := api.Group("/admin/warranties")
	warranty.GET("", handlers.GetAllWarrantyTypes)

	api.GET("/static-pages", handlers.GetStaticPages)
	api.GET("/static-pages/:static_page_id", handlers.GetStaticPageByID)
	api.GET("/partners", handlers.GetAllPartners)
	api.POST("/validate-voucher", handlers.ValidateVoucherCodeHandler)
}

// SetupMiscRoutes configures miscellaneous routes
