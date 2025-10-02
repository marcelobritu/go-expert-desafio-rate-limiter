package limiter

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/marcelobritu/go-expert-desafio-rate-limiter/config"
	"github.com/marcelobritu/go-expert-desafio-rate-limiter/strategy"
)

// RateLimiter handles rate limiting logic
type RateLimiter struct {
	storage strategy.StorageStrategy
	config  *config.Config
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(storage strategy.StorageStrategy, config *config.Config) *RateLimiter {
	return &RateLimiter{
		storage: storage,
		config:  config,
	}
}

// CheckResult represents the result of a rate limit check
type CheckResult struct {
	Allowed   bool          `json:"allowed"`
	Remaining int           `json:"remaining"`
	ResetTime time.Time     `json:"reset_time"`
	BlockTime time.Duration `json:"block_time,omitempty"`
	Reason    string        `json:"reason,omitempty"`
}

// CheckIPRateLimit checks rate limit for an IP address
func (rl *RateLimiter) CheckIPRateLimit(ctx context.Context, ip string) (*CheckResult, error) {
	key := strategy.GetKeyWithPrefix("ip", ip)

	// Increment counter first (Redis will handle TTL automatically)
	newCount, err := rl.storage.Increment(ctx, key, time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to increment counter: %w", err)
	}

	// Check if limit is exceeded after increment
	if newCount > rl.config.RateLimit.IPLimit {
		// Return rate limit exceeded (no permanent blocking)
		now := time.Now()
		resetTime := now.Add(time.Second)

		return &CheckResult{
			Allowed:   false,
			Remaining: 0,
			ResetTime: resetTime,
			Reason:    "IP rate limit exceeded",
		}, nil
	}

	remaining := rl.config.RateLimit.IPLimit - newCount
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time (current time + 1 second)
	resetTime := time.Now().Add(time.Second)

	return &CheckResult{
		Allowed:   true,
		Remaining: remaining,
		ResetTime: resetTime,
	}, nil
}

// CheckTokenRateLimit checks rate limit for a token
func (rl *RateLimiter) CheckTokenRateLimit(ctx context.Context, token string) (*CheckResult, error) {
	key := strategy.GetKeyWithPrefix("token", token)

	// Get token-specific configuration
	tokenConfig, exists := rl.config.RateLimit.TokenLimits[token]
	if !exists {
		// Token not configured, use IP limits as fallback
		return nil, fmt.Errorf("token not configured")
	}

	// Increment counter first (Redis will handle TTL automatically)
	newCount, err := rl.storage.Increment(ctx, key, time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to increment counter: %w", err)
	}

	// Check if limit is exceeded after increment
	if newCount > tokenConfig.Limit {
		// Return rate limit exceeded (no permanent blocking)
		now := time.Now()
		resetTime := now.Add(time.Second)

		return &CheckResult{
			Allowed:   false,
			Remaining: 0,
			ResetTime: resetTime,
			Reason:    "Token rate limit exceeded",
		}, nil
	}

	remaining := tokenConfig.Limit - newCount
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time (current time + 1 second)
	resetTime := time.Now().Add(time.Second)

	return &CheckResult{
		Allowed:   true,
		Remaining: remaining,
		ResetTime: resetTime,
	}, nil
}

// CheckRateLimit checks rate limit for both IP and token, prioritizing token limits
func (rl *RateLimiter) CheckRateLimit(ctx context.Context, ip, token string) (*CheckResult, error) {
	// If token is provided, check token limits first
	if token != "" {
		log.Printf("Checking token rate limit for token: %s", token)
		tokenResult, err := rl.CheckTokenRateLimit(ctx, token)
		if err == nil {
			log.Printf("Token rate limit result: Allowed=%t, Remaining=%d", tokenResult.Allowed, tokenResult.Remaining)
			return tokenResult, nil
		}
		log.Printf("Token rate limit failed: %v, falling back to IP", err)
		// If token check fails (e.g., token not configured), fall back to IP check
	}

	// Check IP limits
	log.Printf("Checking IP rate limit for IP: %s", ip)
	return rl.CheckIPRateLimit(ctx, ip)
}

// ResetRateLimit resets rate limit for a specific key
func (rl *RateLimiter) ResetRateLimit(ctx context.Context, key string) error {
	return rl.storage.Delete(ctx, key)
}

// GetRateLimitInfo returns current rate limit information for a key
func (rl *RateLimiter) GetRateLimitInfo(ctx context.Context, key string) (*strategy.RateLimitInfo, error) {
	return rl.storage.Get(ctx, key)
}
