package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FahadAkash/100M-Systems/internal/shortener"
	"github.com/FahadAkash/100M-Systems/internal/storage"
)

func setupTestHandler() (*Handler, *http.ServeMux) {
	store := storage.NewMemoryStore()
	svc := shortener.NewService(store)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(svc, "http://localhost:8080", logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shorten", handler.Shorten)
	mux.HandleFunc("GET /api/urls/{code}", handler.GetURL)
	mux.HandleFunc("GET /{code}", handler.Redirect)

	return handler, mux
}

func TestHandler_Shorten(t *testing.T) {
	_, mux := setupTestHandler()

	// 1. Success case
	body := bytes.NewBufferString(`{"url":"https://golang.org"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp shortenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code == "" || resp.OriginalURL != "https://golang.org" {
		t.Errorf("unexpected response: %+v", resp)
	}

	// 2. Invalid URL case
	badBody := bytes.NewBufferString(`{"url":"invalid-url"}`)
	badReq := httptest.NewRequest(http.MethodPost, "/api/shorten", badBody)
	badW := httptest.NewRecorder()

	mux.ServeHTTP(badW, badReq)

	if badW.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for bad URL, got %d", badW.Code)
	}
}

func TestHandler_RedirectAndGet(t *testing.T) {
	_, mux := setupTestHandler()

	// Shorten first
	body := bytes.NewBufferString(`{"url":"https://google.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp shortenResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	// Test Redirect
	redirectReq := httptest.NewRequest(http.MethodGet, "/"+resp.Code, nil)
	redirectW := httptest.NewRecorder()
	mux.ServeHTTP(redirectW, redirectReq)

	if redirectW.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", redirectW.Code)
	}
	if loc := redirectW.Header().Get("Location"); loc != "https://google.com" {
		t.Errorf("expected Location https://google.com, got %s", loc)
	}

	// Test GetURL
	getReq := httptest.NewRequest(http.MethodGet, "/api/urls/"+resp.Code, nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", getW.Code)
	}

	// Test Not Found
	notFoundReq := httptest.NewRequest(http.MethodGet, "/api/urls/nonexist", nil)
	notFoundW := httptest.NewRecorder()
	mux.ServeHTTP(notFoundW, notFoundReq)

	if notFoundW.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", notFoundW.Code)
	}
}

