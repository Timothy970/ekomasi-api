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
func SetCache(key string, value interface{}, customExpiration ...time.Duration) error {
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
func GetCache(key string, dest interface{}) error {
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
	return RedisClient.Del(ctx, key).Err()
}

// Delete by prefix
func DeleteCacheByPrefix(prefix string) error {
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
