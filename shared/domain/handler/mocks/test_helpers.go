package mocks

/*import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"shared/handler"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestRequest creates a test request with sensible defaults
func TestRequest(requestType string, payload interface{}) handler.Request {
	payloadBytes, _ := json.Marshal(payload)

	return handler.Request{
		ID:        uuid.New().String(),
		Source:    "test",
		Type:      requestType,
		Payload:   payloadBytes,
		Metadata:  map[string]string{"test": "true"},
		Timestamp: time.Now().UTC(),
	}
}

// TestSuccessResponse creates a test success response
func TestSuccessResponse(requestID string, data interface{}) handler.Response {
	resp, _ := handler.NewSuccessResponse(requestID, data)
	return resp
}

// TestErrorResponse creates a test error response
func TestErrorResponse(requestID string, code string, message string) handler.Response {
	return handler.NewErrorResponse(requestID, code, message, "test error")
}

// AssertRequestEqual asserts two requests are equal (ignoring timestamp)
func AssertRequestEqual(t *testing.T, expected, actual handler.Request) {
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.Source, actual.Source)
	assert.Equal(t, expected.Type, actual.Type)
	assert.JSONEq(t, string(expected.Payload), string(actual.Payload))
	assert.Equal(t, expected.Metadata, actual.Metadata)
}

// AssertResponseEqual asserts two responses are equal (ignoring timestamp)
func AssertResponseEqual(t *testing.T, expected, actual handler.Response) {
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.Success, actual.Success)

	if expected.Data != nil {
		assert.JSONEq(t, string(expected.Data), string(actual.Data))
	}

	if expected.Error != nil {
		assert.Equal(t, expected.Error.Code, actual.Error.Code)
		assert.Equal(t, expected.Error.Message, actual.Error.Message)
	}

	assert.Equal(t, expected.Metadata, actual.Metadata)
}

// MockContext creates a context with common test values
func MockContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "test-request-123")
	ctx = context.WithValue(ctx, "trace_id", "test-trace-456")
	ctx = context.WithValue(ctx, "worker", "test-worker")
	return ctx
}*/
