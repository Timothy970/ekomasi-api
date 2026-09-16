package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"errors"
)

func GetOrderByID(db DBExecutor, orderID string) (*dtos.Order, error) {
	// Validate order exists
	err := IsOrderThere(db, orderID)
	if err != nil {
		return nil, err
	}

	var userID sql.NullString
	// Query with LEFT JOIN to deliveries
	query := `
        SELECT 
            o.order_id,
            o.total_amount,
            o.total_discount,
            o.delivery_id,
            o.status,
            d.status AS delivery_status,
            o.payment_method,
            d.delivery_charge,
            d.delivery_address,
            o.guest_delivery_address,
            o.guest_personal_details,
            o.created_at,
            o.user_id,
			o.is_guest_order,
			o.source
        FROM orders o
        LEFT JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.order_id = ?`

	var ord dtos.Order
	var totalAmount float64
	var guestAddrStr, guestDetailsStr string

	// Scan order fields including nullable user_id
	err = DB.QueryRow(query, orderID).Scan(
		&ord.OrderID,
		&totalAmount,
		&ord.TotalDiscount,
		&ord.DeliveryID,
		&ord.OrderStatus,
		&ord.DeliveryStatus,
		&ord.PaymentMethod,
		&ord.DeliveryCharge,
		&ord.DeliveryAddress,
		&guestAddrStr,
		&guestDetailsStr,
		&ord.CreatedAt,
		&userID,
		&ord.IsGuestOrder,
		&ord.OrderSource,
	)

	// Calculate tax and subtotal
	ord.TotalAmount = totalAmount
	estimatedTax, _ := GetEstimatedTax(DB)
	ord.EstimatedTax = ord.TotalAmount * estimatedTax / 100
	ord.SubTotal = ord.TotalAmount + ord.TotalDiscount - ptrToFloat(ord.DeliveryCharge) - ord.EstimatedTax

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Parse guest JSON fields
	if guestAddrStr != "" {
		_ = json.Unmarshal([]byte(guestAddrStr), &ord.GuestDeliveryAddress)
	}
	if guestDetailsStr != "" {
		_ = json.Unmarshal([]byte(guestDetailsStr), &ord.GuestPersonalDetails)
	}

	// Fetch user address if this is a registered user order
	if userID.Valid {
		address, err := GetUserAddresses(db, userID.String)
		if err != nil {
			return nil, err
		}
		ord.UserAddress = &address
		ord.UserID = &userID.String

	}

	// Fetch items with product details
	items, err := getOrderProducts(db, orderID, userID.String)
	if err != nil {
		return nil, err
	}
	ord.Items = items

	return &ord, nil
}

// getOrderProducts is an internal helper that fetches order items with product details.
//
// This function retrieves all products in an order with complete information including
// images, warranties, and review status (if userID provided).
//
// Parameters:
//   - orderID: string - The order whose items to fetch
//   - userID: string - User ID for checking review status (empty string to skip review checks)
//
// Returns:
//   - []dtos.OrderProduct: Array of order items with:
//   - Product details (ID, Name, Description, SKU, CategoryID, Price)
//   - StockQuantity: Actually contains the ordered quantity (not current stock)
//   - Images: Array of product images
//   - Warranty: Product warranty information
//   - IsReviewed: Boolean if user has reviewed this product
//   - ReviewID: The user's review ID if reviewed
//   - error: Database error or nil on success
func getOrderProducts(db DBExecutor, orderID string, userID string) ([]dtos.OrderProduct, error) {
	// Query joins order_items with products to get complete product info
	itemsQuery := `
        SELECT 
            p.product_id,
            p.name,
            p.description,
            p.sku,
            oi.unit_price,
            p.category_id,
            oi.quantity,
            p.search_vector,
            p.created_at,
            p.last_updated_at
        FROM order_items oi
        JOIN products p ON oi.product_id = p.product_id
        WHERE oi.order_id = ?`

	rows, err := db.Query(itemsQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dtos.OrderProduct
	for rows.Next() {
		var item dtos.OrderProduct

		// Scan product details and ordered quantity
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.SKU,
			&item.Price,
			&item.CategoryID,
			&item.StockQuantity, // Actually ordered quantity from oi.quantity
			&item.SearchVector,
			&item.CreatedAt,
			&item.LastUpdated,
		); err != nil {
			return nil, err
		}

		// Fetch product images
		images, err := fetchProductImages(db, item.ID)
		if err != nil {
			return nil, err
		}
		item.Images = images

		// Fetch product warranty
		warranty, err := FetchProductWarranties(db, item.ID)
		if err != nil {
			return nil, err
		}
		item.Warranty = &warranty

		// Check if user has reviewed this product
		if userID != "" {
			item.IsReviewed, item.ReviewID = checkIfReviewed(db, item.ID, userID)
		}

		items = append(items, item)
	}

	return items, nil
}

// checkIfReviewed is an internal helper to check if a user has reviewed a product.
//
// This function queries the product_reviews table to determine if the user
// has already submitted a review for the given product.
//
// Parameters:
//   - productID: string - The product to check
//   - userID: string - The user to check
//
// Returns:
//   - bool: true if user has reviewed the product, false otherwise
//   - string: The review_id if reviewed, empty string otherwise
func checkIfReviewed(db DBExecutor, productID, userID string) (bool, string) {
	query := `SELECT review_id FROM product_reviews WHERE product_id = ? AND user_id = ?`
	var reviewID string
	err := db.QueryRow(query, productID, userID).Scan(&reviewID)
	if err != nil {
		return false, ""
	}
	if reviewID != "" {
		return true, reviewID
	}
	return false, ""
}

// AdminOrderParameters defines filtering and pagination options for admin order listing.
//
// This structure supports comprehensive order filtering with multiple criteria,
// time range selections, full-text search, and pagination.
//
// Fields:
//   - OrderStatus: string - Filter by order status ("pending", "completed", etc.)
//   - PaymentStatus: string - Filter by payment status ("paid", "pending", etc.)
//   - DeliveryStatus: string - Filter by delivery status ("Delivered", "Shipped", etc.)
//   - PaymentMethod: string - Filter by payment method ("card", "mpesa", "cash", etc.)
//   - TimeRange: string - Predefined time range:
//   - "today": Orders from today only
//   - "this_week": Orders from current week (Monday-Sunday)
//   - "this_month": Orders from current month
//   - "last_month": Orders from previous month
//   - "this_year": Orders from current year
//   - OrderID: string - Exact order ID match
//   - Q: string - Full-text search across:
//   - Guest personal details
//   - Order ID, Delivery ID
//   - Delivery addresses (guest and registered)
//   - Payment method
//   - User email, phone
//   - User names (supports "first last" and "last first" order)
//   - StartDate: string - Custom date range start (ISO format)
//   - EndDate: string - Custom date range end (ISO format)
//   - Page: int - Page number (1-indexed)
//   - Limit: int - Orders per page
type AdminOrderParameters struct {
	OrderStatus    string
	PaymentStatus  string
	DeliveryStatus string
	PaymentMethod  string
	TimeRange      string
	OrderID        string
	Q              string
	Page           int
	Limit          int
	StartDate      string
	EndDate        string
	Past           string
	RiderID        string
	UserID         string
}

// ListOrdersByAdmin retrieves paginated orders with advanced filtering for admin dashboards.
//
// This function provides comprehensive order management capabilities including:
//   - Multiple status filters (order, payment, delivery)
//   - Predefined and custom time ranges
//   - Full-text search across orders, users, and addresses
//   - Pagination with metadata
//
// Parameters:
//   - params: AdminOrderParameters with filter and pagination options
//
// Returns:
//   - []dtos.AdminOrder: Array of orders with:
//   - Complete order and delivery information
//   - User details (if registered user)
//   - Items array with full product details
//   - ItemsCount: Number of items in order
//   - DeliveredAt: Actual delivery timestamp
//   - Guest details (if guest order)
//   - *dtos.PaginationMeta: Pagination with HasNext, HasPrev, TotalItems, TotalPages
//   - error: Database error or nil on success
//
// Search Behavior:
//   - Searches across guest details, order IDs, addresses, payment method, user info
//   - Supports name search in both "first last" and "last first" order
//   - Uses LIKE queries with wildcards for flexible matching
