package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FahadAkash/100M-Systems/internal/shortener"
	"github.com/FahadAkash/100M-Systems/internal/storage"
	"github.com/FahadAkash/100M-Systems/pkg/events"
	"github.com/FahadAkash/100M-Systems/pkg/metrics"
)

type Handler struct {
	service   *shortener.Service
	producer  events.ClickProducer
	analytics storage.AnalyticsStore
	baseURL   string
	log       *slog.Logger
}

func NewHandler(
	service *shortener.Service,
	producer events.ClickProducer,
	analytics storage.AnalyticsStore,
	baseURL string,
	log *slog.Logger,
) *Handler {
	if producer == nil {
		producer = &events.NoopProducer{}
	}
	return &Handler{
		service:   service,
		producer:  producer,
		analytics: analytics,
		baseURL:   strings.TrimRight(baseURL, "/"),
		log:       log,
	}
}

type shortenRequest struct {
	URL            string `json:"url"`
	CustomCode     string `json:"custom_code,omitempty"`
	ExpiresInHours int    `json:"expires_in_hours,omitempty"`
}

type shortenResponse struct {
	Code        string     `json:"code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) Shorten(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req shortenRequest
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	// Reject additional JSON values after the first object.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresInHours > 0 {
		exp := time.Now().UTC().Add(time.Duration(req.ExpiresInHours) * time.Hour)
		expiresAt = &exp
	}

	record, err := h.service.ShortenWithOptions(r.Context(), req.URL, req.CustomCode, expiresAt)
	if err != nil {
		if errors.Is(err, shortener.ErrInvalidURL) {
			writeError(w, http.StatusBadRequest, "invalid URL")
			return
		}
		if errors.Is(err, shortener.ErrInvalidAlias) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, shortener.ErrAliasTaken) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		h.log.Error("failed to shorten URL", "error", err)
		writeError(w, http.StatusInternalServerError, "could not shorten URL")
		return
	}

	metrics.URLShortenedTotal.Inc()

	writeJSON(w, http.StatusCreated, shortenResponse{
		Code:        record.Code,
		ShortURL:    h.baseURL + "/" + record.Code,
		OriginalURL: record.Original,
		ExpiresAt:   record.ExpiresAt,
	})
}

func (h *Handler) Redirect(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	code := r.PathValue("code")
	record, err := h.service.Resolve(r.Context(), code)

	if err != nil {
		metrics.URLRedirectsTotal.WithLabelValues("not_found").Inc()
		if errors.Is(err, storage.ErrNotFound) ||
			errors.Is(err, shortener.ErrInvalidCode) {
			writeError(w, http.StatusNotFound, "short URL not found")
			return
		}

		h.log.Error("failed to resolve URL", "error", err)
		writeError(w, http.StatusInternalServerError, "could not resolve URL")
		return
	}

	metrics.URLRedirectsTotal.WithLabelValues("found").Inc()

	// Emit asynchronous click event to Kafka streaming pipeline (non-blocking)
	if h.producer != nil {
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		} else if idx := strings.Index(clientIP, ","); idx != -1 {
			clientIP = strings.TrimSpace(clientIP[:idx])
		}

		h.producer.Publish(events.ClickEvent{
			Code:        record.Code,
			OriginalURL: record.Original,
			IP:          clientIP,
			UserAgent:   r.UserAgent(),
			Referer:     r.Referer(),
			Timestamp:   time.Now().UTC(),
		})
	}

	http.Redirect(w, r, record.Original, http.StatusFound)
}

func (h *Handler) GetURL(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	code := r.PathValue("code")
	record, err := h.service.Resolve(r.Context(), code)

	if err != nil {
		if errors.Is(err, storage.ErrNotFound) ||
			errors.Is(err, shortener.ErrInvalidCode) {
			writeError(w, http.StatusNotFound, "short URL not found")
			return
		}

		h.log.Error("failed to get URL", "error", err)
		writeError(w, http.StatusInternalServerError, "could not get URL")
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (h *Handler) GetAnalytics(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.analytics == nil {
		writeError(w, http.StatusNotImplemented, "analytics engine is not configured")
		return
	}

	code := r.PathValue("code")
	summary, err := h.analytics.GetAnalytics(r.Context(), code)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, shortener.ErrInvalidCode) {
			writeError(w, http.StatusNotFound, "short URL not found")
			return
		}

		h.log.Error("failed to get analytics", "code", code, "error", err)
		writeError(w, http.StatusInternalServerError, "could not retrieve analytics")
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
