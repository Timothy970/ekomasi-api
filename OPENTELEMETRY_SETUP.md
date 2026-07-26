# OpenTelemetry Setup for Ekomasi Backend

This document explains how OpenTelemetry is configured in the Ekomasi backend to export spans, logs, and metrics to Uptrace.

## Overview

The project now includes comprehensive OpenTelemetry instrumentation that captures:
- **Traces**: HTTP requests, database queries, Redis operations
- **Logs**: Structured logging with trace correlation
- **Metrics**: Business metrics, performance metrics, error rates

## Configuration

### Environment Variables

Add these environment variables to your `.env` file:

```bash
# Uptrace Configuration
UPTRACE_DSN=https://your-token@api.uptrace.dev?grpc=4317

# Environment
ENVIRONMENT=development

# Other existing variables...
```

### Uptrace Setup

1. Sign up for Uptrace at [uptrace.dev](https://uptrace.dev)
2. Create a new project
3. Copy your DSN from the project settings
4. Set the `UPTRACE_DSN` environment variable

## Features Implemented

### 1. HTTP Request Tracing
- Automatic tracing of all HTTP requests
- Request/response timing
- Status code tracking
- User agent and IP address capture

### 2. Database Instrumentation
- Wrapped database operations with tracing
- Query timing and error tracking
- Database connection monitoring

### 3. Redis Instrumentation
- Wrapped Redis operations with tracing
- Operation timing and error tracking
- Cache performance monitoring

### 4. Business Metrics
- Order creation tracking
- Product view metrics
- User registration metrics
- Cart operations
- Payment processing
- Error rates

### 5. Structured Logging
- OpenTelemetry-integrated logging
- Trace correlation in logs
- Structured JSON output

## Usage Examples

### Basic Tracing
```go
func MyHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Create a span
    ctx, span := tracer.Start(ctx, "my.operation")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(
        attribute.String("user.id", "123"),
        attribute.String("operation.type", "data_processing"),
    )
    
    // Your business logic here
    
    // Record metrics
    utils.RecordProductViewed(ctx, "product-123", "category-456")
}
```

### Database Operations
```go
// The wrapped database client automatically traces operations
result, err := models.DB.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "John")
```

### Redis Operations
```go
// The wrapped Redis client automatically traces operations
val, err := handlers.Redis.Get(ctx, "user:123").Result()
```

### Recording Metrics
```go
// Record business metrics
utils.RecordOrderCreated(ctx, "order-123", 99.99)
utils.RecordPaymentProcessed(ctx, "payment-456", "credit_card", 99.99, true)
utils.RecordError(ctx, "validation", "user_registration", err)
```

## Available Endpoints

### Health Check
```
GET /api/health
```
Returns system health status with trace information.

### Telemetry Example
```
GET /api/telemetry/example
```
Demonstrates OpenTelemetry features with sample operations.

## Monitoring

### Uptrace Dashboard
1. Log into your Uptrace account
2. Navigate to your project dashboard
3. View traces, metrics, and logs in real-time

### Key Metrics to Monitor
- HTTP request latency and error rates
- Database query performance
- Redis operation timing
- Business metrics (orders, users, etc.)
- Error rates by operation type

## Customization

### Adding Custom Metrics
Edit `utils/metrics.go` to add new business metrics:

```go
var customMetric, _ = meter.Int64Counter(
    "custom_metric_total",
    metric.WithDescription("Description of your metric"),
    metric.WithUnit("1"),
)

func RecordCustomMetric(ctx context.Context, value int64) {
    customMetric.Add(ctx, value)
}
```

### Adding Custom Spans
Use the tracer in your handlers:

```go
ctx, span := tracer.Start(ctx, "custom.operation")
defer span.End()

span.SetAttributes(
    attribute.String("custom.attribute", "value"),
)
```

## Troubleshooting

### Common Issues

1. **No traces appearing in Uptrace**
   - Check that `UPTRACE_DSN` is set correctly
   - Verify network connectivity to Uptrace
   - Check application logs for OpenTelemetry errors

2. **High memory usage**
   - Adjust batch processor settings in `main.go`
   - Reduce sampling rate if needed

3. **Missing database traces**
   - Ensure you're using the wrapped database client
   - Check that database operations are using context

### Debug Mode
Set `OTEL_LOG_LEVEL=debug` to see OpenTelemetry debug information.

## Performance Considerations

- OpenTelemetry adds minimal overhead (~1-2ms per request)
- Batch processing reduces network overhead
- Sampling can be configured to reduce data volume
- Metrics are collected asynchronously

## Next Steps

1. Set up alerting rules in Uptrace
2. Create custom dashboards for business metrics
3. Add more specific business metrics as needed
4. Consider adding distributed tracing across microservices
5. Set up log aggregation and analysis
