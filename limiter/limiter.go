package limiter

import (
	"context"
	"fmt"
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

	// Check if IP is currently blocked
	blocked, blockUntil, err := rl.storage.IsBlocked(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to check if IP is blocked: %w", err)
	}

	if blocked {
		return &CheckResult{
			Allowed:   false,
			Remaining: 0,
			ResetTime: blockUntil,
			BlockTime: time.Until(blockUntil),
			Reason:    "IP is currently blocked",
		}, nil
	}

	// Get current rate limit info
	info, err := rl.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit info: %w", err)
	}

	// Check if we need to reset the counter (new time window)
	now := time.Now()
	if now.After(info.ResetTime) {
		// Reset counter for new time window
		info.Count = 0
		info.ResetTime = now.Add(time.Second)
	}

	// Check if limit is exceeded
	if info.Count >= rl.config.RateLimit.IPLimit {
		// Block the IP
		blockDuration := rl.config.RateLimit.IPBlockTime
		blockUntil := now.Add(blockDuration)

		if err := rl.storage.SetBlocked(ctx, key, blockUntil); err != nil {
			return nil, fmt.Errorf("failed to block IP: %w", err)
		}

		return &CheckResult{
			Allowed:   false,
			Remaining: 0,
			ResetTime: blockUntil,
			BlockTime: blockDuration,
			Reason:    "IP rate limit exceeded",
		}, nil
	}

	// Increment counter
	newCount, err := rl.storage.Increment(ctx, key, time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to increment counter: %w", err)
	}

	remaining := rl.config.RateLimit.IPLimit - newCount
	if remaining < 0 {
		remaining = 0
	}

	return &CheckResult{
		Allowed:   true,
		Remaining: remaining,
		ResetTime: info.ResetTime,
	}, nil
}

// CheckTokenRateLimit checks rate limit for a token
func (rl *RateLimiter) CheckTokenRateLimit(ctx context.Context, token string) (*CheckResult, error) {
	key := strategy.GetKeyWithPrefix("token", token)

	// Check if token is currently blocked
	blocked, blockUntil, err := rl.storage.IsBlocked(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to check if token is blocked: %w", err)
	}

	if blocked {
		return &CheckResult{
			Allowed:   false,
			Remaining: 0,
			ResetTime: blockUntil,
			BlockTime: time.Until(blockUntil),
			Reason:    "Token is currently blocked",
		}, nil
	}

	// Get token-specific configuration
	tokenConfig, exists := rl.config.RateLimit.TokenLimits[token]
	if !exists {
		// Token not configured, use IP limits as fallback
		return nil, fmt.Errorf("token not configured")
	}

	// Get current rate limit info
	info, err := rl.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit info: %w", err)
	}

	// Check if we need to reset the counter (new time window)
	now := time.Now()
	if now.After(info.ResetTime) {
		// Reset counter for new time window
		info.Count = 0
		info.ResetTime = now.Add(time.Second)
	}

	// Check if limit is exceeded
	if info.Count >= tokenConfig.Limit {
		// Block the token
		blockDuration := tokenConfig.BlockTime
		blockUntil := now.Add(blockDuration)

		if err := rl.storage.SetBlocked(ctx, key, blockUntil); err != nil {
			return nil, fmt.Errorf("failed to block token: %w", err)
		}

		return &CheckResult{
			Allowed:   false,
			Remaining: 0,
			ResetTime: blockUntil,
			BlockTime: blockDuration,
			Reason:    "Token rate limit exceeded",
		}, nil
	}

	// Increment counter
	newCount, err := rl.storage.Increment(ctx, key, time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to increment counter: %w", err)
	}

	remaining := tokenConfig.Limit - newCount
	if remaining < 0 {
		remaining = 0
	}

	return &CheckResult{
		Allowed:   true,
		Remaining: remaining,
		ResetTime: info.ResetTime,
	}, nil
}

// CheckRateLimit checks rate limit for both IP and token, prioritizing token limits
func (rl *RateLimiter) CheckRateLimit(ctx context.Context, ip, token string) (*CheckResult, error) {
	// If token is provided, check token limits first
	if token != "" {
		tokenResult, err := rl.CheckTokenRateLimit(ctx, token)
		if err == nil {
			return tokenResult, nil
		}
		// If token check fails (e.g., token not configured), fall back to IP check
	}

	// Check IP limits
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
