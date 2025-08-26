package observability

import (
	"bytes"
	"shared/observability/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProvider(t *testing.T) {
	config := &Config{
		ServiceName: "test-service",
		Environment: "test",
		LogLevel:    "info",
	}

	provider := NewProvider(config)

	assert.NotNil(t, provider)
	assert.Implements(t, (*Provider)(nil), provider)
}

func TestDefaultProvider_Logger(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		ServiceName: "test",
		Environment: "test",
		LogLevel:    "info",
		LogOutput:   &buf,
		AdditionalFields: types.Fields{
			"version": "1.0.0",
		},
	}

	provider := NewProvider(config)
	defer provider.Close()

	// Get logger for same component twice
	logger1 := provider.Logger("downloader")
	logger2 := provider.Logger("downloader")

	assert.NotNil(t, logger1)
	assert.NotNil(t, logger2)
	// Should be the same instance
	assert.Equal(t, logger1, logger2)

	// Get logger for different component
	logger3 := provider.Logger("scraper")
	assert.NotNil(t, logger3)
	// Should be different instance
	assert.NotEqual(t, logger1, logger3)
}

func TestDefaultProvider_Metrics(t *testing.T) {
	config := &Config{
		ServiceName: "test",
		Environment: "test",
	}

	provider := NewProvider(config)
	defer provider.Close()

	// Get metrics for same component twice
	metrics1 := provider.Metrics("downloader")
	metrics2 := provider.Metrics("downloader")

	assert.NotNil(t, metrics1)
	assert.NotNil(t, metrics2)
	// Should be the same instance
	assert.Equal(t, metrics1, metrics2)

	// Get metrics for different component
	metrics3 := provider.Metrics("scraper")
	assert.NotNil(t, metrics3)
	// Should be different instance
	assert.NotEqual(t, metrics1, metrics3)
}

func TestDefaultProvider_Close(t *testing.T) {
	t.Run("close with stdout", func(t *testing.T) {
		config := &Config{
			ServiceName: "test",
		}

		provider := NewProvider(config)
		err := provider.Close()
		assert.NoError(t, err)
	})

	t.Run("close with buffer", func(t *testing.T) {
		var buf bytes.Buffer
		config := &Config{
			ServiceName: "test",
			LogOutput:   &buf,
		}

		provider := NewProvider(config)
		err := provider.Close()
		assert.NoError(t, err)
	})
}
