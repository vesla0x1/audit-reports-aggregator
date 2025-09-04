package ports

import (
	"context"
	"encoding/json"
	"time"
)

type RuntimeRequest struct {
	ID        string            `json:"id"`
	Source    string            `json:"source"`
	Type      string            `json:"type"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
}

func (r *RuntimeRequest) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Payload, v)
}

type RuntimeResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Handler processes requests through middleware chains
type Handler interface {
	Handle(ctx context.Context, req RuntimeRequest) (RuntimeResponse, error)
}

// Adapter interface for platform-specific adapters
type Runtime interface {
	Start() error
}
