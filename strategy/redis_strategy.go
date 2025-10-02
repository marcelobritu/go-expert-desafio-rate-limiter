package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStrategy implements StorageStrategy using Redis
type RedisStrategy struct {
	client *redis.Client
}

// NewRedisStrategy creates a new Redis strategy instance
func NewRedisStrategy(host, port, password string, db int) *RedisStrategy {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})

	return &RedisStrategy{
		client: rdb,
	}
}

// Get retrieves rate limit information for a given key
func (r *RedisStrategy) Get(ctx context.Context, key string) (*RateLimitInfo, error) {
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return &RateLimitInfo{
				Count:     0,
				ResetTime: time.Now().Add(time.Second),
				Blocked:   false,
			}, nil
		}
		return nil, err
	}

	var info RateLimitInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// Set stores rate limit information for a given key with expiration
func (r *RedisStrategy) Set(ctx context.Context, key string, info *RateLimitInfo, expiration time.Duration) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, expiration).Err()
}

// Increment increments the count for a given key
func (r *RedisStrategy) Increment(ctx context.Context, key string, expiration time.Duration) (int, error) {
	// Use Redis pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Increment counter
	incrCmd := pipe.Incr(ctx, key)

	// Set expiration if this is the first increment
	pipe.Expire(ctx, key, expiration)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return int(incrCmd.Val()), nil
}

// SetBlocked sets a key as blocked until a specific time
func (r *RedisStrategy) SetBlocked(ctx context.Context, key string, blockUntil time.Time) error {
	blockKey := fmt.Sprintf("blocked:%s", key)
	blockDuration := time.Until(blockUntil)

	if blockDuration <= 0 {
		return nil
	}

	return r.client.Set(ctx, blockKey, "1", blockDuration).Err()
}

// IsBlocked checks if a key is currently blocked
func (r *RedisStrategy) IsBlocked(ctx context.Context, key string) (bool, time.Time, error) {
	blockKey := fmt.Sprintf("blocked:%s", key)

	ttl, err := r.client.TTL(ctx, blockKey).Result()
	if err != nil {
		return false, time.Time{}, err
	}

	if ttl <= 0 {
		return false, time.Time{}, nil
	}

	blockUntil := time.Now().Add(ttl)
	return true, blockUntil, nil
}

// Delete removes a key from storage
func (r *RedisStrategy) Delete(ctx context.Context, key string) error {
	blockKey := fmt.Sprintf("blocked:%s", key)

	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.Del(ctx, blockKey)

	_, err := pipe.Exec(ctx)
	return err
}

// Close closes the Redis connection
func (r *RedisStrategy) Close() error {
	return r.client.Close()
}

// Ping tests the Redis connection
func (r *RedisStrategy) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetKeyWithPrefix creates a key with a prefix for different types of rate limiting
func GetKeyWithPrefix(prefix, identifier string) string {
	return fmt.Sprintf("%s:%s", prefix, identifier)
}

// ParseTokenFromHeader extracts token from API_KEY header
func ParseTokenFromHeader(headerValue string) (string, error) {
	if headerValue == "" {
		return "", fmt.Errorf("empty header value")
	}

	// The header value is just the token itself
	return headerValue, nil
}
