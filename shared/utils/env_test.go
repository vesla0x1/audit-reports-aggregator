package utils

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "environment variable set",
			key:          "TEST_ENV_VAR",
			envValue:     "test_value",
			defaultValue: "default",
			expected:     "test_value",
		},
		{
			name:         "environment variable not set",
			key:          "UNSET_VAR",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "empty environment variable returns default",
			key:          "EMPTY_VAR",
			envValue:     "",
			defaultValue: "fallback",
			expected:     "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset environment variable
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			key:          "TEST_INT",
			envValue:     "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "negative integer",
			key:          "TEST_NEG_INT",
			envValue:     "-100",
			defaultValue: 10,
			expected:     -100,
		},
		{
			name:         "invalid integer returns default",
			key:          "TEST_INVALID_INT",
			envValue:     "not_a_number",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "empty value returns default",
			key:          "TEST_EMPTY_INT",
			envValue:     "",
			defaultValue: 25,
			expected:     25,
		},
		{
			name:         "float value returns default",
			key:          "TEST_FLOAT_AS_INT",
			envValue:     "3.14",
			defaultValue: 10,
			expected:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetEnvInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "true lowercase",
			key:          "TEST_BOOL",
			envValue:     "true",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "true uppercase",
			key:          "TEST_BOOL_UPPER",
			envValue:     "TRUE",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "true as 1",
			key:          "TEST_BOOL_1",
			envValue:     "1",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "false lowercase",
			key:          "TEST_BOOL_FALSE",
			envValue:     "false",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "false as 0",
			key:          "TEST_BOOL_0",
			envValue:     "0",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "invalid bool returns default",
			key:          "TEST_INVALID_BOOL",
			envValue:     "yes",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "empty value returns default",
			key:          "TEST_EMPTY_BOOL",
			envValue:     "",
			defaultValue: true,
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetEnvBool(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue string
		expected     time.Duration
	}{
		{
			name:         "valid duration seconds",
			key:          "TEST_DURATION",
			envValue:     "30s",
			defaultValue: "10s",
			expected:     30 * time.Second,
		},
		{
			name:         "valid duration minutes",
			key:          "TEST_DURATION_MIN",
			envValue:     "5m",
			defaultValue: "1m",
			expected:     5 * time.Minute,
		},
		{
			name:         "valid duration hours",
			key:          "TEST_DURATION_HOUR",
			envValue:     "2h",
			defaultValue: "1h",
			expected:     2 * time.Hour,
		},
		{
			name:         "complex duration",
			key:          "TEST_DURATION_COMPLEX",
			envValue:     "2h45m30s",
			defaultValue: "1h",
			expected:     2*time.Hour + 45*time.Minute + 30*time.Second,
		},
		{
			name:         "invalid duration returns default",
			key:          "TEST_INVALID_DURATION",
			envValue:     "not_a_duration",
			defaultValue: "15s",
			expected:     15 * time.Second,
		},
		{
			name:         "empty value returns default",
			key:          "TEST_EMPTY_DURATION",
			envValue:     "",
			defaultValue: "1m",
			expected:     1 * time.Minute,
		},
		{
			name:         "milliseconds",
			key:          "TEST_DURATION_MS",
			envValue:     "500ms",
			defaultValue: "100ms",
			expected:     500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetEnvDuration(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
