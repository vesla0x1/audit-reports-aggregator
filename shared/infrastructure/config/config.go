package config

import (
	"fmt"
)

// Singleton instance management
var (
	instance *Config
	loaded   bool
)

// Load loads configuration from environment variables and .env files
// This should be called once at application startup
func Load() (*Config, error) {
	if loaded {
		return instance, nil
	}

	// Load .env files in order of precedence
	if err := loadEnvFiles(); err != nil {
		return nil, fmt.Errorf("failed to load env files: %w", err)
	}

	// Parse configuration from environment
	cfg, err := parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults based on environment
	applyDefaults(cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	instance = cfg
	loaded = true
	return cfg, nil
}

// IsLoaded returns whether configuration has been loaded
func IsLoaded() bool {
	return loaded
}
