package handlers

import (
	"bytes"
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestProcessSplitPaymentHandler(t *testing.T) {
	// Sample request payload
	amount := 100.0
	validPayload := dtos.SplitPaymentRequest{
		OrderID: "ord-1",
		PaymentMethods: []dtos.PaymentMethod{
			{Type: "cash", Amount: 100.0},
		},
	}

	t.Run("Success_CashOnly", func(t *testing.T) {
		_, mock, teardown := setupTestDB(t)
		defer teardown()

		// --- 1. GetOrderByID Mocks ---

		// IsOrderThere
		mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM orders WHERE order_id = \\?\\)").
			WithArgs("ord-1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Main Order Query
		orderRows := sqlmock.NewRows([]string{
			"order_id", "total_amount", "total_discount", "delivery_id", "status",
			"delivery_status", "payment_method", "delivery_charge", "delivery_address",
			"guest_delivery_address", "guest_personal_details", "created_at", "user_id", "is_guest_order", "source",
		}).AddRow(
			"ord-1", amount, 0.0, "del-1", "pending",
			"pending", "", 0.0, "Addr",
			"", "", time.Now(), nil, true, "POS",
		)

		mock.ExpectQuery("SELECT .* FROM orders o .* WHERE o.order_id = ?").
			WithArgs("ord-1").
			WillReturnRows(orderRows)

		// GetEstimatedTax
		mock.ExpectQuery("SELECT charge_value FROM charges WHERE charge_name = \"tax\"").
			WillReturnRows(sqlmock.NewRows([]string{"charge_value"}).AddRow(16.0))

		// getOrderProducts -> Select items
		itemRows := sqlmock.NewRows([]string{
			"product_id", "name", "description", "sku", "unit_price", "category_id",
			"quantity", "search_vector", "created_at", "last_updated_at",
		}) // Empty items for simplicity, dealing with payment only
		mock.ExpectQuery("SELECT .* FROM order_items .* WHERE oi.order_id = ?").
			WithArgs("ord-1").
			WillReturnRows(itemRows)

		// Note: getOrderProducts loop is skipped if no items returned.

		// --- 2. Transaction Start ---
		mock.ExpectBegin()

		// --- 3. ProcessCashPayment Logic ---

		// UpdateOrderStatus calls IsOrderThere FIRST
		mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM orders WHERE order_id = \\?\\)").
			WithArgs("ord-1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// UPDATE orders
		// Arguments: status, payment_method, payment_status, order_id
		mock.ExpectExec("UPDATE orders SET status = \\?, payment_method = \\?, payment_status = \\? WHERE order_id = \\?").
			WithArgs("COMPLETED", "CASH", "SUCCESS", "ord-1").
			WillReturnResult(sqlmock.NewResult(1, 1))

		// InsertTransaction
		mock.ExpectExec("INSERT INTO transactions").
			WillReturnResult(sqlmock.NewResult(1, 1))

		// --- 4. Commit ---
		mock.ExpectCommit()

		// Expect Log Insert (Success Response)
		mock.ExpectExec("INSERT INTO logs").
			WithArgs(sqlmock.AnyArg(), "INFO", "Payment processed successfully", sqlmock.AnyArg(), sqlmock.AnyArg(), "Payments", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// --- Execution ---
		body, _ := json.Marshal(validPayload)
		req, _ := http.NewRequest("POST", "/api/pos/split-payment", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(middleware.ContextWithUser(req.Context(), middleware.AuthenticatedUser{
			Role:  "admin",
			Email: "admin@ekomasi.shop",
		}))

		rr := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rr)
		c.Request = req
		ProcessSplitPaymentHandler(c)

		// --- Assertions ---
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			t.Errorf("Response: %s", rr.Body.String())
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
