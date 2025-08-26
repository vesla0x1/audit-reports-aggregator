package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequest(t *testing.T) {
	t.Run("successful creation with struct payload", func(t *testing.T) {
		payload := struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}{
			Name:  "test",
			Value: 42,
		}

		req, err := NewRequest("test-type", payload)

		assert.NoError(t, err)
		assert.NotEmpty(t, req.ID)
		assert.Equal(t, "test-type", req.Type)
		assert.NotNil(t, req.Metadata)
		assert.NotZero(t, req.Timestamp)

		// Verify payload was marshaled correctly
		var unmarshaledPayload map[string]interface{}
		err = json.Unmarshal(req.Payload, &unmarshaledPayload)
		assert.NoError(t, err)
		assert.Equal(t, "test", unmarshaledPayload["name"])
		assert.Equal(t, float64(42), unmarshaledPayload["value"])
	})

	t.Run("successful creation with map payload", func(t *testing.T) {
		payload := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}

		req, err := NewRequest("map-type", payload)

		assert.NoError(t, err)
		assert.NotEmpty(t, req.ID)
		assert.Equal(t, "map-type", req.Type)

		var unmarshaledPayload map[string]interface{}
		err = json.Unmarshal(req.Payload, &unmarshaledPayload)
		assert.NoError(t, err)
		assert.Equal(t, "value1", unmarshaledPayload["key1"])
		assert.Equal(t, float64(123), unmarshaledPayload["key2"])
	})

	t.Run("successful creation with nil payload", func(t *testing.T) {
		req, err := NewRequest("nil-type", nil)

		assert.NoError(t, err)
		assert.NotEmpty(t, req.ID)
		assert.Equal(t, "nil-type", req.Type)
		assert.Equal(t, json.RawMessage("null"), req.Payload)
	})

	t.Run("error with unmarshalable payload", func(t *testing.T) {
		// Create a channel which cannot be marshaled to JSON
		payload := make(chan int)

		_, err := NewRequest("error-type", payload)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json")
	})

	t.Run("UUID uniqueness", func(t *testing.T) {
		req1, err1 := NewRequest("type1", nil)
		req2, err2 := NewRequest("type2", nil)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, req1.ID, req2.ID)
	})

	t.Run("timestamp is recent", func(t *testing.T) {
		before := time.Now().UTC()
		req, err := NewRequest("time-type", nil)
		after := time.Now().UTC()

		assert.NoError(t, err)
		assert.True(t, req.Timestamp.After(before.Add(-time.Second)))
		assert.True(t, req.Timestamp.Before(after.Add(time.Second)))
	})
}

func TestRequest_Unmarshal(t *testing.T) {
	t.Run("successful unmarshal to struct", func(t *testing.T) {
		type TestPayload struct {
			Name    string                 `json:"name"`
			Age     int                    `json:"age"`
			Active  bool                   `json:"active"`
			Tags    []string               `json:"tags"`
			Details map[string]interface{} `json:"details"`
		}

		payload := TestPayload{
			Name:   "John Doe",
			Age:    30,
			Active: true,
			Tags:   []string{"tag1", "tag2"},
			Details: map[string]interface{}{
				"location": "NYC",
				"score":    95.5,
			},
		}

		payloadBytes, _ := json.Marshal(payload)
		req := Request{
			Payload: payloadBytes,
		}

		var result TestPayload
		err := req.Unmarshal(&result)

		assert.NoError(t, err)
		assert.Equal(t, payload.Name, result.Name)
		assert.Equal(t, payload.Age, result.Age)
		assert.Equal(t, payload.Active, result.Active)
		assert.Equal(t, payload.Tags, result.Tags)
		assert.Equal(t, payload.Details, result.Details)
	})

	t.Run("successful unmarshal to map", func(t *testing.T) {
		req := Request{
			Payload: json.RawMessage(`{"key1": "value1", "key2": 42}`),
		}

		var result map[string]interface{}
		err := req.Unmarshal(&result)

		assert.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, float64(42), result["key2"])
	})

	t.Run("error with invalid JSON", func(t *testing.T) {
		req := Request{
			Payload: json.RawMessage(`{invalid json}`),
		}

		var result map[string]interface{}
		err := req.Unmarshal(&result)

		assert.Error(t, err)
	})

	t.Run("error with type mismatch", func(t *testing.T) {
		req := Request{
			Payload: json.RawMessage(`{"name": "test"}`),
		}

		var result []string
		err := req.Unmarshal(&result)

		assert.Error(t, err)
	})

	t.Run("unmarshal empty payload", func(t *testing.T) {
		req := Request{
			Payload: json.RawMessage(`{}`),
		}

		var result map[string]interface{}
		err := req.Unmarshal(&result)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("unmarshal null payload", func(t *testing.T) {
		req := Request{
			Payload: json.RawMessage(`null`),
		}

		var result *string
		err := req.Unmarshal(&result)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestResponse_Marshal(t *testing.T) {
	t.Run("successful marshal from struct", func(t *testing.T) {
		resp := Response{}
		data := struct {
			Message string `json:"message"`
			Count   int    `json:"count"`
		}{
			Message: "success",
			Count:   10,
		}

		err := resp.Marshal(data)

		assert.NoError(t, err)
		assert.NotNil(t, resp.Data)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Data, &result)
		assert.NoError(t, err)
		assert.Equal(t, "success", result["message"])
		assert.Equal(t, float64(10), result["count"])
	})

	t.Run("successful marshal from map", func(t *testing.T) {
		resp := Response{}
		data := map[string]interface{}{
			"status": "ok",
			"items":  []int{1, 2, 3},
		}

		err := resp.Marshal(data)

		assert.NoError(t, err)
		assert.NotNil(t, resp.Data)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Data, &result)
		assert.NoError(t, err)
		assert.Equal(t, "ok", result["status"])
	})

	t.Run("marshal nil data", func(t *testing.T) {
		resp := Response{}

		err := resp.Marshal(nil)

		assert.NoError(t, err)
		assert.Equal(t, json.RawMessage("null"), resp.Data)
	})

	t.Run("error with unmarshalable data", func(t *testing.T) {
		resp := Response{}
		data := make(chan int) // channels cannot be marshaled

		err := resp.Marshal(data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json")
	})

	t.Run("overwrites existing data", func(t *testing.T) {
		resp := Response{
			Data: json.RawMessage(`{"old": "data"}`),
		}

		newData := map[string]string{"new": "data"}
		err := resp.Marshal(newData)

		assert.NoError(t, err)

		var result map[string]string
		json.Unmarshal(resp.Data, &result)
		assert.Equal(t, "data", result["new"])
		assert.Empty(t, result["old"])
	})
}

func TestNewErrorResponse(t *testing.T) {
	t.Run("creates error response with all fields", func(t *testing.T) {
		resp := NewErrorResponse(
			"req-123",
			"VALIDATION_ERROR",
			"Invalid input",
			"Field 'name' is required",
		)

		assert.Equal(t, "req-123", resp.ID)
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
		assert.Equal(t, "Invalid input", resp.Error.Message)
		assert.Equal(t, "Field 'name' is required", resp.Error.Details)
		assert.False(t, resp.Error.Retryable)
		assert.NotZero(t, resp.ProcessedAt)
	})

	t.Run("creates error response with empty details", func(t *testing.T) {
		resp := NewErrorResponse(
			"req-456",
			"NOT_FOUND",
			"Resource not found",
			"",
		)

		assert.Equal(t, "req-456", resp.ID)
		assert.False(t, resp.Success)
		assert.Equal(t, "NOT_FOUND", resp.Error.Code)
		assert.Equal(t, "Resource not found", resp.Error.Message)
		assert.Empty(t, resp.Error.Details)
		assert.False(t, resp.Error.Retryable)
	})

	t.Run("sets retryable for retryable error codes", func(t *testing.T) {
		retryableCodes := []string{
			"TIMEOUT",
			"NETWORK_ERROR",
			"RATE_LIMITED",
			"TEMPORARY_ERROR",
			"SERVICE_UNAVAILABLE",
		}

		for _, code := range retryableCodes {
			resp := NewErrorResponse("test", code, "message", "details")
			assert.True(t, resp.Error.Retryable, "Code %s should be retryable", code)
		}
	})

	t.Run("does not set retryable for non-retryable error codes", func(t *testing.T) {
		nonRetryableCodes := []string{
			"VALIDATION_ERROR",
			"NOT_FOUND",
			"UNAUTHORIZED",
			"FORBIDDEN",
			"INTERNAL_ERROR",
		}

		for _, code := range nonRetryableCodes {
			resp := NewErrorResponse("test", code, "message", "details")
			assert.False(t, resp.Error.Retryable, "Code %s should not be retryable", code)
		}
	})

	t.Run("timestamp is recent", func(t *testing.T) {
		before := time.Now().UTC()
		resp := NewErrorResponse("test", "ERROR", "msg", "")
		after := time.Now().UTC()

		assert.True(t, resp.ProcessedAt.After(before.Add(-time.Second)))
		assert.True(t, resp.ProcessedAt.Before(after.Add(time.Second)))
	})
}

func TestNewSuccessResponse(t *testing.T) {
	t.Run("creates success response with data", func(t *testing.T) {
		data := map[string]interface{}{
			"result": "success",
			"count":  42,
		}

		resp, err := NewSuccessResponse("req-789", data)

		assert.NoError(t, err)
		assert.Equal(t, "req-789", resp.ID)
		assert.True(t, resp.Success)
		assert.Nil(t, resp.Error)
		assert.NotNil(t, resp.Data)
		assert.NotNil(t, resp.Metadata)
		assert.NotZero(t, resp.ProcessedAt)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Data, &result)
		assert.NoError(t, err)
		assert.Equal(t, "success", result["result"])
		assert.Equal(t, float64(42), result["count"])
	})

	/*t.Run("creates success response with nil data", func(t *testing.T) {
		resp, err := NewSuccessResponse("req-nil", nil)

		assert.NoError(t, err)
		assert.Equal(t, "req-nil", resp.ID)
		assert.True(t, resp.Success)
		assert.Nil(t, resp.Error)
		assert.Equal(t, json.RawMessage("null"), resp.Data)
	})*/

	t.Run("creates success response with struct data", func(t *testing.T) {
		type Result struct {
			Name   string    `json:"name"`
			Score  float64   `json:"score"`
			Active bool      `json:"active"`
			Tags   []string  `json:"tags"`
			Time   time.Time `json:"time"`
		}

		now := time.Now().UTC().Truncate(time.Second)
		data := Result{
			Name:   "test",
			Score:  98.5,
			Active: true,
			Tags:   []string{"tag1", "tag2"},
			Time:   now,
		}

		resp, err := NewSuccessResponse("req-struct", data)

		assert.NoError(t, err)
		assert.True(t, resp.Success)

		var result Result
		err = json.Unmarshal(resp.Data, &result)
		assert.NoError(t, err)
		assert.Equal(t, data.Name, result.Name)
		assert.Equal(t, data.Score, result.Score)
		assert.Equal(t, data.Active, result.Active)
		assert.Equal(t, data.Tags, result.Tags)
		assert.Equal(t, data.Time.Unix(), result.Time.Unix())
	})

	t.Run("error with unmarshalable data", func(t *testing.T) {
		data := make(chan int) // channels cannot be marshaled

		_, err := NewSuccessResponse("req-error", data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json")
	})

	t.Run("metadata is initialized", func(t *testing.T) {
		resp, err := NewSuccessResponse("req-meta", nil)

		assert.NoError(t, err)
		assert.NotNil(t, resp.Metadata)
		assert.IsType(t, map[string]string{}, resp.Metadata)
	})

	t.Run("timestamp is recent", func(t *testing.T) {
		before := time.Now().UTC()
		resp, err := NewSuccessResponse("req-time", nil)
		after := time.Now().UTC()

		require.NoError(t, err)
		assert.True(t, resp.ProcessedAt.After(before.Add(-time.Second)))
		assert.True(t, resp.ProcessedAt.Before(after.Add(time.Second)))
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		code      string
		retryable bool
	}{
		{"TIMEOUT", true},
		{"NETWORK_ERROR", true},
		{"RATE_LIMITED", true},
		{"TEMPORARY_ERROR", true},
		{"SERVICE_UNAVAILABLE", true},
		{"VALIDATION_ERROR", false},
		{"NOT_FOUND", false},
		{"UNAUTHORIZED", false},
		{"FORBIDDEN", false},
		{"INTERNAL_ERROR", false},
		{"UNKNOWN_ERROR", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := isRetryableError(tt.code)
			assert.Equal(t, tt.retryable, result, "Code %s retryable should be %v", tt.code, tt.retryable)
		})
	}
}

func TestRequest_SetMetadata(t *testing.T) {
	t.Run("sets metadata on nil map", func(t *testing.T) {
		req := Request{}

		req.SetMetadata("key1", "value1")

		assert.NotNil(t, req.Metadata)
		assert.Equal(t, "value1", req.Metadata["key1"])
	})

	t.Run("sets metadata on existing map", func(t *testing.T) {
		req := Request{
			Metadata: map[string]string{
				"existing": "value",
			},
		}

		req.SetMetadata("new", "data")

		assert.Equal(t, "value", req.Metadata["existing"])
		assert.Equal(t, "data", req.Metadata["new"])
	})

	t.Run("overwrites existing metadata", func(t *testing.T) {
		req := Request{
			Metadata: map[string]string{
				"key": "old",
			},
		}

		req.SetMetadata("key", "new")

		assert.Equal(t, "new", req.Metadata["key"])
	})

	t.Run("sets multiple metadata values", func(t *testing.T) {
		req := Request{}

		req.SetMetadata("key1", "value1")
		req.SetMetadata("key2", "value2")
		req.SetMetadata("key3", "value3")

		assert.Len(t, req.Metadata, 3)
		assert.Equal(t, "value1", req.Metadata["key1"])
		assert.Equal(t, "value2", req.Metadata["key2"])
		assert.Equal(t, "value3", req.Metadata["key3"])
	})

	t.Run("handles empty strings", func(t *testing.T) {
		req := Request{}

		req.SetMetadata("", "value")
		req.SetMetadata("key", "")

		assert.Equal(t, "value", req.Metadata[""])
		assert.Equal(t, "", req.Metadata["key"])
	})
}

func TestRequest_GetMetadata(t *testing.T) {
	t.Run("gets existing metadata", func(t *testing.T) {
		req := Request{
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}

		val, ok := req.GetMetadata("key1")

		assert.True(t, ok)
		assert.Equal(t, "value1", val)
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		req := Request{
			Metadata: map[string]string{
				"key1": "value1",
			},
		}

		val, ok := req.GetMetadata("nonexistent")

		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("returns false for nil metadata", func(t *testing.T) {
		req := Request{}

		val, ok := req.GetMetadata("any")

		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("handles empty string key", func(t *testing.T) {
		req := Request{
			Metadata: map[string]string{
				"": "empty-key-value",
			},
		}

		val, ok := req.GetMetadata("")

		assert.True(t, ok)
		assert.Equal(t, "empty-key-value", val)
	})

	t.Run("handles empty string value", func(t *testing.T) {
		req := Request{
			Metadata: map[string]string{
				"key": "",
			},
		}

		val, ok := req.GetMetadata("key")

		assert.True(t, ok)
		assert.Empty(t, val)
	})
}

func TestRequest_CompleteScenario(t *testing.T) {
	t.Run("complete request-response flow", func(t *testing.T) {
		// Create request
		requestData := map[string]interface{}{
			"action": "download",
			"url":    "https://example.com/file.pdf",
		}

		req, err := NewRequest("download", requestData)
		require.NoError(t, err)

		// Add metadata
		req.SetMetadata("user_id", "user123")
		req.SetMetadata("priority", "high")

		// Verify request
		assert.NotEmpty(t, req.ID)
		assert.Equal(t, "download", req.Type)

		val, ok := req.GetMetadata("user_id")
		assert.True(t, ok)
		assert.Equal(t, "user123", val)

		// Unmarshal payload
		var payload map[string]interface{}
		err = req.Unmarshal(&payload)
		assert.NoError(t, err)
		assert.Equal(t, "download", payload["action"])
		assert.Equal(t, "https://example.com/file.pdf", payload["url"])

		// Create success response
		responseData := map[string]interface{}{
			"status":   "completed",
			"file_id":  "file123",
			"size":     1024,
			"checksum": "abc123",
		}

		resp, err := NewSuccessResponse(req.ID, responseData)
		require.NoError(t, err)

		// Verify response
		assert.Equal(t, req.ID, resp.ID)
		assert.True(t, resp.Success)
		assert.Nil(t, resp.Error)

		var respData map[string]interface{}
		err = json.Unmarshal(resp.Data, &respData)
		assert.NoError(t, err)
		assert.Equal(t, "completed", respData["status"])
		assert.Equal(t, "file123", respData["file_id"])
	})

	t.Run("complete error flow", func(t *testing.T) {
		// Create request
		req, err := NewRequest("process", map[string]string{"file": "test.pdf"})
		require.NoError(t, err)

		// Simulate processing error
		errorResp := NewErrorResponse(
			req.ID,
			"NETWORK_ERROR",
			"Failed to download file",
			"Connection timeout after 30 seconds",
		)

		// Verify error response
		assert.Equal(t, req.ID, errorResp.ID)
		assert.False(t, errorResp.Success)
		assert.NotNil(t, errorResp.Error)
		assert.True(t, errorResp.Error.Retryable)
		assert.Equal(t, "NETWORK_ERROR", errorResp.Error.Code)
		assert.Contains(t, errorResp.Error.Details, "timeout")
	})
}

func TestJSONSerialization(t *testing.T) {
	t.Run("request JSON roundtrip", func(t *testing.T) {
		original := Request{
			ID:      "test-123",
			Source:  "http",
			Type:    "process",
			Payload: json.RawMessage(`{"data": "test"}`),
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Timestamp: time.Now().UTC().Truncate(time.Second),
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var decoded Request
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Compare
		assert.Equal(t, original.ID, decoded.ID)
		assert.Equal(t, original.Source, decoded.Source)
		assert.Equal(t, original.Type, decoded.Type)
		assert.JSONEq(t, string(original.Payload), string(decoded.Payload))
		assert.Equal(t, original.Metadata, decoded.Metadata)
		assert.Equal(t, original.Timestamp.Unix(), decoded.Timestamp.Unix())
	})

	t.Run("response JSON roundtrip", func(t *testing.T) {
		original := Response{
			ID:      "resp-456",
			Success: true,
			Data:    json.RawMessage(`{"result": "ok"}`),
			Error:   nil,
			Metadata: map[string]string{
				"meta1": "value1",
			},
			ProcessedAt: time.Now().UTC().Truncate(time.Second),
			Duration:    5 * time.Second,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var decoded Response
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Compare
		assert.Equal(t, original.ID, decoded.ID)
		assert.Equal(t, original.Success, decoded.Success)
		assert.JSONEq(t, string(original.Data), string(decoded.Data))
		assert.Equal(t, original.Metadata, decoded.Metadata)
		assert.Equal(t, original.ProcessedAt.Unix(), decoded.ProcessedAt.Unix())
		assert.Equal(t, original.Duration, decoded.Duration)
	})

	t.Run("error response JSON roundtrip", func(t *testing.T) {
		original := NewErrorResponse(
			"err-789",
			"VALIDATION_ERROR",
			"Invalid input",
			"Field 'name' is required",
		)

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var decoded Response
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Compare
		assert.Equal(t, original.ID, decoded.ID)
		assert.Equal(t, original.Success, decoded.Success)
		assert.NotNil(t, decoded.Error)
		assert.Equal(t, original.Error.Code, decoded.Error.Code)
		assert.Equal(t, original.Error.Message, decoded.Error.Message)
		assert.Equal(t, original.Error.Details, decoded.Error.Details)
		assert.Equal(t, original.Error.Retryable, decoded.Error.Retryable)
	})
}
