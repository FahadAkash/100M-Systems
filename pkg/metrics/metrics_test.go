package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/api/shorten", "/api/shorten"},
		{"/api/urls/abc12345", "/api/urls/:code"},
		{"/api/urls/test", "/api/urls/:code"},
		{"/metrics", "/metrics"},
		{"/abc12345", "/:code"},
		{"/xyz", "/:code"},
		{"/", "other"},
		{"/a/b/c", "other"},
	}

	for _, tt := range tests {
		got := NormalizePath(tt.input)
		if got != tt.want {
			t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMetricsMiddlewareAndHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /metrics", Handler())

	handler := Middleware(mux)

	// 1. Call test endpoint
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// 2. Scrape /metrics endpoint
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsW := httptest.NewRecorder()
	handler.ServeHTTP(metricsW, metricsReq)

	if metricsW.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /metrics, got %d", metricsW.Code)
	}

	body := metricsW.Body.String()
	if !contains(body, "http_requests_total") {
		t.Errorf("expected metrics output to contain http_requests_total")
	}
	if !contains(body, "http_request_duration_seconds") {
		t.Errorf("expected metrics output to contain http_request_duration_seconds")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && searchSubstr(s, substr))
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

