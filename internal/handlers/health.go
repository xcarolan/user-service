package handlers

import (
	"encoding/json"
	"net/http"
	"time"

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
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"service":     "user-service",
		"users_count": h.userService.GetUsersCount(),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
