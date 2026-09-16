package handlers

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

func AtomicGetAndDelete(ctx context.Context, key string) (string, error) {
	script := `
		local val = redis.call('GET', KEYS[1])
		if val then
			redis.call('DEL', KEYS[1])
		end
		return val
	`

	val, err := Redis.Eval(ctx, script, []string{key}).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}

	if val == nil {
		return "", nil
	}

	res, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("unexpected value type from Redis")
	}

	return res, nil
}
