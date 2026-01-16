package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"adenzo_backend/handlers"
	"adenzo_backend/middleware"
)

// SetupReportsRoutes configures all reporting and analytics routes
func SetupReportsRoutes(api *mux.Router) {
	reports := api.PathPrefix("/reports").Subrouter()

	// Financial reports
	api.Handle("/reports/balance-sheet", middleware.AuthenticateToken(http.HandlerFunc(handlers.BalanceSheet))).Methods("GET")
	api.Handle("/reports/income-statement", middleware.AuthenticateToken(http.HandlerFunc(handlers.IncomeStatement))).Methods("GET")
	api.Handle("/reports/cash-flow", middleware.AuthenticateToken(http.HandlerFunc(handlers.CashFlow))).Methods("POST")
	api.Handle("/reports/ledger/{account_id}", middleware.AuthenticateToken(http.HandlerFunc(handlers.Ledger))).Methods("GET")

	// Analytics reports
	reports.HandleFunc("/analytics/cart-abandonment", handlers.CartAbandonmentReport).Methods("GET")
	reports.HandleFunc("/analytics/cart-abandonment/trend", handlers.CartAbandonmentTrendReport).Methods("GET")
	reports.HandleFunc("/analytics/product-performance", handlers.GetProductPerformanceSummary).Methods("GET")
	reports.HandleFunc("/analytics/product-performance/{product_id}", handlers.GetIndividualProductPerformanceSummary).Methods("GET")
	reports.HandleFunc("/analytics/sales-trends", handlers.GetSalesTrendsSummary).Methods("GET")
	reports.HandleFunc("/analytics/sales-trends/trend", handlers.GetSalesTrendsOverTime).Methods("GET")

	// Promotion reports
	reports.HandleFunc("/promotions/effectiveness/{promotion_id}", handlers.GetEffectiveness).Methods("GET")
	reports.HandleFunc("/promotions/comparison/{promotion_id}", handlers.GetComparison).Methods("GET")
	reports.HandleFunc("/promotions/summary", handlers.GetSummary).Methods("GET")

	// Customer reports
	reports.HandleFunc("/customers/retention", handlers.GetCustomerRetention).Methods("GET")
	reports.HandleFunc("/customers/retention/trend", handlers.GetCustomerRetentionTrends).Methods("GET")
	reports.HandleFunc("/customers/retention/summary", handlers.GetCustomerRetentionSummary).Methods("GET")

	// Inventory reports
	reports.HandleFunc("/inventory/turnover", handlers.GetInventoryTurnover).Methods("GET")
	reports.HandleFunc("/inventory/turnover/{product_id}", handlers.GetInventoryTurnoverByProduct).Methods("GET")

	// Segmentation reports
	reports.HandleFunc("/customer/segmentation", handlers.GetCustomerSegmentation).Methods("GET")
	reports.HandleFunc("/sales/segmentation", handlers.GetSalesByRegion).Methods("GET")

	// Sales reports
	reports.Handle("/products/top-selling", middleware.AuthenticateToken(http.HandlerFunc(handlers.TopSellingProductsReport))).Methods("GET")
	reports.Handle("/sales/overview", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetSalesOverview))).Methods("GET")
	reports.Handle("/sales/orders-vs-sales", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetSalesVsOrdersPerMonth))).Methods("GET")
	reports.Handle("/sales/revenue-vs-expenses", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetRevenueVsExpenses))).Methods("GET")
	reports.Handle("/sales/revenue-customers-orders", middleware.AuthenticateToken(http.HandlerFunc(handlers.GetRevenueCustomersOrdersOverview))).Methods("GET")
}
