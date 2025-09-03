package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserService handles user-related operations
type UserService struct {
	mu    sync.RWMutex
	users map[int]User
}

// NewUserService creates a new user service with initial data
func NewUserService() *UserService {
	return &UserService{
		users: map[int]User{
			1: {ID: 1, Name: "John Doe", Email: "john@example.com"},
			2: {ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
			3: {ID: 3, Name: "Bob Johnson", Email: "bob@example.com"},
		},
	}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(w http.ResponseWriter, r *http.Request) {
	// Extract and validate ID parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("Missing id parameter from %s", r.RemoteAddr)
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Invalid id parameter '%s' from %s", idStr, r.RemoteAddr)
		http.Error(w, "invalid id parameter", http.StatusBadRequest)
		return
	}

	// Thread-safe user lookup
	s.mu.RLock()
	user, exists := s.users[id]
	s.mu.RUnlock()

	if !exists {
		log.Printf("User %d not found, requested by %s", id, r.RemoteAddr)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Set response headers and encode JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("Failed to encode user %d: %v", id, err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully returned user %d to %s", id, r.RemoteAddr)
}

// ListUsers returns all users (with pagination support)
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
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully returned %d users to %s", len(users), r.RemoteAddr)
}

// Health check endpoint
func (s *UserService) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "user-service",
	}
	json.NewEncoder(w).Encode(response)
}

// Middleware for logging requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer that captures the status code
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

// Rate limiting middleware
func rateLimitMiddleware(limiter *rate.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				log.Printf("Rate limit exceeded for %s", r.RemoteAddr)
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware for cross-origin requests
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

// Recovery middleware to handle panics
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// setupRoutes configures all routes with middleware
func setupRoutes(service *UserService) *http.ServeMux {
	mux := http.NewServeMux()

	// Create rate limiter (10 requests per second, burst of 20)
	limiter := rate.NewLimiter(10, 20)

	// Apply middleware chain
	var handler http.Handler = mux
	handler = recoveryMiddleware(handler)
	handler = corsMiddleware(handler)
	handler = rateLimitMiddleware(limiter)(handler)
	handler = loggingMiddleware(handler)

	// Register routes
	mux.HandleFunc("/user", service.GetUser)
	mux.HandleFunc("/users", service.ListUsers)
	mux.HandleFunc("/health", service.HealthCheck)

	// Wrap the final handler
	finalMux := http.NewServeMux()
	finalMux.Handle("/", handler)

	return finalMux
}

func main() {
	// Setup structured logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting user service...")

	// Create service
	userService := NewUserService()

	// Setup routes with middleware
	mux := setupRoutes(userService)

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
