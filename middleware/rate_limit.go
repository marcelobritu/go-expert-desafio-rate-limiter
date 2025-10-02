package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/marcelobritu/go-expert-desafio-rate-limiter/limiter"
	"github.com/marcelobritu/go-expert-desafio-rate-limiter/strategy"
)

// RateLimitMiddleware creates a rate limiting middleware for go-chi
func RateLimitMiddleware(rateLimiter *limiter.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()

			// Get client IP
			clientIP := getClientIP(r)

			// Get token from header
			token := ""
			if apiKey := r.Header.Get("API_KEY"); apiKey != "" {
				var err error
				token, err = strategy.ParseTokenFromHeader(apiKey)
				if err != nil {
					// Invalid token format, continue with IP-only rate limiting
					token = ""
				}
			}

			// Check rate limit
			result, err := rateLimiter.CheckRateLimit(ctx, clientIP, token)
			if err != nil {
				// Log error but don't block the request
				w.Header().Set("X-RateLimit-Error", "Rate limit check failed")
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
			w.Header().Set("X-RateLimit-Reset", result.ResetTime.Format(time.RFC3339))

			if result.BlockTime > 0 {
				w.Header().Set("X-RateLimit-Block-Time", result.BlockTime.String())
			}

			// Check if request is allowed
			if !result.Allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)

				response := map[string]interface{}{
					"error":   "Rate limit exceeded",
					"message": "you have reached the maximum number of requests or actions allowed within a certain time frame",
					"details": map[string]interface{}{
						"reason":     result.Reason,
						"reset_time": result.ResetTime,
						"block_time": result.BlockTime,
					},
				}

				json.NewEncoder(w).Encode(response)
				return
			}

			// Request is allowed, continue
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitInfoMiddleware provides rate limit information without blocking
func RateLimitInfoMiddleware(rateLimiter *limiter.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()

			// Get client IP
			clientIP := getClientIP(r)

			// Get token from header
			token := ""
			if apiKey := r.Header.Get("API_KEY"); apiKey != "" {
				var err error
				token, err = strategy.ParseTokenFromHeader(apiKey)
				if err != nil {
					token = ""
				}
			}

			// Get rate limit info without incrementing
			var info *strategy.RateLimitInfo
			var err error

			if token != "" {
				key := strategy.GetKeyWithPrefix("token", token)
				info, err = rateLimiter.GetRateLimitInfo(ctx, key)
			} else {
				key := strategy.GetKeyWithPrefix("ip", clientIP)
				info, err = rateLimiter.GetRateLimitInfo(ctx, key)
			}

			if err == nil && info != nil {
				w.Header().Set("X-RateLimit-Count", fmt.Sprintf("%d", info.Count))
				w.Header().Set("X-RateLimit-Reset", info.ResetTime.Format(time.RFC3339))
				w.Header().Set("X-RateLimit-Blocked", fmt.Sprintf("%t", info.Blocked))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := len(xff); idx > 0 {
			for i, c := range xff {
				if c == ',' {
					idx = i
					break
				}
			}
			return xff[:idx]
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := len(ip); idx > 0 {
		for i, c := range ip {
			if c == ':' {
				idx = i
				break
			}
		}
		return ip[:idx]
	}

	return ip
}
