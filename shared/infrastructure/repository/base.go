package repository

import (
	"context"
	"database/sql"
	"fmt"
	"shared/application/ports"

	"github.com/Masterminds/squirrel"
)

type baseRepository[T any] struct {
	db      ports.Database
	logger  ports.Logger
	metrics ports.Metrics
	table   string
	qb      squirrel.StatementBuilderType
}

func newBaseRepository[T any](db ports.Database, logger ports.Logger, metrics ports.Metrics, table string) *baseRepository[T] {
	return &baseRepository[T]{
		db:      db,
		logger:  logger,
		metrics: metrics,
		table:   table,
		qb:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// Create inserts a new entity - using sqlx NamedExec for simplicity
func (r *baseRepository[T]) Create(ctx context.Context, entity *T) error {
	panic("Create must be implemented by concrete repository")
}

// Get retrieves an entity by ID - using sqlx for auto-scanning
func (r *baseRepository[T]) Get(ctx context.Context, id int64) (*T, error) {
	var entity T

	r.logger.Info("Getting entity", "table", r.table, "id", id)
	r.metrics.IncrementCounter(fmt.Sprintf("repository.%s.get", r.table), nil)

	// Build query with Squirrel
	query := r.qb.
		Select("*").
		From(r.table).
		Where(squirrel.Eq{"id": id})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	// Execute and scan with sqlx
	err = r.db.Get(ctx, &entity, sqlQuery, args...)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("entity not found")
	}
	if err != nil {
		r.logger.Error("Failed to get entity", "error", err)
		r.metrics.IncrementCounter(fmt.Sprintf("repository.%s.errors", r.table), nil)
		return nil, fmt.Errorf("get entity: %w", err)
	}

	return &entity, nil
}

// Update modifies an entity - using Squirrel for dynamic updates
func (r *baseRepository[T]) Update(ctx context.Context, entity *T) error {
	panic("Update must be implemented by concrete repository")
}

// Delete removes an entity - simple enough for plain sqlx
func (r *baseRepository[T]) Delete(ctx context.Context, id int64) error {
	r.logger.Info("Deleting entity", "table", r.table, "id", id)
	r.metrics.IncrementCounter(fmt.Sprintf("repository.%s.delete", r.table), nil)

	// Build delete with Squirrel
	query := r.qb.
		Delete(r.table).
		Where(squirrel.Eq{"id": id})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.Execute(ctx, sql, args...)
	if err != nil {
		r.logger.Error("Failed to delete entity", "error", err)
		r.metrics.IncrementCounter(fmt.Sprintf("repository.%s.errors", r.table), nil)
		return fmt.Errorf("delete entity: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entity not found")
	}

	return nil
}

// List retrieves multiple entities - using Squirrel for flexible filtering
func (r *baseRepository[T]) ListAll(ctx context.Context) ([]*T, error) {
	r.logger.Info("Listing entities", "table", r.table)
	r.metrics.IncrementCounter(fmt.Sprintf("repository.%s.list", r.table), nil)

	query := r.qb.
		Select("*").
		From(r.table)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	// Use sqlx for scanning multiple rows
	var entities []T
	err = r.db.Select(ctx, &entities, sql, args...)
	if err != nil {
		r.logger.Error("Failed to list entities", "error", err)
		r.metrics.IncrementCounter(fmt.Sprintf("repository.%s.errors", r.table), nil)
		return nil, fmt.Errorf("list entities: %w", err)
	}

	result := make([]*T, len(entities))
	for i := range entities {
		result[i] = &entities[i]
	}

	return result, nil
}

// Count returns the number of entities - Squirrel for conditional counting
func (r *baseRepository[T]) CountAll(ctx context.Context) (int64, error) {
	r.logger.Info("Counting all entities", "table", r.table)
	r.metrics.IncrementCounter(fmt.Sprintf("repository.%s.count_all", r.table), nil)

	// Build query with Squirrel
	query := r.qb.
		Select("COUNT(*)").
		From(r.table)

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build query: %w", err)
	}

	// Use sqlx to scan the count
	var count int64
	err = r.db.Get(ctx, &count, sql, args...)
	if err != nil {
		r.logger.Error("Failed to count all entities", "error", err)
		r.metrics.IncrementCounter(fmt.Sprintf("repository.%s.errors", r.table), nil)
		return 0, fmt.Errorf("count all entities: %w", err)
	}

	return count, nil
}
