package handlers

import (
	"encoding/json"
	"log"
	"net/http"

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
	// Extract and validate ID parameter
	idStr := r.URL.Query().Get("id")
	id, err := models.ParseUserID(idStr)
	if err != nil {
		log.Printf("Invalid id parameter '%s' from %s: %v", idStr, r.RemoteAddr, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user from service
	user, err := h.userService.GetUser(id)
	if err != nil {
		log.Printf("User %d not found, requested by %s", id, r.RemoteAddr)
		http.Error(w, err.Error(), http.StatusNotFound)
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

// ListUsers handles GET /users requests
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users := h.userService.ListUsers()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"users": users,
		"total": len(users),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode users list: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully returned %d users to %s", len(users), r.RemoteAddr)
}
