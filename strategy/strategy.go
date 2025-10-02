package strategy

import (
	"context"
	"time"
)

// RateLimitInfo holds information about rate limiting for a key
type RateLimitInfo struct {
	Count      int       `json:"count"`
	ResetTime  time.Time `json:"reset_time"`
	Blocked    bool      `json:"blocked"`
	BlockUntil time.Time `json:"block_until,omitempty"`
}

// StorageStrategy defines the interface for different storage mechanisms
type StorageStrategy interface {
	// Get retrieves rate limit information for a given key
	Get(ctx context.Context, key string) (*RateLimitInfo, error)

	// Set stores rate limit information for a given key with expiration
	Set(ctx context.Context, key string, info *RateLimitInfo, expiration time.Duration) error

	// Increment increments the count for a given key
	Increment(ctx context.Context, key string, expiration time.Duration) (int, error)

	// SetBlocked sets a key as blocked until a specific time
	SetBlocked(ctx context.Context, key string, blockUntil time.Time) error

	// IsBlocked checks if a key is currently blocked
	IsBlocked(ctx context.Context, key string) (bool, time.Time, error)

	// Delete removes a key from storage
	Delete(ctx context.Context, key string) error

	// Close closes the storage connection
	Close() error
}
