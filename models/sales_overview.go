// Package models provides data access functions for the Ekomasi e-commerce backend.
//
// sales.go handles sales analytics and reporting including:
//   - Sales trends and growth analysis
//   - Revenue vs expenses tracking
//   - Customer segmentation by region
//   - Sales overview dashboards
//   - Period-over-period comparisons
package models

import (
	"time"
)

// getISOWeek returns the ISO week number for a given date.
//
// Parameters:
//   - t: time.Time - Date to get week number for
//
// Returns:
//   - int: ISO week number (1-53)
func getISOWeek(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

// RevenueCustomersOrdersOverview provides comprehensive business metrics dashboard.
// Aggregates key performance indicators for revenue, customers, and orders.
type RevenueCustomersOrdersOverview struct {
	Revenue   RevenueOverview   `json:"revenue"`   // Revenue metrics with growth
	Customers CustomersOverview `json:"customers"` // Customer acquisition metrics
	Orders    OrdersOverview    `json:"orders"`    // Order volume metrics
}

// RevenueOverview tracks revenue performance and growth.
type RevenueOverview struct {
	TotalRevenue            float64 `json:"total_revenue"`             // All-time revenue from delivered orders
	MonthlyRevenue          float64 `json:"monthly_revenue"`           // Current month revenue
	RevenuePercentageChange float64 `json:"revenue_percentage_change"` // Month-over-month % change
}

// CustomersOverview tracks customer acquisition and growth.
type CustomersOverview struct {
	TotalCustomers            int     `json:"total_customers"`             // Total registered users
	MonthlyNewCustomers       int     `json:"monthly_new_customers"`       // New users this month
	CustomersPercentageChange float64 `json:"customers_percentage_change"` // Month-over-month % change
}

// OrdersOverview tracks order volume and growth.
type OrdersOverview struct {
	TotalOrders            int     `json:"total_orders"`             // All-time order count
	MonthlyOrders          int     `json:"monthly_orders"`           // Orders this month
	OrdersPercentageChange float64 `json:"orders_percentage_change"` // Month-over-month % change
}

// GetRevenueCustomersOrdersOverview provides comprehensive business metrics.
//
// This dashboard function aggregates:
//   - Revenue metrics (total, monthly, growth %)
//   - Customer metrics (total, new this month, growth %)
//   - Order metrics (total, monthly, growth %)
//
// All metrics include month-over-month comparison.
//
// Returns:
//   - RevenueCustomersOrdersOverview: Complete dashboard metrics
//   - error: Database error or nil on success
func GetRevenueCustomersOrdersOverview() (RevenueCustomersOrdersOverview, error) {
	overview := RevenueCustomersOrdersOverview{}

	// ----- REVENUE METRICS -----
	// Total revenue from all delivered orders (all time)
	var totalRevenue float64
	err := DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.status IN ('Delivered', 'delivered', 'DELIVERED')
	`).Scan(&totalRevenue)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Revenue.TotalRevenue = totalRevenue

	// Revenue for current month from delivered orders
	var monthlyRevenue float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.status IN ('Delivered', 'delivered', 'DELIVERED')
		  AND MONTH(o.created_at) = MONTH(CURRENT_DATE())
		  AND YEAR(o.created_at) = YEAR(CURRENT_DATE())
	`).Scan(&monthlyRevenue)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Revenue.MonthlyRevenue = monthlyRevenue

	// Previous month revenue for comparison
	var prevMonthlyRevenue float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.unit_price * oi.quantity), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.status IN ('Delivered', 'delivered', 'DELIVERED')
		  AND MONTH(o.created_at) = MONTH(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND YEAR(o.created_at) = YEAR(CURRENT_DATE() - INTERVAL 1 MONTH)
	`).Scan(&prevMonthlyRevenue)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}

	// Calculate revenue growth percentage
	percentageChange := 0.0
	if prevMonthlyRevenue > 0 {
		percentageChange = ((monthlyRevenue - prevMonthlyRevenue) / prevMonthlyRevenue) * 100
	}
	overview.Revenue.RevenuePercentageChange = percentageChange

	// ----- CUSTOMERS METRICS -----
	// Total registered customers (all time)
	var totalCustomers int
	err = DB.QueryRow(`SELECT COUNT(DISTINCT user_id) FROM users WHERE status = 'active' AND role = 'customer'`).Scan(&totalCustomers)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Customers.TotalCustomers = totalCustomers

	// New customers registered this month
	var monthlyNewCustomers int
	err = DB.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM users
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE())
		  AND YEAR(created_at) = YEAR(CURRENT_DATE())
		  AND status = 'active'
		  AND role = 'customer'
	`).Scan(&monthlyNewCustomers)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Customers.MonthlyNewCustomers = monthlyNewCustomers

	// Previous month new customers for comparison
	var prevMonthlyNewCustomers int
	err = DB.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM users
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND YEAR(created_at) = YEAR(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND status = 'active'
		  AND role = 'customer'
	`).Scan(&prevMonthlyNewCustomers)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}

	// Calculate customer growth percentage
	percentageChange = 0.0
	if prevMonthlyNewCustomers > 0 {
		percentageChange = ((float64(monthlyNewCustomers) - float64(prevMonthlyNewCustomers)) / float64(prevMonthlyNewCustomers)) * 100
	}
	overview.Customers.CustomersPercentageChange = percentageChange

	// ----- ORDERS METRICS -----
	// Total orders (all time, any status)
	var totalOrders int
	err = DB.QueryRow(`SELECT COUNT(order_id) FROM orders`).Scan(&totalOrders)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Orders.TotalOrders = totalOrders

	// Orders created this month
	var monthlyOrders int
	err = DB.QueryRow(`
		SELECT COUNT(order_id)
		FROM orders
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE())
		  AND YEAR(created_at) = YEAR(CURRENT_DATE())
	`).Scan(&monthlyOrders)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}
	overview.Orders.MonthlyOrders = monthlyOrders

	// Previous month orders for comparison
	var prevMonthlyOrders int
	err = DB.QueryRow(`
		SELECT COUNT(order_id)
		FROM orders
		WHERE MONTH(created_at) = MONTH(CURRENT_DATE() - INTERVAL 1 MONTH)
		  AND YEAR(created_at) = YEAR(CURRENT_DATE() - INTERVAL 1 MONTH)
	`).Scan(&prevMonthlyOrders)
	if err != nil {
		return RevenueCustomersOrdersOverview{}, err
	}

	// Calculate order growth percentage
	percentageChange = 0.0
	if prevMonthlyOrders > 0 {
		percentageChange = ((float64(monthlyOrders) - float64(prevMonthlyOrders)) / float64(prevMonthlyOrders)) * 100
	}
	overview.Orders.OrdersPercentageChange = percentageChange

	return overview, nil
}
