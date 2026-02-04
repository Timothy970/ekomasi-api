package utils

import "adenzo_backend/dtos"

// supportedPermissions defines the complete permission hierarchy for the application.
// Organized by functional categories for granular access control.
// Each permission has a unique key (e.g., "inventory.create"), category, and description.
// Used as fallback when database permissions are unavailable.
var SupportedPermissions = []dtos.AvailablePermission{
	// INVENTORY
	{Category: "Inventory", Key: "inventory.create", Description: "Add new inventory item"},
	{Category: "Inventory", Key: "inventory.view", Description: "View inventory list and details"},
	{Category: "Inventory", Key: "inventory.update", Description: "Edit inventory item"},
	{Category: "Inventory", Key: "inventory.delete", Description: "Delete inventory item"},

	// ORDERS
	{Category: "Orders", Key: "orders.create", Description: "Create new order"},
	{Category: "Orders", Key: "orders.view", Description: "View orders and order details"},
	{Category: "Orders", Key: "orders.update", Description: "Update or modify order"},
	{Category: "Orders", Key: "orders.delete", Description: "Cancel or delete order"},

	// PRODUCTS
	{Category: "Products", Key: "products.create", Description: "Add new product"},
	{Category: "Products", Key: "products.view", Description: "View products list"},
	{Category: "Products", Key: "products.update", Description: "Edit product details"},
	{Category: "Products", Key: "products.delete", Description: "Delete product"},

	// CATEGORIES
	{Category: "Categories", Key: "categories.create", Description: "Add new product category"},
	{Category: "Categories", Key: "categories.view", Description: "View product categories"},
	{Category: "Categories", Key: "categories.update", Description: "Edit product category"},
	{Category: "Categories", Key: "categories.delete", Description: "Delete product category"},

	// USERS & ROLES
	{Category: "Users", Key: "users.create", Description: "Add new user"},
	{Category: "Users", Key: "users.view", Description: "View users"},
	{Category: "Users", Key: "users.update", Description: "Edit user info"},
	{Category: "Users", Key: "users.delete", Description: "Delete user"},
	{Category: "Users", Key: "roles.create", Description: "Create new role"},
	{Category: "Users", Key: "roles.view", Description: "View roles and permissions"},
	{Category: "Users", Key: "roles.update", Description: "Update existing roles"},
	{Category: "Users", Key: "roles.delete", Description: "Delete a role"},

	// REPORTS
	{Category: "Reports", Key: "reports.view", Description: "View all reports"},

	// PAYMENTS
	{Category: "Payments", Key: "payments.create", Description: "Initiate new payment"},
	{Category: "Payments", Key: "payments.view", Description: "View payments"},
	{Category: "Payments", Key: "payments.refund", Description: "Process payment refund"},
	{Category: "Payments", Key: "payments.update", Description: "Update payment status"},
	{Category: "Payments", Key: "payments.delete", Description: "Delete payment record"},

	// CHARGES
	{Category: "Charges", Key: "charges.create", Description: "Create new charge"},
	{Category: "Charges", Key: "charges.view", Description: "View charges"},
	{Category: "Charges", Key: "charges.update", Description: "Update charge details"},
	{Category: "Charges", Key: "charges.delete", Description: "Delete charge"},

	// NOTIFICATIONS
	{Category: "Notifications", Key: "notifications.create", Description: "Add new notification"},
	{Category: "Notifications", Key: "notifications.view", Description: "View notifications"},
	{Category: "Notifications", Key: "notifications.update", Description: "Edit notification details"},
	{Category: "Notifications", Key: "notifications.delete", Description: "Delete notification"},

	// SUPPLIERS
	{Category: "Suppliers", Key: "suppliers.create", Description: "Add new supplier"},
	{Category: "Suppliers", Key: "suppliers.view", Description: "View supplier list"},
	{Category: "Suppliers", Key: "suppliers.update", Description: "Edit supplier details"},
	{Category: "Suppliers", Key: "suppliers.delete", Description: "Remove supplier"},

	// SETTINGS
	// {Category: "Settings", Key: "settings.view", Description: "View system settings"},
	// {Category: "Settings", Key: "settings.update", Description: "Modify application settings"},

	// WAREHOUSE
	{Category: "Warehouse", Key: "warehouse.view", Description: "View warehouse stock and details"},
	{Category: "Warehouse", Key: "warehouse.update", Description: "Update warehouse information"},
	{Category: "Warehouse", Key: "warehouse.create", Description: "Add new warehouse"},
	{Category: "Warehouse", Key: "warehouse.delete", Description: "Delete warehouse"},

	// LOGS & AUDIT
	// {Category: "Logs", Key: "logs.view", Description: "View system or user activity logs"},
	// {Category: "Logs", Key: "logs.export", Description: "Export system logs for analysis"},

	//ACCOUNTS
	{Category: "Accounts", Key: "accounts.view", Description: "View account details and balances"},
	{Category: "Accounts", Key: "accounts.update", Description: "Update account information"},
	{Category: "Accounts", Key: "accounts.delete", Description: "Delete an account"},
	{Category: "Accounts", Key: "accounts.create", Description: "Create a new account"},

	//PROMOTIONS
	{Category: "Promotions & Vouchers", Key: "promotions.create", Description: "Create new promotion or discount"},
	{Category: "Promotions & Vouchers", Key: "promotions.view", Description: "View promotions and discounts"},
	{Category: "Promotions & Vouchers", Key: "promotions.update", Description: "Edit existing promotions"},
	{Category: "Promotions & Vouchers", Key: "promotions.delete", Description: "Remove a promotion"},

	// CONTENT (Blogs & Static Pages)
	{Category: "Blogs & Static pages", Key: "content.create", Description: "Create new content"},
	{Category: "Blogs & Static pages", Key: "content.view", Description: "View content"},
	{Category: "Blogs & Static pages", Key: "content.update", Description: "Edit existing content"},
	{Category: "Blogs & Static pages", Key: "content.delete", Description: "Delete content"},
}
