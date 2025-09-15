package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"user-service/internal/middleware"
	"user-service/internal/models"
	"user-service/internal/services"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUser handles GET /user requests
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(middleware.RequestIDKey).(string)

	// Extract and validate ID parameter
	idStr := r.URL.Query().Get("id")
	id, err := models.ParseUserID(idStr)
	if err != nil {
		slog.Warn("Invalid id parameter", "error", err, "id", idStr, "remote_addr", r.RemoteAddr, "request_id", requestID)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user from service
	user, err := h.userService.GetUser(id)
	if err != nil {
		slog.Warn("User not found", "id", id, "remote_addr", r.RemoteAddr, "request_id", requestID)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Set response headers and encode JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		slog.Error("Failed to encode user", "error", err, "id", id, "request_id", requestID)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	slog.Info("Successfully returned user", "id", id, "remote_addr", r.RemoteAddr, "request_id", requestID)
}

// ListUsers handles GET /users requests
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(middleware.RequestIDKey).(string)

	users, err := h.userService.ListUsers()
	if err != nil {
		slog.Error("Failed to list users", "error", err, "request_id", requestID)
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"users": users,
		"total": len(users),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode users list", "error", err, "request_id", requestID)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	slog.Info("Successfully returned users list", "count", len(users), "remote_addr", r.RemoteAddr, "request_id", requestID)
}
