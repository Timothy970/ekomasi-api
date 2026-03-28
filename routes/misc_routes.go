package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
)

// SetupMiscRoutes configures miscellaneous routes
func SetupMiscRoutes(api *mux.Router) {
	// Shipping
	shipping := api.PathPrefix("/shipping").Subrouter()
	shipping.HandleFunc("/fee", handlers.GetShippingCostHandler).Methods("POST")

	// Refunds
	api.Handle("/request-refund", middleware.AuthenticateToken(http.HandlerFunc(handlers.RequestRefund))).Methods("POST")
	api.HandleFunc("/refunds", handlers.ListRefundsHandler).Methods("GET")
	api.HandleFunc("/refunds/{refund_id}", handlers.GetRefundByIDHandler).Methods("GET")

	// Vouchers (public)
	vouchers := api.PathPrefix("/vouchers").Subrouter()
	api.HandleFunc("/vouchers/designs", handlers.GetAllVoucherDesigns).Methods("GET")
	api.HandleFunc("/vouchers/designs/{voucher_id}", handlers.GetVoucherDesignByID).Methods("GET")
	vouchers.Handle("/me/{voucher_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetUserVoucherHandler))).Methods("GET")
	vouchers.Handle("/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListUserVoucherHandler))).Methods("GET")
	vouchers.Handle("/buy-voucher", middleware.AuthenticateToken(http.HandlerFunc(handlers.BuyVoucherHandler))).Methods("POST")
	vouchers.Handle("/buy-voucher/{voucher_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.BuyVoucherUpdateHandler))).Methods("PATCH")
	vouchers.Handle("/redeem", middleware.AuthenticateToken(http.HandlerFunc(handlers.RedeemVoucherHandler))).Methods("POST")

	// Locations
	api.HandleFunc("/locations", handlers.ListLocations).Methods("GET")
	api.HandleFunc("/locations/{location_id:[0-9]+}", handlers.GetLocation).Methods("GET")

	// Deliveries
	deliveries := api.PathPrefix("/deliveries").Subrouter()
	deliveries.HandleFunc("", handlers.ListDeliveriesHandler).Methods("GET")
	deliveries.Handle("/user/{user_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListUserDeliveriesHandler))).Methods("GET")
	deliveries.HandleFunc("/{delivery_id}", handlers.GetDeliveryHandler).Methods("GET")
	deliveries.Handle("/feedback", middleware.AuthenticateToken(http.HandlerFunc(handlers.SubmitFeedbackHandler))).Methods("POST")
	deliveries.HandleFunc("/feedback/{delivery_id}", handlers.GetDeliveryFeedbacks).Methods("GET")
	deliveries.Handle("/feedback/{feedback_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DeleteFeedbackHandler))).Methods("DELETE")
	deliveries.HandleFunc("/feedback/user/{user_id}", handlers.GetUserDeliveryFeedbacks).Methods("GET")

	// Restock notifications
	restockNotifications := api.PathPrefix("/restock-notifications").Subrouter()
	restockNotifications.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.RequestRestockNotification))).Methods("POST")
	restockNotifications.HandleFunc("/{user_id}", handlers.ListUserRestockNotifications).Methods("GET")
	restockNotifications.Handle("/{user_id}/{notification_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.CancelRestockNotification))).Methods("DELETE")
	restockNotifications.Handle("/trigger", middleware.AuthenticateToken(http.HandlerFunc(handlers.TriggerRestockNotifications))).Methods("POST")

	// Inventory
	api.Handle("/inventory", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListInventory))).Methods("GET")
	api.Handle("/inventory/{inventory_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetInventory))).Methods("GET")
	api.Handle("/inventory/csv/{inventory_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DownloadInventoryCSV))).Methods("GET")
	api.Handle("/inventory/pdf/{inventory_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DownloadInventoryPDF))).Methods("GET")
	api.Handle("/inventory/stock-summary/{inventory_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetInventoryStockSummary))).Methods("GET")
	api.Handle("/inventory/stock-history/{inventory_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetInventoryStockHistory))).Methods("GET")

	// Purchase orders
	api.Handle("/purchase-orders", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListPurchaseOrders))).Methods("GET")
	api.Handle("/purchase-orders/{po_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetPurchaseOrder))).Methods("GET")

	// Warehouses
	warehouses := api.PathPrefix("/warehouses").Subrouter()
	warehouses.HandleFunc("", handlers.ListWarehouses).Methods("GET")
	warehouses.HandleFunc("/{warehouse_id}", handlers.GetWarehouse).Methods("GET")

	// Stock transfers
	stockTransfers := api.PathPrefix("/stock_transfers").Subrouter()
	stockTransfers.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListStockTransfers))).Methods("GET")
	stockTransfers.HandleFunc("/{transfer_id}", handlers.GetStockTransfer).Methods("GET")

	// Suppliers
	api.Handle("/suppliers", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListSuppliers))).Methods("GET")
	api.Handle("/suppliers/{supplier_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetSupplierByID))).Methods("GET")

	// Accounts
	api.Handle("/accounts", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListAccounts))).Methods("GET")
	api.Handle("/accounts/csv", middleware.AuthenticateToken(http.HandlerFunc(handlers.ExportAccountsCSVHandler))).Methods("GET")
	api.Handle("/accounts/next-code", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetNextAccountCode))).Methods("GET")
	api.HandleFunc("/accounts/{account_id}", handlers.GetAccount).Methods("GET")

	// Entries
	api.Handle("/entries", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListEntries))).Methods("GET")
	api.Handle("/entries/csv", middleware.AuthenticateToken(http.HandlerFunc(handlers.ExportJournalEntriesCSVHandler))).Methods("GET")
	api.HandleFunc("/entries/{entry_id}", handlers.GetEntry).Methods("GET")

	// Blogs
	api.HandleFunc("/blogs", handlers.ListBlogsHandler).Methods("GET")
	api.HandleFunc("/blogs/{blog_id}", handlers.GetBlogHandler).Methods("GET")

	// Telemetry and health
	api.HandleFunc("/telemetry/example", handlers.ExampleHandler).Methods("GET")
	api.HandleFunc("/health", handlers.HealthCheckHandler).Methods("GET")

	// Utilities
	api.HandleFunc("/updateimages", handlers.MigrateImageURLs).Methods("GET")
	api.HandleFunc("/subscribe", handlers.AddSubscriber).Methods("POST")
	api.HandleFunc("/autocomplete", handlers.AutoCompleteHandler).Methods("GET")

	// Image uploads
	api.Handle("/upload-image", middleware.AuthenticateToken(http.HandlerFunc(handlers.UploadImageHandler2))).Methods("POST")
	api.Handle("/upload-image2", middleware.AuthenticateToken(http.HandlerFunc(handlers.UploadImageHandler))).Methods("POST")

	// POS routes
	pos := api.PathPrefix("/pos/").Subrouter()
	pos.Handle("/scan/product", middleware.AuthenticateToken(http.HandlerFunc(handlers.ScanProductsHandler))).Methods("GET")
	pos.Handle("/cash/payment", middleware.AuthenticateToken(http.HandlerFunc(handlers.ProcessCashPaymentHandler))).Methods("POST")
	pos.Handle("/credit/payment", middleware.AuthenticateToken(http.HandlerFunc(handlers.ProcessCreditPaymentHandler))).Methods("POST")
	pos.Handle("/split/payment", middleware.AuthenticateToken(http.HandlerFunc(handlers.ProcessSplitPaymentHandler))).Methods("POST")
	pos.Handle("/voucher/payment", middleware.AuthenticateToken(http.HandlerFunc(handlers.ProcessVoucherPaymentHandler))).Methods("POST")
	pos.Handle("/print/{order_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.PrintReceiptHandler))).Methods("POST")
	pos.Handle("/download/receipt/{order_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.DownloadReceiptHandler))).Methods("POST")
	pos.Handle("/hold/order/{order_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.HoldOrderHandler))).Methods("POST")
	pos.Handle("/hold/order/{order_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.ReleaseOrderHandler))).Methods("GET")

	// Returns
	returnsPath := api.PathPrefix("/returns").Subrouter()
	returnsPathWithID := api.PathPrefix("/returns/{return_id}").Subrouter()
	returnsPath.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.CreateReturnsHandler))).Methods("POST")
	returnsPathWithID.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetReturnByIDHandler))).Methods("GET")
	returnsPathWithID.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateReturnStatusHandler))).Methods("PATCH")
	returnsPath.Handle("", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListAllReturnsHandler))).Methods("GET")
	api.Handle("/returns/owner/me", middleware.AuthenticateToken(http.HandlerFunc(handlers.ListOwnerReturnsHandler))).Methods("GET")
	api.Handle("/returns/owner/me/{return_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetOwnerReturnsHandler))).Methods("GET")

	// Transactions
	api.Handle("/transactions", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetAllTransactionHandler))).Methods("GET")
	api.Handle("/transactions/{transaction_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetTransactionByIDHandler))).Methods("GET")
	api.Handle("/transactions/{transaction_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.UpdateTransactionStatusHandler))).Methods("PATCH")
	api.Handle("/transactions/mpesa/status", middleware.AuthenticateToken(http.HandlerFunc(handlers.HandleMpesaTransactionStatus))).Methods("POST")
	api.HandleFunc("/transactions/status/callback", handlers.HandleTransactionStatusCallback).Methods("POST")

	// Charges
	charges := api.PathPrefix("/admin/charges").Subrouter()
	charges.HandleFunc("", handlers.GetAllChargesHandler).Methods("GET")
	charges.HandleFunc("/{charge_id}", handlers.GetChargeByIDHandler).Methods("GET")

	// Warranties
	warranty := api.PathPrefix("/admin/warranties").Subrouter()
	warranty.HandleFunc("", handlers.GetAllWarrantyTypes).Methods("GET")

	// Static pages
	api.HandleFunc("/static-pages", handlers.GetStaticPages).Methods("GET")
	api.HandleFunc("/static-pages/{static_page_id}", handlers.GetStaticPageByID).Methods("GET")

	// Partners
	api.HandleFunc("/partners", handlers.GetAllPartners).Methods("GET")

	//validate voucher code
	api.HandleFunc("/validate-voucher", handlers.ValidateVoucherCodeHandler).Methods("POST")
}
