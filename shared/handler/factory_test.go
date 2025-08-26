package handler

import (
	"os"
	"testing"
	"time"

	"shared/observability/mocks"

	"github.com/stretchr/testify/assert"
)

func TestNewFactory(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(mocks.MockProvider)

	factory := NewFactory(worker, provider)

	assert.NotNil(t, factory)
	assert.Equal(t, worker, factory.worker)
	assert.Equal(t, provider, factory.provider)
	assert.NotNil(t, factory.config)
}

func TestFactory_WithConfig(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(mocks.MockProvider)

	customConfig := &Config{
		Timeout:     60 * time.Second,
		Environment: "production",
		Platform:    "lambda",
	}

	factory := NewFactory(worker, provider).WithConfig(customConfig)

	assert.Equal(t, customConfig, factory.config)
}

func TestFactory_Create(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(mocks.MockProvider)

	factory := NewFactory(worker, provider)
	handler := factory.Create()

	assert.NotNil(t, handler)
	assert.Equal(t, worker, handler.worker)
	assert.NotEmpty(t, handler.middlewares) // Should have default middleware
}

func TestFactory_CreateHTTP(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(mocks.MockProvider)

	factory := NewFactory(worker, provider)
	handler := factory.CreateHTTP()

	assert.NotNil(t, handler)
	assert.Equal(t, "http", handler.config.Platform)
}

func TestFactory_CreateOpenFaaS(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(mocks.MockProvider)

	factory := NewFactory(worker, provider)
	handler := factory.CreateOpenFaaS()

	assert.NotNil(t, handler)
	assert.Equal(t, "openfaas", handler.config.Platform)
}

func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "OpenFaaS",
			envVars:  map[string]string{"OPENFAAS_FUNCTION_NAME": "my-function"},
			expected: "openfaas",
		},
		{
			name:     "Knative",
			envVars:  map[string]string{"K_SERVICE": "my-service"},
			expected: "knative",
		},
		{
			name:     "Default",
			envVars:  map[string]string{},
			expected: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			platform := DetectPlatform()
			assert.Equal(t, tt.expected, platform)
		})
	}
}
