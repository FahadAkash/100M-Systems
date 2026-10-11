package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_InMemory(t *testing.T) {
	// Set limit: 3 requests per 60 seconds
	limiter := NewRateLimiter(nil, 3, 60)

	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// 1. First 3 requests should pass
	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d expected status 200, got %d", i, w.Code)
		}
	}

	// 2. 4th request should be throttled (429 Too Many Requests)
	req4 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req4.RemoteAddr = "192.0.2.1:12345"
	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, req4)

	if w4.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 on 4th request, got %d", w4.Code)
	}

	if w4.Header().Get("Retry-After") == "" {
		t.Errorf("expected Retry-After header to be set")
	}

	// 3. Different IP should still be allowed
	reqOther := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqOther.RemoteAddr = "192.0.2.2:54321"
	wOther := httptest.NewRecorder()
	handler.ServeHTTP(wOther, reqOther)

	if wOther.Code != http.StatusOK {
		t.Errorf("expected status 200 for different IP, got %d", wOther.Code)
	}

	// 4. /metrics endpoint should bypass rate limiting
	for i := 0; i < 5; i++ {
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsReq.RemoteAddr = "192.0.2.1:12345"
		metricsW := httptest.NewRecorder()
		handler.ServeHTTP(metricsW, metricsReq)

		if metricsW.Code != http.StatusOK {
			t.Fatalf("/metrics expected status 200, got %d", metricsW.Code)
		}
	}
}

