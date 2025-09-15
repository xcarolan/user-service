package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
	"user-service/internal/metrics"
)

// Logging middleware
func Logging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapper, r)
			duration := time.Since(start)

			requestID, _ := r.Context().Value(RequestIDKey).(string)

			slog.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapper.statusCode,
				"duration", duration,
				"remote_addr", r.RemoteAddr,
				"request_id", requestID,
			)
		})
	}
}

// Metrics middleware
func Metrics(metricsCollector *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Track requests in flight
			metricsCollector.RecordRequestInFlight(1)
			defer metricsCollector.RecordRequestInFlight(-1)

			// Update last request time
			metricsCollector.UpdateLastRequestTime()

			// Create response writer wrapper to capture status code
			wrapper := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapper, r)

			// Record metrics after request completion
			duration := time.Since(start)
			endpoint := r.URL.Path
			method := r.Method
			statusCode := strconv.Itoa(wrapper.statusCode)

			// Record request metrics
			metricsCollector.RecordRequest(method, endpoint, statusCode, duration)
		})
	}
}

// RateLimit middleware
func RateLimit(limiter *rate.Limiter, metricsCollector *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				slog.Warn("Rate limit exceeded", "remote_addr", r.RemoteAddr)
				metricsCollector.RecordRateLimitHit()
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware
func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Recovery middleware
func Recovery(metricsCollector *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID, _ := r.Context().Value(RequestIDKey).(string)
					slog.Error("Panic recovered", "error", err, "request_id", requestID)
					metricsCollector.RecordPanicRecovery()
					metricsCollector.RecordError("panic", r.URL.Path)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Response writer wrapper to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Metrics response writer wrapper
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
