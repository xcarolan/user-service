package services

import (
	"fmt"
	"sync"

	"user-service/internal/metrics"
	"user-service/internal/models"
)

// UserService handles user-related business logic
type UserService struct {
	mu      sync.RWMutex
	users   map[int]models.User
	metrics *metrics.Metrics
}

// NewUserService creates a new user service with initial data and metrics
func NewUserService(metricsCollector *metrics.Metrics) *UserService {
	service := &UserService{
		users: map[int]models.User{
			1: {ID: 1, Name: "John Doe", Email: "john@example.com"},
			2: {ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
			3: {ID: 3, Name: "Bob Johnson", Email: "bob@example.com"},
			4: {ID: 4, Name: "Sylvester Carolan", Email: "sly@example.com"},
		},
		metrics: metricsCollector,
	}

	// Initialize users count metric
	service.metrics.SetUsersTotal(float64(len(service.users)))

	return service
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(id int) (models.User, error) {
	s.mu.RLock()
	user, exists := s.users[id]
	s.mu.RUnlock()

	if !exists {
		s.metrics.RecordUserLookup("not_found")
		return models.User{}, fmt.Errorf("user not found")
	}

	s.metrics.RecordUserLookup("found")
	return user, nil
}

// ListUsers returns all users
func (s *UserService) ListUsers() []models.User {
	s.mu.RLock()
	users := make([]models.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	s.mu.RUnlock()

	return users
}

// GetUsersCount returns the current number of users
func (s *UserService) GetUsersCount() int {
	s.mu.RLock()
	count := len(s.users)
	s.mu.RUnlock()
	return count
}

// AddUser adds a new user (for future use)
func (s *UserService) AddUser(user models.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	s.users[user.ID] = user
	s.metrics.SetUsersTotal(float64(len(s.users)))
	s.mu.Unlock()

	return nil
}
