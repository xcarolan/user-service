package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-service/internal/config"
	"user-service/internal/handlers"
	"user-service/internal/metrics"
	"user-service/internal/middleware"
	"user-service/internal/services"
)

func main() {
	// Setup structured logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting user service...")

	// Load configuration
	cfg := config.Load()

	// Initialize metrics
	metricsCollector := metrics.New()
	log.Println("Metrics initialized")

	// Create service
	userService := services.NewUserService(metricsCollector)

	// Setup routes with middleware
	mux := setupRoutes(userService, metricsCollector, cfg)

	// Configure server
	server := &http.Server{
		Addr:           cfg.Port,
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

func setupRoutes(userService *services.UserService, metricsCollector *metrics.Metrics, cfg *config.Config) *http.ServeMux {
	mux := http.NewServeMux()

	// Create handlers
	userHandler := handlers.NewUserHandler(userService)
	healthHandler := handlers.NewHealthHandler(userService)

	// Apply middleware chain
	var handler http.Handler = mux
	handler = middleware.Recovery(metricsCollector)(handler)
	handler = middleware.CORS()(handler)
	handler = middleware.RateLimit(cfg.RateLimit, metricsCollector)(handler)
	handler = middleware.Metrics(metricsCollector)(handler)
	handler = middleware.Logging()(handler)

	// Register application routes
	mux.HandleFunc("/user", userHandler.GetUser)
	mux.HandleFunc("/users", userHandler.ListUsers)
	mux.HandleFunc("/health", healthHandler.Health)

	// Register metrics endpoint
	mux.Handle("/metrics", metricsCollector.Handler())

	// Wrap the final handler
	finalMux := http.NewServeMux()
	finalMux.Handle("/", handler)

	return finalMux
}
