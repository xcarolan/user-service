package services

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"user-service/internal/metrics"
	"user-service/internal/models"
)

func TestUserService(t *testing.T) {
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.New(reg, reg)
	userService := NewUserService(metricsCollector)

	t.Run("get user", func(t *testing.T) {
		user, err := userService.GetUser(1)
		if err != nil {
			t.Errorf("unexpected error getting user: %v", err)
		}
		if user.ID != 1 {
			t.Errorf("expected user ID to be 1, got %d", user.ID)
		}
	})

	t.Run("get non-existent user", func(t *testing.T) {
		_, err := userService.GetUser(100)
		if err == nil {
			t.Error("expected an error getting non-existent user, got nil")
		}
	})

	t.Run("list users", func(t *testing.T) {
		users := userService.ListUsers()
		if len(users) != 4 {
			t.Errorf("expected 3 users, got %d", len(users))
		}
	})

	t.Run("get users count", func(t *testing.T) {
		count := userService.GetUsersCount()
		if count != 4 {
			t.Errorf("expected 3 users, got %d", count)
		}
	})

	t.Run("add user", func(t *testing.T) {
		user := models.User{ID: 5, Name: "Test User", Email: "test@user.com"}
		err := userService.AddUser(user)
		if err != nil {
			t.Errorf("unexpected error adding user: %v", err)
		}

		count := userService.GetUsersCount()
		if count != 5 {
			t.Errorf("expected 5 users, got %d", count)
		}
	})
}
