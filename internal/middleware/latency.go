package middleware

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"time"
)

// LatencySimulator injects simulated real-world network and data-processing latency
// so local LAN requests behave with realistic cloud/datacenter response times.
type LatencySimulator struct {
	enabled bool
	minMs   int
	maxMs   int
}

// NewLatencySimulator constructs a new LatencySimulator.
func NewLatencySimulator(enabled bool, minMs, maxMs int) *LatencySimulator {
	if minMs <= 0 {
		minMs = 15
	}
	if maxMs < minMs {
		maxMs = minMs + 25
	}
	return &LatencySimulator{
		enabled: enabled,
		minMs:   minMs,
		maxMs:   maxMs,
	}
}

// Middleware returns an http.Handler that simulates realistic production latency with random jitter.
func (ls *LatencySimulator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass latency simulation for health/monitoring endpoints
		if !ls.enabled || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		delta := int64(ls.maxMs - ls.minMs)
		var jitter int64
		if delta > 0 {
			n, err := rand.Int(rand.Reader, big.NewInt(delta+1))
			if err == nil {
				jitter = n.Int64()
			}
		}

		delay := time.Duration(int64(ls.minMs)+jitter) * time.Millisecond
		time.Sleep(delay)

		next.ServeHTTP(w, r)
	})
}

