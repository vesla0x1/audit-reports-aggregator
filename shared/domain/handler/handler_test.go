package handler

/*import (
	"context"
	"errors"
	"testing"

	"shared/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockWorker for testing
type mockWorker struct {
	mock.Mock
}

func (m *mockWorker) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockWorker) Process(ctx context.Context, req Request) (Response, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(Response), args.Error(1)
}

func (m *mockWorker) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewHandler(t *testing.T) {
	worker := &mockWorker{}
	cfg := config.DefaultHandlerConfig()

	handler := NewHandler(worker, nil, &cfg)

	assert.NotNil(t, handler)
	assert.Equal(t, worker, handler.worker)
	assert.Equal(t, &cfg, handler.config)
	assert.Empty(t, handler.middlewares)
}

func TestNewHandler_WithConfig(t *testing.T) {
	worker := &mockWorker{}
	cfg := config.DefaultHandlerConfig()

	handler := NewHandler(worker, nil, &cfg)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.config)
	assert.Equal(t, &cfg, handler.config)
}

func TestHandler_Handle(t *testing.T) {
	worker := &mockWorker{}
	worker.On("Name").Return("test-worker")

	req := Request{
		ID:   "test-123",
		Type: "test",
	}

	expectedResp := Response{
		ID:      "test-123",
		Success: true,
	}

	worker.On("Process", mock.Anything, req).Return(expectedResp, nil)

	cfg := config.DefaultHandlerConfig()
	handler := NewHandler(worker, nil, &cfg)

	ctx := context.Background()
	resp, err := handler.Handle(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
	worker.AssertExpectations(t)
}

func TestHandler_Handle_WithError(t *testing.T) {
	worker := &mockWorker{}
	worker.On("Name").Return("test-worker")

	req := Request{
		ID:   "test-123",
		Type: "test",
	}

	expectedErr := errors.New("processing failed")
	worker.On("Process", mock.Anything, req).Return(Response{}, expectedErr)

	cfg := config.DefaultHandlerConfig()
	handler := NewHandler(worker, nil, &cfg)

	ctx := context.Background()
	_, err := handler.Handle(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	worker.AssertExpectations(t)
}

func TestHandler_ContextValues(t *testing.T) {
	worker := &mockWorker{}
	worker.On("Name").Return("test-worker")

	req := Request{
		ID:   "test-123",
		Type: "test",
	}

	// Capture context to verify values
	var capturedCtx context.Context
	worker.On("Process", mock.Anything, req).Run(func(args mock.Arguments) {
		capturedCtx = args.Get(0).(context.Context)
	}).Return(Response{ID: "test-123", Success: true}, nil)

	cfg := config.DefaultHandlerConfig()
	cfg.Platform = "http"
	handler := NewHandler(worker, nil, &cfg)

	ctx := context.Background()
	_, err := handler.Handle(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "test-123", capturedCtx.Value("request_id"))
	assert.Equal(t, "test-worker", capturedCtx.Value("worker"))
	assert.Equal(t, "http", capturedCtx.Value("platform"))
}

func TestHandler_Use(t *testing.T) {
	defaultCfg := config.DefaultHandlerConfig()
	handler := NewHandler(&mockWorker{}, nil, &defaultCfg)

	// Add middleware
	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
			return next(ctx, req)
		}
	}

	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
			return next(ctx, req)
		}
	}

	handler.Use(middleware1)
	handler.Use(middleware2)

	assert.Len(t, handler.middlewares, 2)
}

func TestHandler_MiddlewareChain(t *testing.T) {
	worker := &mockWorker{}
	worker.On("Name").Return("test-worker")

	req := Request{ID: "test-123", Type: "test"}

	// Track middleware execution order
	var executionOrder []string

	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
			executionOrder = append(executionOrder, "mw1-before")
			resp, err := next(ctx, req)
			executionOrder = append(executionOrder, "mw1-after")
			return resp, err
		}
	}

	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
			executionOrder = append(executionOrder, "mw2-before")
			resp, err := next(ctx, req)
			executionOrder = append(executionOrder, "mw2-after")
			return resp, err
		}
	}

	worker.On("Process", mock.Anything, req).Run(func(args mock.Arguments) {
		executionOrder = append(executionOrder, "worker")
	}).Return(Response{ID: "test-123", Success: true}, nil)

	cfg := config.DefaultHandlerConfig()
	handler := NewHandler(worker, nil, &cfg)
	handler.Use(middleware1)
	handler.Use(middleware2)

	ctx := context.Background()
	_, err := handler.Handle(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, []string{
		"mw1-before",
		"mw2-before",
		"worker",
		"mw2-after",
		"mw1-after",
	}, executionOrder)
}

func TestHandler_Health(t *testing.T) {
	worker := &mockWorker{}
	worker.On("Health", mock.Anything).Return(nil)

	cfg := config.DefaultHandlerConfig()
	handler := NewHandler(worker, nil, &cfg)

	ctx := context.Background()
	err := handler.Health(ctx)

	assert.NoError(t, err)
	worker.AssertExpectations(t)
}

func TestHandler_Health_WithError(t *testing.T) {
	worker := &mockWorker{}
	expectedErr := errors.New("unhealthy")
	worker.On("Health", mock.Anything).Return(expectedErr)

	cfg := config.DefaultHandlerConfig()
	handler := NewHandler(worker, nil, &cfg)

	ctx := context.Background()
	err := handler.Health(ctx)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	worker.AssertExpectations(t)
}*/
