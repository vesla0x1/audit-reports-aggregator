package mocks

/*import (
	"context"

	"shared/config"
	"shared/domain/handler"

	"github.com/stretchr/testify/mock"
)

// MockHandler is a mock implementation of the Handler.
// Use this to test platform adapters without real handler logic.
type MockHandler struct {
	mock.Mock
	config *config.HandlerConfig
	worker handler.Worker
}

// NewMockHandler creates a new mock handler with optional config
func NewMockHandler(cfg *config.HandlerConfig, worker handler.Worker) *MockHandler {
	if cfg == nil {
		defaultCfg := config.DefaultHandlerConfig()
		cfg = &defaultCfg
	}
	if worker == nil {
		worker = &MockWorker{}
	}
	return &MockHandler{
		config: cfg,
		worker: worker,
	}
}

// Handle mocks the request handling
func (m *MockHandler) Handle(ctx context.Context, req handler.Request) (handler.Response, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(handler.Response), args.Error(1)
}

// Health mocks the health check
func (m *MockHandler) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Config returns the handler configuration
func (m *MockHandler) Config() *config.HandlerConfig {
	return m.config
}

// Worker returns the underlying worker
func (m *MockHandler) Worker() handler.Worker {
	return m.worker
}

// Use mocks adding middleware (no-op for mock)
func (m *MockHandler) Use(middleware handler.Middleware) {
	m.Called(middleware)
}*/
