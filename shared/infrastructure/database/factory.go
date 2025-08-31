package infradatabase

import (
	"fmt"

	"shared/config"
	"shared/domain/database"
	"shared/domain/observability"
	"shared/infrastructure/database/adapters/postgres"
)

// Factory implements database.DatabaseFactory
type Factory struct {
	logger  observability.Logger
	metrics observability.Metrics
}

// NewFactory creates a new database factory
func NewFactory(logger observability.Logger, metrics observability.Metrics) database.DatabaseFactory {
	return &Factory{
		logger:  logger,
		metrics: metrics,
	}
}

// Create creates a database instance based on configuration
func (f *Factory) Create(cfg *config.Config) (database.Database, error) {
	switch cfg.Adapters.Database {
	case "postgres":
		f.logger.Info("Creating PostgreSQL database connection")
		return postgres.New(&cfg.Database, f.logger, f.metrics)
	default:
		return nil, fmt.Errorf("unsupported database adapter: %s", cfg.Adapters.Database)
	}
}
