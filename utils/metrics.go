package utils

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("ekomasi-backend")

// Business metrics
var (
	// HTTP request metrics
	httpRequestsTotal, _ = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)

	httpRequestDuration, _ = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)

	// Database metrics
	dbQueriesTotal, _ = meter.Int64Counter(
		"db_queries_total",
		metric.WithDescription("Total number of database queries"),
		metric.WithUnit("1"),
	)

	dbQueryDuration, _ = meter.Float64Histogram(
		"db_query_duration_seconds",
		metric.WithDescription("Database query duration in seconds"),
		metric.WithUnit("s"),
	)

	// Redis metrics
	redisOperationsTotal, _ = meter.Int64Counter(
		"redis_operations_total",
		metric.WithDescription("Total number of Redis operations"),
		metric.WithUnit("1"),
	)

	redisOperationDuration, _ = meter.Float64Histogram(
		"redis_operation_duration_seconds",
		metric.WithDescription("Redis operation duration in seconds"),
		metric.WithUnit("s"),
	)

	// Business metrics
	ordersCreated, _ = meter.Int64Counter(
		"orders_created_total",
		metric.WithDescription("Total number of orders created"),
		metric.WithUnit("1"),
	)

	productsViewed, _ = meter.Int64Counter(
		"products_viewed_total",
		metric.WithDescription("Total number of product views"),
		metric.WithUnit("1"),
	)

	usersRegistered, _ = meter.Int64Counter(
		"users_registered_total",
		metric.WithDescription("Total number of user registrations"),
		metric.WithUnit("1"),
	)

	cartItemsAdded, _ = meter.Int64Counter(
		"cart_items_added_total",
		metric.WithDescription("Total number of items added to cart"),
		metric.WithUnit("1"),
	)

	paymentsProcessed, _ = meter.Int64Counter(
		"payments_processed_total",
		metric.WithDescription("Total number of payments processed"),
		metric.WithUnit("1"),
	)

	// Error metrics
	errorsTotal, _ = meter.Int64Counter(
		"errors_total",
		metric.WithDescription("Total number of errors"),
		metric.WithUnit("1"),
	)
)

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(ctx context.Context, method, path, statusCode string, duration float64) {
	attrs := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.route", path),
		attribute.String("http.status_code", statusCode),
	}

	httpRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	httpRequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
}

// RecordDatabaseQuery records database query metrics
func RecordDatabaseQuery(ctx context.Context, operation, table string, duration float64, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("db.operation", operation),
		attribute.String("db.table", table),
	}

	if err != nil {
		attrs = append(attrs, attribute.String("error", err.Error()))
	}

	dbQueriesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	dbQueryDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
}

// RecordRedisOperation records Redis operation metrics
func RecordRedisOperation(ctx context.Context, operation, key string, duration float64, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("redis.operation", operation),
		attribute.String("redis.key", key),
	}

	if err != nil {
		attrs = append(attrs, attribute.String("error", err.Error()))
	}

	redisOperationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	redisOperationDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
}

// RecordOrderCreated records order creation metrics
func RecordOrderCreated(ctx context.Context, orderID string, amount float64) {
	attrs := []attribute.KeyValue{
		attribute.String("order.id", orderID),
		attribute.Float64("order.amount", amount),
	}

	ordersCreated.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordProductViewed records product view metrics
func RecordProductViewed(ctx context.Context, productID string, categoryID string) {
	attrs := []attribute.KeyValue{
		attribute.String("product.id", productID),
		attribute.String("product.category_id", categoryID),
	}

	productsViewed.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordUserRegistered records user registration metrics
func RecordUserRegistered(ctx context.Context, userID string, method string) {
	attrs := []attribute.KeyValue{
		attribute.String("user.id", userID),
		attribute.String("registration.method", method),
	}

	usersRegistered.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCartItemAdded records cart item addition metrics
func RecordCartItemAdded(ctx context.Context, userID, productID string, quantity int64) {
	attrs := []attribute.KeyValue{
		attribute.String("user.id", userID),
		attribute.String("product.id", productID),
		attribute.Int64("quantity", quantity),
	}

	cartItemsAdded.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordPaymentProcessed records payment processing metrics
func RecordPaymentProcessed(ctx context.Context, paymentID, method string, amount float64, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("payment.id", paymentID),
		attribute.String("payment.method", method),
		attribute.Float64("payment.amount", amount),
		attribute.Bool("payment.success", success),
	}

	paymentsProcessed.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordError records error metrics
func RecordError(ctx context.Context, errorType, operation string, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("error.type", errorType),
		attribute.String("error.operation", operation),
		attribute.String("error.message", err.Error()),
	}

	errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}
