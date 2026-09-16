package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ekomasi_backend/dtos"

	"strings"

	"github.com/gin-gonic/gin"
)

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	ip := r.RemoteAddr
	if pos := strings.LastIndex(ip, ":"); pos != -1 {
		ip = ip[:pos]
	}
	return ip
}

// GinRateLimiter creates a native Gin Redis token-bucket rate limiter middleware
func GinRateLimiter(maxRequests int, windowSeconds int, keyPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if dtos.Redis == nil {
			c.Next()
			return
		}

		ip := getClientIP(c.Request)

		tenantID := TenantIDFromContext(c.Request.Context())
		redisKey := fmt.Sprintf("rate_limit:%s:tenant_%d:ip_%s", keyPrefix, tenantID, ip)

		ctx := context.Background()
		val, err := dtos.Redis.Incr(ctx, redisKey).Result()
		if err != nil {
			c.Next()
			return
		}

		if val == 1 {
			dtos.Redis.Expire(ctx, redisKey, time.Duration(windowSeconds)*time.Second)
		}

		if val > int64(maxRequests) {
			c.Header("Retry-After", fmt.Sprintf("%d", windowSeconds))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status_code": 429,
				"message":     fmt.Sprintf("Too many requests. Please try again in %d seconds.", windowSeconds),
				"module":      "RateLimiter",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimiter creates a Redis token-bucket rate limiter middleware
func RateLimiter(maxRequests int, windowSeconds int, keyPrefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if dtos.Redis == nil {
				// If Redis is disabled or offline, bypass rate limiting safely
				next.ServeHTTP(w, r)
				return
			}

			// Get IP address
			ip := getClientIP(r)

			tenantID := TenantIDFromContext(r.Context())
			redisKey := fmt.Sprintf("rate_limit:%s:tenant_%d:ip_%s", keyPrefix, tenantID, ip)

			ctx := context.Background()
			val, err := dtos.Redis.Incr(ctx, redisKey).Result()
			if err != nil {
				// Fail open on Redis error so legitimate traffic is not blocked
				next.ServeHTTP(w, r)
				return
			}

			if val == 1 {
				dtos.Redis.Expire(ctx, redisKey, time.Duration(windowSeconds)*time.Second)
			}

			if val > int64(maxRequests) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", windowSeconds))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write(fmt.Appendf(nil, `{"status_code":429,"message":"Too many requests. Please try again in %d seconds.","module":"RateLimiter"}`, windowSeconds))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
