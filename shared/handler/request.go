package handler

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Request represents a generic incoming request to a worker.
// It provides a platform-agnostic way to handle different input formats
// from various sources (HTTP, Lambda, OpenFaaS, etc.).
type Request struct {
	// ID is a unique identifier for the request (for tracing)
	ID string `json:"id"`

	// Source identifies where the request came from (http, lambda, openfaas, etc.)
	Source string `json:"source"`

	// Type identifies the type of request/event (e.g., "download", "process")
	Type string `json:"type"`

	// Payload contains the actual request data as raw JSON
	Payload json.RawMessage `json:"payload"`

	// Metadata contains additional context (headers, attributes, etc.)
	Metadata map[string]string `json:"metadata,omitempty"`

	// Timestamp when the request was created
	Timestamp time.Time `json:"timestamp"`
}

// Response represents a generic response from a worker.
// It provides a platform-agnostic way to return results that can be
// adapted to different output formats for various platforms.
type Response struct {
	// ID correlates with the request ID
	ID string `json:"id"`

	// Success indicates if processing was successful
	Success bool `json:"success"`

	// Data contains the response payload (only if Success is true)
	Data json.RawMessage `json:"data,omitempty"`

	// Error contains error information if Success is false
	Error *ErrorResponse `json:"error,omitempty"`

	// Metadata contains additional response context
	Metadata map[string]string `json:"metadata,omitempty"`

	// ProcessedAt timestamp
	ProcessedAt time.Time `json:"processed_at"`

	// Duration of processing (optional)
	Duration time.Duration `json:"duration,omitempty"`
}

// ErrorResponse represents structured error information.
// This provides consistent error reporting across all workers and platforms.
type ErrorResponse struct {
	// Code is a machine-readable error code (e.g., "VALIDATION_ERROR")
	Code string `json:"code"`

	// Message is a human-readable error message
	Message string `json:"message"`

	// Details provides additional error context (optional)
	Details string `json:"details,omitempty"`

	// Retryable indicates if the operation can be retried
	Retryable bool `json:"retryable,omitempty"`
}

// NewRequest creates a new request with generated ID and timestamp.
func NewRequest(requestType string, payload interface{}) (Request, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Request{}, err
	}

	return Request{
		ID:        uuid.New().String(),
		Type:      requestType,
		Payload:   payloadBytes,
		Metadata:  make(map[string]string),
		Timestamp: time.Now().UTC(),
	}, nil
}

// Unmarshal is a helper to unmarshal request payload into a specific type.
func (r *Request) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Payload, v)
}

// Marshal is a helper to marshal response data from a specific type.
func (r *Response) Marshal(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Data = data
	return nil
}

// NewErrorResponse creates an error response.
func NewErrorResponse(id string, code string, message string, details string) Response {
	return Response{
		ID:      id,
		Success: false,
		Error: &ErrorResponse{
			Code:      code,
			Message:   message,
			Details:   details,
			Retryable: isRetryableError(code),
		},
		ProcessedAt: time.Now().UTC(),
	}
}

// NewSuccessResponse creates a success response.
func NewSuccessResponse(id string, data interface{}) (Response, error) {
	resp := Response{
		ID:          id,
		Success:     true,
		ProcessedAt: time.Now().UTC(),
		Metadata:    make(map[string]string),
	}

	if data != nil {
		if err := resp.Marshal(data); err != nil {
			return Response{}, err
		}
	}

	return resp, nil
}

// isRetryableError determines if an error code represents a retryable error.
func isRetryableError(code string) bool {
	retryableCodes := map[string]bool{
		"TIMEOUT":             true,
		"NETWORK_ERROR":       true,
		"RATE_LIMITED":        true,
		"TEMPORARY_ERROR":     true,
		"SERVICE_UNAVAILABLE": true,
	}
	return retryableCodes[code]
}

// SetMetadata adds or updates metadata on the request.
func (r *Request) SetMetadata(key, value string) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]string)
	}
	r.Metadata[key] = value
}

// GetMetadata retrieves metadata from the request.
func (r *Request) GetMetadata(key string) (string, bool) {
	if r.Metadata == nil {
		return "", false
	}
	val, ok := r.Metadata[key]
	return val, ok
}
