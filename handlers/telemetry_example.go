package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/utils"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("ekomasi-backend")

// ExampleHandler demonstrates OpenTelemetry usage
func ExampleHandler(c *gin.Context) {
	ctx := c.Request.Context()

	// Create a span for this operation
	ctx, span := tracer.Start(ctx, "example.operation",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("operation.type", "example"),
			attribute.String("user.agent", c.Request.UserAgent()),
		))
	defer span.End()

	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	// Add attributes to the span
	span.SetAttributes(
		attribute.String("operation.status", "processing"),
		attribute.Int("operation.duration_ms", 100),
	)

	// Example database operation with tracing
	utils.LogInfo(ctx, "Starting database operation example")
	if err := exampleDatabaseOperation(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		utils.RecordError(ctx, "database", "example_operation", err)
		utils.LogError(ctx, "Database operation failed", "error", err)
		c.String(http.StatusInternalServerError, "Database error")
		return
	}
	utils.LogInfo(ctx, "Database operation completed successfully")

	// Example Redis operation with tracing
	utils.LogInfo(ctx, "Starting Redis operation example")
	if err := exampleRedisOperation(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		utils.RecordError(ctx, "redis", "example_operation", err)
		utils.LogError(ctx, "Redis operation failed", "error", err)
		c.String(http.StatusInternalServerError, "Redis error")
		return
	}
	utils.LogInfo(ctx, "Redis operation completed successfully")

	// Record business metrics
	utils.RecordProductViewed(ctx, "example-product-123", "example-category")
	utils.RecordUserRegistered(ctx, "example-user-456", "api")

	// Set span status to success
	span.SetStatus(codes.Ok, "Operation completed successfully")
	span.SetAttributes(attribute.String("operation.status", "completed"))

	// Return response
	response := map[string]any{
		"message":   "Example operation completed",
		"timestamp": time.Now().Unix(),
		"trace_id":  span.SpanContext().TraceID().String(),
	}

	c.JSON(http.StatusOK, response)
}

// exampleDatabaseOperation demonstrates database tracing
func exampleDatabaseOperation(ctx context.Context) error {
	// This would typically use your actual database connection
	// For this example, we'll simulate a database operation

	ctx, span := tracer.Start(ctx, "db.example_query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "mysql"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "users"),
		))
	defer span.End()

	// Simulate database work
	time.Sleep(50 * time.Millisecond)

	// Record metrics
	utils.RecordDatabaseQuery(ctx, "SELECT", "users", 0.05, nil)

	span.SetStatus(codes.Ok, "")
	return nil
}

// exampleRedisOperation demonstrates Redis tracing
func exampleRedisOperation(ctx context.Context) error {
	// This would typically use your actual Redis connection
	// For this example, we'll simulate a Redis operation

	ctx, span := tracer.Start(ctx, "redis.example_operation",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "GET"),
			attribute.String("db.key", "example:key"),
		))
	defer span.End()

	// Simulate Redis work
	time.Sleep(25 * time.Millisecond)

	// Record metrics
	utils.RecordRedisOperation(ctx, "GET", "example:key", 0.025, nil)

	span.SetStatus(codes.Ok, "")
	return nil
}

// HealthCheckHandler provides a health check endpoint with telemetry
func HealthCheckHandler(c *gin.Context) {
	ctx := c.Request.Context()

	ctx, span := tracer.Start(ctx, "health.check",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// Check database health
	dbHealthy := checkDatabaseHealth(ctx)

	// Check Redis health
	redisHealthy := checkRedisHealth(ctx)

	// Determine overall health
	healthy := dbHealthy && redisHealthy
	status := "healthy"
	if !healthy {
		status = "unhealthy"
		span.SetStatus(codes.Error, "Health check failed")
	} else {
		span.SetStatus(codes.Ok, "Health check passed")
	}

	response := map[string]any{
		"status":    status,
		"timestamp": time.Now().Unix(),
		"checks": map[string]bool{
			"database": dbHealthy,
			"redis":    redisHealthy,
		},
		"trace_id": span.SpanContext().TraceID().String(),
	}

	statusCode := http.StatusOK
	if !healthy {
		statusCode = http.StatusServiceUnavailable
	}
	c.JSON(statusCode, response)
}

// checkDatabaseHealth checks database connectivity
func checkDatabaseHealth(ctx context.Context) bool {
	ctx, span := tracer.Start(ctx, "health.db_check",
		trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// This would use your actual database connection
	// For this example, we'll simulate a ping
	time.Sleep(10 * time.Millisecond)

	span.SetStatus(codes.Ok, "")
	return true
}

// checkRedisHealth checks Redis connectivity
func checkRedisHealth(ctx context.Context) bool {
	ctx, span := tracer.Start(ctx, "health.redis_check",
		trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// This would use your actual Redis connection
	// For this example, we'll simulate a ping
	time.Sleep(5 * time.Millisecond)

	span.SetStatus(codes.Ok, "")
	return true
}
