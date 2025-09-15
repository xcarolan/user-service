package database

import (
	"context"
	"log/slog"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// DBTX is an interface for database operations, allowing for both real connections and mocks.
// It is satisfied by *pgx.Conn.
type DBTX interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

func NewConnection(databaseUrl string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), databaseUrl)
	if err != nil {
		return nil, err
	}

	slog.Info("Database connection established")
	return conn, nil
}
