package handler

import (
	"encoding/json"
	"time"
)

type Request struct {
	ID        string            `json:"id"`
	Source    string            `json:"source"`
	Type      string            `json:"type"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
}

func (r *Request) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Payload, v)
}

type Response struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *ErrorInfo      `json:"error,omitempty"`
}

type ErrorInfo struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// Helper functions for creating responses
func NewSuccessResponse(requestID string, data interface{}) (Response, error) {
	marshaled, err := json.Marshal(data)
	if err != nil {
		return Response{}, err
	}

	return Response{
		Success: true,
		Data:    marshaled,
	}, nil
}

func NewErrorResponse(code, message string, retryable bool) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      code,
			Message:   message,
			Retryable: retryable,
		},
	}
}
