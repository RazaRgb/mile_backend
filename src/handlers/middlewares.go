package handlers

import (
	"backend/src/auth"
	"backend/src/utils"
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// withLogging logs method, path, status, and duration for every request.
func WithLogging(next http.Handler) http.Handler {
	loggingHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration", time.Since(start).String(),
			)
		},
	)
	return loggingHandler
}

// authUserKey is the context key for the authenticated user's claims.
type authUserKey struct{}

// WithAuth requires a valid `Authorization: Bearer <token>` header. On success
// the claims are stored in the request context (see UserFromContext).
func WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		tokenStr, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || strings.TrimSpace(tokenStr) == "" {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}

		claims, err := auth.ParseToken(strings.TrimSpace(tokenStr))
		if err != nil {
			slog.Warn("rejected invalid token", "error", err)
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
			return
		}

		ctx := context.WithValue(r.Context(), authUserKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext returns the authenticated user's claims, if present.
func UserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(authUserKey{}).(*auth.Claims)
	return claims, ok
}

// WithCORS sets permissive CORS headers so the mobile/desktop Wails clients
// can call the API cross-origin (there is no dev proxy on mobile). Allowed
// origins come from CORS_ALLOWED_ORIGIN (comma separated); defaults to * for
// development.
func WithCORS(next http.Handler) http.Handler {
	allowed := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowed == "" {
		allowed = "*"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		// Short-circuit browser preflight requests.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// turns panics into 500 responses instead of crashing the server
func WithRecovery(next http.Handler) http.Handler {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
	return handler
}

// status writer wraps response writer to capture the written status code (check withLogging())
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
