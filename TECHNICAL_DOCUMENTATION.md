# 📖 Ekomasi Technical Documentation

Welcome to the comprehensive technical guide for the **Ekomasi E-Commerce Backend**. This document is designed to give you a complete understanding of how the system is built, how it works, and where everything is located.

---

## 🏗 System Overview & Architecture

Ekomasi is a high-performance, modular backend built with **Go**. It follows a layered architecture to ensure that the code is easy to maintain and scale.

### 🔌 API Layer (How we handle requests)
- **Framework:** We use `gorilla/mux` for fast and flexible routing.
- **Routing:** All API endpoints are defined in the `routes/` folder, neatly separated by what they do (Admin, Products, Orders, etc.).
- **Middleware:** These are special functions that run before a request reaches its final destination. They handle things like checking if a user is logged in (**Auth**), tracking performance (**Telemetry**), and catching errors.

### 🧠 Domain Logic (The "Brain" of the App)
- **Handlers:** Located in `handlers/`, these files contain the actual logic for features like adding to a cart or processing an order. 
- **DTOs:** We use **Data Transfer Objects** (`dtos/`) to define exactly what data should come in and go out of our API, keeping our database models clean.

### 🗄 Storage Layer (Where data lives)
- **Database:** We use **MySQL** with **GORM**, which makes it easy to talk to the database using Go code.
- **Caching:** **Redis** is used to speed things up, like storing user sessions and temporary checkout data.
- **Migrations:** All changes to the database structure are tracked in the `migrations/` folder as SQL files.

---

## 📂 Project Structure (Where is everything?)

To find your way around the codebase, here is a quick map:

| Folder | What's inside? |
| :--- | :--- |
| **`handlers/`** | The core logic for every feature (Orders, Cart, M-Pesa, etc.). |
| **`models/`** | The database table definitions and relationships. |
| **`routes/`** | Where we define our API paths (e.g., `/api/products`). |
| **`dtos/`** | Data structures for API requests and responses. |
| **`middleware/`** | Functions for security, logging, and monitoring. |
| **`migrations/`** | SQL scripts to update or rollback the database. |
| **`utils/`** | Helpful tools like the Logger, Redis client, and WebSockets. |
| **`docs/`** | Automatically generated Swagger API documentation. |

## 🗃 Detailed File Breakdown

This section provides a granular look at the files within each major directory and their specific responsibilities.

### 📁 `handlers/` (Business Logic)
This folder contains the "actions" of the application.
- **`auth.go`**: Handles user registration, login, and password resets.
- **`order.go`**: Manages the lifecycle of an order, from creation to status updates.
- **`cart.go`**: Controls shopping cart operations (add, remove, update quantities).
- **`products.go`**: CRUD operations for the product catalog.
- **`mpesa.go`**: Direct integration with M-Pesa API for payments.
- **`inventory.go`**: Tracks stock levels and handles stock-in/stock-out logic.
- **`reports.go`**: Generates sales, customer, and inventory reports.
- **`deals.go` / `promotions.go`**: Logic for managing discounts, coupons, and special offers.
- **`reviews.go`**: Handles customer feedback and product ratings.
- **`vouchers.go`**: Manages the purchase and redemption of digital vouchers.
- **`transactions.go`**: Logs and tracks all financial movements within the system.
- **`users.go` / `roles.go`**: Profile management and permission control.

### 📁 `models/` (Database Schema)
These files define the structure of our data.
- **`database.go`**: Sets up the GORM connection and configuration.
- **`products.go` / `variants.go`**: Defines product attributes, price variants, and descriptions.
- **`order.go` / `sales.go`**: Schema for customer orders and individual sale items.
- **`cart.go`**: Storage structure for persistent user carts.
- **`users.go` / `accounts.go`**: User details, credentials, and account statuses.
- **`categories.go` / `brands.go`**: Taxonomic data for organizing products.
- **`mpesa.go`**: Storage for M-Pesa transaction logs and callback data.
- **`warehouse.go` / `inventory.go`**: Tracks physical locations and stock counts.

### 📁 `routes/` (API Endpoints)
Mapping URLs to the logic in our handlers.
- **`routes.go`**: The "Master" router that combines all sub-routes.
- **`auth_routes.go`**: Endpoints for signing in and out.
- **`admin_routes.go`**: Protected routes accessible only by managers/owners.
- **`product_routes.go`**: Public-facing catalog endpoints.
- **`order_routes.go`**: Endpoints for users to place and view orders.
- **`payment_routes.go`**: Handles payment initiation and webhook callbacks.
- **`reports_routes.go`**: Secured routes for viewing business intelligence data.

### 📁 `dtos/` (Data Transfer Objects)
Defines how data is formatted when it enters or leaves the API.
- **`auth.go`**: Structure for login requests and registration forms.
- **`order.go`**: How an order is presented in the API response.
- **`products.go`**: Defines searchable product fields and filter formats.
- **`homepage.go`**: Aggregated data structure for the landing page.
- **`validation.go`**: Shared rules for cleaning and checking user input.

### 📁 `middleware/` (Security & Helpers)
Shared logic that protects our routes.
- **`auth_middleware.go`**: Verifies JWT tokens and checks user roles.
- **`telemetry_middleware.go`**: Records the speed and success rate of every request.
- **`error_handling.go`**: Ensures that every error returns a clean, understandable message.
- **`ip_restriction.go`**: Limits access to sensitive APIs based on IP address.

### 📁 `utils/` (Core Tools)
The engine's utility belt.
- **`logger.go`**: Sets up the system for detailed "flight recording" (tracing).
- **`redis.go`**: Simplified helper for talking to our high-speed cache.
- **`websocket.go`**: Enables "live" updates without refreshing the page.
- **`pdf.go` / `csv.go`**: Helpers for exporting receipts and reports.
- **`image_compression.go`**: Automatically shrinks uploaded images for faster loading.

---

## 🔄 Core Business Flows

### 🛒 The Shopping Journey
1. **Browsing:** Users find products managed by `products.go` handlers.
2. **Cart:** Items are added to a persistent cart managed by `cart.go`.
3. **Checkout:** The `CheckoutHandler` starts a session and prepares for payment.
4. **Payment:** We integrate with **M-Pesa** for secure transactions.
5. **Fulfillment:** Once paid, the order moves to `OrderHandler` for processing and shipping.

### 🔐 Security & Access
- **Authentication:** Users sign in and get a **JWT** (token).
- **Session Caching:** Tokens and session data are stored in **Redis** for lightning-fast verification.
- **Roles:** Different users (Admins, Staff, Customers) have different permissions checked via middleware.

---

## 🛠 Advanced Features

### 📡 Monitoring & Observability
- **OpenTelemetry:** We track every single request to see how long it takes and where it might be slow using `uptrace-go`.
- **Logging:** We use structured logging (`utils/logger.go`) so that logs are easy to search and analyze.

### ⏰ Background Tasks (Schedulers)
The system automatically handles repetitive tasks in the background:
- **Low Stock Alerts:** Notifies the team when items are running out.
- **Voucher Cleanup:** Removes expired discounts automatically.
- **Abandoned Carts:** Reminds customers who left items in their cart without buying.

---

## 🔗 Integrations

- **M-Pesa:** Custom built to handle STK push and payment confirmations.
- **WebSockets:** Used for real-time updates—like when an order status changes.
- **Social Login:** Ready for WhatsApp and Google authentication.

---

## 🛠 Developer Guide

### How to add a new feature:
1. **Define the Data:** Create a new `DTO` for your request and response.
2. **Write the Logic:** Add a new file in `handlers/` for your business logic.
3. **Register the Route:** Add your new endpoint to a file in `routes/`.
4. **Update the DB:** If you need a new table, add a migration file in `migrations/`.

### Debugging Tips:
- **API Test:** Use the `/swagger/` page to test your endpoints in the browser.
- **Traces:** Check the **Uptrace** dashboard to see exactly what happened during a request.
