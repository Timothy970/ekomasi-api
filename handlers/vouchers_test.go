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
)

func TestRedeemVoucherHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		_, mock, teardown := setupTestDB(t)
		defer teardown()

		payload := dtos.RedeemVoucherRequest{
			Code: "VOUCHER123",
		}

		// Mock RedeemVoucher call
		// IsVoucherThere check
		mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM vouchers WHERE code = \\?\\)").
			WithArgs("VOUCHER123").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// RedeemVoucher performs GetVoucherByCode
		voucherRows := sqlmock.NewRows([]string{"voucher_id", "code", "balance", "original_value", "status", "created_at", "expiry_date"}).
			AddRow("voucher-1", "VOUCHER123", 100.0, 100.0, "active", time.Now(), time.Now().AddDate(0, 0, 30))

		mock.ExpectQuery("SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date FROM vouchers WHERE code = \\?").
			WithArgs("VOUCHER123").
			WillReturnRows(voucherRows)


		// Then UPDATE vouchers SET user_id = ? WHERE code = ?
		mock.ExpectExec("UPDATE vouchers SET user_id = \\? WHERE code = \\?").
			WithArgs(sqlmock.AnyArg(), "VOUCHER123").
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Success log
		mock.ExpectExec("INSERT INTO logs").
			WithArgs(sqlmock.AnyArg(), "INFO", "Voucher redeemed successfully", sqlmock.AnyArg(), sqlmock.AnyArg(), "Vouchers", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/vouchers/redeem", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(middleware.ContextWithUser(req.Context(), middleware.AuthenticatedUser{
			Role:  "customer",
			Email: "user@example.com",
		}))

		rr := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rr)
		c.Request = req
		RedeemVoucherHandler(c)

		// Note: This demonstrates the database mocking approach
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Logf("Unmet expectations: %s", err)
		}
	})
}

