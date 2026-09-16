package routes

import (
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupReportsGinRoutes configures all reporting and analytics routes using native Gin router groups
func SetupReportsGinRoutes(api *gin.RouterGroup) {
	reports := api.Group("/reports")

	// Financial reports
	api.GET("/reports/balance-sheet", middleware.GinAuthenticateToken(), handlers.BalanceSheet)
	api.GET("/reports/balance-sheet/csv", middleware.GinAuthenticateToken(), handlers.ExportBalanceSheetCSVHandler)
	api.GET("/reports/income-statement", middleware.GinAuthenticateToken(), handlers.IncomeStatement)
	api.POST("/reports/cash-flow", middleware.GinAuthenticateToken(), handlers.CashFlow)
	api.GET("/reports/ledger/:account_id", middleware.GinAuthenticateToken(), handlers.Ledger)

	// Analytics reports
	reports.GET("/analytics/cart-abandonment", handlers.CartAbandonmentReport)
	reports.GET("/analytics/cart-abandonment/trend", handlers.CartAbandonmentTrendReport)
	reports.GET("/analytics/product-performance", handlers.GetProductPerformanceSummary)
	reports.GET("/analytics/product-performance/:product_id", handlers.GetIndividualProductPerformanceSummary)
	reports.GET("/analytics/sales-trends", handlers.GetSalesTrendsSummary)
	reports.GET("/analytics/sales-trends/trend", handlers.GetSalesTrendsOverTime)

	// Promotion reports
	reports.GET("/promotions/effectiveness/:promotion_id", handlers.GetEffectiveness)
	reports.GET("/promotions/comparison/:promotion_id", handlers.GetComparison)
	reports.GET("/promotions/summary", handlers.GetSummary)

	// Customer reports
	reports.GET("/customers/retention", handlers.GetCustomerRetention)
	reports.GET("/customers/retention/trend", handlers.GetCustomerRetentionTrends)
	reports.GET("/customers/retention/summary", handlers.GetCustomerRetentionSummary)

	// Inventory reports
	reports.GET("/inventory/turnover", handlers.GetInventoryTurnover)
	reports.GET("/inventory/turnover/:product_id", handlers.GetInventoryTurnoverByProduct)

	// Segmentation reports
	reports.GET("/customer/segmentation", handlers.GetCustomerSegmentation)
	reports.GET("/sales/segmentation", handlers.GetSalesByRegion)

	// Sales reports
	reports.GET("/products/top-selling", middleware.GinAuthenticateToken(), handlers.TopSellingProductsReport)
	reports.GET("/sales/overview", middleware.GinAuthenticateToken(), handlers.GetSalesOverview)
	reports.GET("/sales/orders-vs-sales", middleware.GinAuthenticateToken(), handlers.GetSalesVsOrdersPerMonth)
	reports.GET("/sales/revenue-vs-expenses", middleware.GinAuthenticateToken(), handlers.GetRevenueVsExpenses)
	reports.GET("/sales/revenue-customers-orders", middleware.GinAuthenticateToken(), handlers.GetRevenueCustomersOrdersOverview)
}

// SetupReportsRoutes configures all reporting and analytics routes
