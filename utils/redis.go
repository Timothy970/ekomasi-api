package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

// RedisClient is a global redis client from main.go
var RedisClient *redis.Client

// ExpirationTime holds the default expiration time for cache entries.
var ExpirationTime = 60 * time.Minute

// SetCache stores any struct or value in Redis with expiration
var SetCache = func(key string, value any, customExpiration ...time.Duration) error {
	if RedisClient == nil {
		return nil
	}
	// Determine expiration
	exp := ExpirationTime

	if len(customExpiration) > 0 {
		// Use the passed expiration
		exp = customExpiration[0]
	} else {
		// Otherwise check environment
		redisMinutesStr := os.Getenv("REDIS_TIME")
		if redisMinutesStr != "" {
			minutes, err := strconv.Atoi(redisMinutesStr)
			if err != nil {
				log.Printf("Warning: Invalid value for REDIS_TIME: '%s'. Using default of %v.", redisMinutesStr, ExpirationTime)
			} else {
				exp = time.Duration(minutes) * time.Minute
			}
		}
	}

	// Marshal the value
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Store with expiration
	return RedisClient.Set(ctx, key, data, exp).Err()
}

// GetCache retrieves a value from Redis and unmarshals into the provided destination
var GetCache = func(key string, dest any) error {
	if RedisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	data, err := RedisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found")
		}
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}
	return nil
}

// DeleteCache invalidates a key from Redis
func DeleteCache(key string) error {
	if RedisClient == nil {
		return nil
	}
	return RedisClient.Del(ctx, key).Err()
}

// Delete by prefix
var DeleteCacheByPrefix = func(prefix string) error {
	if RedisClient == nil {
		return nil
	}
	var cursor uint64
	var keys []string
	var err error

	for {
		var batch []string
		// Use SCAN instead of KEYS to avoid blocking Redis on large datasets
		batch, cursor, err = RedisClient.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}

		if len(batch) > 0 {
			keys = append(keys, batch...)
		}

		if cursor == 0 {
			break
		}
	}

	if len(keys) > 0 {
		if err := RedisClient.Del(ctx, keys...).Err(); err != nil {
			return err
		}
	}

	return nil
}

// CheckRateLimit checks if the given key has exceeded the limit within the window.
// It returns (isAllowed, retryAfterSeconds, error).
func CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	if RedisClient == nil {
		return true, 0, nil // Bypass if Redis not available
	}

	redisKey := "rate_limit:" + key

	// Lua script to increment and set expiry atomically if new
	luaScript := `
		       local current
		       current = redis.call("INCR", KEYS[1])
		       if tonumber(current) == 1 then
			   redis.call("PEXPIRE", KEYS[1], ARGV[1])
		       end
		       return current
	       `

	// Convert window to milliseconds
	windowMs := int(window / time.Millisecond)
	result, err := RedisClient.Eval(ctx, luaScript, []string{redisKey}, windowMs).Result()
	if err != nil {
		return true, 0, err // Fail open on Redis error
	}

	count, ok := result.(int64)
	if !ok {
		// Try to convert if returned as another type
		switch v := result.(type) {
		case int:
			count = int64(v)
		case string:
			count, _ = strconv.ParseInt(v, 10, 64)
		}
	}

	if int(count) > limit {
		ttl, _ := RedisClient.TTL(ctx, redisKey).Result()
		return false, int(ttl.Seconds()), nil
	}

	return true, 0, nil
}

func ClearRateLimit(ctx context.Context, key string) error {
	if RedisClient == nil {
		return nil // Nothing to clear if Redis not available
	}
	return RedisClient.Del(ctx, "rate_limit:"+key).Err()
}

// // Store data
// if err := redisutils.SetCache("user:1", user, 10*time.Minute); err != nil {
// 	fmt.Println("Error storing:", err)
// }

// // Retrieve data
// var cachedUser User
// if err := redisutils.GetCache("user:1", &cachedUser); err != nil {
// 	fmt.Println("Error retrieving:", err)
// } else {
// 	fmt.Println("Retrieved user:", cachedUser)
// }

// // Delete key
// if err := redisutils.DeleteCache("user:1"); err != nil {
// 	fmt.Println("Error deleting:", err)
// } else {
// 	fmt.Println("Key deleted")
// }
