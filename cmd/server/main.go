package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/marcelobritu/go-expert-desafio-rate-limiter/config"
	"github.com/marcelobritu/go-expert-desafio-rate-limiter/limiter"
	ratelimitMiddleware "github.com/marcelobritu/go-expert-desafio-rate-limiter/middleware"
	"github.com/marcelobritu/go-expert-desafio-rate-limiter/strategy"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Redis strategy
	redisStrategy := strategy.NewRedisStrategy(
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisStrategy.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	// Initialize rate limiter
	rateLimiter := limiter.NewRateLimiter(redisStrategy, cfg)

	// Setup Chi router
	router := chi.NewRouter()

	// Add standard middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoint (without rate limiting)
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
		})
	})

	// Rate limit info endpoint
	router.Route("/rate-limit", func(r chi.Router) {
		r.Use(ratelimitMiddleware.RateLimitInfoMiddleware(rateLimiter))
		r.Get("/info", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Rate limit information in headers",
				"headers": map[string]string{
					"X-RateLimit-Count":   w.Header().Get("X-RateLimit-Count"),
					"X-RateLimit-Reset":   w.Header().Get("X-RateLimit-Reset"),
					"X-RateLimit-Blocked": w.Header().Get("X-RateLimit-Blocked"),
				},
			})
		})
	})

	// Protected endpoints
	router.Route("/api", func(r chi.Router) {
		r.Use(ratelimitMiddleware.RateLimitMiddleware(rateLimiter))

		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "This is a protected endpoint",
				"ip":      getClientIP(r),
				"token":   r.Header.Get("API_KEY"),
				"time":    time.Now(),
			})
		})

		r.Post("/data", func(w http.ResponseWriter, r *http.Request) {
			var requestData map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid JSON",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Data received successfully",
				"data":    requestData,
				"ip":      getClientIP(r),
				"time":    time.Now(),
			})
		})

		r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "API is working",
				"rate_limit": map[string]string{
					"remaining": w.Header().Get("X-RateLimit-Remaining"),
					"reset":     w.Header().Get("X-RateLimit-Reset"),
				},
			})
		})
	})

	// Admin endpoints for testing
	router.Route("/admin", func(r chi.Router) {
		r.Post("/reset/{key}", func(w http.ResponseWriter, r *http.Request) {
			key := chi.URLParam(r, "key")
			if err := rateLimiter.ResetRateLimit(ctx, key); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Failed to reset rate limit",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Rate limit reset successfully",
				"key":     key,
			})
		})
	})

	// Start server
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server started on port %s", cfg.Server.Port)
	log.Println("Available endpoints:")
	log.Println("  GET  /health - Health check")
	log.Println("  GET  /rate-limit/info - Rate limit information")
	log.Println("  GET  /api/test - Test protected endpoint")
	log.Println("  POST /api/data - Test POST endpoint")
	log.Println("  GET  /api/status - API status")
	log.Println("  POST /admin/reset/{key} - Reset rate limit for key")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close Redis connection
	if err := redisStrategy.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}

	log.Println("Server exited")
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
