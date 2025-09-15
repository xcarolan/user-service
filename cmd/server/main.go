package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-service/internal/config"
	"user-service/internal/database"
	"user-service/internal/handlers"
	"user-service/internal/metrics"
	"user-service/internal/middleware"
	"user-service/internal/services"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting user service...")

	// Load configuration
	cfg := config.Load()

	// Initialize database connection
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close(context.Background())

	// Initialize metrics
	metricsCollector := metrics.New(nil, nil)
	slog.Info("Metrics initialized")

	// Create service
	userService := services.NewUserService(db, metricsCollector)

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
		slog.Info("Server starting", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-quit
	slog.Info("Received signal, shutting down gracefully...", "signal", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	} else {
		slog.Info("Server shutdown complete")
	}
}

func setupRoutes(userService *services.UserService, metricsCollector *metrics.Metrics, cfg *config.Config) *http.ServeMux {
	mux := http.NewServeMux()

	// Create handlers
	userHandler := handlers.NewUserHandler(userService)
	healthHandler := handlers.NewHealthHandler(userService)

	// Apply middleware chain
	var handler http.Handler = mux
	handler = middleware.RequestID()(handler)
	handler = middleware.Recovery(metricsCollector)(handler)
	handler = middleware.CORS()(handler)
	handler = middleware.RateLimit(cfg.GetRateLimiter(), metricsCollector)(handler)
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
