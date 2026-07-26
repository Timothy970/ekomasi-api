package utils

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("ekomasi-backend")

// DatabaseTracing wraps database operations with OpenTelemetry tracing
func DatabaseTracing(ctx context.Context, operation string, query string, fn func() error) error {
	ctx, span := tracer.Start(ctx, "db."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "mysql"),
			attribute.String("db.operation", operation),
			attribute.String("db.statement", query),
		))
	defer span.End()

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// RedisTracing wraps Redis operations with OpenTelemetry tracing
func RedisTracing(ctx context.Context, operation string, key string, fn func() error) error {
	ctx, span := tracer.Start(ctx, "redis."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", operation),
			attribute.String("db.key", key),
		))
	defer span.End()

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// WrappedDB wraps sql.DB with tracing
type WrappedDB struct {
	*sql.DB
}

// NewWrappedDB creates a new wrapped database connection
func NewWrappedDB(db *sql.DB) *WrappedDB {
	return &WrappedDB{DB: db}
}

// Exec wraps sql.DB.Exec with tracing
func (w *WrappedDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error
	start := time.Now()

	err = DatabaseTracing(context.Background(), "exec", query, func() error {
		result, err = w.DB.Exec(query, args...)
		return err
	})

	duration := time.Since(start).Milliseconds()
	LogDatabaseOperation(context.Background(), "exec", query, duration, err)

	return result, err
}

// Query wraps sql.DB.Query with tracing
func (w *WrappedDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error
	start := time.Now()

	err = DatabaseTracing(context.Background(), "query", query, func() error {
		rows, err = w.DB.Query(query, args...)
		return err
	})

	duration := time.Since(start).Milliseconds()
	LogDatabaseOperation(context.Background(), "query", query, duration, err)

	return rows, err
}

// QueryRow wraps sql.DB.QueryRow with tracing
func (w *WrappedDB) QueryRow(query string, args ...interface{}) *sql.Row {
	var row *sql.Row
	start := time.Now()

	DatabaseTracing(context.Background(), "query_row", query, func() error {
		row = w.DB.QueryRow(query, args...)
		return row.Err()
	})

	duration := time.Since(start).Milliseconds()
	LogDatabaseOperation(context.Background(), "query_row", query, duration, row.Err())

	return row
}

// Prepare wraps sql.DB.Prepare with tracing
func (w *WrappedDB) Prepare(query string) (*sql.Stmt, error) {
	var stmt *sql.Stmt
	var err error
	start := time.Now()

	err = DatabaseTracing(context.Background(), "prepare", query, func() error {
		stmt, err = w.DB.Prepare(query)
		return err
	})

	duration := time.Since(start).Milliseconds()
	LogDatabaseOperation(context.Background(), "prepare", query, duration, err)

	return stmt, err
}

// PrepareContext wraps sql.DB.PrepareContext with tracing
func (w *WrappedDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	var stmt *sql.Stmt
	var err error
	start := time.Now()

	err = DatabaseTracing(ctx, "prepare", query, func() error {
		stmt, err = w.DB.PrepareContext(ctx, query)
		return err
	})

	duration := time.Since(start).Milliseconds()
	LogDatabaseOperation(ctx, "prepare", query, duration, err)

	return stmt, err
}

// ExecContext wraps sql.DB.ExecContext with tracing
func (w *WrappedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error
	start := time.Now()

	err = DatabaseTracing(ctx, "exec", query, func() error {
		result, err = w.DB.ExecContext(ctx, query, args...)
		return err
	})

	duration := time.Since(start).Milliseconds()
	LogDatabaseOperation(ctx, "exec", query, duration, err)

	return result, err
}

// QueryContext wraps sql.DB.QueryContext with tracing
func (w *WrappedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error
	start := time.Now()

	err = DatabaseTracing(ctx, "query", query, func() error {
		rows, err = w.DB.QueryContext(ctx, query, args...)
		return err
	})

	duration := time.Since(start).Milliseconds()
	LogDatabaseOperation(ctx, "query", query, duration, err)

	return rows, err
}

// QueryRowContext wraps sql.DB.QueryRowContext with tracing
func (w *WrappedDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	var row *sql.Row
	start := time.Now()

	DatabaseTracing(ctx, "query_row", query, func() error {
		row = w.DB.QueryRowContext(ctx, query, args...)
		return row.Err()
	})

	duration := time.Since(start).Milliseconds()
	LogDatabaseOperation(ctx, "query_row", query, duration, row.Err())

	return row
}

// WrappedRedis wraps redis.Client with tracing
type WrappedRedis struct {
	*redis.Client
}

// Get wraps redis.Client.Get with tracing
func (w *WrappedRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := w.Client.Get(ctx, key)
	start := time.Now()

	RedisTracing(ctx, "get", key, func() error {
		return cmd.Err()
	})

	duration := time.Since(start).Milliseconds()
	LogRedisOperation(ctx, "get", key, duration, cmd.Err())

	return cmd
}

// Set wraps redis.Client.Set with tracing
func (w *WrappedRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := w.Client.Set(ctx, key, value, expiration)
	start := time.Now()

	RedisTracing(ctx, "set", key, func() error {
		return cmd.Err()
	})

	duration := time.Since(start).Milliseconds()
	LogRedisOperation(ctx, "set", key, duration, cmd.Err())

	return cmd
}

// Del wraps redis.Client.Del with tracing
func (w *WrappedRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := w.Client.Del(ctx, keys...)
	start := time.Now()

	RedisTracing(ctx, "del", keys[0], func() error {
		return cmd.Err()
	})

	duration := time.Since(start).Milliseconds()
	LogRedisOperation(ctx, "del", keys[0], duration, cmd.Err())

	return cmd
}

// HGet wraps redis.Client.HGet with tracing
func (w *WrappedRedis) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	cmd := w.Client.HGet(ctx, key, field)
	start := time.Now()

	RedisTracing(ctx, "hget", key, func() error {
		return cmd.Err()
	})

	duration := time.Since(start).Milliseconds()
	LogRedisOperation(ctx, "hget", key, duration, cmd.Err())

	return cmd
}

// HSet wraps redis.Client.HSet with tracing
func (w *WrappedRedis) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	cmd := w.Client.HSet(ctx, key, values...)
	start := time.Now()

	RedisTracing(ctx, "hset", key, func() error {
		return cmd.Err()
	})

	duration := time.Since(start).Milliseconds()
	LogRedisOperation(ctx, "hset", key, duration, cmd.Err())

	return cmd
}

// HDel wraps redis.Client.HDel with tracing
func (w *WrappedRedis) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	cmd := w.Client.HDel(ctx, key, fields...)
	start := time.Now()

	RedisTracing(ctx, "hdel", key, func() error {
		return cmd.Err()
	})

	duration := time.Since(start).Milliseconds()
	LogRedisOperation(ctx, "hdel", key, duration, cmd.Err())

	return cmd
}
