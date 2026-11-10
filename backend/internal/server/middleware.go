package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"plewtus/internal/model"
)

// responseWriterInterceptor captures the HTTP status code for logging.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterInterceptor) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs HTTP requests with method, path, status, and duration.
func LoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriterInterceptor{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		logger.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Duration("duration", time.Since(start)),
			slog.String("remote_addr", r.RemoteAddr),
		)
	})
}

// RecoveryMiddleware recovers from panics and returns a 500 Internal Server Error JSON response.
func RecoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					slog.Any("error", rec),
					slog.String("stack", string(debug.Stack())),
				)
				WriteJSON(w, http.StatusInternalServerError, model.ErrorResponse{
					Error:   "internal_server_error",
					Message: "An unexpected error occurred",
					Code:    http.StatusInternalServerError,
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// jsonErrorWriter rewrites the router's plain-text 404 and 405 responses as
// the API's standard JSON error payload.
type jsonErrorWriter struct {
	http.ResponseWriter
	intercepted bool
}

func (w *jsonErrorWriter) WriteHeader(code int) {
	isJSON := strings.HasPrefix(w.Header().Get("Content-Type"), "application/json")
	if (code == http.StatusNotFound || code == http.StatusMethodNotAllowed) && !isJSON {
		w.intercepted = true
		w.Header().Del("X-Content-Type-Options")
		token, message := "not_found", "No such endpoint. Available: POST /search, GET /health."
		if code == http.StatusMethodNotAllowed {
			token, message = "method_not_allowed", "Method not allowed for this endpoint."
		}
		WriteJSON(w.ResponseWriter, code, model.ErrorResponse{Error: token, Message: message, Code: code})
		return
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *jsonErrorWriter) Write(b []byte) (int, error) {
	if w.intercepted {
		return len(b), nil // drop the router's plain-text body
	}
	return w.ResponseWriter.Write(b)
}

// JSONErrorMiddleware makes unknown routes and methods return JSON errors.
func JSONErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&jsonErrorWriter{ResponseWriter: w}, r)
	})
}

// CORSMiddleware sets standard permissive CORS headers for AI agent and browser clients.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-API-Key")
		w.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, X-Quota-Limit, X-Quota-Remaining, X-Quota-Reset, Retry-After")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WriteJSON writes a JSON response with proper headers and status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
