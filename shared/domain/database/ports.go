package database

import (
	"context"
	"database/sql"

	"shared/config"
)

// Database represents a database connection
type Database interface {
	// Execute runs a query that doesn't return rows
	Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Query runs a query that returns rows
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// QueryRow runs a query that returns at most one row
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row

	// Transaction executes a function within a transaction
	Transaction(ctx context.Context, fn func(tx Transaction) error) error

	// Ping verifies the connection
	Ping(ctx context.Context) error

	// Close closes the database connection
	Close() error
}

// Transaction represents a database transaction
type Transaction interface {
	Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Commit() error
	Rollback() error
}

// DatabaseFactory creates database instances
type DatabaseFactory interface {
	Create(cfg *config.Config) (Database, error)
}
