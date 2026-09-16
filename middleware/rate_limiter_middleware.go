package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

// RateLimiterMiddleware creates a Redis token-bucket rate limiter for sensitive routes
// Param limit: Max allowed requests per window
// Param windowDuration: Rate limit window (e.g. 1 minute)
func RateLimiterMiddleware(limit int, windowDuration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		path := c.FullPath()
		key := fmt.Sprintf("rate_limit:%s:%s", path, ip)

		ctx := context.Background()
		if RedisClient == nil {
			c.Next()
			return
		}

		currentCount, err := RedisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if currentCount == 1 {
			RedisClient.Expire(ctx, key, windowDuration)
		}

		if currentCount > int64(limit) {
			c.Header("Retry-After", fmt.Sprintf("%d", int(windowDuration.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too Many Requests",
				"message":     fmt.Sprintf("Rate limit exceeded. Please wait %d seconds before retrying.", int(windowDuration.Seconds())),
				"retry_after": int(windowDuration.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
