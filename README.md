
# 🚀 Adenzo E-Commerce Backend

Welcome to the **Adenzo E-Commerce Backend**, a high-performance, scalable server-side application built with **Go (Golang)**. This system provides a robust API foundation for a complete e-commerce ecosystem, featuring sophisticated order management, real-time inventory tracking, and seamless payment integrations.

---

## 🏗 System Architecture

Adenzo is architected for modularity and performance:
- **Framework:** `gorilla/mux` for efficient routing.
- **Database:** MySQL for relational data persistence.
- **Caching:** Redis for high-speed data access and session management.
- **Observability:** Integrated with OpenTelemetry (Uptrace) for deep tracing and metrics.
- **Background Tasks:** Built-in schedulers for automated workflows like low-stock alerts and voucher emails.

---

## ✨ Key Features

- **🔐 Advanced Authentication:** Secure JWT-based auth with RBAC (Customer, Staff, Admin).
- **📦 Product Excellence:** Complex product management including variants, bundles, and reviews.
- **🛒 Dynamic Shopping:** Persistent carts, wishlists, and real-time coupon applications.
- **💳 Multi-Payment Support:** Seamless integration with M-Pesa and other traditional payment methods.
- **📊 Business Intelligence:** Comprehensive reporting tools for sales, inventory, and customer behavior.
- **🔔 Live Notifications:** WebSocket support for real-time order updates and stock alerts.
- **📝 Logistics & Fulfillment:** Built-in shipping, returns, and warehouse management systems.

---

## 🛠 Tech Stack

| Component | Technology |
| :--- | :--- |
| **Language** | Go v1.23+ |
| **Routing** | Gorilla Mux |
| **Database** | MySQL |
| **Cache** | Redis |
| **Monitoring** | OpenTelemetry, SLOs |
| **Documentation** | Swagger / OpenAPI |

---

## 🚀 Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) (v1.23 or higher)
- [MySQL](https://www.mysql.com/)
- [Redis](https://redis.io/)

### Installation

1. **Clone the Repository**
   ```bash
   git clone https://github.com/Roamtech-Solutions/adenzo-be.git
   cd adenzo-be
   ```

2. **Setup Environment Variables**
   Copy `.env.example` to `.env` and configure your database and API keys.
   ```bash
   cp .env.example .env
   ```

3. **Install Dependencies**
   ```bash
   go mod tidy
   ```

4. **Initialize Database**
   ```bash
   go run . -migrate=true
   ```

5. **Start the Engine**
   ```bash
   go run .
   ```

---

## 📖 API Documentation

Adenzo uses Swagger for interactive API documentation. Once the server is running, navigate to:
`http://localhost:8000/swagger/`

---

## 🧪 Documentation & Resources

- [Technical Deep Dive](file:///c:/Users/Timothy/adenzo-backend/TECHNICAL_DOCUMENTATION.md)
- [Migrations Guide](file:///c:/Users/Timothy/adenzo-backend/migrations/)

---

## 🤝 Contact

Developed by **@Timothy** at Roamtech Solutions.
