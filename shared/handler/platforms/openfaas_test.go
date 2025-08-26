package platforms

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"shared/handler"
	"shared/handler/mocks"
	obmocks "shared/observability/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOpenFaaSAdapter_BuildRequest(t *testing.T) {
	// Create adapter with minimal setup
	mockWorker := &mocks.MockWorker{}
	mockProvider := &obmocks.MockProvider{}
	h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
	adapter := NewOpenFaaSAdapter(h)

	t.Run("JSON request", func(t *testing.T) {
		input := handler.Request{
			ID:      "test-123",
			Type:    "test",
			Payload: json.RawMessage(`{"key": "value"}`),
		}

		inputBytes, _ := json.Marshal(input)
		req, err := adapter.buildRequest(inputBytes)

		assert.NoError(t, err)
		assert.Equal(t, "test-123", req.ID)
		assert.Equal(t, "test", req.Type)
		assert.JSONEq(t, `{"key": "value"}`, string(req.Payload))
	})

	t.Run("raw payload - non-JSON", func(t *testing.T) {
		// Use invalid JSON to force the raw payload path
		input := []byte(`not valid json`)

		// Set environment variables
		os.Setenv("OPENFAAS_FUNCTION_NAME", "test-function")
		defer os.Unsetenv("OPENFAAS_FUNCTION_NAME")

		req, err := adapter.buildRequest(input)

		assert.NoError(t, err)
		assert.NotEmpty(t, req.ID)
		assert.Equal(t, "openfaas", req.Source)
		assert.Equal(t, "test-function", req.Type)
		// The invalid JSON is stored as-is in the payload
		assert.Equal(t, json.RawMessage(`not valid json`), req.Payload)
	})

	t.Run("raw payload - valid JSON but not Request structure", func(t *testing.T) {
		// This JSON is valid but doesn't match Request structure
		// It will be partially unmarshaled, resulting in empty Request fields
		input := []byte(`{"raw": "data"}`)

		req, err := adapter.buildRequest(input)

		assert.NoError(t, err)
		// When unmarshaling succeeds but fields are empty:
		// - ID will be generated if empty
		// - Timestamp will be set if zero
		// - Other fields remain empty/nil
		assert.NotEmpty(t, req.ID)       // Generated because empty
		assert.NotZero(t, req.Timestamp) // Set because zero
		// These fields are not in the JSON, so they remain empty
		assert.Empty(t, req.Type)
		assert.Empty(t, req.Source)
		// The original JSON is NOT preserved as Payload when unmarshal succeeds
		assert.Nil(t, req.Payload)
	})

	t.Run("partial Request structure", func(t *testing.T) {
		// JSON with some Request fields but not all
		input := []byte(`{"type": "partial", "source": "test"}`)

		req, err := adapter.buildRequest(input)

		assert.NoError(t, err)
		assert.NotEmpty(t, req.ID) // Generated because not provided
		assert.Equal(t, "partial", req.Type)
		assert.Equal(t, "test", req.Source)
		assert.Nil(t, req.Payload)       // Not provided in JSON
		assert.NotZero(t, req.Timestamp) // Set because zero
	})
}

func TestOpenFaaSAdapter_ExtractMetadataFromEnv(t *testing.T) {
	// Create adapter with minimal setup
	mockWorker := &mocks.MockWorker{}
	mockProvider := &obmocks.MockProvider{}
	h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
	adapter := NewOpenFaaSAdapter(h)

	// Set environment variables
	os.Setenv("OPENFAAS_FUNCTION_NAME", "my-function")
	os.Setenv("OPENFAAS_NAMESPACE", "default")
	os.Setenv("HOSTNAME", "function-pod-123")
	os.Setenv("Http_Path", "/function")
	os.Setenv("Http_Method", "POST")
	os.Setenv("Http_X_Request_Id", "req-456")

	defer func() {
		os.Unsetenv("OPENFAAS_FUNCTION_NAME")
		os.Unsetenv("OPENFAAS_NAMESPACE")
		os.Unsetenv("HOSTNAME")
		os.Unsetenv("Http_Path")
		os.Unsetenv("Http_Method")
		os.Unsetenv("Http_X_Request_Id")
	}()

	metadata := adapter.extractMetadataFromEnv()

	assert.Equal(t, "my-function", metadata["function_name"])
	assert.Equal(t, "default", metadata["namespace"])
	assert.Equal(t, "function-pod-123", metadata["hostname"])
	assert.Equal(t, "/function", metadata["http_path"])
	assert.Equal(t, "POST", metadata["http_method"])
	assert.Equal(t, "req-456", metadata["request_id"])
}

func TestOpenFaaSAdapter_ExtractRequestType(t *testing.T) {
	// Create adapter with minimal setup
	mockWorker := &mocks.MockWorker{}
	mockProvider := &obmocks.MockProvider{}
	h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
	adapter := NewOpenFaaSAdapter(h)

	t.Run("from header env", func(t *testing.T) {
		os.Setenv("Http_X_Request_Type", "download")
		defer os.Unsetenv("Http_X_Request_Type")

		reqType := adapter.extractRequestType()
		assert.Equal(t, "download", reqType)
	})

	t.Run("from function name", func(t *testing.T) {
		os.Setenv("OPENFAAS_FUNCTION_NAME", "processor")
		defer os.Unsetenv("OPENFAAS_FUNCTION_NAME")

		reqType := adapter.extractRequestType()
		assert.Equal(t, "processor", reqType)
	})

	t.Run("from path", func(t *testing.T) {
		os.Setenv("Http_Path", "/analyze")
		defer os.Unsetenv("Http_Path")

		reqType := adapter.extractRequestType()
		assert.Equal(t, "analyze", reqType)
	})

	t.Run("default", func(t *testing.T) {
		reqType := adapter.extractRequestType()
		assert.Equal(t, "function", reqType)
	})
}

func TestOpenFaaSAdapter_WriteResponse(t *testing.T) {
	// Create adapter with minimal setup
	mockWorker := &mocks.MockWorker{}
	mockProvider := &obmocks.MockProvider{}
	h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
	adapter := NewOpenFaaSAdapter(h)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resp := handler.Response{
		ID:      "test-123",
		Success: true,
		Data:    json.RawMessage(`{"result": "ok"}`),
	}

	err := adapter.writeResponse(resp)
	assert.NoError(t, err)

	// Restore stdout and read output
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify output
	var output handler.Response
	err = json.Unmarshal(buf.Bytes(), &output)
	assert.NoError(t, err)
	assert.Equal(t, resp.ID, output.ID)
	assert.True(t, output.Success)
}

func TestOpenFaaSAdapter_ServeHTTP(t *testing.T) {
	// Create mock worker
	mockWorker := &mocks.MockWorker{}
	mockWorker.On("Name").Return("test-worker")

	expectedResp := handler.Response{
		ID:      "test-123",
		Success: true,
		Data:    json.RawMessage(`{"result": "success"}`),
	}

	mockWorker.On("Process", mock.Anything, mock.MatchedBy(func(req handler.Request) bool {
		return req.Source == "openfaas"
	})).Return(expectedResp, nil)

	// Create mock observability provider
	mockProvider := &obmocks.MockProvider{}
	mockLogger := &obmocks.MockLogger{}

	// Setup logger expectations (optional, depending on if middleware is used)
	mockProvider.On("Logger", mock.Anything).Return(mockLogger).Maybe()
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger).Maybe()
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Maybe()

	// Create handler and adapter
	h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
	adapter := NewOpenFaaSAdapter(h)

	// Prepare request
	body := bytes.NewBufferString(`{"test": "data"}`)
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("X-Function-Name", "test-function")
	req.Header.Set("X-Request-ID", "test-123")

	// Execute
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "test-123", w.Header().Get("X-Request-ID"))

	var resp handler.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)

	mockWorker.AssertExpectations(t)
}

func TestOpenFaaSAdapter_ServeHTTP_Error(t *testing.T) {
	// Create mock worker
	mockWorker := &mocks.MockWorker{}
	mockWorker.On("Name").Return("test-worker").Maybe() // Add Name() expectation

	errorResp := handler.NewErrorResponse(
		"test-123",
		"PROCESSING_ERROR",
		"Something went wrong",
		"Details here",
	)

	mockWorker.On("Process", mock.Anything, mock.Anything).Return(errorResp, nil)

	// Create mock observability provider
	mockProvider := &obmocks.MockProvider{}
	mockLogger := &obmocks.MockLogger{}

	// Setup logger expectations
	mockProvider.On("Logger", mock.Anything).Return(mockLogger).Maybe()
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger).Maybe()
	mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything).Maybe()

	// Create handler and adapter
	h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
	adapter := NewOpenFaaSAdapter(h)

	// Prepare request
	body := bytes.NewBufferString(`{"test": "data"}`)
	req := httptest.NewRequest("POST", "/", body)

	// Execute
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp handler.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "PROCESSING_ERROR", resp.Error.Code)

	mockWorker.AssertExpectations(t)
}

func TestOpenFaaSAdapter_BuildRequestFromHTTP(t *testing.T) {
	// Create adapter with minimal setup
	mockWorker := &mocks.MockWorker{}
	mockProvider := &obmocks.MockProvider{}
	h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
	adapter := NewOpenFaaSAdapter(h)

	t.Run("with headers", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(`{"data": "test"}`))
		req.Header.Set("X-Function-Name", "my-function")
		req.Header.Set("X-Request-ID", "req-123")
		req.Header.Set("X-Request-Type", "process")
		req.Header.Set("X-Custom", "value")

		body := []byte(`{"data": "test"}`)
		result, err := adapter.buildRequestFromHTTP(req, body)

		assert.NoError(t, err)
		assert.Equal(t, "req-123", result.ID)
		assert.Equal(t, "process", result.Type)
		assert.Equal(t, "openfaas", result.Source)
		assert.Equal(t, "my-function", result.Metadata["function_name"])
		assert.Equal(t, "value", result.Metadata["x_custom"])
		assert.JSONEq(t, `{"data": "test"}`, string(result.Payload))
	})

	t.Run("without request ID", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("X-Call-Id", "call-456")

		body := []byte(`{}`)
		result, err := adapter.buildRequestFromHTTP(req, body)

		assert.NoError(t, err)
		assert.Equal(t, "call-456", result.ID)
	})

	t.Run("with environment function name", func(t *testing.T) {
		os.Setenv("OPENFAAS_FUNCTION_NAME", "env-function")
		defer os.Unsetenv("OPENFAAS_FUNCTION_NAME")

		req := httptest.NewRequest("POST", "/", nil)
		body := []byte(`{}`)
		result, err := adapter.buildRequestFromHTTP(req, body)

		assert.NoError(t, err)
		assert.Equal(t, "env-function", result.Type)
	})
}

func TestOpenFaaSAdapter_CreateContext(t *testing.T) {
	// Create adapter with minimal setup
	mockWorker := &mocks.MockWorker{}
	mockProvider := &obmocks.MockProvider{}
	h := handler.NewHandler(mockWorker, mockProvider, handler.DefaultConfig())
	adapter := NewOpenFaaSAdapter(h)

	t.Run("with environment variables", func(t *testing.T) {
		os.Setenv("OPENFAAS_FUNCTION_NAME", "test-func")
		os.Setenv("OPENFAAS_NAMESPACE", "test-ns")
		os.Setenv("OPENFAAS_TIMEOUT", "10s")
		defer func() {
			os.Unsetenv("OPENFAAS_FUNCTION_NAME")
			os.Unsetenv("OPENFAAS_NAMESPACE")
			os.Unsetenv("OPENFAAS_TIMEOUT")
		}()

		ctx := adapter.createContext()

		assert.Equal(t, "test-func", ctx.Value("function_name"))
		assert.Equal(t, "test-ns", ctx.Value("namespace"))

		// Check that timeout was set
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.NotZero(t, deadline)
	})

	t.Run("without timeout", func(t *testing.T) {
		ctx := adapter.createContext()

		// No timeout should be set
		_, ok := ctx.Deadline()
		assert.False(t, ok)
	})
}
