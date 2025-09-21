package services

import (
	"context"
	"testing"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"user-service/internal/database/mocks"
	"user-service/internal/metrics"
	"user-service/internal/models"
)

func TestUserService(t *testing.T) {
	dbMock := &mocks.MockDBTX{}
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.New(reg, reg)
	userService := NewUserService(dbMock, metricsCollector)

	t.Run("get user", func(t *testing.T) {
		row := &mocks.MockRow{}
		row.On("Scan", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(0).([]interface{})
			*arg[0].(*int) = 1
			*arg[1].(*string) = "John Doe"
			*arg[2].(*string) = "john@example.com"
		})

		dbMock.On("QueryRow", context.Background(), "SELECT id, name, email FROM users WHERE id = $1", 1).Return(row)

		user, err := userService.GetUser(1)
		assert.NoError(t, err)
		assert.Equal(t, 1, user.ID)
		dbMock.AssertExpectations(t)
	})

	t.Run("get non-existent user", func(t *testing.T) {
		row := &mocks.MockRow{}
		row.On("Scan", mock.Anything).Return(pgx.ErrNoRows)
		dbMock.On("QueryRow", context.Background(), "SELECT id, name, email FROM users WHERE id = $1", 100).Return(row)

		_, err := userService.GetUser(100)
		assert.Error(t, err)
		dbMock.AssertExpectations(t)
	})

	t.Run("list users", func(t *testing.T) {
		rows := &mocks.MockRows{}
		rows.On("Close").Return()
		rows.On("Next").Return(true).Once()
		rows.On("Next").Return(true).Once()
		rows.On("Next").Return(false).Once()
		rows.On("Scan", mock.Anything).Return(nil).Times(2)

		dbMock.On("Query", context.Background(), "SELECT id, name, email FROM users").Return(rows, nil)

		users, err := userService.ListUsers()
		assert.NoError(t, err)
		assert.Len(t, users, 2)
		dbMock.AssertExpectations(t)
	})

	t.Run("get users count", func(t *testing.T) {
		row := &mocks.MockRow{}
		row.On("Scan", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(0).([]interface{})
			*arg[0].(*int) = 5
		})
		dbMock.On("QueryRow", context.Background(), "SELECT COUNT(*) FROM users").Return(row)

		count, err := userService.GetUsersCount()
		assert.NoError(t, err)
		assert.Equal(t, 5, count)
		dbMock.AssertExpectations(t)
	})

	t.Run("add user", func(t *testing.T) {
		dbMock.On("Exec", context.Background(), "INSERT INTO users (name, email) VALUES ($1, $2)", "Test User", "test@user.com").Return(pgconn.CommandTag{}, nil)

		user := models.User{Name: "Test User", Email: "test@user.com"}
		err := userService.AddUser(user)
		assert.NoError(t, err)
		dbMock.AssertExpectations(t)
	})

	t.Run("add user validation error", func(t *testing.T) {
		// Test with invalid user data - create separate service to avoid mock conflicts
		dbMockValidation := &mocks.MockDBTX{}
		userServiceValidation := NewUserService(dbMockValidation, metricsCollector)
		user := models.User{Name: "", Email: "invalid-email"} // Empty name and invalid email
		err := userServiceValidation.AddUser(user)
		assert.Error(t, err)
		// Should not call database since validation fails
	})

	t.Run("add user database error", func(t *testing.T) {
		dbMockAddError := &mocks.MockDBTX{}
		userServiceAddError := NewUserService(dbMockAddError, metricsCollector)
		dbMockAddError.On("Exec", context.Background(), "INSERT INTO users (name, email) VALUES ($1, $2)", "Test User", "test@example.com").Return(pgconn.CommandTag{}, assert.AnError)

		user := models.User{Name: "Test User", Email: "test@example.com"}
		err := userServiceAddError.AddUser(user)
		assert.Error(t, err)
		dbMockAddError.AssertExpectations(t)
	})

	t.Run("get user database error", func(t *testing.T) {
		dbMockGetError := &mocks.MockDBTX{}
		userServiceGetError := NewUserService(dbMockGetError, metricsCollector)
		row := &mocks.MockRow{}
		row.On("Scan", mock.Anything).Return(assert.AnError)
		dbMockGetError.On("QueryRow", context.Background(), "SELECT id, name, email FROM users WHERE id = $1", 999).Return(row)

		_, err := userServiceGetError.GetUser(999)
		assert.Error(t, err)
		dbMockGetError.AssertExpectations(t)
	})

	t.Run("list users database error", func(t *testing.T) {
		dbMock2 := &mocks.MockDBTX{}
		userService2 := NewUserService(dbMock2, metricsCollector)
		dbMock2.On("Query", context.Background(), "SELECT id, name, email FROM users").Return(nil, assert.AnError)

		_, err := userService2.ListUsers()
		assert.Error(t, err)
		dbMock2.AssertExpectations(t)
	})

	t.Run("list users scan error", func(t *testing.T) {
		dbMock3 := &mocks.MockDBTX{}
		userService3 := NewUserService(dbMock3, metricsCollector)
		rows := &mocks.MockRows{}
		rows.On("Close").Return()
		rows.On("Next").Return(true).Once()
		rows.On("Scan", mock.Anything).Return(assert.AnError)

		dbMock3.On("Query", context.Background(), "SELECT id, name, email FROM users").Return(rows, nil)

		_, err := userService3.ListUsers()
		assert.Error(t, err)
		dbMock3.AssertExpectations(t)
	})

	t.Run("get users count database error", func(t *testing.T) {
		dbMock4 := &mocks.MockDBTX{}
		userService4 := NewUserService(dbMock4, metricsCollector)
		row := &mocks.MockRow{}
		row.On("Scan", mock.Anything).Return(assert.AnError)
		dbMock4.On("QueryRow", context.Background(), "SELECT COUNT(*) FROM users").Return(row)

		_, err := userService4.GetUsersCount()
		assert.Error(t, err)
		dbMock4.AssertExpectations(t)
	})
}
