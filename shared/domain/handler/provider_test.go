package handler

/*import (
	"os"
	"testing"
	"time"

	"shared/config"
	//"shared/domain/observability/mocks"

	"github.com/stretchr/testify/assert"
)

func TestNewFactory(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(MockProvider)

	factory := NewFactory(worker, provider)

	assert.NotNil(t, factory)
	assert.Equal(t, worker, factory.worker)
	assert.Equal(t, provider, factory.provider)
	assert.NotNil(t, factory.handlerCfg)
	assert.NotNil(t, factory.retryCfg)
}

func TestFactory_WithHandlerConfig(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(MockProvider)

	customConfig := config.HandlerConfig{
		Timeout:  60 * time.Second,
		Platform: "lambda",
	}

	factory := NewFactory(worker, provider).WithHandlerConfig(customConfig)

	assert.Equal(t, customConfig, factory.handlerCfg)
}

func TestFactory_WithRetryConfig(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(MockProvider)

	retryConfig := config.RetryConfig{
		MaxAttempts:    5,
		InitialBackoff: 200 * time.Millisecond,
	}

	factory := NewFactory(worker, provider).WithRetryConfig(retryConfig)

	assert.Equal(t, retryConfig, factory.retryCfg)
}

func TestFactory_Create(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(ockProvider)

	factory := NewFactory(worker, provider)
	handler := factory.Create()

	assert.NotNil(t, handler)
	assert.Equal(t, worker, handler.worker)
	assert.NotEmpty(t, handler.middlewares) // Should have default middleware
}

func TestFactory_CreateHTTP(t *testing.T) {
	worker := &TestWorker{name: "test"}
	provider := new(.MockProvider)

	factory := NewFactory(worker, provider)
	handler := factory.CreateHTTP()

	assert.NotNil(t, handler)
	assert.Equal(t, "http", handler.config.Platform)
}

func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
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
}*/
