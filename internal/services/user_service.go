package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"user-service/internal/database"
	"user-service/internal/metrics"
	"user-service/internal/models"
)

// UserService handles user-related business logic
type UserService struct {
	db      database.DBTX
	metrics *metrics.Metrics
}

// NewUserService creates a new user service with a database connection and metrics
func NewUserService(db database.DBTX, metricsCollector *metrics.Metrics) *UserService {
	return &UserService{
		db:      db,
		metrics: metricsCollector,
	}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(id int) (models.User, error) {
	var user models.User
	err := s.db.QueryRow(context.Background(), "SELECT id, name, email FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.metrics.RecordUserLookup("not_found")
			return models.User{}, fmt.Errorf("user not found")
		}
		return models.User{}, err
	}

	s.metrics.RecordUserLookup("found")
	return user, nil
}

// ListUsers returns all users
func (s *UserService) ListUsers() ([]models.User, error) {
	rows, err := s.db.Query(context.Background(), "SELECT id, name, email FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUsersCount returns the current number of users
func (s *UserService) GetUsersCount() (int, error) {
	var count int
	err := s.db.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// AddUser adds a new user (for future use)
func (s *UserService) AddUser(user models.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	_, err := s.db.Exec(context.Background(), "INSERT INTO users (name, email) VALUES ($1, $2)", user.Name, user.Email)
	if err != nil {
		return err
	}

	return nil
}
