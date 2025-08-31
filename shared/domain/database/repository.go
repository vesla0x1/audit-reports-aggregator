package database

import (
	"context"
	"database/sql"
)

// Repository provides basic database operations for any entity
type Repository struct {
	db        Database
	tableName string
}

// NewRepository creates a new repository instance
func NewRepository(db Database, tableName string) *Repository {
	return &Repository{
		db:        db,
		tableName: tableName,
	}
}

// Execute runs a query that doesn't return rows
func (r *Repository) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return r.db.Execute(ctx, query, args...)
}

// Query runs a query that returns rows
func (r *Repository) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return r.db.Query(ctx, query, args...)
}

// QueryRow runs a query that returns at most one row
func (r *Repository) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return r.db.QueryRow(ctx, query, args...)
}

// Transaction executes a function within a database transaction
func (r *Repository) Transaction(ctx context.Context, fn func(tx Transaction) error) error {
	return r.db.Transaction(ctx, fn)
}

// GetDB returns the underlying database connection
func (r *Repository) GetDB() Database {
	return r.db
}

// GetTableName returns the table name
func (r *Repository) GetTableName() string {
	return r.tableName
}
