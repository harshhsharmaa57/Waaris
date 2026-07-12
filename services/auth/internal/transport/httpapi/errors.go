package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/auth/internal/application"
)

type contextKey string

const principalKey contextKey = "principal"

func withPrincipal(ctx context.Context, principal application.Principal) context.Context {
	return context.WithValue(ctx, principalKey, principal)
}
func principalFromContext(ctx context.Context) application.Principal {
	principal, _ := ctx.Value(principalKey).(application.Principal)
	return principal
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := validCorrelationID(r.Header.Get("X-Correlation-Id"))
		w.Header().Set("X-Correlation-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey("correlation_id"), id)))
	})
}

func validCorrelationID(value string) string {
	if len(value) == 0 || len(value) > 128 {
		return uuid.NewString()
	}
	for _, character := range value {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-", character) {
			return uuid.NewString()
		}
	}
	return value
}
func correlationID(ctx context.Context) string {
	value, _ := ctx.Value(contextKey("correlation_id")).(string)
	return value
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message, "correlationId": correlationID(r.Context())})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(value []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(value)
}

func withHTTPProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		started := time.Now()
		recorder := &statusWriter{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("request panicked", "method", r.Method, "path", r.URL.Path, "correlation_id", correlationID(r.Context()))
				if recorder.status == 0 {
					writeError(recorder, r, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
				}
			}
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			slog.Info("request completed", "method", r.Method, "path", r.URL.Path, "status", status, "duration_ms", time.Since(started).Milliseconds(), "correlation_id", correlationID(r.Context()))
		}()
		next.ServeHTTP(recorder, r)
	})
}
