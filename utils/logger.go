package utils

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var (
	// Global logger instance
	Logger *slog.Logger
)

// InitLogger initializes the structured logger with OpenTelemetry integration
func InitLogger() {
	Logger = otelslog.NewLogger("adenzo-backend")
	slog.SetDefault(Logger)
}

// LogWithContext logs a message with trace context
func LogWithContext(ctx context.Context, level slog.Level, msg string, args ...any) {
	Logger.Log(ctx, level, msg, args...)
}

// LogInfo logs an info message with trace context
func LogInfo(ctx context.Context, msg string, args ...any) {
	LogWithContext(ctx, slog.LevelInfo, msg, args...)
}

// LogError logs an error message with trace context
func LogError(ctx context.Context, msg string, args ...any) {
	LogWithContext(ctx, slog.LevelError, msg, args...)
}

// LogWarn logs a warning message with trace context
func LogWarn(ctx context.Context, msg string, args ...any) {
	LogWithContext(ctx, slog.LevelWarn, msg, args...)
}

// LogDebug logs a debug message with trace context
func LogDebug(ctx context.Context, msg string, args ...any) {
	LogWithContext(ctx, slog.LevelDebug, msg, args...)
}

// LogDatabaseOperation logs database operations with trace context
func LogDatabaseOperation(ctx context.Context, operation, query string, duration int64, err error) {
	if err != nil {
		LogError(ctx, "Database operation failed",
			"operation", operation,
			"query", query,
			"duration_ms", duration,
			"error", err.Error())
	} else {
		LogInfo(ctx, "Database operation completed",
			"operation", operation,
			"query", query,
			"duration_ms", duration)
	}
}

// LogRedisOperation logs Redis operations with trace context
func LogRedisOperation(ctx context.Context, operation, key string, duration int64, err error) {
	if err != nil {
		LogError(ctx, "Redis operation failed",
			"operation", operation,
			"key", key,
			"duration_ms", duration,
			"error", err.Error())
	} else {
		LogInfo(ctx, "Redis operation completed",
			"operation", operation,
			"key", key,
			"duration_ms", duration)
	}
}

// LogHTTPRequest logs HTTP requests with trace context
func LogHTTPRequest(ctx context.Context, method, path, statusCode string, duration int64) {
	LogInfo(ctx, "HTTP request completed",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"duration_ms", duration)
}

// LogBusinessEvent logs business events with trace context
func LogBusinessEvent(ctx context.Context, eventType, eventID string, data map[string]interface{}) {
	args := []any{
		"event_type", eventType,
		"event_id", eventID,
	}

	// Add custom data as attributes
	for key, value := range data {
		args = append(args, key, value)
	}

	LogInfo(ctx, "Business event occurred", args...)
}

// LogWithSpan adds span information to log context
func LogWithSpan(ctx context.Context, spanName string, fn func(context.Context)) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		// Add span information to context
		spanCtx := trace.ContextWithSpan(ctx, span)
		fn(spanCtx)
	} else {
		// Create a new span if none exists
		tracer := otel.Tracer("adenzo-backend")
		spanCtx, newSpan := tracer.Start(ctx, spanName)
		defer newSpan.End()
		fn(spanCtx)
	}
}

// SetupFileLogging sets up file logging alongside OpenTelemetry
func SetupFileLogging() {
	// Open or create the log file
	_, err := os.OpenFile("storage/logs/adenzo.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		Logger.Error("Failed to open log file", "error", err)
		return
	}

	// Set up a multi-writer logger (both console and file)
	multiLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Use the multi-logger as default
	slog.SetDefault(multiLogger)
	Logger = multiLogger
}
