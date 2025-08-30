package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getInt gets environment variable as int with default value
func getInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getBool gets environment variable as bool with default value
func getBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// getFloat64 gets environment variable as float64 with default value
func getFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

// getDuration gets environment variable as duration with default value
func getDuration(key, defaultValue string) time.Duration {
	value := getEnv(key, defaultValue)
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}
	// Fallback to default if parsing fails
	if duration, err := time.ParseDuration(defaultValue); err == nil {
		return duration
	}
	return 30 * time.Second // Ultimate fallback
}

// Environment detection methods
func (c *Config) IsLocal() bool {
	env := strings.ToLower(c.Environment)
	return env == "local" || env == "development" || env == "dev"
}

func (c *Config) IsStaging() bool {
	env := strings.ToLower(c.Environment)
	return env == "staging" || env == "stage"
}

func (c *Config) IsProduction() bool {
	env := strings.ToLower(c.Environment)
	return env == "production" || env == "prod"
}

func (c *Config) IsTest() bool {
	env := strings.ToLower(c.Environment)
	return env == "test" || env == "testing"
}

// isLambdaEnvironment detects if running in AWS Lambda
func IsLambda() bool {
	// AWS Lambda sets these environment variables
	return os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" ||
		os.Getenv("LAMBDA_TASK_ROOT") != "" ||
		os.Getenv("AWS_EXECUTION_ENV") != ""
}
