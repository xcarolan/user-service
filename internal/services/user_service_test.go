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
}
