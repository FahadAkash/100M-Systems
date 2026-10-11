package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestsTotal counts all HTTP requests processed by method, normalized path, and status code.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed by method, normalized path, and status code.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks latency of HTTP requests in seconds.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestsInFlight tracks currently executing requests.
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of in-flight HTTP requests.",
		},
	)

	// URLShortenedTotal counts total short links generated.
	URLShortenedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_urls_created_total",
			Help: "Total number of shortened URLs generated.",
		},
	)

	// URLRedirectsTotal counts redirects executed by status.
	URLRedirectsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_shortener_redirects_total",
			Help: "Total number of short link redirections handled.",
		},
		[]string{"status"},
	)

	// CacheHitsTotal tracks Redis cache hits.
	CacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_cache_hits_total",
			Help: "Total number of Redis cache hits for URL lookups.",
		},
	)

	// CacheMissesTotal tracks Redis cache misses.
	CacheMissesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_cache_misses_total",
			Help: "Total number of Redis cache misses requiring DB fetch.",
		},
	)

	// RateLimitedTotal counts requests rejected due to rate limiting.
	RateLimitedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_rate_limited_total",
			Help: "Total number of HTTP requests rejected due to rate limiting.",
		},
		[]string{"path"},
	)

	// KafkaEventsProducedTotal counts click events published to Kafka.
	KafkaEventsProducedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_events_produced_total",
			Help: "Total number of click events published to Kafka topic.",
		},
	)

	// KafkaEventsConsumedTotal counts click events consumed and batched to database.
	KafkaEventsConsumedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_events_consumed_total",
			Help: "Total number of click events consumed from Kafka and batched to database.",
		},
	)
)

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// NormalizePath avoids Prometheus cardinality explosion by grouping dynamic route patterns.
func NormalizePath(path string) string {
	if path == "/api/shorten" {
		return "/api/shorten"
	}
	if strings.HasPrefix(path, "/api/urls/") {
		if strings.HasSuffix(path, "/analytics") {
			return "/api/urls/:code/analytics"
		}
		return "/api/urls/:code"
	}
	if path == "/metrics" {
		return "/metrics"
	}
	if path != "/" && !strings.Contains(path[1:], "/") {
		return "/:code"
	}
	return "other"
}

// Middleware wraps an http.Handler with Prometheus request tracking metrics.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		start := time.Now()
		rw := &statusResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		normPath := NormalizePath(r.URL.Path)
		statusStr := strconv.Itoa(rw.statusCode)

		HTTPRequestsTotal.WithLabelValues(r.Method, normPath, statusStr).Inc()
		HTTPRequestDuration.WithLabelValues(r.Method, normPath, statusStr).Observe(duration)
	})
}

// Handler returns the HTTP handler for the Prometheus metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

