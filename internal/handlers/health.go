package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"user-service/internal/middleware"
	"user-service/internal/services"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	userService *services.UserService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(userService *services.UserService) *HealthHandler {
	return &HealthHandler{
		userService: userService,
	}
}

// Health handles GET /health requests
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(middleware.RequestIDKey).(string)

	w.Header().Set("Content-Type", "application/json")

	usersCount, err := h.userService.GetUsersCount()
	if err != nil {
		slog.Error("Failed to get users count for health check", "error", err, "request_id", requestID)
		http.Error(w, "Failed to get users count", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"service":     "user-service",
		"users_count": usersCount,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode health response", "error", err, "request_id", requestID)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
