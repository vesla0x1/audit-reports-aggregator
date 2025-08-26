package platforms

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"shared/handler"
	"shared/handler/mocks"
	"shared/observability"
	obmocks "shared/observability/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHTTPAdapter_ServeHTTP(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")
		mockWorker.On("Process", mock.Anything, mock.MatchedBy(func(req handler.Request) bool {
			return req.Type == "test"
		})).Return(handler.Response{
			ID:      "test-123",
			Success: true,
			Data:    json.RawMessage(`{"result": "success"}`),
		}, nil)

		// Create mock observability provider
		mockProvider := &obmocks.MockProvider{}
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}

		mockProvider.On("Logger", mock.Anything).Return(mockLogger)
		mockProvider.On("Metrics", mock.Anything).Return(mockMetrics)

		// Setup logger expectations (if logging middleware is used)
		mockLogger.On("WithFields", mock.Anything).Return(mockLogger).Maybe()
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Maybe()
		mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything).Maybe()

		// Create real handler with mocked dependencies
		h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
		adapter := NewHTTPAdapter(h)

		// Prepare request
		body := bytes.NewBufferString(`{"test": "data"}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("X-Request-ID", "test-123")
		req.Header.Set("Content-Type", "application/json")

		// Execute
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, "test-123", w.Header().Get("X-Request-ID"))

		var resp handler.Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)

		mockWorker.AssertExpectations(t)
	})

	t.Run("health check", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")
		mockWorker.On("Health", mock.Anything).Return(nil)

		// Create mock observability provider
		mockProvider := &obmocks.MockProvider{}

		// Create real handler
		h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
		adapter := NewHTTPAdapter(h)

		// Execute health check
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var health map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &health)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", health["status"])
		assert.Equal(t, "test-worker", health["worker"])

		mockWorker.AssertExpectations(t)
	})

	t.Run("unhealthy service", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker").Maybe()
		mockWorker.On("Health", mock.Anything).Return(assert.AnError)

		// Create mock observability provider
		mockProvider := &obmocks.MockProvider{}

		// Create real handler
		h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
		adapter := NewHTTPAdapter(h)

		// Execute health check
		req := httptest.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var health map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &health)
		assert.NoError(t, err)
		assert.Equal(t, "unhealthy", health["status"])

		mockWorker.AssertExpectations(t)
	})

	t.Run("error response", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker").Maybe()
		mockWorker.On("Process", mock.Anything, mock.Anything).Return(
			handler.NewErrorResponse(
				"test-123",
				"VALIDATION_ERROR",
				"Invalid input",
				"Missing required field",
			), nil)

		// Create mock observability provider
		mockProvider := &obmocks.MockProvider{}
		mockLogger := &obmocks.MockLogger{}

		mockProvider.On("Logger", mock.Anything).Return(mockLogger).Maybe()
		mockLogger.On("WithFields", mock.Anything).Return(mockLogger).Maybe()
		mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything).Maybe()

		// Create real handler
		h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
		adapter := NewHTTPAdapter(h)

		// Execute request
		body := bytes.NewBufferString(`{"test": "data"}`)
		req := httptest.NewRequest("POST", "/test", body)
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp handler.Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)

		mockWorker.AssertExpectations(t)
	})

	t.Run("request ID generation", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")
		mockWorker.On("Process", mock.Anything, mock.MatchedBy(func(req handler.Request) bool {
			// Verify request ID was generated
			return req.ID != ""
		})).Return(handler.Response{ID: "generated", Success: true}, nil)

		// Create mock observability provider
		mockProvider := &obmocks.MockProvider{}
		mockLogger := &obmocks.MockLogger{}

		mockProvider.On("Logger", mock.Anything).Return(mockLogger).Maybe()
		mockLogger.On("WithFields", mock.Anything).Return(mockLogger).Maybe()
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Maybe()

		// Create real handler
		h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
		adapter := NewHTTPAdapter(h)

		// Execute request without X-Request-ID header
		body := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest("POST", "/", body)
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockWorker.AssertExpectations(t)
	})

	t.Run("extract request type from path", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")
		mockWorker.On("Process", mock.Anything, mock.MatchedBy(func(req handler.Request) bool {
			return req.Type == "download"
		})).Return(handler.Response{Success: true}, nil)

		// Create mock observability provider
		mockProvider := &obmocks.MockProvider{}
		mockLogger := &obmocks.MockLogger{}

		mockProvider.On("Logger", mock.Anything).Return(mockLogger).Maybe()
		mockLogger.On("WithFields", mock.Anything).Return(mockLogger).Maybe()
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Maybe()

		// Create real handler
		h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
		adapter := NewHTTPAdapter(h)

		// Execute
		body := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest("POST", "/download", body)
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockWorker.AssertExpectations(t)
	})

	t.Run("max request size", func(t *testing.T) {
		// Create mock worker (won't be called due to size limit)
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")

		// Create mock observability provider
		mockProvider := &obmocks.MockProvider{}

		// Create handler with small max request size
		config := &handler.Config{
			MaxRequestSize: 100, // 100 bytes limit
		}
		h := handler.NewHandler(mockWorker, mockProvider, config)
		adapter := NewHTTPAdapter(h)

		// Create a body larger than limit
		largeBody := bytes.NewBuffer(make([]byte, 200))
		req := httptest.NewRequest("POST", "/", largeBody)
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)

		// Should get an error for body too large
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("metadata extraction", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")
		mockWorker.On("Process", mock.Anything, mock.MatchedBy(func(req handler.Request) bool {
			// Verify metadata was extracted
			return req.Metadata["http_method"] == "POST" &&
				req.Metadata["http_path"] == "/test" &&
				req.Metadata["header_user_agent"] == "test-agent" &&
				req.Metadata["query_param1"] == "value1"
		})).Return(handler.Response{Success: true}, nil)

		// Create mock observability provider
		mockProvider := &obmocks.MockProvider{}
		mockLogger := &obmocks.MockLogger{}

		mockProvider.On("Logger", mock.Anything).Return(mockLogger).Maybe()
		mockLogger.On("WithFields", mock.Anything).Return(mockLogger).Maybe()
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Maybe()

		// Create real handler
		h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
		adapter := NewHTTPAdapter(h)

		// Execute
		body := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest("POST", "/test?param1=value1&param2=value2", body)
		req.Header.Set("User-Agent", "test-agent")
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockWorker.AssertExpectations(t)
	})
}

func TestHTTPAdapter_DetermineStatusCode(t *testing.T) {
	// Create a minimal adapter for testing
	mockWorker := &mocks.MockWorker{}
	mockWorker.On("Name").Return("test-worker")
	mockProvider := &obmocks.MockProvider{}
	h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
	adapter := NewHTTPAdapter(h)

	tests := []struct {
		name     string
		response handler.Response
		expected int
	}{
		{
			name:     "success",
			response: handler.Response{Success: true},
			expected: http.StatusOK,
		},
		{
			name:     "validation error",
			response: handler.NewErrorResponse("", "VALIDATION_ERROR", "", ""),
			expected: http.StatusBadRequest,
		},
		{
			name:     "not found",
			response: handler.NewErrorResponse("", "NOT_FOUND", "", ""),
			expected: http.StatusNotFound,
		},
		{
			name:     "unauthorized",
			response: handler.NewErrorResponse("", "UNAUTHORIZED", "", ""),
			expected: http.StatusUnauthorized,
		},
		{
			name:     "rate limited",
			response: handler.NewErrorResponse("", "RATE_LIMITED", "", ""),
			expected: http.StatusTooManyRequests,
		},
		{
			name:     "timeout",
			response: handler.NewErrorResponse("", "TIMEOUT", "", ""),
			expected: http.StatusGatewayTimeout,
		},
		{
			name:     "unknown error",
			response: handler.NewErrorResponse("", "UNKNOWN", "", ""),
			expected: http.StatusInternalServerError,
		},
		{
			name:     "error without code",
			response: handler.Response{Success: false},
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := adapter.determineStatusCode(tt.response)
			assert.Equal(t, tt.expected, status)
		})
	}
}

// Helper function to create a test handler with minimal setup
func createTestHandler(worker handler.Worker, provider observability.Provider) *handler.Handler {
	if provider == nil {
		provider = &obmocks.MockProvider{}
	}
	if worker == nil {
		worker = &mocks.MockWorker{}
	}
	return handler.NewHandler(worker, provider, handler.DefaultConfig())
}
