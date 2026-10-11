package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/FahadAkash/100M-Systems/internal/shortener"
	"github.com/FahadAkash/100M-Systems/internal/storage"
)

type Handler struct {
	service *shortener.Service
	baseURL string
	log     *slog.Logger
}

func NewHandler(
	service *shortener.Service,
	baseURL string,
	log *slog.Logger,
) *Handler {
	return &Handler{
		service: service,
		baseURL: strings.TrimRight(baseURL, "/"),
		log:     log,
	}
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code        string `json:"code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
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

	record, err := h.service.Shorten(r.Context(), req.URL)
	if err != nil {
		if errors.Is(err, shortener.ErrInvalidURL) {
			writeError(w, http.StatusBadRequest, "invalid URL")
			return
		}

		h.log.Error("failed to shorten URL", "error", err)
		writeError(w, http.StatusInternalServerError, "could not shorten URL")
		return
	}

	writeJSON(w, http.StatusCreated, shortenResponse{
		Code:        record.Code,
		ShortURL:    h.baseURL + "/" + record.Code,
		OriginalURL: record.Original,
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
		if errors.Is(err, storage.ErrNotFound) ||
			errors.Is(err, shortener.ErrInvalidCode) {
			writeError(w, http.StatusNotFound, "short URL not found")
			return
		}

		h.log.Error("failed to resolve URL", "error", err)
		writeError(w, http.StatusInternalServerError, "could not resolve URL")
		return
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

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
