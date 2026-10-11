package middleware

import (
	"log/slog"
	"net/http"
)

func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter, r *http.Request) {
			log.Info("Incoming request",
				"method", r.Method,
				"url", r.URL.String(),
				"remote_addr", r.RemoteAddr,
			)
			next.ServeHTTP(w, r)
		})
	}
}

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("Panic occurred",
						"error", err,
						"method", r.Method,
						"url", r.URL.String(),
						"remote_addr", r.RemoteAddr,
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
