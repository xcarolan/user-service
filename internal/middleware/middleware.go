package middleware

import (
	"log"
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
			log.Printf("%s %s %d %v %s",
				r.Method, r.URL.Path, wrapper.statusCode, duration, r.RemoteAddr)
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
				log.Printf("Rate limit exceeded for %s", r.RemoteAddr)
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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

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
					log.Printf("Panic recovered: %v", err)
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
