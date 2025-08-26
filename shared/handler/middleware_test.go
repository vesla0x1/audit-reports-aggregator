package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"shared/observability/mocks"
	"shared/observability/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test TimeoutMiddleware
func TestTimeoutMiddleware(t *testing.T) {
	timeout := 100 * time.Millisecond
	middleware := TimeoutMiddleware(timeout)

	t.Run("success within timeout", func(t *testing.T) {
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			// Quick operation
			time.Sleep(10 * time.Millisecond)
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{ID: "test-123"}
		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("timeout exceeded", func(t *testing.T) {
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			// Slow operation
			time.Sleep(200 * time.Millisecond)
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{ID: "test-123"}
		resp, err := handler(context.Background(), req)

		assert.Error(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "TIMEOUT", resp.Error.Code)
	})

	t.Run("context cancellation", func(t *testing.T) {
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			select {
			case <-ctx.Done():
				return Response{}, ctx.Err()
			case <-time.After(1 * time.Second):
				return NewSuccessResponse(req.ID, nil)
			}
		})

		req := Request{ID: "test-123"}
		resp, err := handler(context.Background(), req)

		assert.Error(t, err)
		assert.Equal(t, "TIMEOUT", resp.Error.Code)
	})
}

// Test LoggingMiddleware
func TestLoggingMiddleware(t *testing.T) {
	mockProvider := new(mocks.MockProvider)
	mockLogger := new(mocks.MockLogger)

	mockProvider.On("Logger", "handler").Return(mockLogger)

	// Setup logger expectations
	mockLogger.On("WithFields", types.Fields{
		"request_id": "test-123",
		"type":       "test",
		"source":     "unit-test",
		"worker":     "",
		"platform":   "",
	}).Return(mockLogger)

	mockLogger.On("Info", mock.Anything, "Processing request", types.Fields{
		"payload_size": 2, // Length of "{}"
	}).Return()

	mockLogger.On("Info", mock.Anything, "Request completed successfully", mock.MatchedBy(func(fields types.Fields) bool {
		// Check that duration_ms exists and is reasonable
		if ms, ok := fields["duration_ms"].(int64); ok {
			return ms >= 0 && ms < 1000
		}
		return false
	})).Return()

	middleware := LoggingMiddleware(mockProvider)

	handler := middleware(func(ctx context.Context, req Request) (Response, error) {
		return NewSuccessResponse(req.ID, nil)
	})

	req := Request{
		ID:      "test-123",
		Type:    "test",
		Source:  "unit-test",
		Payload: []byte("{}"),
	}

	resp, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.True(t, resp.Success)
	mockProvider.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

// Test MetricsMiddleware
func TestMetricsMiddleware(t *testing.T) {
	mockProvider := new(mocks.MockProvider)
	mockMetrics := new(mocks.MockMetrics)

	mockProvider.On("Metrics", "handler").Return(mockMetrics)

	t.Run("successful request", func(t *testing.T) {
		mockMetrics.On("StartOperation", "test-worker").Return()
		mockMetrics.On("EndOperation", "test-worker").Return()
		mockMetrics.On("RecordDuration", "test-worker", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordSuccess", "test-worker").Return()

		middleware := MetricsMiddleware(mockProvider)

		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			return NewSuccessResponse(req.ID, nil)
		})

		ctx := context.WithValue(context.Background(), "worker", "test-worker")
		req := Request{ID: "test-123", Type: "test"}

		resp, err := handler(ctx, req)

		assert.NoError(t, err)
		assert.True(t, resp.Success)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("failed request", func(t *testing.T) {
		mockMetrics := new(mocks.MockMetrics)
		mockProvider := new(mocks.MockProvider)
		mockProvider.On("Metrics", "handler").Return(mockMetrics)

		mockMetrics.On("StartOperation", "test-worker").Return()
		mockMetrics.On("EndOperation", "test-worker").Return()
		mockMetrics.On("RecordDuration", "test-worker", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordError", "test-worker", "processing_error").Return()

		middleware := MetricsMiddleware(mockProvider)

		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			return Response{}, errors.New("test error")
		})

		ctx := context.WithValue(context.Background(), "worker", "test-worker")
		req := Request{ID: "test-123", Type: "test"}

		_, err := handler(ctx, req)

		assert.Error(t, err)
		mockMetrics.AssertExpectations(t)
	})
}

// Test RecoveryMiddleware
func TestRecoveryMiddleware(t *testing.T) {
	mockProvider := new(mocks.MockProvider)
	mockLogger := new(mocks.MockLogger)
	mockMetrics := new(mocks.MockMetrics)

	mockProvider.On("Logger", "handler").Return(mockLogger)
	mockProvider.On("Metrics", "handler").Return(mockMetrics)

	mockLogger.On("Error", mock.Anything, "Panic recovered", mock.AnythingOfType("*errors.errorString"), mock.MatchedBy(func(fields types.Fields) bool {
		// Verify required fields exist
		return fields["request_id"] == "test-123" && fields["stack"] != nil
	})).Return()

	mockMetrics.On("RecordError", "panic", "panic_recovered").Return()

	middleware := RecoveryMiddleware(mockProvider)

	handler := middleware(func(ctx context.Context, req Request) (Response, error) {
		panic("test panic")
	})

	req := Request{ID: "test-123"}
	resp, err := handler(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.False(t, resp.Success)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)

	mockProvider.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

// Test ValidationMiddleware
func TestValidationMiddleware(t *testing.T) {
	middleware := ValidationMiddleware()

	t.Run("valid request", func(t *testing.T) {
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{
			ID:      "test-123",
			Type:    "test",
			Payload: []byte(`{"key": "value"}`),
		}

		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("missing request ID", func(t *testing.T) {
		var capturedReq Request
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			capturedReq = req
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{
			Type:    "test",
			Payload: []byte(`{}`),
		}

		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, capturedReq.ID) // ID should be generated
	})

	t.Run("missing request type", func(t *testing.T) {
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{
			ID:      "test-123",
			Payload: []byte(`{}`),
		}

		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "type is required")
	})

	t.Run("missing payload", func(t *testing.T) {
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{
			ID:   "test-123",
			Type: "test",
		}

		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "payload is required")
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{
			ID:      "test-123",
			Type:    "test",
			Payload: []byte(`{invalid json`),
		}

		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "Invalid JSON")
	})
}

// Test TracingMiddleware
func TestTracingMiddleware(t *testing.T) {
	middleware := TracingMiddleware()

	t.Run("generates trace ID", func(t *testing.T) {
		var capturedCtx context.Context
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			capturedCtx = ctx
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{
			ID:       "test-123",
			Metadata: make(map[string]string),
		}

		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.True(t, resp.Success)

		// Check context has trace ID
		traceID := capturedCtx.Value("trace_id")
		assert.NotNil(t, traceID)
		assert.NotEmpty(t, traceID.(string))

		// Check response metadata has trace ID
		assert.NotEmpty(t, resp.Metadata["trace_id"])
		assert.NotEmpty(t, resp.Metadata["span_id"])
	})

	t.Run("uses existing trace ID", func(t *testing.T) {
		var capturedCtx context.Context
		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			capturedCtx = ctx
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{
			ID: "test-123",
			Metadata: map[string]string{
				"trace_id": "existing-trace-123",
			},
		}

		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.True(t, resp.Success)

		// Check context has existing trace ID
		traceID := capturedCtx.Value("trace_id")
		assert.Equal(t, "existing-trace-123", traceID)

		// Check response metadata
		assert.Equal(t, "existing-trace-123", resp.Metadata["trace_id"])
	})
}

// Test RetryMiddleware
func TestRetryMiddleware(t *testing.T) {
	t.Run("success on first try", func(t *testing.T) {
		callCount := 0
		middleware := RetryMiddleware(DefaultRetryConfig())

		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			callCount++
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{ID: "test-123"}
		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 1, callCount)
	})

	t.Run("retry on failure then success", func(t *testing.T) {
		callCount := 0
		middleware := RetryMiddleware(DefaultRetryConfig())

		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			callCount++
			if callCount < 3 {
				return NewErrorResponse(req.ID, "TEMPORARY_ERROR", "Temporary failure", ""), nil
			}
			return NewSuccessResponse(req.ID, nil)
		})

		req := Request{ID: "test-123"}
		start := time.Now()
		resp, err := handler(context.Background(), req)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 3, callCount)
		// Should have some delay due to backoff
		assert.True(t, duration >= 30*time.Millisecond) // 10ms + 20ms backoff
	})

	t.Run("max retries exhausted", func(t *testing.T) {
		callCount := 0
		middleware := RetryMiddleware(DefaultRetryConfig())

		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			callCount++
			return NewErrorResponse(req.ID, "TEMPORARY_ERROR", "Always fails", ""), nil
		})

		req := Request{ID: "test-123"}
		resp, err := handler(context.Background(), req)

		assert.NoError(t, err) // No error, but response indicates failure
		assert.False(t, resp.Success)
		assert.Equal(t, 4, callCount) // Initial + 3 retries
		assert.Contains(t, resp.Error.Details, "Failed after 3 retries")
	})

	t.Run("non-retryable error", func(t *testing.T) {
		callCount := 0
		middleware := RetryMiddleware(DefaultRetryConfig())

		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			callCount++
			return NewErrorResponse(req.ID, "VALIDATION_ERROR", "Bad request", ""), nil
		})

		req := Request{ID: "test-123"}
		resp, err := handler(context.Background(), req)

		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, 1, callCount) // No retries for non-retryable errors
	})

	t.Run("context cancellation", func(t *testing.T) {
		callCount := 0
		middleware := RetryMiddleware(DefaultRetryConfig())

		handler := middleware(func(ctx context.Context, req Request) (Response, error) {
			callCount++
			return NewErrorResponse(req.ID, "TEMPORARY_ERROR", "Temporary failure", ""), nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		req := Request{ID: "test-123"}
		resp, err := handler(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "CANCELLED", resp.Error.Code)
		assert.Equal(t, 1, callCount) // Should stop retrying on context cancellation
	})
}
