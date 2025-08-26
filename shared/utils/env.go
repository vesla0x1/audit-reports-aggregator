package utils

import (
	"os"
	"strconv"
	"time"
)

// GetEnv returns the value of an environment variable or a default value if not set.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt returns the value of an environment variable as an integer,
// or a default value if not set or if parsing fails.
func GetEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intVal
}

// GetEnvBool returns the value of an environment variable as a boolean,
// or a default value if not set or if parsing fails.
// Accepts: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
func GetEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return boolVal
}

// GetEnvDuration returns the value of an environment variable as a time.Duration,
// or a default value if not set or if parsing fails.
// Accepts formats like: "300ms", "1.5h", "2h45m"
func GetEnvDuration(key string, defaultValue string) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		duration, _ := time.ParseDuration(defaultValue)
		return duration
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		// Fall back to default if parsing fails
		duration, _ = time.ParseDuration(defaultValue)
		return duration
	}

	return duration
}

// GetEnvFloat64 returns the value of an environment variable as a float64,
// or a default value if not set or if parsing fails.
func GetEnvFloat64(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	floatVal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}

	return floatVal
}
