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
func Load() error {
	if loaded {
		return nil // Already loaded
	}

	// Load .env files in order of precedence
	if err := loadEnvFiles(); err != nil {
		return fmt.Errorf("failed to load env files: %w", err)
	}

	// Parse configuration from environment
	cfg, err := parse()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults based on environment
	applyDefaults(cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	instance = cfg
	loaded = true
	return nil
}

// MustLoad loads configuration and panics on error
// Use this for application initialization where errors are fatal
func MustLoad() {
	if err := Load(); err != nil {
		panic(fmt.Sprintf("failed to load configuration: %v", err))
	}
}

// Get returns the current configuration
// Returns error if configuration hasn't been loaded
func Get() (*Config, error) {
	if !loaded || instance == nil {
		return nil, fmt.Errorf("configuration not loaded; call Load() first")
	}

	return instance, nil
}

// MustGet returns the configuration or panics if not loaded
// Use this when you're certain configuration has been loaded
func MustGet() *Config {
	cfg, err := Get()
	if err != nil {
		panic(fmt.Sprintf("failed to get configuration: %v", err))
	}
	return cfg
}

// IsLoaded returns whether configuration has been loaded
func IsLoaded() bool {
	return loaded
}

// Reset clears the configuration (useful for testing)
func Reset() {
	instance = nil
	loaded = false
}

// Reload reloads configuration from environment
// Useful for configuration updates without restart (use with caution)
func Reload() error {
	// Parse configuration from current environment
	cfg, err := parse()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	applyDefaults(cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	instance = cfg
	return nil
}
