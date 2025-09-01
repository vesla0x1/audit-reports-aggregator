package database

import (
	"fmt"

	"shared/application/ports"
	"shared/infrastructure/config"
)

// Create creates a database instance based on configuration
func CreateDB(cfg *config.Config, obs ports.Observability) (ports.Database, error) {
	switch cfg.Adapters.Database {
	case "postgres":
		return NewPostgresAdapter(&cfg.Database, obs)
	default:
		return nil, fmt.Errorf("unsupported database adapter: %s", cfg.Adapters.Database)
	}
}
