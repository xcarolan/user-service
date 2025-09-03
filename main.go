package main

import (
	"context"
	"encoding/json"
	_ "fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Metrics structure to hold all Prometheus metrics
type Metrics struct {
	// HTTP request metrics
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge

	// Business metrics
	usersTotal  prometheus.Gauge
	userLookups *prometheus.CounterVec
	errorRate   *prometheus.CounterVec

	// System metrics
	rateLimitHits   prometheus.Counter
	panicRecoveries prometheus.Counter

	// Custom application metrics
	lastRequestTime prometheus.Gauge
	uptime          prometheus.Counter
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
			},
			[]string{"method", "endpoint"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
		usersTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "users_total",
				Help: "Total number of users in the system",
			},
		),
		userLookups: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_lookups_total",
				Help: "Total number of user lookup operations",
			},
			[]string{"result"}, // "found" or "not_found"
		),
		errorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "errors_total",
				Help: "Total number of errors by type",
			},
			[]string{"type", "endpoint"},
		),
		rateLimitHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "rate_limit_hits_total",
				Help: "Total number of rate limit violations",
			},
		),
		panicRecoveries: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "panic_recoveries_total",
				Help: "Total number of panic recoveries",
			},
		),
		lastRequestTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "last_request_time_seconds",
				Help: "Unix timestamp of the last request",
			},
		),
		uptime: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "uptime_seconds_total",
				Help: "Total uptime in seconds",
			},
		),
	}

	// Register all metrics with Prometheus
	prometheus.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
		m.usersTotal,
		m.userLookups,
		m.errorRate,
		m.rateLimitHits,
		m.panicRecoveries,
		m.lastRequestTime,
		m.uptime,
	)

	// Start uptime counter
	go m.updateUptime()

	return m
}

// Update uptime counter every second
func (m *Metrics) updateUptime() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.uptime.Inc()
	}
}

// UserService handles user-related operations with metrics
type UserService struct {
	mu      sync.RWMutex
	users   map[int]User
	metrics *Metrics
}

// NewUserService creates a new user service with initial data and metrics
func NewUserService(metrics *Metrics) *UserService {
	service := &UserService{
		users: map[int]User{
			1: {ID: 1, Name: "John Doe", Email: "john@example.com"},
			2: {ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
			3: {ID: 3, Name: "Bob Johnson", Email: "bob@example.com"},
		},
		metrics: metrics,
	}

	// Initialize users count metric
	service.metrics.usersTotal.Set(float64(len(service.users)))

	return service
}

// GetUser retrieves a user by ID with metrics collection
func (s *UserService) GetUser(w http.ResponseWriter, r *http.Request) {
	// Extract and validate ID parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("Missing id parameter from %s", r.RemoteAddr)
		s.metrics.errorRate.WithLabelValues("missing_parameter", "get_user").Inc()
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Invalid id parameter '%s' from %s", idStr, r.RemoteAddr)
		s.metrics.errorRate.WithLabelValues("invalid_parameter", "get_user").Inc()
		http.Error(w, "invalid id parameter", http.StatusBadRequest)
		return
	}

	// Thread-safe user lookup with metrics
	s.mu.RLock()
	user, exists := s.users[id]
	s.mu.RUnlock()

	if !exists {
		log.Printf("User %d not found, requested by %s", id, r.RemoteAddr)
		s.metrics.userLookups.WithLabelValues("not_found").Inc()
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Record successful lookup
	s.metrics.userLookups.WithLabelValues("found").Inc()

	// Set response headers and encode JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("Failed to encode user %d: %v", id, err)
		s.metrics.errorRate.WithLabelValues("encoding_error", "get_user").Inc()
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully returned user %d to %s", id, r.RemoteAddr)
}

// ListUsers returns all users with metrics
func (s *UserService) ListUsers(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"total": len(users),
	}); err != nil {
		log.Printf("Failed to encode users list: %v", err)
		s.metrics.errorRate.WithLabelValues("encoding_error", "list_users").Inc()
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully returned %d users to %s", len(users), r.RemoteAddr)
}

// Health check endpoint with metrics
func (s *UserService) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"service":     "user-service",
		"users_count": len(s.users),
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		return
	}
}

// Metrics middleware that wraps HTTP handlers
func metricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Track requests in flight
			metrics.requestsInFlight.Inc()
			defer metrics.requestsInFlight.Dec()

			// Update last request time
			metrics.lastRequestTime.SetToCurrentTime()

			// Create response writer wrapper to capture status code
			wrapper := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapper, r)

			// Record metrics after request completion
			duration := time.Since(start).Seconds()
			endpoint := r.URL.Path
			method := r.Method
			statusCode := strconv.Itoa(wrapper.statusCode)

			// Record request metrics
			metrics.requestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
			metrics.requestDuration.WithLabelValues(method, endpoint).Observe(duration)
		})
	}
}

// Response writer wrapper for metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Enhanced logging middleware with metrics
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapper, r)
		duration := time.Since(start)
		log.Printf("%s %s %d %v %s",
			r.Method, r.URL.Path, wrapper.statusCode, duration, r.RemoteAddr)
	})
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

// Rate limiting middleware with metrics
func rateLimitMiddleware(limiter *rate.Limiter, metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				log.Printf("Rate limit exceeded for %s", r.RemoteAddr)
				metrics.rateLimitHits.Inc()
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
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

// Recovery middleware with metrics
func recoveryMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Panic recovered: %v", err)
					metrics.panicRecoveries.Inc()
					metrics.errorRate.WithLabelValues("panic", r.URL.Path).Inc()
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Setup routes with metrics-enabled middleware
func setupRoutes(service *UserService, metrics *Metrics) *http.ServeMux {
	mux := http.NewServeMux()

	// Create rate limiter
	limiter := rate.NewLimiter(200, 20)

	// Apply middleware chain with metrics
	var handler http.Handler = mux
	handler = recoveryMiddleware(metrics)(handler)
	handler = corsMiddleware(handler)
	handler = rateLimitMiddleware(limiter, metrics)(handler)
	handler = metricsMiddleware(metrics)(handler)
	handler = loggingMiddleware(handler)

	// Register application routes
	mux.HandleFunc("/user", service.GetUser)
	mux.HandleFunc("/users", service.ListUsers)
	mux.HandleFunc("/health", service.HealthCheck)

	// Register metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Wrap the final handler
	finalMux := http.NewServeMux()
	finalMux.Handle("/", handler)

	return finalMux
}

func main() {
	// Setup structured logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting user service with metrics...")

	// Initialize metrics
	metrics := NewMetrics()
	log.Println("Metrics initialized")

	// Create service with metrics
	userService := NewUserService(metrics)

	// Setup routes with metrics-enabled middleware
	mux := setupRoutes(userService, metrics)

	// Configure server
	server := &http.Server{
		Addr:           ":8080",
		Handler:        mux,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", server.Addr)
		log.Printf("Metrics available at http://localhost%s/metrics", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-quit
	log.Printf("Received signal %v, shutting down gracefully...", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server shutdown complete")
	}
}
