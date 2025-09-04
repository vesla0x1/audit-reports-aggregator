package dto

import (
	"time"
)

// ExtractRequest represents the message payload for requesting extraction of audit reports
type ExtractRequest struct {
	// EventID is the unique identifier for this message (for idempotency)
	EventID string `json:"event_id"`

	// EventType identifies the extraction stage
	// Can be: "root", "intermediate", "leaf"
	EventType string `json:"event_type"`

	// URL is the page to extract from
	URL string `json:"url"`

	// Metadata carries data extracted in intermediate phases
	// Used to pass information from root/intermediate stages to leaf stage
	Metadata map[string]interface{} `json:"metadata"`

	// Timestamp when the message was created
	Timestamp time.Time `json:"timestamp"`
}

// GetMetadataString safely retrieves a string value from metadata
func (r *ExtractRequest) GetMetadataString(key string) string {
	if r.Metadata == nil {
		return ""
	}
	if val, ok := r.Metadata[key].(string); ok {
		return val
	}
	return ""
}

// GetMetadataInt64 safely retrieves an int64 value from metadata
func (r *ExtractRequest) GetMetadataInt64(key string) int64 {
	if r.Metadata == nil {
		return 0
	}
	switch v := r.Metadata[key].(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case int:
		return int64(v)
	default:
		return 0
	}
}
