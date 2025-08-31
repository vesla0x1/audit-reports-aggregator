package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"shared/config"
	"shared/domain/database"
	"shared/domain/observability"
)

// DB implements the Database interface for PostgreSQL
type DB struct {
	conn    *sql.DB
	cfg     *config.DatabaseConfig
	logger  observability.Logger
	metrics observability.Metrics
}

// New creates a new PostgreSQL database connection
func New(cfg *config.DatabaseConfig, logger observability.Logger, metrics observability.Metrics) (database.Database, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	logger.Info("Connecting to PostgreSQL database",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("Failed to open database connection", "error", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		logger.Error("Failed to ping database", "error", err)
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Successfully connected to PostgreSQL database")
	metrics.IncrementCounter("database.connection.success", map[string]string{"type": "postgres"})

	return &DB{
		conn:    conn,
		cfg:     cfg,
		logger:  logger,
		metrics: metrics,
	}, nil
}

// Execute runs a query that doesn't return rows
func (d *DB) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	startTime := time.Now()

	result, err := d.conn.ExecContext(ctx, query, args...)

	d.recordMetrics("execute", time.Since(startTime), err)

	if err != nil {
		d.logger.Error("Failed to execute query", "error", err)
		return nil, err
	}

	return result, nil
}

// Query runs a query that returns rows
func (d *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	startTime := time.Now()

	rows, err := d.conn.QueryContext(ctx, query, args...)

	d.recordMetrics("query", time.Since(startTime), err)

	if err != nil {
		d.logger.Error("Failed to query", "error", err)
		return nil, err
	}

	return rows, nil
}

// QueryRow runs a query that returns at most one row
func (d *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	startTime := time.Now()
	row := d.conn.QueryRowContext(ctx, query, args...)

	d.metrics.RecordHistogram("database.query_row.duration_ms",
		float64(time.Since(startTime).Milliseconds()), nil)

	return row
}

// Transaction executes a function within a transaction
func (d *DB) Transaction(ctx context.Context, fn func(tx database.Transaction) error) error {
	startTime := time.Now()

	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		d.logger.Error("Failed to begin transaction", "error", err)
		return err
	}

	ptx := &pgTx{tx: tx, logger: d.logger, metrics: d.metrics}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(ptx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			d.logger.Error("Failed to rollback", "error", rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		d.logger.Error("Failed to commit", "error", err)
		return err
	}

	d.recordMetrics("transaction", time.Since(startTime), nil)
	return nil
}

// Ping verifies the connection
func (d *DB) Ping(ctx context.Context) error {
	return d.conn.PingContext(ctx)
}

// Close closes the database connection
func (d *DB) Close() error {
	d.logger.Info("Closing database connection")
	return d.conn.Close()
}

// recordMetrics records operation metrics
func (d *DB) recordMetrics(operation string, duration time.Duration, err error) {
	d.metrics.RecordHistogram(
		fmt.Sprintf("database.%s.duration_ms", operation),
		float64(duration.Milliseconds()),
		nil,
	)

	if err != nil {
		d.metrics.IncrementCounter(fmt.Sprintf("database.%s.errors", operation), nil)
	} else {
		d.metrics.IncrementCounter(fmt.Sprintf("database.%s.success", operation), nil)
	}
}

// pgTx implements the Transaction interface
type pgTx struct {
	tx      *sql.Tx
	logger  observability.Logger
	metrics observability.Metrics
}

func (t *pgTx) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *pgTx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *pgTx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *pgTx) Commit() error {
	return t.tx.Commit()
}

func (t *pgTx) Rollback() error {
	return t.tx.Rollback()
}
