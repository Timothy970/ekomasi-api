package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
)

func TestGetAllTransactionHandler(t *testing.T) {
	_, mock, teardown := setupTestDB(t)
	defer teardown()

	// Default query with no filters
	// Page 1, Limit 10 (default in handler if not specified? Or handler uses params)
	// Handler defaults: page=1, limit=10.

	// Count Query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM transactions WHERE 1=1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Data Query
	rows := sqlmock.NewRows([]string{
		"transaction_id", "order_id", "mpesa_reference", "transaction_reference",
		"phone_number", "amount", "account_number", "status", "created_at", "payment_method",
	}).AddRow(
		"tx-1", "ord-1", "MPESA1", "TX1", "1234567890", 100.0, "ACC1", "completed", time.Now(), "mpesa",
	).AddRow(
		"tx-2", "ord-2", "MPESA2", "TX2", "0987654321", 200.0, "ACC2", "pending", time.Now(), "mpesa",
	)

	mock.ExpectQuery("SELECT .* FROM transactions .* LIMIT \\? OFFSET \\?").
		WithArgs(10, 0).
		WillReturnRows(rows)

	// Logging
	// Handler typically creates request summary but might not log success unless configured.
	// We'll check code. Assuming no log/log success.

	// Need empty body to avoid nil pointer in GetRequestSummary
	req, _ := http.NewRequest("GET", "/api/admin/transactions?page=1&limit=10", bytes.NewBuffer([]byte{}))
	req = req.WithContext(middleware.ContextWithUser(req.Context(), middleware.AuthenticatedUser{
		Role:  "admin",
		Email: "admin@ekomasi.shop",
	}))
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	GetAllTransactionHandler(c)


	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Errorf("Response: %s", rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}

func TestUpdateTransactionStatusHandler_Success(t *testing.T) {
	_, mock, teardown := setupTestDB(t)
	defer teardown()

	payload := dtos.UpdateTransactionStatus{
		Status: "completed",
	}

	// Expect Transaction Start
	mock.ExpectBegin()

	// 1. UpdateTransactionStatusByID
	// Update transactions
	mock.ExpectExec("UPDATE transactions SET status = \\? WHERE transaction_id = \\?").
		WithArgs("completed", "tx-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Get OrderID
	mock.ExpectQuery("SELECT order_id FROM transactions WHERE transaction_id = \\?").
		WithArgs("tx-1").
		WillReturnRows(sqlmock.NewRows([]string{"order_id"}).AddRow("ord-1"))

	// 2. UpdateOrderPaymentStatus
	// Update order
	mock.ExpectExec("UPDATE orders SET payment_status = \\? WHERE order_id = \\?").
		WithArgs("completed", "ord-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect Commit
	mock.ExpectCommit()

	// 3. Success Log
	mock.ExpectExec("INSERT INTO logs").
		WithArgs(sqlmock.AnyArg(), "INFO", "Transaction status updated successfully", sqlmock.AnyArg(), sqlmock.AnyArg(), "Transactions", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PATCH", "/api/admin/transactions/tx-1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.ContextWithUser(req.Context(), middleware.AuthenticatedUser{
		Role:  "admin",
		Email: "admin@ekomasi.shop",
	}))
	req = mux.SetURLVars(req, map[string]string{"transaction_id": "tx-1"})

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "transaction_id", Value: "tx-1"}}

	UpdateTransactionStatusHandler(c)


	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Errorf("Response: %s", rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}
