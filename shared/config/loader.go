package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// loadEnvFiles loads .env files in order of precedence
func loadEnvFiles() error {
	if IsLambda() {
		return nil
	}

	// Load base .env file (optional)
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return fmt.Errorf("failed to load .env: %w", err)
		}
	}

	// Load environment-specific file (optional)
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env != "" {
		envFile := fmt.Sprintf(".env.%s", env)
		if _, err := os.Stat(envFile); err == nil {
			if err := godotenv.Overload(envFile); err != nil {
				return fmt.Errorf("failed to load %s: %w", envFile, err)
			}
		}
	}

	// Load .env.local for local overrides (highest precedence, optional)
	if _, err := os.Stat(".env.local"); err == nil {
		if err := godotenv.Overload(".env.local"); err != nil {
			return fmt.Errorf("failed to load .env.local: %w", err)
		}
	}

	return nil
}
