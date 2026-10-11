package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FahadAkash/100M-Systems/pkg/metrics"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides distributed rate limiting backed by Redis with an in-memory token bucket fallback.
type RateLimiter struct {
	redisClient   *redis.Client
	maxRequests   int
	windowSeconds int
	inMemoryStore map[string]*clientLimiter
	mu            sync.Mutex
}

func NewRateLimiter(rdb *redis.Client, maxRequests int, windowSeconds int) *RateLimiter {
	rl := &RateLimiter{
		redisClient:   rdb,
		maxRequests:   maxRequests,
		windowSeconds: windowSeconds,
		inMemoryStore: make(map[string]*clientLimiter),
	}

	// Periodically cleanup inactive in-memory entries
	go rl.cleanupLoop()

	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-5 * time.Minute)
		for ip, cl := range rl.inMemoryStore {
			if cl.lastSeen.Before(cutoff) {
				delete(rl.inMemoryStore, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) getMemoryLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cl, exists := rl.inMemoryStore[ip]
	if !exists {
		// Rate in requests per second
		r := rate.Limit(float64(rl.maxRequests) / float64(rl.windowSeconds))
		cl = &clientLimiter{
			limiter:  rate.NewLimiter(r, rl.maxRequests),
			lastSeen: time.Now(),
		}
		rl.inMemoryStore[ip] = cl
	} else {
		cl.lastSeen = time.Now()
	}

	return cl.limiter
}

// Limit wraps an http.Handler with rate limiting enforcement.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Exclude Prometheus scraper endpoint from rate limits
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := getClientIP(r)

		// 1. Redis Distributed Rate Limiting (if available)
		if rl.redisClient != nil {
			now := time.Now().Unix()
			windowBucket := now / int64(rl.windowSeconds)
			resetTime := (windowBucket + 1) * int64(rl.windowSeconds)
			key := fmt.Sprintf("ratelimit:%s:%d", clientIP, windowBucket)

			pipe := rl.redisClient.Pipeline()
			incrCmd := pipe.Incr(r.Context(), key)
			pipe.Expire(r.Context(), key, time.Duration(rl.windowSeconds*2)*time.Second)
			_, err := pipe.Exec(r.Context())

			if err == nil {
				count := int(incrCmd.Val())
				remaining := rl.maxRequests - count
				if remaining < 0 {
					remaining = 0
				}

				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.maxRequests))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))

				if count > rl.maxRequests {
					retryAfter := resetTime - now
					if retryAfter <= 0 {
						retryAfter = 1
					}
					w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
					metrics.RateLimitedTotal.WithLabelValues(metrics.NormalizePath(r.URL.Path)).Inc()
					writeRateLimitError(w, "rate limit exceeded, please retry later")
					return
				}

				next.ServeHTTP(w, r)
				return
			}
			// If Redis had a connection error, fall through to in-memory limiter gracefully
		}

		// 2. In-Memory Token Bucket Fallback
		limiter := rl.getMemoryLimiter(clientIP)
		if !limiter.Allow() {
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.maxRequests))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", strconv.Itoa(rl.windowSeconds))
			metrics.RateLimitedTotal.WithLabelValues(metrics.NormalizePath(r.URL.Path)).Inc()
			writeRateLimitError(w, "rate limit exceeded, please retry later")
			return
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.maxRequests))
		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeRateLimitError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

