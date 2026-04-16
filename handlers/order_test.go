package handlers

import (
	"adenzo_backend/dtos"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
)

func TestNewCreateOrderHandler(t *testing.T) {
	isGuest := true
	// Sample valid request payload
	validPayload := dtos.CreateOrderPayload{
		OrderItems: []dtos.OrderItemPayload{
			{ProductID: "prod-1", Quantity: 2},
		},
		IsGuestOrder:      &isGuest,
		DeliveryAddressID: nil, // Skip delivery charge fetch
	}

	t.Run("Success", func(t *testing.T) {
		// Setup mock DB for this subtest
		_, mock, teardown := setupTestDB(t)
		defer teardown()

		// --- Mocks for logic BEFORE transaction ---

		// --- 1. First Call: getOrderItems -> GetProductByID ---

		// IsProductThere
		mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM products WHERE product_id = \\?\\)").
			WithArgs("prod-1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Main Product Query
		rows1 := sqlmock.NewRows([]string{
			"product_id", "name", "description", "sku", "price", "category_id",
			"stock_quantity", "search_vector", "created_at", "last_updated_at",
			"category_name", "tag", "details", "discount", "discount_type",
			"weight", "dimensions", "manufacturer", "weight_limit",
		}).AddRow(
			"prod-1", "Test Product", "Desc", "SKU1", 100.0, "cat-1",
			10, "", time.Now(), time.Now(),
			"Category", "Tag", nil, 0.0, "",
			1.0, "10x10", "Manu", 10.0,
		)
		mock.ExpectQuery("SELECT .* FROM products .* WHERE p.product_id = ?").
			WithArgs("prod-1").
			WillReturnRows(rows1)

		// Enrich calls (1st pass)
		mock.ExpectQuery("SELECT .* FROM product_images").WillReturnRows(sqlmock.NewRows([]string{"image_url"}))
		mock.ExpectQuery("SELECT .* FROM product_warranties").WillReturnRows(sqlmock.NewRows([]string{"warranty_id"}))
		mock.ExpectQuery("SELECT .* FROM product_features").WillReturnRows(sqlmock.NewRows([]string{"feature"}))

		// Variants - Must provide all columns
		mock.ExpectQuery("SELECT .* FROM product_variants").
			WillReturnRows(sqlmock.NewRows([]string{"variant_id", "variant_type", "name", "hex_code", "additional_price", "stock_quantity"}))

		// Tax - Uses product_charges table
		mock.ExpectQuery("SELECT .* FROM product_charges").WillReturnRows(sqlmock.NewRows([]string{"charge_id", "charge_name", "charge_value"}))

		// --- 2. Second Call: checkStockAvailability -> GetProductByID ---

		// IsProductThere
		mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM products WHERE product_id = \\?\\)").
			WithArgs("prod-1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Main Product Query
		rows2 := sqlmock.NewRows([]string{
			"product_id", "name", "description", "sku", "price", "category_id",
			"stock_quantity", "search_vector", "created_at", "last_updated_at",
			"category_name", "tag", "details", "discount", "discount_type",
			"weight", "dimensions", "manufacturer", "weight_limit",
		}).AddRow(
			"prod-1", "Test Product", "Desc", "SKU1", 100.0, "cat-1",
			10, "", time.Now(), time.Now(),
			"Category", "Tag", nil, 0.0, "",
			1.0, "10x10", "Manu", 10.0,
		)
		mock.ExpectQuery("SELECT .* FROM products .* WHERE p.product_id = ?").
			WithArgs("prod-1").
			WillReturnRows(rows2)

		// Enrich calls (2nd pass)
		mock.ExpectQuery("SELECT .* FROM product_images").WillReturnRows(sqlmock.NewRows([]string{"image_url"}))
		mock.ExpectQuery("SELECT .* FROM product_warranties").WillReturnRows(sqlmock.NewRows([]string{"warranty_id"}))
		mock.ExpectQuery("SELECT .* FROM product_features").WillReturnRows(sqlmock.NewRows([]string{"feature"}))

		// Variants
		mock.ExpectQuery("SELECT .* FROM product_variants").
			WillReturnRows(sqlmock.NewRows([]string{"variant_id", "variant_type", "name", "hex_code", "additional_price", "stock_quantity"}))

		// Tax
		mock.ExpectQuery("SELECT .* FROM product_charges").WillReturnRows(sqlmock.NewRows([]string{"charge_id", "charge_name", "charge_value"}))

		// 3. GetProductDiscount (called in processOrderItems)
		mock.ExpectQuery("SELECT dp.discount, dp.discount_type FROM deal_products dp").
			WithArgs("prod-1").
			WillReturnRows(sqlmock.NewRows([]string{"discount", "discount_type"})) // No rows = no discount

		// 3. GetProductPromotionData
		mock.ExpectQuery("SELECT .* FROM promotion_products").
			WithArgs("prod-1").
			WillReturnRows(sqlmock.NewRows([]string{"promotion_type", "promotion_value"})) // Empty rows

		// --- TRANSACTION START ---

		mock.ExpectBegin()

		// Expect CreateOrder (INSERT INTO orders)
		mock.ExpectExec("INSERT INTO orders").
			WithArgs(sqlmock.AnyArg(), nil, true, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect CreateOrderItem (INSERT INTO order_items)
		mock.ExpectExec("INSERT INTO order_items").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "prod-1", "", 2, 100.0). // Quantity 2, Price 100
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect CreateDeliveries (INSERT INTO deliveries)
		mock.ExpectExec("INSERT INTO deliveries").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 0.0, sqlmock.AnyArg(), sqlmock.AnyArg()). // DeliveryCharge 0
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect DeductProductStock (UPDATE products)
		mock.ExpectExec("UPDATE products").
			WithArgs(2, "prod-1").
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect Log Insert (Success)
		mock.ExpectExec("INSERT INTO logs").
			WithArgs(sqlmock.AnyArg(), "INFO", "Order created successfully", sqlmock.AnyArg(), sqlmock.AnyArg(), "Orders", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect Commit
		mock.ExpectCommit()

		// Prepare request
		body, _ := json.Marshal(validPayload)
		req, _ := http.NewRequest("POST", "/api/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{})

		// Recorder
		rr := httptest.NewRecorder()

		// Execute Handler
		NewCreateOrderHandler(rr, req)

		// Assertions
		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
			t.Errorf("Response: %s", rr.Body.String())
		}

		// Verify that all expectations were met
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("RollbackOnError", func(t *testing.T) {
		// Setup mock DB for this subtest
		_, mock, teardown := setupTestDB(t)
		defer teardown()

		// --- 1. First Call: getOrderItems -> GetProductByID ---

		// IsProductThere
		mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM products WHERE product_id = \\?\\)").
			WithArgs("prod-1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Main Product Query
		rows1 := sqlmock.NewRows([]string{
			"product_id", "name", "description", "sku", "price", "category_id",
			"stock_quantity", "search_vector", "created_at", "last_updated_at",
			"category_name", "tag", "details", "discount", "discount_type",
			"weight", "dimensions", "manufacturer", "weight_limit",
		}).AddRow(
			"prod-1", "Test Product", "Desc", "SKU1", 100.0, "cat-1",
			10, "", time.Now(), time.Now(),
			"Category", "Tag", nil, 0.0, "",
			1.0, "10x10", "Manu", 10.0,
		)
		mock.ExpectQuery("SELECT .* FROM products .* WHERE p.product_id = ?").
			WithArgs("prod-1").
			WillReturnRows(rows1)

		// Enrich calls (1st pass)
		mock.ExpectQuery("SELECT .* FROM product_images").WillReturnRows(sqlmock.NewRows([]string{"image_url"}))
		mock.ExpectQuery("SELECT .* FROM product_warranties").WillReturnRows(sqlmock.NewRows([]string{"warranty_id"}))
		mock.ExpectQuery("SELECT .* FROM product_features").WillReturnRows(sqlmock.NewRows([]string{"feature"}))

		// Variants
		mock.ExpectQuery("SELECT .* FROM product_variants").
			WillReturnRows(sqlmock.NewRows([]string{"variant_id", "variant_type", "name", "hex_code", "additional_price", "stock_quantity"}))

		// Tax
		mock.ExpectQuery("SELECT .* FROM product_charges").WillReturnRows(sqlmock.NewRows([]string{"charge_id", "charge_name", "charge_value"}))

		// --- 2. Second Call: checkStockAvailability -> GetProductByID ---

		// IsProductThere
		mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM products WHERE product_id = \\?\\)").
			WithArgs("prod-1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Main Product Query
		rows2 := sqlmock.NewRows([]string{
			"product_id", "name", "description", "sku", "price", "category_id",
			"stock_quantity", "search_vector", "created_at", "last_updated_at",
			"category_name", "tag", "details", "discount", "discount_type",
			"weight", "dimensions", "manufacturer", "weight_limit",
		}).AddRow(
			"prod-1", "Test Product", "Desc", "SKU1", 100.0, "cat-1",
			10, "", time.Now(), time.Now(),
			"Category", "Tag", nil, 0.0, "",
			1.0, "10x10", "Manu", 10.0,
		)
		mock.ExpectQuery("SELECT .* FROM products .* WHERE p.product_id = ?").
			WithArgs("prod-1").
			WillReturnRows(rows2)

		// Enrich calls (2nd pass)
		mock.ExpectQuery("SELECT .* FROM product_images").WillReturnRows(sqlmock.NewRows([]string{"image_url"}))
		mock.ExpectQuery("SELECT .* FROM product_warranties").WillReturnRows(sqlmock.NewRows([]string{"warranty_id"}))
		mock.ExpectQuery("SELECT .* FROM product_features").WillReturnRows(sqlmock.NewRows([]string{"feature"}))

		// Variants
		mock.ExpectQuery("SELECT .* FROM product_variants").
			WillReturnRows(sqlmock.NewRows([]string{"variant_id", "variant_type", "name", "hex_code", "additional_price", "stock_quantity"}))

		// Tax
		mock.ExpectQuery("SELECT .* FROM product_charges").WillReturnRows(sqlmock.NewRows([]string{"charge_id", "charge_name", "charge_value"}))

		// GetProductDiscount (called in processOrderItems)
		mock.ExpectQuery("SELECT dp.discount, dp.discount_type FROM deal_products dp").
			WithArgs("prod-1").
			WillReturnRows(sqlmock.NewRows([]string{"discount", "discount_type"}))

		// GetProductPromotionData
		mock.ExpectQuery("SELECT .* FROM promotion_products").
			WithArgs("prod-1").
			WillReturnRows(sqlmock.NewRows([]string{"promotion_type", "promotion_value"}))

		// Start Transaction
		mock.ExpectBegin()

		// Expect CreateOrder (INSERT INTO orders)
		mock.ExpectExec("INSERT INTO orders").
			WithArgs(sqlmock.AnyArg(), nil, true, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// CreateOrderItem Failure
		mock.ExpectExec("INSERT INTO order_items").
			WillReturnError(errors.New("db error"))

		// Expect Log Insert (triggered by respondInternalServerError)
		mock.ExpectExec("INSERT INTO logs").
			WithArgs(sqlmock.AnyArg(), "ERROR", "db error", sqlmock.AnyArg(), sqlmock.AnyArg(), "Orders", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect Rollback
		mock.ExpectRollback()

		// Prepare request
		body, _ := json.Marshal(validPayload)
		req, _ := http.NewRequest("POST", "/api/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		NewCreateOrderHandler(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
