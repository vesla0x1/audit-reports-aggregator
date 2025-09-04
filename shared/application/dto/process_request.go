package dto

import "time"

// ProcessRequest represents the message payload for requesting file processing
type ProcessRequest struct {
	// EventID is the unique identifier for this message (for idempotency)
	EventID string `json:"event_id"`

	// EventType identifies this as a process request
	EventType string `json:"event_type"`

	// ProcessID is the database ID of the process record to handle
	ProcessID int64 `json:"process_id"`

	// Timestamp when the message was created
	Timestamp time.Time `json:"timestamp"`
}
